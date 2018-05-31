package adcdb

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"runtime/debug"

	"time"

	"encoding/json"
	"io/ioutil"
	parent "olived/acquisition"
	"olived/core"
)

const (
	rawDataSizeLimit uint32 = 10 * 1024 * 1024
)

func makeChannelsStatus(config *parent.UnitConfig) map[string]parent.ChannelStatus {
	channels := make(map[string]parent.ChannelStatus, len(config.Channels))
	for name, channel := range config.Channels {
		defaultRange := config.Ranges[channel.RangesSupported[0]]
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
		channels[name] = parent.ChannelSettings{
			Enabled:      true,
			SamplingRate: channel.SamplingRatesSupported[0],
			Range:        channel.RangesSupported[0],
			Coupling:     channel.CouplingsSupported[0],
		}
	}
	return channels
}

func min_int(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func NewADCDBUnit(config *parent.UnitConfig) (parent.Unit, error) {
	return &adcdbUnit{
		config: *config,
		status: parent.UnitStatus{
			Running:  false,
			Channels: makeChannelsStatus(config),
		},
		settings: parent.UnitSettings{
			Channels: makeChannelsSettings(config),
		},
		ProcessorHostImpl: parent.NewProcessorHostImpl(),

		postDataCh: make(chan parent.AcquiredData, 10),

		stopDataPosterCh: make(chan struct{}),
		doneDataPosterCh: make(chan struct{}),
	}, nil
}

type adcdbUnit struct {
	parent.ProcessorHostImpl

	config   parent.UnitConfig
	status   parent.UnitStatus
	settings parent.UnitSettings

	command    *exec.Cmd
	reader     io.Reader
	writer     io.WriteCloser
	postDataCh chan parent.AcquiredData

	stopSamplerCh chan struct{}
	doneSamplerCh chan struct{}

	stopDataPosterCh chan struct{}
	doneDataPosterCh chan struct{}
}

func (u *adcdbUnit) Status() parent.UnitStatus {
	return u.status
}
func (u *adcdbUnit) Config() parent.UnitConfig {
	return u.config
}
func (u *adcdbUnit) Settings() parent.UnitSettings {
	return u.settings
}

func (u *adcdbUnit) Start() error {
	if u.status.Running {
		return fmt.Errorf("Already running")
	}

	var adcdbPath string
	value, ok := u.config.Parameters["adcdb_path"]
	if !ok {
		return fmt.Errorf("adcdb_path must be specified in the configuration")
	}
	if adcdbPath, ok = value.(string); !ok {
		return fmt.Errorf("adcdb_path must be string")
	}
	adcdbArgs := []string{}
	if value, ok := u.config.Parameters["adcdb_args"]; ok {
		adcdbArgs = core.ToStringSlice(value.([]interface{}))
	}

	log.Printf("adc-db command: %s %v", adcdbPath, adcdbArgs)

	command := exec.Command(adcdbPath, adcdbArgs...)
	command.Stderr = os.Stderr
	reader, err := command.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Failed to pipe stdout.")
	}
	writer, err := command.StdinPipe()
	if err != nil {
		return fmt.Errorf("Failed to pipe stdin.")
	}
	err = command.Start()
	if err != nil {
		return fmt.Errorf("Failed to start process. %s", err.Error())
	}

	u.command = command
	u.reader = reader
	u.writer = writer
	u.status.Running = true

	go u.adcdbSampler()
	go u.adcdbDataPoster()

	u.ProcessorHostImpl.NotifyStarted(u)
	return nil
}
func (u *adcdbUnit) ChangeSettings(settings *parent.UnitSettings) error {

	if err := u.ProcessorHostImpl.NotifySettingsChanging(u, settings); err != nil {
		return err
	}

	u.settings.Enabled = settings.Enabled
	u.settings.Comparing = settings.Comparing
	u.settings.Trigger = settings.Trigger
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
	u.ProcessorHostImpl.NotifySettingsChanged(u, settings)
	if u.status.Running {
		changedSettings, _ := json.Marshal(u.settings)
		tmpFile, err := ioutil.TempFile("", "olive_changed_settings_")
		if err != nil {
			log.Fatalln(err)
		}
		err = ioutil.WriteFile(tmpFile.Name(), changedSettings, 0644)
		if err != nil {
			log.Fatalln(err)
		}
		io.WriteString(u.writer, tmpFile.Name()+"\n")

	}
	return nil
}
func (u *adcdbUnit) Stop() error {
	if !u.status.Running {
		return fmt.Errorf("Not running")
	}
	u.status.Running = false

	u.stopSamplerCh <- struct{}{}
	<-u.doneSamplerCh
	u.stopDataPosterCh <- struct{}{}
	<-u.doneDataPosterCh

	u.ProcessorHostImpl.NotifyStopped(u)
	return nil
}

func (u *adcdbUnit) dataArrived(data parent.AcquiredData) {
	u.postDataCh <- data
}

type adcdbRawData struct {
	Raw      []byte
	DataType parent.AcquiredDataType
	parent.FrameData
	Channels []int
	IsFirst  bool
	IsLast   bool
}

type adcdbRawParsedData struct {
	index   int
	parent  *adcdbRawData
	channel int
	data    []byte
}

func (d *adcdbRawData) RawData() parent.ReadOnlyData {
	return parent.ReadOnlyData(d.Raw)
}
func (d *adcdbRawData) Type() parent.AcquiredDataType {
	return d.DataType
}
func (d *adcdbRawData) Parse() []parent.ParsedData {
	channelCount := len(d.Channels)
	parsedDatas := make([]parent.ParsedData, channelCount)
	if channelCount == 0 {
		return parsedDatas
	}
	channelLength := (len(d.Raw) - 8) / channelCount
	for i, ch := range d.Channels {
		parsedDatas[i] = &adcdbRawParsedData{
			index:   i,
			channel: ch,
			data:    d.Raw[8+i*channelLength : 8+(i+1)*channelLength],
			parent:  d,
		}
	}
	return parsedDatas
}
func (d *adcdbRawData) IsFrameFirstData() bool {
	return d.IsFirst
}
func (d *adcdbRawData) IsFrameLastData() bool {
	return d.IsLast
}
func (d *adcdbRawParsedData) Channel() string {
	return fmt.Sprintf("ch%d", d.channel+1)
}
func (d *adcdbRawParsedData) Length() uint64 {
	return uint64(len(d.data))
}
func (d *adcdbRawParsedData) Read(p []byte) (int, error) {
	bytesToRead := min_int(int(d.Length()), len(p) & ^7)
	raw := d.data
	copy(p[:bytesToRead], raw[:bytesToRead])
	return bytesToRead, nil
}
func (d *adcdbRawParsedData) NumberOfItems() int {
	return int(d.Length()) / 8
}
func (d *adcdbRawParsedData) ReadAll() (interface{}, error) {
	samples := d.NumberOfItems()
	result := make([]float64, samples)
	for i := 0; i < samples; i++ {
		bits := binary.LittleEndian.Uint64(d.data[i*8 : (i+1)*8])
		result[i] = math.Float64frombits(bits)
	}
	return result, nil
}
func (d *adcdbRawParsedData) Iterate(from int, toExclusive int, iter func(index int, value float64)) {
	samples := d.NumberOfItems()
	if toExclusive > samples {
		panic(fmt.Errorf("Invalid iteration range. limit:%d, specified:%d", samples, toExclusive))
	}
	for i := from; i < toExclusive; i++ {
		bits := binary.LittleEndian.Uint64(d.data[i*8 : (i+1)*8])
		iter(i, math.Float64frombits(bits))
	}
}

type adcdbSummaryData struct {
	raw     []byte
	isFirst bool
	isLast  bool
	parent.FrameData
}

func (d *adcdbSummaryData) RawData() parent.ReadOnlyData {
	return parent.ReadOnlyData(d.raw)
}
func (d *adcdbSummaryData) Type() parent.AcquiredDataType {
	return parent.TimeSeriesSummary
}
func (d *adcdbSummaryData) IsFrameFirstData() bool {
	return d.isFirst
}
func (d *adcdbSummaryData) IsFrameLastData() bool {
	return d.isLast
}
func (d *adcdbSummaryData) Parse() []parent.ParsedData {
	return nil
}

func (u *adcdbUnit) adcdbSampler() {
	defer func() { u.doneSamplerCh <- struct{}{} }()

	numberOfChannels := len(u.settings.Channels)

	const HeaderSize uint32 = 8
	var headerBuffer [HeaderSize]byte

	frameData := parent.NewFrameData(make(map[string]interface{}))

	for {
		select {
		case <-u.stopSamplerCh:
			return
		default:
			if n, err := u.reader.Read(headerBuffer[:]); err != nil || n < len(headerBuffer) {
				break
			}
			dataType := binary.LittleEndian.Uint16(headerBuffer[0:2])
			flags := binary.LittleEndian.Uint16(headerBuffer[2:4])
			lengthBytes := binary.LittleEndian.Uint32(headerBuffer[4:8])
			switch dataType {
			case 0: // Summary data
				//log.Printf("Summary received, length=%d", lengthBytes)
				buffer := make([]byte, lengthBytes+HeaderSize)
				copy(buffer[:HeaderSize], headerBuffer[:])
				bytesRead := uint32(0)
				for bytesRead < lengthBytes {
					n, err := u.reader.Read(buffer[bytesRead+HeaderSize:])
					if err != nil {
						break
					}
					bytesRead += uint32(n)
				}
				if bytesRead == uint32(2*4*numberOfChannels) {
					acquiredData := adcdbSummaryData{
						raw:     buffer,
						isFirst: false,
						isLast:  false,
					}
					u.dataArrived(&acquiredData)
				}
			case 3: // RAW data
				if lengthBytes <= rawDataSizeLimit {
					buffer := make([]byte, lengthBytes)
					bytesRead := uint32(0)
					for bytesRead < lengthBytes {
						n, err := u.reader.Read(buffer[bytesRead:])
						if err != nil {
							break
						}
						bytesRead += uint32(n)
					}
					if bytesRead == lengthBytes {
						channelMaps := binary.LittleEndian.Uint64(buffer[0:])
						channelCount := 0
						channelIndex := 0
						channels := make([]int, 64)
						// Count number of enabled channels and its index
						for ; channelMaps != 0; channelMaps >>= 1 {
							if (channelMaps & 1) != 0 {
								channels[channelCount] = channelIndex
								channelCount++
							}
							channelIndex++
						}
						channelMaps = binary.LittleEndian.Uint64(buffer[0:])
						//log.Printf("map: %08x, count: %d\n", channelMaps, channelCount)
						// If no channels are enabled, do not send any data.
						if channelCount == 0 {
							break
						}
						//log.Printf("Length - %d\n", lengthBytes)
						if (lengthBytes-8)%uint32(channelCount) != 0 {
							// Invalid length.
							log.Printf("Invalid length - %d\n", (lengthBytes - 8))
							break
						}
						if flags == 1 {
							// New frame data
							metadata := make(map[string]interface{})
							metadata["magicNumber"] = uint32(0x0ADCDB)
							metadata["headerSize"] = uint32(48)
							metadata["dataSize"] = uint32(lengthBytes / uint32(channelCount))
							metadata["startTime"] = uint32(time.Now().Unix())
							metadata["flag"] = uint32(0)
							metadata["code"] = uint32(0)
							metadata["palette"] = uint32(0)
							metadata["pos"] = uint32(0)
							metadata["dataErrCnt"] = uint32(0)
							metadata["fftErrCnt"] = uint32(0)

							frameData = parent.NewFrameData(metadata)
						}
						acquiredData := adcdbRawData{
							Raw:       buffer,
							DataType:  parent.TimeSeries,
							IsFirst:   flags == 1,
							IsLast:    flags == 2,
							Channels:  channels[:channelCount],
							FrameData: frameData,
						}
						u.dataArrived(&acquiredData)
					}
				}
			}
		}
	}
}

func (u *adcdbUnit) adcdbDataPoster() {
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
