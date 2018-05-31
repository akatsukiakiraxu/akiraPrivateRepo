package file

import (
	"encoding/binary"
	"fmt"
	"io"
)

type WaveWriter struct {
	channels      map[string]ChannelDefinition
	writer        io.Writer
	sampleIndices []uint32
}

func serializeChannelDefinition(writer io.Writer, channel *ChannelDefinition) error {
	if err := binary.Write(writer, binary.LittleEndian, uint16(channel.DataType)); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.LittleEndian, uint16(len(channel.Name))); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.LittleEndian, []byte(channel.Name)); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.LittleEndian, channel.SamplingRate); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.LittleEndian, channel.Range.RangeMin); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.LittleEndian, channel.Range.RangeMax); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.LittleEndian, channel.Range.ValueMin); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.LittleEndian, channel.Range.ValueMax); err != nil {
		return err
	}
	return nil
}
func sizeOfChannelDefinition(channel *ChannelDefinition) int {
	return 2 + 2 + len(channel.Name) + 4 + 8 + 8 + binary.Size(channel.Range.ValueMin) + binary.Size(channel.Range.ValueMax)
}

func putBlockHeader(writer io.Writer, blockType uint32, blockLength uint32) error {
	if err := binary.Write(writer, binary.LittleEndian, blockType); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.LittleEndian, blockLength); err != nil {
		return err
	}
	return nil
}

func getSampleDataTypeAndNumSamples(data interface{}) (dataType DataType, numSamples uint32, err error) {
	err = nil
	dataType = 0
	numSamples = 0
	switch v := data.(type) {
	case []uint16:
		dataType = DataType_Uint16
		numSamples = uint32(len(v))
	case []int16:
		dataType = DataType_Int16
		numSamples = uint32(len(v))
	case []float32:
		dataType = DataType_Float32
		numSamples = uint32(len(v))
	case []float64:
		dataType = DataType_Float64
		numSamples = uint32(len(v))
	default:
		err = fmt.Errorf("unsupported format")
	}
	return
}
func NewWaveWriter(writer io.Writer, channels []ChannelDefinition) (*WaveWriter, error) {
	channelMap := make(map[string]ChannelDefinition, len(channels))
	channelDefLength := 4
	for _, channel := range channels {
		channelMap[channel.Name] = channel
		channelDefLength += sizeOfChannelDefinition(&channel)
	}

	// Write-out header
	header := FileHeader{
		Magic:         HeaderMagic,
		FormatVersion: 1,
		Flags:         0,
		DataLength:    0,
	}
	if err := binary.Write(writer, binary.LittleEndian, &header); err != nil {
		return nil, err
	}

	// Write-out channel definitions
	if err := putBlockHeader(writer, BlockType_Channel, uint32(channelDefLength)); err != nil {
		return nil, err
	}
	if err := binary.Write(writer, binary.LittleEndian, uint32(len(channels))); err != nil {
		return nil, err
	}
	for _, channel := range channels {
		if err := serializeChannelDefinition(writer, &channel); err != nil {
			return nil, err
		}
	}

	return &WaveWriter{
		channels:      channelMap,
		writer:        writer,
		sampleIndices: make([]uint32, len(channels)),
	}, nil
}

func (w *WaveWriter) PutFrameData(frameData *FrameData) error {
	length := 0x20 + frameData.MetadataLength
	if err := putBlockHeader(w.writer, BlockType_FrameData, length); err != nil {
		return err
	}
	if err := binary.Write(w.writer, binary.LittleEndian, frameData.FrameId); err != nil {
		return err
	}
	if err := binary.Write(w.writer, binary.LittleEndian, frameData.TimeStamp); err != nil {
		return err
	}
	if err := binary.Write(w.writer, binary.LittleEndian, frameData.MetadataType); err != nil {
		return err
	}
	if err := binary.Write(w.writer, binary.LittleEndian, frameData.MetadataLength); err != nil {
		return err
	}
	if frameData.MetadataLength > 0 {
		if err := binary.Write(w.writer, binary.LittleEndian, frameData.Metadata); err != nil {
			return err
		}
	}
	return nil
}

func (w *WaveWriter) PutWaveformData(channelIndex int, data interface{}) error {
	if channelIndex >= len(w.channels) {
		return fmt.Errorf("channelIndex out of range (%d >= %d)", channelIndex, len(w.channels))
	}
	dataType, numSamples, err := getSampleDataTypeAndNumSamples(data)
	if err != nil {
		return err
	}
	length := 0x0c + uint32(dataType.Size())*numSamples
	if err := putBlockHeader(w.writer, BlockType_WaveData, length); err != nil {
		return err
	}

	if err := binary.Write(w.writer, binary.LittleEndian, uint32(channelIndex)); err != nil {
		return err
	}
	if err := binary.Write(w.writer, binary.LittleEndian, uint32(numSamples)); err != nil {
		return err
	}
	if err := binary.Write(w.writer, binary.LittleEndian, w.sampleIndices[channelIndex]); err != nil {
		return err
	}
	w.sampleIndices[channelIndex] += numSamples
	if err := binary.Write(w.writer, binary.LittleEndian, data); err != nil {
		return err
	}
	return nil
}
