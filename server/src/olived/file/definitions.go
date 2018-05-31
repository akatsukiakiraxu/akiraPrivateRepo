package file

import (
	"fmt"

	"github.com/google/uuid"
)

const (
	HeaderMagic uint32 = 0x64766c4f
)

type ValueRange struct {
	RangeMin float64     `json:"range_min"`
	RangeMax float64     `json:"range_max"`
	ValueMin interface{} `json:"value_min"`
	ValueMax interface{} `json:"value_max"`
}

type DataType uint16

const (
	DataType_Uint16  DataType = 0x0000
	DataType_Int16   DataType = 0x0001
	DataType_Float32 DataType = 0x0002
	DataType_Float64 DataType = 0x0003
)

const (
	BlockType_Channel   uint32 = 0x00000000
	BlockType_WaveData  uint32 = 0x00000001
	BlockType_FrameData uint32 = 0x00000002
)

func (d DataType) String() string {
	switch d {
	case DataType_Uint16:
		return "uint16"
	case DataType_Int16:
		return "int16"
	case DataType_Float32:
		return "float32"
	case DataType_Float64:
		return "float64"
	default:
		return fmt.Sprintf("unknown(0x%04x)", uint16(d))
	}
}
func (d DataType) Size() int {
	switch d {
	case DataType_Uint16:
		return 2
	case DataType_Int16:
		return 2
	case DataType_Float32:
		return 4
	case DataType_Float64:
		return 8
	default:
		panic(fmt.Sprintf("Unknown data type - %x", uint16(d)))
	}
}
func (d DataType) MakeBuffer(count int) interface{} {
	switch d {
	case DataType_Uint16:
		return make([]uint16, count)
	case DataType_Int16:
		return make([]int16, count)
	case DataType_Float32:
		return make([]float32, count)
	case DataType_Float64:
		return make([]float64, count)
	default:
		panic(fmt.Sprintf("Unknown data type - %x", uint16(d)))
	}
}

type MetadataType uint32

const (
	MetadataType_Binary MetadataType = 0x0000
	MetadataType_JSON   MetadataType = 0x0001
)

func (m MetadataType) String() string {
	switch m {
	case MetadataType_JSON:
		return "json"
	default:
		return fmt.Sprintf("unknown(0x%08x)", uint32(m))
	}
}

type ChannelDefinition struct {
	DataType     DataType
	Name         string
	SamplingRate float32
	Range        ValueRange
}

type FrameData struct {
	FrameId        uuid.UUID
	TimeStamp      int64
	MetadataType   MetadataType
	MetadataLength uint32
	Metadata       []byte
}

type WaveformDataHeader struct {
	ChannelIndex uint32
	NumSamples   uint32
	SampleOffset uint32
}
type WaveformData struct {
	WaveformDataHeader
	Samples []byte
}

type FileHeader struct {
	Magic         uint32
	FormatVersion uint32
	Flags         uint32
	DataLength    uint32
}
