package discovery

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"runtime/debug"
	"time"

	parent "olived/acquisition"
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

type discoveryChannelSettings struct {
	Gain      uint16
	Threshold uint16
}

func min_int(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func NewDiscoveryUnit(config *parent.UnitConfig) (parent.Unit, error) {
	var discoveryAddress string
	var discoveryPort float64

	if value, ok := config.Parameters["discovery_address"]; !ok {
		return nil, fmt.Errorf("address of the Discovery endpoint must be specified")
	} else if discoveryAddress, ok = value.(string); !ok {
		return nil, fmt.Errorf("address of the Discovery endpoint must be string")
	}
	if value, ok := config.Parameters["discovery_port"]; !ok {
		return nil, fmt.Errorf("port of the Discovery endpoint must be specified")
	} else if discoveryPort, ok = value.(float64); !ok {
		return nil, fmt.Errorf("port of the Discovery endpoint must be number")
	}

	return &discoveryUnit{
		config: *config,
		status: parent.UnitStatus{
			Running:  false,
			Channels: makeChannelsStatus(config),
		},
		settings: parent.UnitSettings{
			Channels: makeChannelsSettings(config),
		},
		ProcessorHostImpl: parent.NewProcessorHostImpl(),

		discoveryAddress: discoveryAddress,
		discoveryPort:    int(discoveryPort),

		settingsReqCh:  make(chan discoveryChannelSettings),
		settingsDoneCh: make(chan struct{}),

		postDataCh: make(chan parent.AcquiredData, 10),

		stopDataPosterCh: make(chan struct{}),
		doneDataPosterCh: make(chan struct{}),
	}, nil
}

type discoveryUnit struct {
	parent.ProcessorHostImpl

	config   parent.UnitConfig
	status   parent.UnitStatus
	settings parent.UnitSettings

	settingsReqCh  chan discoveryChannelSettings
	settingsDoneCh chan struct{}

	postDataCh chan parent.AcquiredData

	discoveryAddress string
	discoveryPort    int

	stopSamplerCh chan struct{}
	doneSamplerCh chan struct{}

	stopDataPosterCh chan struct{}
	doneDataPosterCh chan struct{}
}

func (u *discoveryUnit) Status() parent.UnitStatus {
	return u.status
}
func (u *discoveryUnit) Config() parent.UnitConfig {
	return u.config
}
func (u *discoveryUnit) Settings() parent.UnitSettings {
	return u.settings
}

func (u *discoveryUnit) Start() error {
	if u.status.Running {
		return fmt.Errorf("Already running")
	}

	log.Printf("Discovery endpoint: %s:%d", u.discoveryAddress, u.discoveryPort)

	go u.discoverySampler()
	go u.discoveryDataPoster()

	u.ProcessorHostImpl.NotifyStarted(u)
	u.status.Running = true
	return nil
}
func (u *discoveryUnit) ChangeSettings(settings *parent.UnitSettings) error {

	if err := u.ProcessorHostImpl.NotifySettingsChanging(u, settings); err != nil {
		return err
	}

	u.settings.Enabled = settings.Enabled
	u.settings.Comparing = settings.Comparing
	var channelSettings *parent.ChannelSettings
	for name, channel := range settings.Channels {
		oldStatus, ok := u.status.Channels[name]
		if !ok {
			continue
		}
		u.status.Channels[name] = parent.ChannelStatus{
			Enabled:        channel.Enabled,
			MinimumVoltage: oldStatus.MinimumVoltage,
			MaximumVoltage: oldStatus.MaximumVoltage,
			MinimumValue:   oldStatus.MinimumValue,
			MaximumValue:   oldStatus.MaximumValue,
		}
		u.settings.Channels[name] = channel
		channelSettings = &channel
	}

	// Update discovery channel settings if running
	if u.status.Running {
		discoveryChannelSettings := discoveryChannelSettings{}
		if value, ok := channelSettings.Parameters["discovery/gain"]; ok {
			if gain, ok := value.(float64); ok {
				discoveryChannelSettings.Gain = uint16(gain)
			}
		}
		if value, ok := channelSettings.Parameters["discovery/threshold"]; ok {
			if threshold, ok := value.(float64); ok {
				discoveryChannelSettings.Threshold = uint16(threshold)
			}
		}
		log.Printf("Discovery channel settings: gain=%d, threshold=%d", discoveryChannelSettings.Gain, discoveryChannelSettings.Threshold)
		u.settingsReqCh <- discoveryChannelSettings
		<-u.settingsDoneCh
	}
	u.ProcessorHostImpl.NotifySettingsChanged(u, settings)
	return nil
}
func (u *discoveryUnit) Stop() error {
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

func (u *discoveryUnit) dataArrived(data parent.AcquiredData) {
	u.postDataCh <- data
}

type discoveryRawData struct {
	Raw      []byte
	DataType parent.AcquiredDataType
	parent.FrameData
	Channels []int
	IsFirst  bool
	IsLast   bool
}

type discoveryRawParsedData struct {
	index   int
	parent  *discoveryRawData
	channel int
	data    []byte
}

func (d *discoveryRawData) RawData() parent.ReadOnlyData {
	return parent.ReadOnlyData(d.Raw)
}
func (d *discoveryRawData) Type() parent.AcquiredDataType {
	return d.DataType
}
func (d *discoveryRawData) Parse() []parent.ParsedData {
	channelCount := 1
	parsedDatas := make([]parent.ParsedData, channelCount)
	parsedDatas[0] = &discoveryRawParsedData{
		index:   0,
		channel: 0,
		data:    d.Raw,
		parent:  d,
	}

	return parsedDatas
}
func (d *discoveryRawData) IsFrameFirstData() bool {
	return d.IsFirst
}
func (d *discoveryRawData) IsFrameLastData() bool {
	return d.IsLast
}

func (d *discoveryRawParsedData) Channel() string {
	return fmt.Sprintf("ch%d", d.channel+1)
}
func (d *discoveryRawParsedData) Length() uint64 {
	return uint64(len(d.data))
}
func (d *discoveryRawParsedData) Read(p []byte) (int, error) {
	bytesToRead := min_int(int(d.Length()), len(p) & ^7)
	raw := d.data
	copy(p[:bytesToRead], raw[:bytesToRead])
	return bytesToRead, nil
}
func (d *discoveryRawParsedData) NumberOfItems() int {
	return int(d.Length()) / 2
}
func (d *discoveryRawParsedData) ReadAll() (interface{}, error) {
	samples := d.NumberOfItems()
	result := make([]float64, samples)
	for i := 0; i < samples; i++ {
		result[i] = float64(binary.LittleEndian.Uint16(d.data[i*2 : (i+1)*2]))
	}
	return result, nil
}
func (d *discoveryRawParsedData) Iterate(from int, toExclusive int, iter func(index int, value float64)) {
	for i := from; i < toExclusive; i++ {
		iter(i, float64(binary.LittleEndian.Uint16(d.data[i*2:(i+1)*2])))
	}
}

func (u *discoveryUnit) discoverySampler() {
	defer func() {
		if err := recover(); err != nil {
			log.Fatalf("Discovery sampler has paniced. %+v, %s", err, debug.Stack())
		}
	}()
	defer func() { u.doneSamplerCh <- struct{}{} }()

	const BufferSize uint32 = 8 * 1024 * 1024
	const HeaderSize = 16
	const Signature = uint32(0x74733181) // ts1?

	var buffer []byte
	localAddress, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", u.discoveryAddress, u.discoveryPort))
	if err != nil {
		log.Printf("Error: failed to resolve server address %s", err.Error())
		return
	}
	listener, err := net.ListenTCP("tcp4", localAddress)
	if err != nil {
		log.Printf("Error: failed to create TCP connection %s", err.Error())
		return
	}
	defer listener.Close()
	var conn *net.TCPConn
	connCh := make(chan *net.TCPConn)
	go func() {
		var settings discoveryChannelSettings
		var conn *net.TCPConn
		sendRequest := func() {
			if conn == nil {
				return
			}
			conn.SetWriteDeadline(time.Now().Add(time.Second))
			settingsBuffer := make([]byte, 0x18)
			binary.LittleEndian.PutUint32(settingsBuffer[0x00:0x04], Signature)
			binary.LittleEndian.PutUint32(settingsBuffer[0x04:0x08], 2)                  // Type
			binary.LittleEndian.PutUint32(settingsBuffer[0x08:0x0c], 0)                  // Flags
			binary.LittleEndian.PutUint32(settingsBuffer[0x0c:0x10], 0x08)               // PayloadLength
			binary.LittleEndian.PutUint32(settingsBuffer[0x10:0x14], 0)                  // ChannelIndex
			binary.LittleEndian.PutUint16(settingsBuffer[0x14:0x16], settings.Gain)      // Gain
			binary.LittleEndian.PutUint16(settingsBuffer[0x16:0x18], settings.Threshold) // Threshold
			var bytesWritten int
			for bytesWritten < len(settingsBuffer) {
				if n, err := conn.Write(settingsBuffer); err != nil {
					log.Printf("Send error: %s", err.Error())
					break
				} else {
					bytesWritten += n
				}
			}
		}
		for {
			select {
			case settings_ := <-u.settingsReqCh:
				settings = settings_
				sendRequest()
				u.settingsDoneCh <- struct{}{}
				log.Printf("Send request")
			case conn_, ok := <-connCh:
				if !ok {
					return
				}
				conn = conn_
				sendRequest()
			}
		}
	}()
	log.Printf("Listening")
	for {
		select {
		case <-u.stopSamplerCh:
			close(connCh)
			return

		default:
			if conn == nil {
				listener.SetDeadline(time.Now().Add(time.Second))
				conn, err = listener.AcceptTCP()
				if err != nil {
					log.Printf("Failed, %v", err)
					conn = nil
					continue
				}
				log.Printf("Connected from discovery.")
				connCh <- conn
			}
			if buffer == nil {
				buffer = make([]byte, BufferSize)
			}
			conn.SetReadDeadline(time.Now().Add(time.Second))
			var bytesRead uint32
			log.Printf("Waiting for incoming data...")
			if n, err := conn.Read(buffer[:HeaderSize]); err != nil || n < HeaderSize {
				if netError, ok := err.(net.Error); ok {
					if !netError.Timeout() {
						conn.Close()
						conn = nil
					}
				} else if err == io.EOF {
					conn.Close()
					conn = nil
				}
				log.Printf("Read error %v", err)
				continue
			} else {
				bytesRead += uint32(n)
			}

			signature := binary.LittleEndian.Uint32(buffer[0:4])
			packetType := binary.LittleEndian.Uint32(buffer[4:8])
			flags := binary.LittleEndian.Uint32(buffer[8:12])
			payloadLength := binary.LittleEndian.Uint32(buffer[12:16])
			log.Printf("Received %08x, %d, %08x, %d, %d\n", signature, packetType, flags, payloadLength, bytesRead)
			if signature != Signature {
				log.Printf("Invalid signature") // Invalid signature
				continue                        // Invalid signature
			}

			// Fill payload
			for bytesRead < payloadLength+HeaderSize {
				conn.SetReadDeadline(time.Now().Add(time.Millisecond * 1000))
				n, err := conn.Read(buffer[bytesRead : payloadLength+HeaderSize])
				if err != nil {
					log.Printf("Read error - %v", err)
					break
				}
				bytesRead += uint32(n)
			}
			if bytesRead != payloadLength+HeaderSize {
				// Not enough data

				log.Printf("Not enough data, received=%d", bytesRead)
				continue
			}

			switch packetType {
			case 0: // Data packet
				if payloadLength < 4 {
					continue // Data packet must have at least sequence number
				}
				// sequence := binary.LittleEndian.Uint32(buffer[16:20])
				acquiredData := discoveryRawData{
					Raw:       buffer[HeaderSize+4 : HeaderSize+payloadLength],
					DataType:  parent.TimeSeries,
					IsFirst:   (flags & 1) != 0,
					IsLast:    (flags & 2) != 0,
					FrameData: parent.NewFrameData(make(map[string]interface{})),
				}
				u.postDataCh <- &acquiredData
			case 1: // Response packet
				if payloadLength < 8 {
					continue
				}
				targetType := binary.LittleEndian.Uint32(buffer[HeaderSize+0 : HeaderSize+4])
				code := binary.LittleEndian.Uint32(buffer[HeaderSize+4 : HeaderSize+8])
				log.Printf("Response received, len=%d, targetType=0x%x, code=0x%x", payloadLength, targetType, code)
			}

		}
	}
}

func (u *discoveryUnit) discoveryDataPoster() {
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
