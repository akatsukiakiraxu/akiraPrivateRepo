package ml

import (
	"encoding/json"
	"io/ioutil"
)

type MLServerConfigs struct {
	Default        string           `json:"default"` // Default connection
	Configurations []MLServerConfig `json:"configurations"`
}

type MLServerConfig struct {
	Name       string       `json:"name"` // Name of this connection
	Type       MLServerType `json:"type"` // Type of ML Server.
	Connection struct {
		Type    string `json:"type"`    // Connection type.
		Address string `json:"address"` // Address of the ML server
		Port    int    `json:"port"`    // Port of the ML server
	} `json:"connection"`
	Settings map[string]interface{} `json:"settings"`
}

type MLInputType string

const (
	Raw MLInputType = "raw"
	FFT             = "fft"
)

type MLActivationFunction string

const (
	Sigmoid MLActivationFunction = "sigmoid"
	ReLU    MLActivationFunction = "relu"
	Linear  MLActivationFunction = "linear"
)

type MLLossFunction string

const (
	MeanSquaredError MLLossFunction = "mean_squared_error"
	L1Error          MLLossFunction = "l1_error"
)

type InputQuantizationParameters struct {
	Enabled           bool `json:"enabled"`
	NumberOfSectionsX uint `json:"number_of_sections_x"`
	NumberOfSectionsY uint `json:"number_of_sections_y"`
}

type MLServerSettings struct {
	TargetChannels     []string                    `json:"target_channels"`
	InputDataOffset    int                         `json:"input_data_offset"`
	InputDataSize      int                         `json:"input_data_size"`
	TrainingCount      int                         `json:"training_count"`
	TrainedNetworkPath *string                     `json:"trained_network_path"`
	InputType          MLInputType                 `json:"input_type"`
	ActivationFunction MLActivationFunction        `json:"activation_function"`
	LossFunction       MLLossFunction              `json:"loss_function"`
	Quantization       InputQuantizationParameters `json:"quantization"`
}

type MLEventType string

const (
	MLEventTrainingProgress MLEventType = "training-progress"
	MLEventTrainingDone                 = "training-done"
	MLEventInferenceResult              = "inference-result"
)

type MLTrainingEvent struct {
	Type         MLEventType `json:"type"`
	TrainedCount int         `json:"trained_count"`
}

type MLInferenceEvent struct {
	Type      MLEventType `json:"type"`
	LossValue float32     `json:"loss_value"`
}

type MLServerInput []float32

type MLServerEventHandler func(server MLServer, event interface{})

type MLServer interface {
	Running() bool
	Settings() MLServerSettings
	DefaultSettings() MLServerSettings

	Configure(config *MLServerConfig) error
	ChangeSettings(settings *MLServerSettings) error

	Start(handler MLServerEventHandler) error
	Stop() error

	Write(input MLServerInput) error
}

// Load MLServerConfig from file.
// path	Path to the file.
// returns a pair of MLServerConfig and error
func LoadMLServerConfig(path string) (*MLServerConfig, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config *MLServerConfig = new(MLServerConfig)
	if err := json.Unmarshal(bytes, config); err != nil {
		return nil, err
	}
	return config, nil
}
