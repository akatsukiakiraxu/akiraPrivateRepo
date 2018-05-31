package mock

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"runtime/debug"
	"strconv"
	"time"

	parent "olived/acquisition"
)

func makeChannelsStatus(config *parent.UnitConfig) map[string]parent.ChannelStatus {
	channels := make(map[string]parent.ChannelStatus, len(config.Channels))
	for name, channel := range config.Channels {
		rangeName := channel.RangesSupported[0]
		defaultRange := config.Ranges[rangeName]
		channels[name] = parent.ChannelStatus{
			Enabled:        true,
			MinimumVoltage: defaultRange.MinimumVoltage,
			MaximumVoltage: defaultRange.MaximumVoltage,
			MinimumValue:   defaultRange.MinimumValue,
			MaximumValue:   defaultRange.MaximumValue,
		}
	}
	return channels
}
func makeChannelsSettings(config *parent.UnitConfig) map[string]parent.ChannelSettings {
	channels := make(map[string]parent.ChannelSettings, len(config.Channels))
	for name, channel := range config.Channels {
		rangeName := channel.RangesSupported[0]
		channels[name] = parent.ChannelSettings{
			Enabled:      true,
			SamplingRate: channel.SamplingRatesSupported[0],
			Range:        rangeName,
			Coupling:     channel.CouplingsSupported[0],
		}
	}
	return channels
}

func NewMockUnit(config *parent.UnitConfig) (parent.Unit, error) {
	var enableDummyFFT bool
	if value, ok := config.Parameters["mock_enable_fft"]; ok {
		if enableDummyFFT, ok = value.(bool); !ok {
			return nil, fmt.Errorf("mock_enable_fft must be a bool value")
		}
	}
	var enableSummary bool
	if value, ok := config.Parameters["mock_enable_dummy_summary"]; ok {
		if enableSummary, ok = value.(bool); !ok {
			return nil, fmt.Errorf("mock_enable_summary must be a bool value")
		}
	}
	return &mockUnit{
		config: *config,
		status: parent.UnitStatus{
			Running:  false,
			Channels: makeChannelsStatus(config),
		},
		settings: parent.UnitSettings{
			Channels: makeChannelsSettings(config),
		},
		ProcessorHostImpl: parent.NewProcessorHostImpl(),

		postDataCh:       make(chan parent.AcquiredData, 10),
		summarizerDataCh: make(chan parent.AcquiredData, 10),

		stopGeneratorCh:  make(chan struct{}),
		doneGeneratorCh:  make(chan struct{}),
		stopSummarizerCh: make(chan struct{}),
		doneSummarizerCh: make(chan struct{}),
		stopDataPosterCh: make(chan struct{}),
		doneDataPosterCh: make(chan struct{}),

		enableDummyFFT: enableDummyFFT,
		enableSummary:  enableSummary,
	}, nil
}

type mockUnit struct {
	parent.ProcessorHostImpl

	config   parent.UnitConfig
	status   parent.UnitStatus
	settings parent.UnitSettings

	postDataCh       chan parent.AcquiredData
	summarizerDataCh chan parent.AcquiredData

	stopGeneratorCh  chan struct{}
	doneGeneratorCh  chan struct{}
	stopSummarizerCh chan struct{}
	doneSummarizerCh chan struct{}
	stopDataPosterCh chan struct{}
	doneDataPosterCh chan struct{}

	enableDummyFFT bool
	enableSummary  bool
}

func (u *mockUnit) Status() parent.UnitStatus {
	return u.status
}
func (u *mockUnit) Config() parent.UnitConfig {
	return u.config
}
func (u *mockUnit) Settings() parent.UnitSettings {
	return u.settings
}

func (u *mockUnit) Start() error {
	if u.status.Running {
		return fmt.Errorf("Already running")
	}
	u.status.Running = true

	go u.generator()
	go u.summarizer()
	go u.dataPoster()

	u.ProcessorHostImpl.NotifyStarted(u)
	return nil
}
func (u *mockUnit) ChangeSettings(settings *parent.UnitSettings) error {
	if err := u.ProcessorHostImpl.NotifySettingsChanging(u, settings); err != nil {
		return err
	}
	u.settings.Enabled = settings.Enabled
	u.settings.Comparing = settings.Comparing
	for name, channel := range settings.Channels {
		oldStatus := u.status.Channels[name]
		u.status.Channels[name] = parent.ChannelStatus{
			Enabled:        channel.Enabled,
			MinimumVoltage: oldStatus.MinimumVoltage,
			MaximumVoltage: oldStatus.MaximumVoltage,
			MinimumValue:   oldStatus.MinimumValue,
			MaximumValue:   oldStatus.MaximumValue,
		}
		u.settings.Channels[name] = channel
	}
	u.settings.Trigger = settings.Trigger
	u.ProcessorHostImpl.NotifySettingsChanged(u, settings)
	return nil
}
func (u *mockUnit) Stop() error {
	if !u.status.Running {
		return fmt.Errorf("Not running")
	}
	u.status.Running = false

	u.stopGeneratorCh <- struct{}{}
	<-u.doneGeneratorCh
	u.stopSummarizerCh <- struct{}{}
	<-u.doneSummarizerCh
	u.stopDataPosterCh <- struct{}{}
	<-u.doneDataPosterCh

	u.ProcessorHostImpl.NotifyStopped(u)
	return nil
}

func (u *mockUnit) dataArrived(data parent.AcquiredData) {
	u.postDataCh <- data
}

type rawAcquiredData struct {
	Raw      []byte
	DataType parent.AcquiredDataType
	parent.FrameData
	IsFirst bool
	IsLast  bool
}

type rawParsedData struct {
	index  int
	parent *rawAcquiredData
}

func min_int(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func (d *rawAcquiredData) RawData() parent.ReadOnlyData {
	return parent.ReadOnlyData(d.Raw)
}
func (d *rawAcquiredData) Type() parent.AcquiredDataType {
	return d.DataType
}
func (d *rawAcquiredData) Parse() []parent.ParsedData {
	parsedDatas := make([]parent.ParsedData, 4)
	for i := 0; i < 4; i++ {
		parsedDatas[i] = &rawParsedData{
			index:  i,
			parent: d,
		}
	}
	return parsedDatas
}
func (d *rawAcquiredData) IsFrameFirstData() bool {
	return d.IsFirst
}
func (d *rawAcquiredData) IsFrameLastData() bool {
	return d.IsLast
}
func (d *rawParsedData) Channel() string {
	return fmt.Sprintf("ch%d", d.index+1)
}
func (d *rawParsedData) Length() uint64 {
	return uint64(len(d.parent.Raw) / 4)
}
func (d *rawParsedData) Read(p []byte) (int, error) {
	bytesToRead := min_int(int(d.Length()), len(p) & ^1)
	raw := d.parent.Raw
	ch := d.index
	for i := 0; i < bytesToRead/4; i++ {
		copy(p[i*4+0:i*4+4], raw[4*4*i+ch*4+0:4*4*i+ch*4+4])
	}
	return bytesToRead, nil
}
func (d *rawParsedData) NumberOfItems() int {
	return int(d.Length()) / 4
}
func (d *rawParsedData) ReadAll() (interface{}, error) {
	samples := d.NumberOfItems()
	raw := d.parent.Raw
	ch := d.index
	result := make([]float64, samples)
	for i := 0; i < samples; i++ {
		result[i] = float64(math.Float32frombits(binary.LittleEndian.Uint32(raw[4*4*i+ch*4+0 : 4*4*i+ch*4+4])))
	}
	return result, nil
}

func (d *rawParsedData) Iterate(from int, toExclusive int, iter func(index int, value float64)) {
	raw := d.parent.Raw
	ch := d.index
	for i := from; i < toExclusive; i++ {
		iter(i, float64(math.Float32frombits(binary.LittleEndian.Uint32(raw[4*4*i+ch*4+0:4*4*i+ch*4+4]))))
	}
}

func (u *mockUnit) generator() {
	defer func() { u.doneGeneratorCh <- struct{}{} }()
	ticker := time.NewTicker(time.Millisecond * 10)
	const channels int = 4
	const samples int = 2500
	const cyclesPerFrame int = 4
	const waveformType string = "sine"
	var counter uint16
	var frameIndex uint16
	isNextFirstPacket := true
	cycleCount := 0
	buffer := bytes.Buffer{}
	buffer.Grow(2 * channels * samples)

	metadata := make(map[string]string)
	wave := func(counter uint16) float32 {
		return float32((math.Sin(float64(counter)/32768.0*2.0*math.Pi) + 1.0) / 2.0)
	}
	for {
		select {
		case <-u.stopGeneratorCh:
			return
		case <-ticker.C:
			buffer.Reset()

			samplesWritten := samples
			isLastPacket := false
			isFirstPacket := isNextFirstPacket
			isNextFirstPacket = false
			metadata["frame_index"] = strconv.Itoa(int(frameIndex))

			for i := 0; i < samples; i++ {
				for ch := 0; ch < channels; ch++ {
					binary.Write(&buffer, binary.LittleEndian, wave(counter))
				}
				if counter < 32767 {
					counter++
				} else {
					counter = 0
					cycleCount++
					if cycleCount == cyclesPerFrame {
						cycleCount = 0
						samplesWritten = i + 1
						isNextFirstPacket = true
						isLastPacket = true
						frameIndex++
					}
				}
			}
			acquiredData := rawAcquiredData{
				Raw:       buffer.Bytes()[:samplesWritten*channels*4],
				DataType:  parent.TimeSeries,
				IsFirst:   isFirstPacket,
				IsLast:    isLastPacket,
				FrameData: parent.NewFrameData(map[string]interface{}{"mock": &metadata}),
			}
			u.dataArrived(&acquiredData)
			u.summarizerDataCh <- &acquiredData
		}

	}
}

func (u *mockUnit) summarizer() {
	const summarizeSamples int = 1024
	var summarized int
	const channels int = 4
	var summary struct {
		Magic    uint16
		Length   uint16
		Padding  uint16
		Channels [channels]struct {
			MaxData     uint16
			MinData     uint16
			MaxExpected uint16
			MinExpected uint16
		}
	}
	var fftCounter int
	summary.Length = uint16(channels * 8)
	for i := 0; i < channels; i++ {
		summary.Channels[i].MaxData = 0
		summary.Channels[i].MinData = 0xffff
		summary.Channels[i].MaxExpected = 0x6fff
		summary.Channels[i].MinExpected = 0x1000
	}
	for {
		select {
		case data := <-u.summarizerDataCh:
			if data.Type() == parent.TimeSeries {
				body := []byte(data.RawData())
				reader := bytes.NewReader(body)
				samples := len(body) / (channels * 2)
				var sample uint16
				for i := 0; i < samples; i++ {
					for ch := 0; ch < channels; ch++ {
						binary.Read(reader, binary.LittleEndian, &sample)
						if summary.Channels[ch].MinData > sample {
							summary.Channels[ch].MinData = sample
						}
						if summary.Channels[ch].MaxData < sample {
							summary.Channels[ch].MaxData = sample
						}
					}

					summarized++
					if summarized == summarizeSamples {
						// Post summarized data if summarized enough data.
						size := binary.Size(&summary)
						buffer := new(bytes.Buffer)
						buffer.Grow(size)
						if err := binary.Write(buffer, binary.LittleEndian, &summary); err != nil {
							log.Printf("Marshaling failed. %s", err.Error())
						} else {
							acquiredData := rawAcquiredData{
								Raw:      buffer.Bytes(),
								DataType: parent.TimeSeriesSummary,
							}
							if u.enableSummary {
								u.dataArrived(&acquiredData)
							}
						}
						summarized = 0
						for i := 0; i < channels; i++ {
							summary.Channels[i].MaxData = 0
							summary.Channels[i].MinData = 0xffff
							summary.Channels[i].MaxExpected = 0x6fff
							summary.Channels[i].MinExpected = 0x1000
						}

						// Post FFT result if enabled
						if u.enableDummyFFT {
							fftBuffer := make([]byte, summarizeSamples*2+2*4)
							binary.LittleEndian.PutUint16(fftBuffer[0:2], uint16(1))
							binary.LittleEndian.PutUint16(fftBuffer[2:4], uint16(2+summarizeSamples*2))
							binary.LittleEndian.PutUint16(fftBuffer[4:6], uint16(0))
							binary.LittleEndian.PutUint16(fftBuffer[6:8], uint16(0))
							binary.LittleEndian.PutUint16(fftBuffer[2*(fftCounter/2)+2*4:2*(fftCounter/2)+2*4+2], uint16(0x500))
							binary.LittleEndian.PutUint16(fftBuffer[2*fftCounter+2*4:2*fftCounter+2*4+2], uint16(0x1000))
							fftCounter++
							if fftCounter == summarizeSamples {
								fftCounter = 0
							}
							fftData := rawAcquiredData{
								Raw:      fftBuffer,
								DataType: parent.FFT,
							}
							u.dataArrived(&fftData)
						}
					}
				}
			}
		}
	}
}

func (u *mockUnit) dataPoster() {
	defer func() { u.doneDataPosterCh <- struct{}{} }()

	for {
		select {
		case <-u.stopDataPosterCh:
			return
		case data := <-u.postDataCh:
			func() {
				defer func() {
					if err := recover(); err != nil {
						log.Fatalf("A processor has paniced in DataArrived handler. %+v, %s", err, debug.Stack())
					}
				}()
				u.ProcessorHostImpl.NotifyDataArrived(u, data)
			}()
		}
	}
}
