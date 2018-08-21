package main

import (
	"encoding/json"
	"fmt"
	_ "log"
)

type Employee struct {
	Name   string `json:"empname"`
	Number int    `json:"empid"`
}
type MLInputType string
type MLActivationFunction string
type MLLossFunction string
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

type InputQuantizationParameters struct {
	Enabled           bool `json:"enabled"`
	NumberOfSectionsX uint `json:"number_of_sections_x"`
	NumberOfSectionsY uint `json:"number_of_sections_y"`
}

type MonitoringSettings struct {
	Duration         float32 `json:"duration"`
	HorizontalPoints int     `json:"horizontal_points"`
}

type RecordingSettings struct {
	StoreByLossValue  float32                             `json:"store_by_loss_value"`
	StoreEverySeconds float32                             `json:"store_every_seconds"`
	Channels          map[string]RecordingChannelSettings `json:"channels"`
}

type RecordingChannelSettings struct {
	Enabled bool `json:"enabled"`
}

type UserProfile struct {
	ActiveProfile string                    `json:"active_profile"`
	Profile       map[string]SettingProfile `json:"profile"`
}

type ProfileCRUD struct {
	ProfileName string `json:"profile_name"`
}

type SettingProfile struct {
	Learned bool   `json:"learned"`
	Date    string `json:"date"`
}

type DisplayProfile struct {
	LossThreshold    float32 `json:"loss_threshold"`
	Duration         float32 `json:"duration"`
	HorizontalPoints int     `json:"horizontal_points"`
	SummarizePoints  int     `json:"summarize_points"`
}

const rootDir = "/usr/local/olive"
const defaultUserName = "admin"

func main() {
	ch := make([]string, 1)
	ch[0] = "ch1"
	ml := &MLServerSettings{ch, 0, 768, 256, nil, "raw", "relu", "mean_squared_error", InputQuantizationParameters{false, 0, 0}}
	e, _ := json.Marshal(ml)
	fmt.Println(string(e))

	disp := &DisplayProfile{0, 1, 1000, 250}
	e, _ = json.Marshal(disp)
	fmt.Println(string(e))

}
