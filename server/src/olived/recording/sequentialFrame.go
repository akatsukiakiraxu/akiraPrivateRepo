package recording

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
	acq "olived/acquisition"
	"os"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"github.com/google/uuid"
)

type recorderWriteSeekCloser interface {
	io.Writer
	io.Seeker
	io.Closer
}

type fdSet struct {
	path        string
	fd          recorderWriteSeekCloser
	bufWriter   *bufio.Writer
	writtenSize uint64
}

type sequentialFrameRecorder struct {
	directory     string
	started       bool
	writeFileMap  map[string]fdSet
	writerLock    sync.RWMutex
	counter       uint32
	valueRangeMap map[string]acq.ValueRange
	chSettingsMap map[string]acq.ChannelSettings

	pendingFrames     map[uuid.UUID][]fdSet
	pendingFramesLock sync.Mutex

	controller RecordingController
}

func NewSequentialFrameRecorder(directory string, unit acq.Unit, controller RecordingController) *sequentialFrameRecorder {
	return &sequentialFrameRecorder{
		directory:     directory,
		writerLock:    sync.RWMutex{},
		counter:       0,
		writeFileMap:  nil,
		valueRangeMap: unit.Config().Ranges,
		chSettingsMap: unit.Settings().Channels,
		pendingFrames: make(map[uuid.UUID][]fdSet),
		controller:    controller,
	}
}

func (r *sequentialFrameRecorder) Type() acq.ProcessorType {
	return acq.Recorder
}
func (r *sequentialFrameRecorder) Start() {
	var err error
	r.writerLock.Lock()
	defer r.writerLock.Unlock()
	if _, err = os.Stat(r.directory); os.IsNotExist(err) {
		if err = os.Mkdir(r.directory, 0777); err != nil {
			log.Fatalf("Mkdir:", err)
		}
	}
	r.started = true
}
func (r *sequentialFrameRecorder) Stop() {
	r.writerLock.Lock()
	defer r.writerLock.Unlock()
	r.started = false
}
func (r *sequentialFrameRecorder) Started(unit acq.Unit) {}
func (r *sequentialFrameRecorder) Stopped(unit acq.Unit) {}

func (r *sequentialFrameRecorder) SettingsChanging(unit acq.Unit, settings *acq.UnitSettings) error {
	return nil
}
func (r *sequentialFrameRecorder) SettingsChanged(unit acq.Unit, settings *acq.UnitSettings) {

}
func (r *sequentialFrameRecorder) DataArrived(unit acq.Unit, data acq.AcquiredData) {
	if r.started {
		if data.Type() == acq.TimeSeries {
			var err error
			parsedData := data.Parse()
			const MaxUint = float64(^uint16(0))
			const sizeofUint16 = uint64(unsafe.Sizeof(uint16(0)))
			const sizePosition = 8
			// check if need to close current file
			if (data.IsFrameFirstData() && r.writeFileMap != nil) || (data.IsFrameLastData() && r.writeFileMap != nil) {
				for _, fdSet := range r.writeFileMap {
					if fdSet.writtenSize > 0 && fdSet.fd != nil && fdSet.bufWriter != nil {
						r.writerLock.RLock()
						defer r.writerLock.RUnlock()
						// finally, write the bit which is means the size of the data, and close
						fdSet.bufWriter.Flush()
						bs := make([]byte, int(unsafe.Sizeof(uint64(0))))
						fdSet.fd.Seek(sizePosition, io.SeekStart)
						binary.LittleEndian.PutUint64(bs, fdSet.writtenSize)
						if _, err := fdSet.fd.Write(bs); err != nil {
							log.Fatalf("file WriteAt err!", err)
						}
						fdSet.fd.Close()
					}
				}
				result := r.controller.FrameRecorded(r, acq.GetFrameData(data))
				switch result {
				case FilterResult_Pending:
					// Pending to commit this file.
					fds := make([]fdSet, len(r.writeFileMap))
					i := 0
					for _, fdSet := range r.writeFileMap {
						fds[i] = fdSet
						i++
					}
					r.pendingFramesLock.Lock()
					r.pendingFrames[data.FrameId()] = fds
					r.pendingFramesLock.Unlock()

				case FilterResult_Rejected:
					// This file is rejected. remove all files related to this frame.
					log.Printf("Frame %v is rejected.", data.FrameId())
					for _, fdSet := range r.writeFileMap {
						if err := os.Remove(fdSet.path); err != nil {
							log.Printf("Error: failed to remove file %s", fdSet.path)
						}
					}
				}
				// when finished the file , drop map and increment counter
				r.writeFileMap = nil
				r.counter++
			}
			// write data only when the data is exist
			if len(parsedData) > 0 {
				r.writerLock.RLock()
				defer r.writerLock.RUnlock()
				isCreateFile := false
				if r.writeFileMap == nil {
					if data.IsFrameFirstData() {
						r.writeFileMap = make(map[string]fdSet)
						isCreateFile = true
					}
				}
				for _, chData := range parsedData {
					if r.chSettingsMap[chData.Channel()].Enabled != true {
						continue
					}
					valueRangeMin := r.valueRangeMap[r.chSettingsMap[chData.Channel()].Range].MinimumValue
					valueRangeMax := r.valueRangeMap[r.chSettingsMap[chData.Channel()].Range].MaximumValue
					fileDescriptorSet := fdSet{"", nil, nil, 0}
					// if not exist the file descriptor and trigger come, create the file name. In this time, should write the meta data
					if isCreateFile {
						channelNum, _ := strconv.ParseUint(strings.Replace(chData.Channel(), "ch", "0", 1), 10, 16)
						fileName := fmt.Sprintf("%s/CH%02d_%05d.bin", r.directory, channelNum-1, r.counter)
						fileDescriptorSet.path = fileName
						fileDescriptorSet.fd, err = os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
						if err != nil {
							log.Fatalf("file open err!", err)
						}
						fileDescriptorSet.bufWriter = bufio.NewWriter(fileDescriptorSet.fd)
						r.writeFileMap[chData.Channel()] = fileDescriptorSet
						metadata := data.FrameMetadata()
						var header = []interface{}{
							uint32(metadata["magicNumber"].(uint32)),
							uint32(metadata["headerSize"].(uint32)),
							uint64(0),
							uint32(metadata["startTime"].(uint32)),
							uint32(metadata["flag"].(uint32)),
							uint32(channelNum - 1),
							uint32(metadata["code"].(uint32)),
							uint32(metadata["palette"].(uint32)),
							uint32(metadata["pos"].(uint32)),
							uint32(metadata["dataErrCnt"].(uint32)),
							uint32(metadata["fftErrCnt"].(uint32)),
							float32(valueRangeMin),
							float32(valueRangeMax),
						}
						for _, v := range header {
							binary.Write(r.writeFileMap[chData.Channel()].fd, binary.LittleEndian, v)
						}
					}
					if r.writeFileMap != nil {
						if r.writeFileMap[chData.Channel()].bufWriter != nil {
							var v float64
							dataPerChannel := make([]byte, chData.Length())
							_, err = chData.Read(dataPerChannel)
							sizePerRawData := int(unsafe.Sizeof(v))
							dataCount := chData.Length() / uint64(sizePerRawData)
							valueRangeAbs := valueRangeMax - valueRangeMin
							coefficient := MaxUint / valueRangeAbs
							deviation := valueRangeMin * coefficient
							for readPos := 0; readPos < len(dataPerChannel); readPos += sizePerRawData {
								v = math.Float64frombits(binary.LittleEndian.Uint64(dataPerChannel[readPos : readPos+sizePerRawData]))
								bs := make([]byte, sizeofUint16)
								binary.LittleEndian.PutUint16(bs, uint16(v*coefficient-deviation))
								r.writeFileMap[chData.Channel()].bufWriter.Write(bs)
							}
							fileDescriptorSet = r.writeFileMap[chData.Channel()]
							fileDescriptorSet.writtenSize += uint64(dataCount * sizeofUint16)
							r.writeFileMap[chData.Channel()] = fileDescriptorSet
						}
					}
				}
			}
		}
	}
}

func (r *sequentialFrameRecorder) AddMetadata(frameID uuid.UUID, key string, value interface{}) error {
	// Not supported.
	return nil
}
func (r *sequentialFrameRecorder) PendingFrameConfirmed(frameID uuid.UUID, result FilterResult) {
	r.pendingFramesLock.Lock()
	defer r.pendingFramesLock.Unlock()
	if frame, ok := r.pendingFrames[frameID]; ok {
		if result == FilterResult_Rejected {
			// This frame was rejected. remove related files.
			for _, fdSet := range frame {
				if err := os.Remove(fdSet.path); err != nil {
					log.Printf("Error: failed to remove file %s, %v", fdSet.path, err)
				}
			}
		}
		// Remove entry from pendingFrames
		delete(r.pendingFrames, frameID)
	}
}
