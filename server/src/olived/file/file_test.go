package file

import (
	"bytes"
	"math/rand"
	"reflect"
	"testing"

	"github.com/google/uuid"
)

var singleChannel = []ChannelDefinition{
	ChannelDefinition{
		DataType:     DataType_Uint16,
		Name:         "Channel1",
		SamplingRate: 250e3,
		Range: ValueRange{
			RangeMin: -10.0,
			RangeMax: 10.0,
			ValueMin: uint16(0),
			ValueMax: uint16(32767),
		},
	},
}
var dualChannel = []ChannelDefinition{
	ChannelDefinition{
		DataType:     DataType_Uint16,
		Name:         "Channel1",
		SamplingRate: 250e3,
		Range: ValueRange{
			RangeMin: -10.0,
			RangeMax: 10.0,
			ValueMin: uint16(0),
			ValueMax: uint16(32767),
		},
	},
	ChannelDefinition{
		DataType:     DataType_Float32,
		Name:         "Channel2",
		SamplingRate: 250e3,
		Range: ValueRange{
			RangeMin: -10.0,
			RangeMax: 10.0,
			ValueMin: float32(-10.0),
			ValueMax: float32(10.0),
		},
	},
}
var multiChannel = []ChannelDefinition{
	ChannelDefinition{
		DataType:     DataType_Uint16,
		Name:         "Channel1",
		SamplingRate: 250e3,
		Range: ValueRange{
			RangeMin: -10.0,
			RangeMax: 10.0,
			ValueMin: uint16(0),
			ValueMax: uint16(32767),
		},
	},
	ChannelDefinition{
		DataType:     DataType_Float32,
		Name:         "Channel2",
		SamplingRate: 100e3,
		Range: ValueRange{
			RangeMin: -10.0,
			RangeMax: 10.0,
			ValueMin: float32(-10.0),
			ValueMax: float32(10.0),
		},
	},
	ChannelDefinition{
		DataType:     DataType_Float64,
		Name:         "Channel3",
		SamplingRate: 50e3,
		Range: ValueRange{
			RangeMin: -20.0,
			RangeMax: 20.0,
			ValueMin: float64(-30.0),
			ValueMax: float64(30.0),
		},
	},
}

var frameDataNoMetadata = FrameData{
	FrameId:        uuid.Must(uuid.Parse("01234567-89ab-cdef-a53b-deadbeefcafe")),
	TimeStamp:      0x324567abdd331386,
	MetadataType:   MetadataType_Binary,
	MetadataLength: 0,
	Metadata:       make([]byte, 0),
}

var jsonMetadata = []byte("{\"lot_id\"=31296537,\"model_name\"=\"P3192-65523\"}")
var frameDataSimple = FrameData{
	FrameId:        uuid.Must(uuid.Parse("01234567-89ab-cdef-a53b-deadbeefcafe")),
	TimeStamp:      0x324567abdd331386,
	MetadataType:   MetadataType_JSON,
	MetadataLength: uint32(len(jsonMetadata)),
	Metadata:       jsonMetadata,
}

func generateUint16Data(length int) []uint16 {
	data := make([]uint16, length)
	for i := 0; i < length; i++ {
		data[i] = uint16(rand.Int())
	}
	return data
}
func generateFloat32Data(length int) []float32 {
	data := make([]float32, length)
	for i := 0; i < length; i++ {
		data[i] = rand.Float32()
	}
	return data
}

func TestReadWriteNoChannels(t *testing.T) {
	file := &bytes.Buffer{}
	var channels [0]ChannelDefinition
	_, err := NewWaveWriter(file, channels[:])
	if err != nil {
		t.Fatalf("failed to create wave writer %#v", err)
	}

	reader := bytes.NewReader(file.Bytes())
	_, err = NewWaveReader(reader)
	if err != nil {
		t.Fatalf("failed to create wave reader %#v", err)
	}
}

func TestReadWriteNoData(t *testing.T) {
	file := &bytes.Buffer{}
	channels := singleChannel
	_, err := NewWaveWriter(file, channels)
	if err != nil {
		t.Fatalf("failed to create wave writer %#v", err)
	}

	reader := bytes.NewReader(file.Bytes())
	r, err := NewWaveReader(reader)
	if err != nil {
		t.Fatalf("failed to create wave reader %#v", err)
	}
	chs := r.Channels()
	if len(chs) != len(channels) {
		t.Fatalf("channel count, expected=%d, actual=%d", len(channels), len(chs))
	}
	for i, channel := range channels {
		if !reflect.DeepEqual(channel, chs[i]) {
			t.Fatalf("Channel data mismatch.\nExpected:%+v\nActual:%+v", channel, chs[i])
		}
	}
}

func TestReadWriteMultiChannelNoData(t *testing.T) {
	file := &bytes.Buffer{}
	channels := multiChannel
	_, err := NewWaveWriter(file, channels)
	if err != nil {
		t.Fatalf("failed to create wave writer %#v", err)
	}

	reader := bytes.NewReader(file.Bytes())
	r, err := NewWaveReader(reader)
	if err != nil {
		t.Fatalf("failed to create wave reader %#v", err)
	}
	chs := r.Channels()
	if len(chs) != len(channels) {
		t.Fatalf("channel count, expected=%d, actual=%d", len(channels), len(chs))
	}
	for i, channel := range channels {
		if !reflect.DeepEqual(channel, chs[i]) {
			t.Fatalf("Channel data mismatch.\nExpected:%+v\nActual:%+v", channel, chs[i])
		}
	}
}

func TestReadWriteNoDataFrameData(t *testing.T) {
	file := &bytes.Buffer{}
	channels := singleChannel
	w, err := NewWaveWriter(file, channels)
	if err != nil {
		t.Fatalf("failed to create wave writer %#v", err)
	}
	if err := w.PutFrameData(&frameDataNoMetadata); err != nil {
		t.Fatalf("failed to put frame data %#v", err)
	}
	reader := bytes.NewReader(file.Bytes())
	r, err := NewWaveReader(reader)
	if err != nil {
		t.Fatalf("failed to create wave reader %#v", err)
	}
	chs := r.Channels()
	if len(chs) != len(channels) {
		t.Fatalf("channel count, expected=%d, actual=%d", len(channels), len(chs))
	}
	for i, channel := range channels {
		if !reflect.DeepEqual(channel, chs[i]) {
			t.Fatalf("Channel data mismatch.\nExpected:%+v\nActual:%+v", channel, chs[i])
		}
	}
	frameData := r.FrameData()
	if frameData == nil {
		t.Fatalf("Frame data must be exist.")
	}
	if !reflect.DeepEqual(*frameData, frameDataNoMetadata) {
		t.Fatalf("FrameData mismatch.\nExpected:%+v\nActual:%+v", frameDataNoMetadata, *frameData)
	}
}

func TestReadWriteNoDataFrameData2(t *testing.T) {
	file := &bytes.Buffer{}
	channels := singleChannel
	w, err := NewWaveWriter(file, channels)
	if err != nil {
		t.Fatalf("failed to create wave writer %#v", err)
	}
	if err := w.PutFrameData(&frameDataSimple); err != nil {
		t.Fatalf("failed to put frame data %#v", err)
	}
	reader := bytes.NewReader(file.Bytes())
	r, err := NewWaveReader(reader)
	if err != nil {
		t.Fatalf("failed to create wave reader %#v", err)
	}
	chs := r.Channels()
	if len(chs) != len(channels) {
		t.Fatalf("channel count, expected=%d, actual=%d", len(channels), len(chs))
	}
	for i, channel := range channels {
		if !reflect.DeepEqual(channel, chs[i]) {
			t.Fatalf("Channel data mismatch.\nExpected:%+v\nActual:%+v", channel, chs[i])
		}
	}
	frameData := r.FrameData()
	if frameData == nil {
		t.Fatalf("Frame data must be exist.")
	}
	if !reflect.DeepEqual(*frameData, frameDataSimple) {
		t.Fatalf("FrameData mismatch.\nExpected:%+v\nActual:%+v", frameDataNoMetadata, *frameData)
	}
}

func TestReadWriteSingleChannel(t *testing.T) {
	file := &bytes.Buffer{}
	channels := singleChannel
	data := generateUint16Data(1024)
	w, err := NewWaveWriter(file, channels)
	if err != nil {
		t.Fatalf("failed to create wave writer %#v", err)
	}
	if err := w.PutFrameData(&frameDataSimple); err != nil {
		t.Fatalf("failed to put frame data %#v", err)
	}
	if err := w.PutWaveformData(0, data); err != nil {
		t.Fatalf("Failed to put waveform data %#v", err)
	}

	reader := bytes.NewReader(file.Bytes())
	r, err := NewWaveReader(reader)
	if err != nil {
		t.Fatalf("failed to create wave reader %#v", err)
	}
	chs := r.Channels()
	if len(chs) != len(channels) {
		t.Fatalf("channel count, expected=%d, actual=%d", len(channels), len(chs))
	}
	for i, channel := range channels {
		if !reflect.DeepEqual(channel, chs[i]) {
			t.Fatalf("Channel data mismatch.\nExpected:%+v\nActual:%+v", channel, chs[i])
		}
	}

	sr, err := r.NewSampleReader(0)
	if err != nil {
		t.Fatalf("failed to create sample reader %#v", err)
	}
	actual := make([]uint16, len(data))
	var bytesRead uint32
	if bytesRead, err = sr.ReadSample(actual); err != nil {
		t.Fatalf("failed to read sample data %#v", err)
	}
	if bytesRead != uint32(len(data)) {
		t.Fatalf("unexpected read length. expected: %d, actual: %d", len(data), bytesRead)
	}
	for i, expected := range data {
		if actual[i] != expected {
			t.Fatalf("data mismatch at 0x%x, expected=%d, actual=%d", i, expected, actual[i])
		}
	}
}

func TestReadWriteSingleChannel3Blocks(t *testing.T) {
	file := &bytes.Buffer{}
	channels := singleChannel
	data := generateUint16Data(1024)
	w, err := NewWaveWriter(file, channels)
	if err != nil {
		t.Fatalf("failed to create wave writer %#v", err)
	}
	if err := w.PutFrameData(&frameDataSimple); err != nil {
		t.Fatalf("failed to put frame data %#v", err)
	}
	if err := w.PutWaveformData(0, data[0:100]); err != nil {
		t.Fatalf("Failed to put waveform data %#v", err)
	}
	if err := w.PutWaveformData(0, data[100:600]); err != nil {
		t.Fatalf("Failed to put waveform data %#v", err)
	}
	if err := w.PutWaveformData(0, data[600:1024]); err != nil {
		t.Fatalf("Failed to put waveform data %#v", err)
	}

	reader := bytes.NewReader(file.Bytes())
	r, err := NewWaveReader(reader)
	if err != nil {
		t.Fatalf("failed to create wave reader %#v", err)
	}
	chs := r.Channels()
	if len(chs) != len(channels) {
		t.Fatalf("channel count, expected=%d, actual=%d", len(channels), len(chs))
	}
	for i, channel := range channels {
		if !reflect.DeepEqual(channel, chs[i]) {
			t.Fatalf("Channel data mismatch.\nExpected:%+v\nActual:%+v", channel, chs[i])
		}
	}

	sr, err := r.NewSampleReader(0)
	if err != nil {
		t.Fatalf("failed to create sample reader %#v", err)
	}
	actual := make([]uint16, len(data))
	var bytesRead uint32
	if bytesRead, err = sr.ReadSample(actual); err != nil {
		t.Fatalf("failed to read sample data %#v", err)
	}
	if bytesRead != uint32(len(data)) {
		t.Fatalf("unexpected read length. expected: %d, actual: %d", len(data), bytesRead)
	}
	for i, expected := range data {
		if actual[i] != expected {
			t.Fatalf("data mismatch at 0x%x, expected=%d, actual=%d", i, expected, actual[i])
		}
	}
}

func TestReadWriteDualChannel3Blocks(t *testing.T) {
	file := &bytes.Buffer{}
	channels := dualChannel
	data0 := generateUint16Data(1024)
	data1 := generateFloat32Data(1024)
	w, err := NewWaveWriter(file, channels)
	if err != nil {
		t.Fatalf("failed to create wave writer %#v", err)
	}
	if err := w.PutFrameData(&frameDataSimple); err != nil {
		t.Fatalf("failed to put frame data %#v", err)
	}
	if err := w.PutWaveformData(0, data0[0:100]); err != nil {
		t.Fatalf("Failed to put waveform data %#v", err)
	}
	if err := w.PutWaveformData(1, data1[0:100]); err != nil {
		t.Fatalf("Failed to put waveform data %#v", err)
	}
	if err := w.PutWaveformData(0, data0[100:600]); err != nil {
		t.Fatalf("Failed to put waveform data %#v", err)
	}
	if err := w.PutWaveformData(1, data1[100:600]); err != nil {
		t.Fatalf("Failed to put waveform data %#v", err)
	}
	if err := w.PutWaveformData(0, data0[600:1024]); err != nil {
		t.Fatalf("Failed to put waveform data %#v", err)
	}
	if err := w.PutWaveformData(1, data1[600:1024]); err != nil {
		t.Fatalf("Failed to put waveform data %#v", err)
	}
	reader := bytes.NewReader(file.Bytes())
	r, err := NewWaveReader(reader)
	if err != nil {
		t.Fatalf("failed to create wave reader %#v", err)
	}
	chs := r.Channels()
	if len(chs) != len(channels) {
		t.Fatalf("channel count, expected=%d, actual=%d", len(channels), len(chs))
	}
	for i, channel := range channels {
		if !reflect.DeepEqual(channel, chs[i]) {
			t.Fatalf("Channel data mismatch.\nExpected:%+v\nActual:%+v", channel, chs[i])
		}
	}
	{
		sr, err := r.NewSampleReader(0)
		if err != nil {
			t.Fatalf("failed to create sample reader %#v", err)
		}
		actual := make([]uint16, len(data0))
		var bytesRead uint32
		if bytesRead, err = sr.ReadSample(actual); err != nil {
			t.Fatalf("failed to read sample data %#v", err)
		}
		if bytesRead != uint32(len(data0)) {
			t.Fatalf("unexpected read length. expected: %d, actual: %d", len(data0), bytesRead)
		}
		for i, expected := range data0 {
			if actual[i] != expected {
				t.Fatalf("data mismatch at 0x%x, expected=%d, actual=%d", i, expected, actual[i])
			}
		}
	}
	{
		sr, err := r.NewSampleReader(1)
		if err != nil {
			t.Fatalf("failed to create sample reader %#v", err)
		}
		actual := make([]float32, len(data1))
		var bytesRead uint32
		if bytesRead, err = sr.ReadSample(actual); err != nil {
			t.Fatalf("failed to read sample data %#v", err)
		}
		if bytesRead != uint32(len(data1)) {
			t.Fatalf("unexpected read length. expected: %d, actual: %d", len(data1), bytesRead)
		}
		for i, expected := range data1 {
			if actual[i] != expected {
				t.Fatalf("data mismatch at 0x%x, expected=%f, actual=%f", i, expected, actual[i])
			}
		}
	}
}
