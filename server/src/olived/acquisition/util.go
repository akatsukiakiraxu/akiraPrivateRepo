package acquisition

import (
	"encoding/json"
	"io/ioutil"
	"sync"
)

// LoadConfig - Load acquisition unit configuration from file.
func LoadConfig(path string) (*UnitConfig, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := new(UnitConfig)
	if err = json.Unmarshal(b, config); err != nil {
		return nil, err
	}

	return config, nil
}

// LoadSettings - Load acquisition unit settings from file.
func LoadSettings(path string) (*UnitSettings, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	settings := new(UnitSettings)
	if err = json.Unmarshal(b, settings); err != nil {
		return nil, err
	}

	return settings, nil
}

type ProcessorHostImpl struct {
	processors     map[Processor]Processor
	processorsLock sync.RWMutex
}

func NewProcessorHostImpl() ProcessorHostImpl {
	return ProcessorHostImpl{
		processors: make(map[Processor]Processor, 2),
	}
}
func (u *ProcessorHostImpl) AddProcessor(processor Processor) error {
	u.processorsLock.Lock()
	defer u.processorsLock.Unlock()
	u.processors[processor] = processor
	return nil
}
func (u *ProcessorHostImpl) RemoveProcessor(processor Processor) {
	u.processorsLock.Lock()
	defer u.processorsLock.Unlock()
	delete(u.processors, processor)
}
func (u *ProcessorHostImpl) Processors() []Processor {
	u.processorsLock.RLock()
	defer u.processorsLock.RUnlock()
	processors := make([]Processor, len(u.processors))
	for _, processor := range u.processors {
		processors = append(processors, processor)
	}
	return processors
}

func (u *ProcessorHostImpl) ProcessorMap() *map[Processor]Processor {
	return &u.processors
}
func (u *ProcessorHostImpl) BeginUsingProcessors() {
	u.processorsLock.RLock()
}

func (u *ProcessorHostImpl) EndUsingProcessors() {
	u.processorsLock.RUnlock()
}

func (u *ProcessorHostImpl) NotifyStarted(unit Unit) {
	defer u.processorsLock.RUnlock()
	u.processorsLock.RLock()
	for _, processor := range u.processors {
		processor.Started(unit)
	}
}
func (u *ProcessorHostImpl) NotifyStopped(unit Unit) {
	defer u.processorsLock.RUnlock()
	u.processorsLock.RLock()
	for _, processor := range u.processors {
		processor.Stopped(unit)
	}
}
func (u *ProcessorHostImpl) NotifyDataArrived(unit Unit, data AcquiredData) {
	defer u.processorsLock.RUnlock()
	u.processorsLock.RLock()
	for _, processor := range u.processors {
		processor.DataArrived(unit, data)
	}
}
func (u *ProcessorHostImpl) NotifySettingsChanging(unit Unit, settings *UnitSettings) error {
	defer u.processorsLock.RUnlock()
	u.processorsLock.RLock()
	for _, processor := range u.processors {
		if err := processor.SettingsChanging(unit, settings); err != nil {
			return err
		}
	}
	return nil
}
func (u *ProcessorHostImpl) NotifySettingsChanged(unit Unit, settings *UnitSettings) {
	defer u.processorsLock.RUnlock()
	u.processorsLock.RLock()
	for _, processor := range u.processors {
		processor.SettingsChanged(unit, settings)
	}
}

func GetParsedDataByChannel(acquiredData AcquiredData, channelName string) ParsedData {
	parsedDatas := acquiredData.Parse()
	if parsedDatas == nil {
		return nil
	}
	for _, parsedData := range parsedDatas {
		if parsedData.Channel() == channelName {
			return parsedData
		}
	}
	return nil
}
