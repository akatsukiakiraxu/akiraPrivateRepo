package recording

import (
	"encoding/binary"
	"io"
	"os"
	"sync"

	acq "olived/acquisition"
)

type simpleRecorder struct {
	path string

	writerLock sync.RWMutex
	rawWriter  io.ReadWriteCloser
	fftWriter  io.ReadWriteCloser
}

func NewSimpleRecorder(path string) *simpleRecorder {
	return &simpleRecorder{
		path:       path,
		writerLock: sync.RWMutex{},
	}
}

func (r *simpleRecorder) Type() acq.ProcessorType {
	return acq.Recorder
}
func (r *simpleRecorder) Started(unit acq.Unit) {}
func (r *simpleRecorder) Stopped(unit acq.Unit) {}

func (r *simpleRecorder) SettingsChanging(unit acq.Unit, settings *acq.UnitSettings) error {
	return nil
}
func (r *simpleRecorder) SettingsChanged(unit acq.Unit, settings *acq.UnitSettings) {
}
func (r *simpleRecorder) DataArrived(unit acq.Unit, data acq.AcquiredData) {
	var writer io.ReadWriteCloser
	switch data.Type() {
	case acq.TimeSeries:
		writer = r.rawWriter
	case acq.FFT:
		writer = r.fftWriter
	}
	if writer == nil {
		return
	}

	r.writerLock.RLock()
	defer r.writerLock.RUnlock()

	raw := []byte(data.RawData())
	length := len(raw)
	binary.Write(writer, binary.LittleEndian, uint32(length))
	totalBytesWritten := 0
	for totalBytesWritten < length {
		bytesWritten, err := writer.Write(raw[totalBytesWritten:])
		if err != nil {
			break
		}
		totalBytesWritten += bytesWritten
	}
}

func (r *simpleRecorder) Start() error {
	var err error
	r.writerLock.Lock()
	defer r.writerLock.Unlock()
	r.fftWriter, err = os.OpenFile(r.path+".fft.bin", os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(0644))
	if err != nil {
		return err
	}
	r.rawWriter, err = os.OpenFile(r.path+".raw.bin", os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(0644))
	if err != nil {
		return err
	}
	return nil
}

func (r *simpleRecorder) Stop() {
	r.writerLock.Lock()
	defer r.writerLock.Unlock()
	if r.fftWriter != nil {
		r.fftWriter.Close()
		r.fftWriter = nil
	}
	if r.rawWriter != nil {
		r.rawWriter.Close()
		r.rawWriter = nil
	}
}
