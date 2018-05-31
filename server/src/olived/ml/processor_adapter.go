package ml

import (
	acq "olived/acquisition"
)

type adapter struct {
	server          MLServer
	buffer          []float32
	samplesSkipped  int
	samplesBuffered int
}

// NewProcessorAdapter
// Create new acquisition processor adapter for a ML server.
// server: an instance of ML server the new adapter wraps.
// settings: an instance of ML server settings the server uses.
// inputCh: a channel to output input data to the ML server.
func NewProcessorAdapter(server MLServer) *adapter {
	return &adapter{
		server: server,
	}
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
func putAcquiredData(input acq.ParsedData, offset int, buffer []float32) (int, error) {

	data, err := input.ReadAll()
	if err != nil {
		return 0, err
	}
	itemsBuffered := 0
	switch typed := data.(type) {
	case []uint16:
		itemsBuffered = min(len(typed)-offset, len(buffer))
		for i := 0; i < itemsBuffered; i++ {
			buffer[i] = float32(typed[offset+i])
		}
	case []uint32:
		itemsBuffered = min(len(typed)-offset, len(buffer))
		for i := 0; i < itemsBuffered; i++ {
			buffer[i] = float32(typed[offset+i])
		}
	case []float32:
		itemsBuffered = min(len(typed)-offset, len(buffer))
		for i := 0; i < itemsBuffered; i++ {
			buffer[i] = typed[offset+i]
		}
	case []float64:
		itemsBuffered = min(len(typed)-offset, len(buffer))
		for i := 0; i < itemsBuffered; i++ {
			buffer[i] = float32(typed[offset+i])
		}
	default:
		return 0, nil // Not supported data type.
	}
	return itemsBuffered, nil
}

// Implement acq.Processor interface

func (a *adapter) Type() acq.ProcessorType {
	return acq.Analyzer
}

func (a *adapter) Started(unit acq.Unit) {
}
func (a *adapter) Stopped(unit acq.Unit) {
}
func (a *adapter) SettingsChanging(unit acq.Unit, settings *acq.UnitSettings) error {
	return nil
}
func (a *adapter) SettingsChanged(unit acq.Unit, settings *acq.UnitSettings) {

}
func (a *adapter) DataArrived(unit acq.Unit, data acq.AcquiredData) {
	if !a.server.Running() {
		return
	}
	settings := a.server.Settings()
	if len(settings.TargetChannels) == 0 || settings.InputDataSize == 0 {
		return
	}
	if data.IsFrameFirstData() && (a.buffer == nil || len(a.buffer) != settings.InputDataSize) {
		a.buffer = make([]float32, settings.InputDataSize)
		a.samplesBuffered = 0
	}

	if a.buffer == nil {
		return
	}

	inputType := a.server.Settings().InputType
	switch inputType {
	case Raw:
		if data.Type() == acq.TimeSeries {
			if data.IsFrameFirstData() {
				a.samplesBuffered = 0
				a.samplesSkipped = 0
			}
			parsedDatas := data.Parse()
			for _, parsedData := range parsedDatas {
				if parsedData.Channel() == settings.TargetChannels[0] {
					numItems := parsedData.NumberOfItems()
					samplesToSkip := 0
					if a.samplesSkipped < settings.InputDataOffset {
						samplesToSkip = min(numItems, settings.InputDataOffset-a.samplesSkipped)
						a.samplesSkipped += samplesToSkip
					}
					numItems -= samplesToSkip
					if numItems > 0 {
						buffered, err := putAcquiredData(parsedData, samplesToSkip, a.buffer[a.samplesBuffered:])
						if err == nil {
							a.samplesBuffered += buffered
						}
						if a.samplesBuffered >= settings.InputDataSize {
							a.server.Write(MLServerInput(a.buffer))
							a.buffer = nil
						}
					}
					break
				}
			}
		}
	case FFT:
		if data.Type() == acq.FFT && data.IsFrameFirstData() {
			parsedDatas := data.Parse()
			for _, parsedData := range parsedDatas {
				if parsedData.Channel() == settings.TargetChannels[0] {
					inputDataSize := settings.InputDataSize
					buffered, err := putAcquiredData(parsedData, 0, a.buffer)
					if err == nil {
						// TODO: Extend or interpolate input data if needed.
						for i := buffered; i < inputDataSize; i++ {
							a.buffer[i] = 0
						}
						a.server.Write(MLServerInput(a.buffer))
						a.buffer = nil
					}
				}
			}
		}
	}
}
