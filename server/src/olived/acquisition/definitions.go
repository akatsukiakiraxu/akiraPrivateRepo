package acquisition

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"olived/core"
)

// CouplingType Channel input coupling type
type CouplingType string

const (
	// DCCoupling - DC Coupling
	DCCoupling CouplingType = "DC"
	// ACCoupling - AC Coupling
	ACCoupling CouplingType = "AC"
)

type TriggerMode int

const (
	Disabled TriggerMode = 0
	Rising               = 1
	Falling              = 2
	Both                 = 3
)

func (m TriggerMode) String() string {
	switch m {
	case Disabled:
		return "Disabled"
	case Rising:
		return "Rising"
	case Falling:
		return "Falling"
	case Both:
		return "Both"
	default:
		return fmt.Sprintf("Unknown(%d)", m)
	}
}
func (m *TriggerMode) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	switch strings.ToLower(s) {
	default:
		return fmt.Errorf("undefined TriggerMode: %s", s)
	case "disabled":
		*m = Disabled
	case "rising":
		*m = Rising
	case "falling":
		*m = Falling
	case "both":
		*m = Both
	}
	return nil
}
func (m TriggerMode) MarshalJSON() ([]byte, error) {
	s := strings.ToLower(m.String())
	return json.Marshal(s)
}

type SignalInputType int

const (
	SingleEnded  SignalInputType = 0
	Differential                 = 1
)

func (m SignalInputType) String() string {
	switch m {
	case SingleEnded:
		return "Single_ended"
	case Differential:
		return "Differential"
	default:
		return fmt.Sprintf("Unknown(%d)", m)
	}
}
func (m *SignalInputType) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	switch strings.ToLower(s) {
	default:
		return fmt.Errorf("undefined SignalInputType: %s", s)
	case "single_ended":
		*m = SingleEnded
	case "differential":
		*m = Differential
	}
	return nil
}

func (m SignalInputType) MarshalJSON() ([]byte, error) {
	s := strings.ToLower(m.String())
	return json.Marshal(s)
}

type ValueRange struct {
	MinimumVoltage float64 `json:"minimum_voltage"`
	MaximumVoltage float64 `json:"maximum_voltage"`
	MinimumValue   float64 `json:"minimum_value"`
	MaximumValue   float64 `json:"maximum_value"`
}

// ChannelConfig - Configuration of a channel in an acquisition unit.
type ChannelConfig struct {
	SamplingRatesSupported []float32      `json:"sampling_rates_supported"`
	RangesSupported        []string       `json:"ranges_supported"`
	CouplingsSupported     []CouplingType `json:"couplings_supported"`
}

// UnitConfig - Configuration of an acquisition unit.
type UnitConfig struct {
	Name       string                   `json:"name"`       // Name of this unit.
	Type       string                   `json:"type"`       // Type of this unit. See units/factory.go for available unit types.
	Ranges     map[string]ValueRange    `json:"ranges"`     // Ranges used in this unit.
	Channels   map[string]ChannelConfig `json:"channels"`   // Channels in this unit.
	Parameters map[string]interface{}   `json:"parameters"` // Parameters specific to the unit type .
}

type UnitStatus struct {
	Running  bool                     `json:"running"`
	Channels map[string]ChannelStatus `json:"channels"`
}

type ChannelStatus struct {
	Enabled        bool    `json:"enabled"`
	MinimumVoltage float64 `json:"minimum_voltage"`
	MaximumVoltage float64 `json:"maximum_voltage"`
	MinimumValue   float64 `json:"minimum_value"`
	MaximumValue   float64 `json:"maximum_value"`
}

// ChannelSettings - Settings of a channel in an acquisition unit.
type ChannelSettings struct {
	Enabled         bool                   `json:"enabled"`
	SamplingRate    float32                `json:"sampling_rate"`
	Range           string                 `json:"range"`
	Coupling        CouplingType           `json:"coupling"`
	RecordRange     float32                `json:"recordRange"`
	SignalInputType SignalInputType        `json:"signal_input_type"`
	Parameters      map[string]interface{} `json:"parameters"`
}

type TriggerSettings struct {
	ChannelName string      `json:"channel_name"`
	Threshold   float64     `json:"threshold"`
	Hysteresis  float64     `json:"hysteresis"`
	DeadTime    float64     `json:"dead_time"`
	Timeout     float64     `json:"timeout"`
	Mode        TriggerMode `json:"trigger_mode"`
}
type UnitSettings struct {
	Enabled   bool                       `json:"enabled"`
	Comparing bool                       `json:"comparing"`
	Trigger   TriggerSettings            `json:"trigger"`
	Channels  map[string]ChannelSettings `json:"channels"`
}

// ReadOnlyData Read only data.
// This type is actually an alias of []byte and you can cast instances of this type to []byte.
// But modifying the contents of the instance is strictly prohibited.
type ReadOnlyData []byte
type AcquiredDataType string

const (
	TimeSeries        AcquiredDataType = "time"
	TimeSeriesSummary                  = "time_summary"
	FFT                                = "fft"
)

type AcquiredData interface {
	RawData() ReadOnlyData
	Type() AcquiredDataType
	Parse() []ParsedData

	FrameId() uuid.UUID
	FrameMetadata() map[string]interface{}

	IsFrameFirstData() bool
	IsFrameLastData() bool
}

type FrameData struct {
	uuid     uuid.UUID
	metadata map[string]interface{}
}

func NewFrameData(metadata map[string]interface{}) FrameData {
	return FrameData{
		uuid:     uuid.New(),
		metadata: metadata,
	}
}
func GetFrameData(acquiredData AcquiredData) *FrameData {
	return &FrameData{
		uuid:     acquiredData.FrameId(),
		metadata: acquiredData.FrameMetadata(),
	}
}
func (d *FrameData) FrameData() *FrameData {
	return d
}
func (d *FrameData) FrameId() uuid.UUID {
	return d.uuid
}
func (d *FrameData) FrameMetadata() map[string]interface{} {
	return d.metadata
}

type ParsedData interface {
	Channel() string
	Length() uint64
	Read(p []byte) (int, error)

	NumberOfItems() int
	ReadAll() (interface{}, error)
}

type TimeSeriesParsedData interface {
	ParsedData
	Iterate(from int, toExclusive int, iter func(index int, value float64))
}

type TimeSeriesSummaryPoint struct {
	Summary core.RangeF32
	Error   core.RangeF32
	Warning core.RangeF32
}
type TimeSeriesSummaryParsedData interface {
	ParsedData
	Iterate(from int, toExclusive int, iter func(index int, value *TimeSeriesSummaryPoint))
}

type FFTParsedData interface {
	ParsedData
	Resolution() float32
}

// ProcessorType Type of processor
type ProcessorType string

const (
	Recorder ProcessorType = "recorder"
	Monitor                = "monitor"
	Analyzer               = "analyzer"
	Filter                 = "filter"
)

type Processor interface {
	Type() ProcessorType
	Started(unit Unit)
	Stopped(unit Unit)
	// SettingsChanging
	// Settings of the parent unit are changing.
	// Processors can return error if they cannot accept the settings.
	SettingsChanging(unit Unit, settings *UnitSettings) error
	SettingsChanged(unit Unit, settings *UnitSettings)
	DataArrived(unit Unit, data AcquiredData)
}

type ProcessorHost interface {
	AddProcessor(processor Processor) error
	RemoveProcessor(processor Processor)
	Processors() []Processor
}

type Unit interface {
	Status() UnitStatus
	Config() UnitConfig
	Settings() UnitSettings

	Start() error
	ChangeSettings(settings *UnitSettings) error
	Stop() error

	ProcessorHost
}
