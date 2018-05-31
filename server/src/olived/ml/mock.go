package ml

import (
	"fmt"
	"log"
	"time"
)

// Mock ML Server
type mockServer struct {
	running       bool
	settings      MLServerSettings
	trainingCount int

	stopCh chan struct{}
	doneCh chan struct{}
}

func mockDefaultSettings() MLServerSettings {
	return MLServerSettings{
		InputDataSize:      768,
		TrainingCount:      512,
		TrainedNetworkPath: nil,
		InputType:          Raw,
		ActivationFunction: Sigmoid,
		LossFunction:       MeanSquaredError,
	}
}

func newMockServer() *mockServer {
	return &mockServer{
		running:       false,
		settings:      mockDefaultSettings(),
		trainingCount: 0,

		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
}

func (s *mockServer) Running() bool {
	return s.running
}
func (s *mockServer) Configure(config *MLServerConfig) error {
	return nil
}
func (s *mockServer) Settings() MLServerSettings {
	return s.settings
}
func (s *mockServer) DefaultSettings() MLServerSettings {
	return mockDefaultSettings()
}
func (s *mockServer) ChangeSettings(settings *MLServerSettings) error {
	if settings.TrainingCount < 0 || settings.TrainingCount == 0 && settings.TrainedNetworkPath == nil {
		return fmt.Errorf("invalid training count")
	}
	switch settings.InputType {
	case Raw:
		break
	case FFT:
		break
	default:
		return fmt.Errorf("invalid input type - %s", settings.InputType)
	}
	if settings.InputDataSize <= 0 {
		return fmt.Errorf("invalid input data size - %d", settings.InputDataSize)
	}
	if len(settings.TargetChannels) != 1 {
		return fmt.Errorf("invalid number of target channels")
	}

	s.settings = *settings

	return nil
}

func (s *mockServer) Write(input MLServerInput) error {
	return nil
}

func (s *mockServer) Start(handler MLServerEventHandler) error {
	if s.running {
		return fmt.Errorf("running")
	}

	s.running = true
	s.trainingCount = 0

	trainFinishedCh := make(chan struct{})
	go func() {
		for ; s.trainingCount <= s.settings.TrainingCount; s.trainingCount++ {
			time.Sleep(time.Millisecond * 10)
		}
		trainFinishedCh <- struct{}{}
	}()

	go func() {
		defer func() { s.doneCh <- struct{}{} }()

		isTraining := true
		var inferenceCount float32

		trainingTick := time.NewTicker(time.Millisecond * 200)
		inferenceTick := time.NewTicker(time.Second * 2)

		for {
			select {
			case <-s.stopCh:
				return
			case <-trainFinishedCh:
				isTraining = false
				log.Printf("ML training finished")
				e := MLTrainingEvent{"training-done", s.settings.TrainingCount}
				handler(s, e)

			case <-trainingTick.C:
				if isTraining {
					log.Printf("ML in progress %d", s.trainingCount)
					e := MLTrainingEvent{"training-progress", s.trainingCount}
					handler(s, e)
				}
			case <-inferenceTick.C:
				if !isTraining {
					inferenceCount += 1.0 / 32
					log.Printf("ML inference %f", inferenceCount)
					e := MLInferenceEvent{"inference-result", inferenceCount}
					handler(s, e)
					if inferenceCount >= 1.0 {
						inferenceCount = 0
					}
				}
			}
		}
	}()

	return nil
}
func (s *mockServer) Stop() error {
	if !s.running {
		return fmt.Errorf("Not running")
	}
	s.stopCh <- struct{}{}
	<-s.doneCh
	s.running = false

	return nil
}
