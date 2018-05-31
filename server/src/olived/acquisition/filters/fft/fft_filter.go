package fft

import (
	"bytes"
	"encoding/binary"
	"log"
	"math"

	acq "olived/acquisition"

	fft "github.com/mjibson/go-dsp/fft"
	window "github.com/mjibson/go-dsp/window"
)

type WindowType int

const (
	Rectangular WindowType = 0
	Hamming                = 1
	Hann                   = 2
)

type fftFilterProcessor struct {
	acq.ProcessorHostImpl

	TargetChannel  string
	NumberOfPoints int
	Window         WindowType

	started        bool
	inputBuffer    []float64
	sampled        int
	samplingRate   float32
	numberOfPoints int
	window         WindowType
	windowValues   []float64

	samplesCh chan *processSamplesRequest
	stopFftCh chan struct{}
	doneFftCh chan struct{}
}

type processSamplesRequest struct {
	Samples []float64
	Unit    acq.Unit
}

func NewFftFilterProcessor() *fftFilterProcessor {
	return &fftFilterProcessor{
		ProcessorHostImpl: acq.NewProcessorHostImpl(),
		samplesCh:         make(chan *processSamplesRequest, 2),
		stopFftCh:         make(chan struct{}),
		doneFftCh:         make(chan struct{}),
	}
}

func (f *fftFilterProcessor) Type() acq.ProcessorType {
	return acq.Filter
}

func (f *fftFilterProcessor) Start() {
	f.started = true
	go f.processSampledData()
}
func (f *fftFilterProcessor) Stop() {
	if f.started {
		f.stopFftCh <- struct{}{}
		<-f.doneFftCh
	}
}
func (f *fftFilterProcessor) Started(unit acq.Unit) {
}
func (f *fftFilterProcessor) Stopped(unit acq.Unit) {
}

func (f *fftFilterProcessor) SettingsChanging(unit acq.Unit, settings *acq.UnitSettings) error {
	for _, processor := range *f.ProcessorHostImpl.ProcessorMap() {
		if err := processor.SettingsChanging(unit, settings); err != nil {
			return err
		}
	}
	return nil
}
func (f *fftFilterProcessor) SettingsChanged(unit acq.Unit, settings *acq.UnitSettings) {
	for _, processor := range *f.ProcessorHostImpl.ProcessorMap() {
		processor.SettingsChanged(unit, settings)
	}
}
func (f *fftFilterProcessor) DataArrived(unit acq.Unit, data acq.AcquiredData) {
	if f.NumberOfPoints <= 0 {
		return
	}
	if data.Type() != acq.TimeSeries {
		return
	}
	if data.IsFrameFirstData() {
		f.sampled = 0
		f.numberOfPoints = f.NumberOfPoints
		if f.inputBuffer == nil || len(f.inputBuffer) < f.numberOfPoints {
			f.inputBuffer = make([]float64, f.numberOfPoints)
		}
		settings := unit.Settings()
		channelSettigns := settings.Channels[f.TargetChannel]
		f.samplingRate = channelSettigns.SamplingRate

		// Initialize window function coefficients.
		if f.windowValues == nil || len(f.windowValues) != f.numberOfPoints || f.window != f.Window {
			switch f.window {
			case Rectangular:
				f.windowValues = window.Rectangular(f.numberOfPoints)
			case Hamming:
				f.windowValues = window.Hamming(f.numberOfPoints)
			case Hann:
				f.windowValues = window.Hann(f.numberOfPoints)
			default:
				f.windowValues = window.Rectangular(f.numberOfPoints)
			}
		}
	}
	if f.sampled >= f.numberOfPoints {
		return
	}

	parsedData := acq.GetParsedDataByChannel(data, f.TargetChannel)
	if parsedData == nil {
		return
	}

	values_, err := parsedData.ReadAll()
	if err != nil {
		return
	}

	values, ok := values_.([]float64)
	if !ok {
		return
	}

	samplesToCopy := len(values)
	if samplesToCopy > f.numberOfPoints-f.sampled {
		samplesToCopy = f.numberOfPoints - f.sampled
	}
	for i := 0; i < samplesToCopy; i++ {
		f.inputBuffer[f.sampled+i] = values[i] * f.windowValues[f.sampled+i]
	}
	f.sampled += samplesToCopy
	if f.sampled == f.numberOfPoints {
		f.samplesCh <- &processSamplesRequest{
			Unit:    unit,
			Samples: f.inputBuffer,
		}
		f.inputBuffer = nil
	}
}

const FFTDataHeaderLength int = 2 + 2 + 4 + 4

type fftData struct {
	Raw     []byte
	Channel string
	acq.FrameData
}

func (d *fftData) Type() acq.AcquiredDataType {
	return acq.FFT
}
func (d *fftData) RawData() acq.ReadOnlyData {
	return acq.ReadOnlyData(d.Raw)
}
func (d *fftData) Parse() []acq.ParsedData {
	return []acq.ParsedData{
		&parsedFFTData{
			Parent: d,
		},
	}
}
func (d *fftData) IsFrameFirstData() bool {
	return true
}
func (d *fftData) IsFrameLastData() bool {
	return false
}

type parsedFFTData struct {
	Parent *fftData
}

func (d *parsedFFTData) Channel() string {
	return d.Parent.Channel
}
func (d *parsedFFTData) Length() uint64 {
	return uint64(len(d.Parent.Raw) - FFTDataHeaderLength)
}
func (d *parsedFFTData) Read(p []byte) (int, error) {
	reader := bytes.NewReader(d.Parent.Raw[FFTDataHeaderLength:])
	return reader.Read(p)
}
func (d *parsedFFTData) NumberOfItems() int {
	return (len(d.Parent.Raw) - FFTDataHeaderLength) / 4
}
func (d *parsedFFTData) ReadAll() (interface{}, error) {
	data := make([]float32, d.NumberOfItems())
	if err := binary.Read(d, binary.LittleEndian, data); err != nil {
		return nil, err
	}
	return data, nil
}
func (d *parsedFFTData) Iterate(from int, toExclusive int, iter func(index int, value float64)) {
	data := d.Parent.Raw[FFTDataHeaderLength:]
	for i := from; i < toExclusive; i++ {
		iter(i, float64(math.Float32frombits(binary.LittleEndian.Uint32(data[i*4:i*4+4]))))
	}
}

func (f *fftFilterProcessor) processSampledData() {
	defer func() { f.doneFftCh <- struct{}{} }()
	for {
		select {
		case <-f.stopFftCh:
			return
		case request := <-f.samplesCh:
			spectrum := fft.FFTReal(request.Samples)
			numberOfPoints := len(spectrum)
			spectrum = spectrum[:numberOfPoints/2] // Get fs/2 points
			outputBuffer := bytes.Buffer{}
			dataLength := len(spectrum) * 4
			outputBuffer.Grow(FFTDataHeaderLength + dataLength)
			// Put header
			binary.Write(&outputBuffer, binary.LittleEndian, uint16(1))
			binary.Write(&outputBuffer, binary.LittleEndian, uint16(0))
			binary.Write(&outputBuffer, binary.LittleEndian, uint32(dataLength+4))
			// Put Number of points
			binary.Write(&outputBuffer, binary.LittleEndian, uint32(len(spectrum)))
			normalizeFactor := (float64(numberOfPoints) / float64(f.samplingRate))
			var max_power float64
			for i, item := range spectrum {
				if i >= outputBuffer.Len() {
					break
				}
				power := (real(item)*real(item) + imag(item)*imag(item)) * normalizeFactor
				if max_power < power {
					max_power = power
				}
				value := float32(power)
				binary.Write(&outputBuffer, binary.LittleEndian, value)
			}
			log.Printf("FFT MaxPower = %f\n", max_power)
			f.ProcessorHostImpl.NotifyDataArrived(
				request.Unit,
				&fftData{Raw: outputBuffer.Bytes(), Channel: f.TargetChannel},
			)
		}
	}
}
