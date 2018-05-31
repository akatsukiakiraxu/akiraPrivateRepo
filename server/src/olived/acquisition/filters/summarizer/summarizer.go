package summarizer

import (
	"encoding/binary"
	"math"
	"sort"

	"bytes"

	acq "olived/acquisition"
)

type channelSummary struct {
	Max float64
	Min float64
}

func (c *channelSummary) Update(value float64) {
	if c.Max < value {
		c.Max = value
	}
	if value < c.Min {
		c.Min = value
	}
}

type summarizeProcessor struct {
	acq.ProcessorHostImpl

	NumberOfPoints int

	started        bool
	numberOfPoints int
	sampled        int
	summaries      map[string]channelSummary
	isFirst        bool
	isLast         bool

	stopSummarizeCh chan struct{}
	doneSummarizeCh chan struct{}
}

type processSamplesRequest struct {
	Samples []float64
	Unit    acq.Unit
}

func NewSummarizeProcessor() *summarizeProcessor {
	return &summarizeProcessor{
		ProcessorHostImpl: acq.NewProcessorHostImpl(),
		stopSummarizeCh:   make(chan struct{}),
		doneSummarizeCh:   make(chan struct{}),
	}
}

func (f *summarizeProcessor) Type() acq.ProcessorType {
	return acq.Filter
}

func (f *summarizeProcessor) Start() {
	f.started = true
}
func (f *summarizeProcessor) Stop() {
	f.started = false
}
func (f *summarizeProcessor) Started(unit acq.Unit) {
}
func (f *summarizeProcessor) Stopped(unit acq.Unit) {
}

func (f *summarizeProcessor) SettingsChanging(unit acq.Unit, settings *acq.UnitSettings) error {
	for _, processor := range *f.ProcessorHostImpl.ProcessorMap() {
		if err := processor.SettingsChanging(unit, settings); err != nil {
			return err
		}
	}
	return nil
}
func (f *summarizeProcessor) SettingsChanged(unit acq.Unit, settings *acq.UnitSettings) {
	for _, processor := range *f.ProcessorHostImpl.ProcessorMap() {
		processor.SettingsChanged(unit, settings)
	}
}

type summaryData struct {
	raw     []byte
	isFirst bool
	isLast  bool
	acq.FrameData
}

func (d *summaryData) RawData() acq.ReadOnlyData {
	return acq.ReadOnlyData(d.raw)
}
func (d *summaryData) Type() acq.AcquiredDataType {
	return acq.TimeSeriesSummary
}
func (d *summaryData) IsFrameFirstData() bool {
	return d.isFirst
}
func (d *summaryData) IsFrameLastData() bool {
	return d.isLast
}
func (d *summaryData) Parse() []acq.ParsedData {
	return nil
}

func min_int(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func (f *summarizeProcessor) DataArrived(unit acq.Unit, data acq.AcquiredData) {
	if !f.started {
		return
	}
	if data.Type() != acq.TimeSeries {
		return
	}
	parsedData := data.Parse()

	totalItemsProcessed := 0
	maxNumberOfItems := 0
	for maxNumberOfItems == 0 || totalItemsProcessed < maxNumberOfItems {
		if data.IsFrameFirstData() || f.summaries == nil || len(f.summaries) != len(parsedData) {
			f.isFirst = data.IsFrameFirstData()
			f.sampled = 0
			f.numberOfPoints = f.NumberOfPoints
			f.summaries = make(map[string]channelSummary, len(parsedData))
			for _, parsedChannel := range parsedData {
				channel := parsedChannel.Channel()
				f.summaries[channel] = channelSummary{
					Max: math.Inf(-1.0),
					Min: math.Inf(+1.0),
				}
			}
		}

		itemsProcessed := 0
		for _, parsedChannel := range parsedData {
			summary := f.summaries[parsedChannel.Channel()]
			numberOfItems := parsedChannel.NumberOfItems()
			if maxNumberOfItems < numberOfItems {
				maxNumberOfItems = numberOfItems
			}

			itemsToProcess := min_int(numberOfItems-totalItemsProcessed, f.numberOfPoints-f.sampled)
			if itemsToProcess >= 0 {
				parsedChannel.(acq.TimeSeriesParsedData).Iterate(totalItemsProcessed, totalItemsProcessed+itemsToProcess, func(_ int, value float64) { summary.Update(value) })
				if itemsProcessed < itemsToProcess {
					itemsProcessed = itemsToProcess
				}
				f.summaries[parsedChannel.Channel()] = summary
			}
		}
		f.sampled += itemsProcessed
		totalItemsProcessed += itemsProcessed

		if f.sampled >= f.numberOfPoints || data.IsFrameLastData() {
			f.isLast = data.IsFrameLastData() && (maxNumberOfItems-totalItemsProcessed) < f.numberOfPoints
			numberOfChannels := len(f.summaries)
			buffer := new(bytes.Buffer)
			index := 0
			keys := make([]string, 0, len(f.summaries))
			for key := range f.summaries {
				keys = append(keys, key)
			}
			sort.Sort(sort.StringSlice(keys))

			flags := uint16(0)
			if f.isFirst {
				flags |= uint16(1)
			}
			binary.Write(buffer, binary.LittleEndian, uint16(0))
			binary.Write(buffer, binary.LittleEndian, flags)
			binary.Write(buffer, binary.LittleEndian, uint32(4*6*numberOfChannels))
			for _, key := range keys {
				summary := f.summaries[key]
				//log.Printf("summary %s, max=%f, min=%f", key, summary.Max, summary.Min)
				binary.Write(buffer, binary.LittleEndian, math.Float32bits(float32(summary.Max)))
				binary.Write(buffer, binary.LittleEndian, math.Float32bits(float32(summary.Min)))
				binary.Write(buffer, binary.LittleEndian, float32(-4)) // TODO: put correct expected data.
				binary.Write(buffer, binary.LittleEndian, float32(+4)) // TODO: put correct expected data.
				binary.Write(buffer, binary.LittleEndian, float32(-2)) // TODO: put correct expected data.
				binary.Write(buffer, binary.LittleEndian, float32(+2)) // TODO: put correct expected data.
				index++
			}
			f.NotifyDataArrived(unit, &summaryData{
				raw:     buffer.Bytes(),
				isFirst: f.isFirst,
				isLast:  f.isLast,
			})
			f.isFirst = false
			f.summaries = nil
		}
	}
}
