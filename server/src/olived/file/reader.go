package file

import (
	"encoding/binary"
	"fmt"
	"io"
)

type waveDataBlockIndex struct {
	FileOffset   int64
	NumSamples   uint32
	SampleOffset uint32
}
type WaveReader struct {
	channels        []ChannelDefinition
	reader          io.ReadSeeker
	waveDataBlocks  [][]waveDataBlockIndex
	totalNumSamples []uint64
	fileOffset      int64
	frameData       *FrameData
}

func readDataType(reader io.Reader, dataType DataType) (value interface{}, err error) {
	err = nil
	value = nil
	switch dataType {
	case DataType_Uint16:
		var typed uint16
		err = binary.Read(reader, binary.LittleEndian, &typed)
		value = typed
	case DataType_Int16:
		var typed int16
		err = binary.Read(reader, binary.LittleEndian, &typed)
		value = typed
	case DataType_Float32:
		var typed float32
		err = binary.Read(reader, binary.LittleEndian, &typed)
		value = typed
	case DataType_Float64:
		var typed float64
		err = binary.Read(reader, binary.LittleEndian, &typed)
		value = typed
	default:
		err = fmt.Errorf("unsupported format")
	}

	return
}

func min_int(a int, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}
func limitSampleLength(samples interface{}, limit int) (int, interface{}) {
	switch v := samples.(type) {
	case []uint16:
		l := min_int(len(v), limit)
		return l, v[:l]
	case []int16:
		l := min_int(len(v), limit)
		return l, v[:l]
	case []float32:
		l := min_int(len(v), limit)
		return l, v[:l]
	case []float64:
		l := min_int(len(v), limit)
		return l, v[:l]
	default:
		panic(fmt.Errorf("unsupported sample format"))
	}
}

func deserializeChannelDefinition(reader io.Reader) (*ChannelDefinition, error) {
	c := ChannelDefinition{}
	if err := binary.Read(reader, binary.LittleEndian, &c.DataType); err != nil {
		return nil, err
	}
	var channelNameLength uint16
	if err := binary.Read(reader, binary.LittleEndian, &channelNameLength); err != nil {
		return nil, err
	}
	channelName := make([]byte, channelNameLength)
	if err := binary.Read(reader, binary.LittleEndian, channelName); err != nil {
		return nil, err
	}
	c.Name = string(channelName)
	if err := binary.Read(reader, binary.LittleEndian, &c.SamplingRate); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &c.Range.RangeMin); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &c.Range.RangeMax); err != nil {
		return nil, err
	}
	var err error
	if c.Range.ValueMin, err = readDataType(reader, c.DataType); err != nil {
		return nil, err
	}
	if c.Range.ValueMax, err = readDataType(reader, c.DataType); err != nil {
		return nil, err
	}

	return &c, nil
}

func readBlockHeader(reader io.Reader) (blockType uint32, blockLength uint32, err error) {
	blockType = 0
	blockLength = 0
	if err = binary.Read(reader, binary.LittleEndian, &blockType); err != nil {
		return
	}
	if err = binary.Read(reader, binary.LittleEndian, &blockLength); err != nil {
		return
	}
	return
}

func NewWaveReader(reader io.ReadSeeker) (*WaveReader, error) {
	var err error

	waveReader := WaveReader{
		reader:         reader,
		channels:       nil,
		waveDataBlocks: nil,
		fileOffset:     0,
		frameData:      nil,
	}
	waveReader.fileOffset, err = reader.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	// Read file header
	fileHeader := FileHeader{}
	if err := binary.Read(reader, binary.LittleEndian, &fileHeader); err != nil {
		return nil, err
	}
	if fileHeader.Magic != HeaderMagic {
		return nil, fmt.Errorf("invalid header magic - %08x", fileHeader.Magic)
	}
	if fileHeader.FormatVersion != 1 {
		return nil, fmt.Errorf("invalid format version - %d", fileHeader.FormatVersion)
	}

	// Read data blocks
	for {
		blockType, blockLength, err := readBlockHeader(reader)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("read error - %s", err.Error())
		}
		switch blockType {
		case BlockType_Channel:
			if blockLength < 4 {
				return nil, fmt.Errorf("channel block length must be greator or equal than 4")
			}
			var numChannels uint32
			if err = binary.Read(reader, binary.LittleEndian, &numChannels); err != nil {
				return nil, fmt.Errorf("cannot read number of channels - %s", err.Error())
			}
			waveReader.channels = make([]ChannelDefinition, numChannels)
			waveReader.waveDataBlocks = make([][]waveDataBlockIndex, numChannels)
			waveReader.totalNumSamples = make([]uint64, numChannels)
			for i := uint32(0); i < numChannels; i++ {
				var ch *ChannelDefinition
				if ch, err = deserializeChannelDefinition(reader); err != nil {
					return nil, fmt.Errorf("failed to read channel definition - %s", err.Error())
				}
				waveReader.channels[i] = *ch
				waveReader.waveDataBlocks[i] = make([]waveDataBlockIndex, 0)
			}

		case BlockType_WaveData:
			if waveReader.channels == nil {
				return nil, fmt.Errorf("WaveData block appears before Channel block")
			}
			if blockLength < 12 {
				return nil, fmt.Errorf("WaveData block length must be greator or equal than 12")
			}
			var channelIndex uint32
			if err = binary.Read(reader, binary.LittleEndian, &channelIndex); err != nil {
				return nil, fmt.Errorf("Failed to read channel index in wave data - %s", err.Error())
			}
			if channelIndex >= uint32(len(waveReader.waveDataBlocks)) {
				return nil, fmt.Errorf("Invalid channel index. expected < %d, got %d", len(waveReader.waveDataBlocks), channelIndex)
			}
			var numSamples uint32
			if err = binary.Read(reader, binary.LittleEndian, &numSamples); err != nil {
				return nil, fmt.Errorf("Failed to read number of samples in wave data - %s", err.Error())
			}
			var expectedBlockLength = numSamples*uint32(waveReader.channels[channelIndex].DataType.Size()) + 12
			if expectedBlockLength != blockLength {
				return nil, fmt.Errorf("Invalid block length. expected=%d, actual=%d", expectedBlockLength, blockLength)
			}
			var sampleOffset uint32
			if err = binary.Read(reader, binary.LittleEndian, &sampleOffset); err != nil {
				return nil, fmt.Errorf("Failed to read sample offset in wave data - %s", err.Error())
			}
			var fileOffset int64
			if fileOffset, err = reader.Seek(0, io.SeekCurrent); err != nil {
				return nil, fmt.Errorf("Failed to get current position in file - %s", err.Error())
			}
			sampleIndex := waveDataBlockIndex{
				FileOffset:   fileOffset,
				NumSamples:   numSamples,
				SampleOffset: sampleOffset,
			}
			waveReader.waveDataBlocks[channelIndex] = append(waveReader.waveDataBlocks[channelIndex], sampleIndex)
			waveReader.totalNumSamples[channelIndex] += uint64(numSamples)
			if _, err = reader.Seek(int64(blockLength-12), io.SeekCurrent); err != nil {
				return nil, fmt.Errorf("Failed to seek to next block - %s", err.Error())
			}
		case BlockType_FrameData:
			if blockLength < 32 {
				return nil, fmt.Errorf("FrameData block length must be greator or equal than 32")
			}
			frameData := FrameData{}
			if err = binary.Read(reader, binary.LittleEndian, &frameData.FrameId); err != nil {
				return nil, fmt.Errorf("Failed to read frame ID - %s", err.Error())
			}
			if err = binary.Read(reader, binary.LittleEndian, &frameData.TimeStamp); err != nil {
				return nil, fmt.Errorf("Failed to read timestamp - %s", err.Error())
			}
			if err = binary.Read(reader, binary.LittleEndian, &frameData.MetadataType); err != nil {
				return nil, fmt.Errorf("Failed to read metadata type - %s", err.Error())
			}
			if err = binary.Read(reader, binary.LittleEndian, &frameData.MetadataLength); err != nil {
				return nil, fmt.Errorf("Failed to read metadata length - %s", err.Error())
			}
			if frameData.MetadataLength+32 != blockLength {
				return nil, fmt.Errorf("Invalid block length. expected=%d, actual=%d", frameData.MetadataLength+32, blockLength)
			}
			frameData.Metadata = make([]byte, frameData.MetadataLength)
			if err = binary.Read(reader, binary.LittleEndian, frameData.Metadata); err != nil {
				return nil, fmt.Errorf("Failed to read metadata - %s", err.Error())
			}
			waveReader.frameData = &frameData
		default:
			// Skip unknown block
			if _, err = reader.Seek(int64(blockLength), io.SeekCurrent); err != nil {
				return nil, fmt.Errorf("Seek error - %s", err.Error())
			}
		}
	}

	return &waveReader, nil
}

func (r *WaveReader) FrameData() *FrameData {
	return r.frameData
}

func (r *WaveReader) Channels() []ChannelDefinition {
	return r.channels
}

type waveSampleReader struct {
	reader          *WaveReader
	channel         int
	currentBlock    int
	currentPosition uint32
	currentIndex    uint64
}

type SampleReader interface {
	io.Reader
	ReadSample(interface{}) (uint32, error)
}

// NewSampleReader - Creates new SampleReader to read sample data in a channel.
// Note that you cannot use multiple SampleReader from multiple threads.
// Because reading sample data with this reader changes the state (position in the file) of the underlying ReadSeeker object.
func (r *WaveReader) NewSampleReader(channel int) (SampleReader, error) {
	if channel >= len(r.channels) {
		return nil, fmt.Errorf("Invalid channel %d", channel)
	}

	return &waveSampleReader{
		reader:  r,
		channel: channel,
	}, nil
}

func (r *waveSampleReader) Read(buffer []byte) (int, error) {
	totalBytesToRead := uint32(len(buffer))
	bytesRemainingToRead := totalBytesToRead
	dataType := r.reader.channels[r.channel].DataType
	dataTypeSize := uint32(dataType.Size())
	if bytesRemainingToRead%dataTypeSize != 0 {
		return 0, fmt.Errorf("Bytes to read must be aligned with size of sample")
	}
	dataBlocks := r.reader.waveDataBlocks[r.channel]
	for bytesRemainingToRead > 0 && r.currentBlock < len(dataBlocks) {
		block := &dataBlocks[r.currentBlock]
		bytesToRead := block.NumSamples*dataTypeSize - r.currentPosition
		if bytesToRead > bytesRemainingToRead {
			bytesToRead = bytesRemainingToRead
		}
		if _, err := r.reader.reader.Seek(block.FileOffset, io.SeekStart); err != nil {
			return 0, err
		}
		if _, err := io.ReadFull(r.reader.reader, buffer[:bytesToRead]); err != nil {
			return 0, err
		}

		buffer = buffer[bytesToRead:]
		bytesRemainingToRead -= bytesToRead

		if bytesRemainingToRead == 0 {
			r.currentPosition = r.currentPosition + bytesToRead
		} else {
			r.currentPosition = 0
			r.currentBlock++
		}
	}

	r.currentIndex += uint64(totalBytesToRead - bytesRemainingToRead)
	return int(totalBytesToRead - bytesRemainingToRead), nil
}

func (r *waveSampleReader) ReadSample(buffer interface{}) (uint32, error) {
	samples := r.reader.totalNumSamples[r.channel]
	remaining := int(samples - r.currentIndex)
	limited, slice := limitSampleLength(buffer, remaining)
	if err := binary.Read(r, binary.LittleEndian, slice); err != nil {
		return 0, err
	}
	return uint32(limited), nil
}
