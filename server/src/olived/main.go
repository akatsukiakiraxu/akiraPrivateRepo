package main

import (
	"flag"
	"fmt"
	"sync"
	"time"

	"io/ioutil"
	"log"
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

	"encoding/binary"
	"io"
	"olived/acquisition"
	"olived/acquisition/filters/fft"
	"olived/acquisition/filters/summarizer"
	"olived/acquisition/units"
	"olived/ml"
	"olived/recording"
	"olived/recording/filters/interval"
	"os"
	"path/filepath"
)

type dataFile struct {
	Name string `json:"name"`
}

type deleteDataList struct {
	Files *[]string `json:"delete_files"`
}

type saveDataList struct {
	Path  *string   `json:"save_path"`
	Files *[]string `json:"save_files"`
}

// MLParameters - Parameters for ML server
type MLParameters struct {
	TargetChannels     *[]string                       `json:"target_channels"`
	InputDataOffset    *float64                        `json:"input_data_offset"`
	InputDataSize      *float64                        `json:"input_data_size"`
	TrainingCount      *int                            `json:"training_count"`
	UseTrainedNetwork  *bool                           `json:"use_trained_network"`
	TrainedNetworkPath *string                         `json:"trained_network_path"`
	InputType          *string                         `json:"input_type"`
	LossThreshold      *float32                        `json:"loss_threshold"`
	Quantization       *ml.InputQuantizationParameters `json:"quantization"`
}

type UnitStatus struct {
	IsAcquiring  bool `json:"is_acquiring"`
	IsMonitoring bool `json:"is_monitoring"`
	IsMLRunning  bool `json:"is_ml_running"`
}

type ChannelSettings struct {
	Enabled         *bool                        `json:"enabled"`
	SamplingRate    *float32                     `json:"sampling_rate"`
	Range           *string                      `json:"range"`
	Coupling        *acquisition.CouplingType    `json:"coupling"`
	SignalInputType *acquisition.SignalInputType `json:"signal_input_type"`
	Parameters      map[string]interface{}       `json:parameters`
}
type UnitSettings struct {
	Enabled   *bool                        `json:"enabled"`
	Comparing *bool                        `json:"comparing"`
	Trigger   *acquisition.TriggerSettings `json:"trigger"`
	Channels  map[string]ChannelSettings   `json:"channels"`
}

type MonitoringSettings struct {
	Duration         float32 `json:"duration"`
	HorizontalPoints int     `json:"horizontal_points"`
}

type RecordingSettings struct {
	StoreAlways       bool                                `json:"store_always"`
	StoreByLossValue  float32                             `json:"store_by_loss_value"`
	StoreEverySeconds float32                             `json:"store_every_seconds"`
	Channels          map[string]RecordingChannelSettings `json:"channels"`
}
type RecordingChannelSettings struct {
	Enabled bool `json:"enabled"`
}

// fillMLParameters - Fills MLParameters by current MLServerSettings.
func fillMLParameters(target *MLParameters, current *ml.MLServerSettings, acquisitionSettings *acquisition.UnitSettings) error {
	targetChannels := make([]string, len(current.TargetChannels))
	copy(targetChannels, current.TargetChannels)
	inputDataOffset := float64(current.InputDataOffset)
	inputDataSize := float64(current.InputDataSize)
	if len(current.TargetChannels) > 0 {
		if current.InputType == ml.Raw {
			targetChannel := current.TargetChannels[0]
			channelSettings, ok := acquisitionSettings.Channels[targetChannel]
			if !ok {
				return fmt.Errorf("invalid target channel - %s", targetChannel)
			}
			if channelSettings.SamplingRate > 0 {
				inputDataOffset /= float64(channelSettings.SamplingRate)
				inputDataSize /= float64(channelSettings.SamplingRate)
			}
		}

	}
	trainingCount := current.TrainingCount
	useTrainedNetwork := current.TrainedNetworkPath != nil
	trainedNetworkPath := current.TrainedNetworkPath
	inputType := string(current.InputType)
	quantization := current.Quantization

	target.TargetChannels = &targetChannels
	target.InputDataOffset = &inputDataOffset
	target.InputDataSize = &inputDataSize
	target.TrainingCount = &trainingCount
	target.UseTrainedNetwork = &useTrainedNetwork
	target.TrainedNetworkPath = trainedNetworkPath
	target.InputType = &inputType
	target.Quantization = &quantization
	return nil
}

// updateMLServerSettings - Update MLServerSettings by MLParameters from client.
func updateMLServerSettings(target *ml.MLServerSettings, params *MLParameters, acquisitionSettings *acquisition.UnitSettings) error {
	if params.InputType == nil {
		return fmt.Errorf("input_type must be specified")
	}
	target.InputType = ml.MLInputType(*params.InputType)

	if params.TargetChannels != nil {
		target.TargetChannels = *params.TargetChannels
	}
	if len(target.TargetChannels) > 0 {
		targetChannel := target.TargetChannels[0]
		channelSettings, ok := acquisitionSettings.Channels[targetChannel]
		if !ok {
			return fmt.Errorf("invalid target channel - %s", targetChannel)
		}
		if params.InputDataOffset != nil {
			if target.InputType == ml.Raw {
				target.InputDataOffset = int(*params.InputDataOffset * float64(channelSettings.SamplingRate))
			} else {
				target.InputDataOffset = int(*params.InputDataOffset)
			}
		}
		if params.InputDataSize != nil {
			if target.InputType == ml.Raw {
				target.InputDataSize = int(*params.InputDataSize * float64(channelSettings.SamplingRate))
			} else {
				target.InputDataSize = int(*params.InputDataSize)
			}
		}
	}
	if params.TrainingCount != nil {
		target.TrainingCount = *params.TrainingCount
	}
	if params.Quantization != nil {
		target.Quantization = *params.Quantization
	}

	useTrainedNetwork := target.TrainedNetworkPath != nil
	if params.UseTrainedNetwork != nil {
		useTrainedNetwork = *params.UseTrainedNetwork
	}
	if useTrainedNetwork {
		if params.TrainedNetworkPath != nil {
			target.TrainedNetworkPath = params.TrainedNetworkPath
		}
	} else {
		target.TrainedNetworkPath = nil
	}

	return nil
}
func apiServer(dataDir *string, acqConfig *acquisition.UnitConfig, acqSettings *acquisition.UnitSettings, mlServerConfig *ml.MLServerConfig, enableDebugRecorder bool, disableFFTFilter bool, enableFrameRecorder bool) {

	acqUnit, err := units.NewUnit(acqConfig)
	if err != nil {
		panic(err)
	}
	// Use unit default settings if acquisition.settings.json is not exist.
	if acqSettings == nil {
		settings := acqUnit.Settings()
		acqSettings = &settings
	}

	monServer := NewMonitoringServer()
	monSettings := MonitoringSettings{
		Duration:         1.0,
		HorizontalPoints: 1000,
	}

	var firstChannelName string
	for name, _ := range acqSettings.Channels {
		firstChannelName = name
		break
	}

	mlInnerServer, err := ml.NewMLServer(mlServerConfig)
	if err != nil {
		panic(err)
	}
	var mlLossThreshold float32
	mlSettings := mlInnerServer.DefaultSettings()
	mlSettings.TargetChannels = []string{firstChannelName}
	if err := mlInnerServer.ChangeSettings(&mlSettings); err != nil {
		panic(err)
	}

	mlServer := ml.NewWSServer(mlInnerServer, "/ml/event")

	// Recording settings
	// TODO: implement RecordingController and manage settings by instance of it.
	recordingSettings := RecordingSettings{
		StoreAlways:       true,
		StoreByLossValue:  0,
		StoreEverySeconds: 0,
		Channels:          make(map[string]RecordingChannelSettings, len(acqConfig.Channels)),
	}
	for name, _ := range acqConfig.Channels {
		recordingSettings.Channels[name] = RecordingChannelSettings{Enabled: true}
	}
	recordingController := recording.NewRecordingController()
	// Construct store always filter.
	storeAlwaysFilter := &recording.FuncRecordingFilter{
		Func: func(controller recording.RecordingController, frameData *acquisition.FrameData) recording.FilterResult {
			if recordingSettings.StoreAlways {
				return recording.FilterResult_Approved
			}
			return recording.FilterResult_NotInterested
		},
	}
	// Construct interval filter.
	intervalFilter := interval.NewIntervalFilter(nil)
	intervalFilter.SetInterval(time.Duration(float64(time.Second) * float64(recordingSettings.StoreEverySeconds)))
	// Add recording filters.
	recordingController.AddFilter(intervalFilter)
	recordingController.AddFilter(storeAlwaysFilter)

	// Instanciate a processor adapter for the ML server.
	mlProcessor := ml.NewProcessorAdapter(mlServer)

	// Instanciate FFT filter processor
	fftFilter := fft.NewFftFilterProcessor()
	fftFilter.TargetChannel = "ch1"
	fftFilter.NumberOfPoints = 1024
	fftFilter.Window = fft.Hann

	// Instanciate summarizer
	summarizer := summarizer.NewSummarizeProcessor()
	summarizer.NumberOfPoints = 200

	// Construct recorder
	recorder := recording.NewSimpleRecorder("debug")
	recorderPF := recording.NewSequentialFrameRecorder(*dataDir, acqUnit, recordingController)

	// Add monitoring server and ML server to acquisition unit's processor list
	acqUnit.AddProcessor(monServer)
	acqUnit.AddProcessor(mlProcessor)
	acqUnit.AddProcessor(fftFilter)
	acqUnit.AddProcessor(recorder)
	acqUnit.AddProcessor(summarizer)
	acqUnit.AddProcessor(recorderPF)

	fftFilter.AddProcessor(monServer)
	fftFilter.AddProcessor(mlProcessor)
	fftFilter.AddProcessor(recorder)
	summarizer.AddProcessor(monServer)

	// Apply current settings
	acqUnit.ChangeSettings(acqSettings)

	mlServerLock := sync.Mutex{}

	e := echo.New()
	// CORS default
	// Allows requests from any origin wth GET, HEAD, PUT, POST or DELETE method.
	e.Use(middleware.CORS())

	e.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, "Alive.\n")
	})
	e.GET("/time/:datetime", func(c echo.Context) error {
		log.Println("set time")
		return c.String(http.StatusOK, "OK.\n")
	})
	e.GET("/rename", func(c echo.Context) error {
		log.Println("rename called...")
		return c.String(http.StatusOK, "OK.\n")
	})
	e.GET("/poweroff", func(c echo.Context) error {
		log.Println("Poweroff called...")
		return c.String(http.StatusOK, "OK.\n")
	})
	e.GET("/datafiles", func(c echo.Context) error {
		files, err := ioutil.ReadDir(*dataDir)
		if err != nil {
			return err
		}

		var jsonfiles []dataFile
		for _, f := range files {
			jsonfiles = append(jsonfiles, dataFile{f.Name()})
		}
		return c.JSON(http.StatusOK, jsonfiles)
	})

	e.POST("/datafiles/delete", func(c echo.Context) error {
		returnCode := http.StatusOK
		if dataDir != nil && *dataDir != "" {
			deleteList := new(deleteDataList)
			if err := c.Bind(deleteList); err != nil {
				return c.String(http.StatusBadRequest, "Invalid parameters.")
			}
			for _, file := range *deleteList.Files {
				filePath := *dataDir + "/" + file
				globFiles, err := filepath.Glob(filePath)
				if err != nil {
					log.Println(err)
					returnCode = http.StatusForbidden
					break
				}
				for _, f := range globFiles {
					if err := os.Remove(f); err != nil {
						log.Println(err)
						returnCode = http.StatusForbidden
						break
					}
				}

			}
		}
		return c.JSON(returnCode, "finished delete.\n")
	})

	e.POST("/datafiles/save", func(c echo.Context) error {
		returnCode := http.StatusOK
		if dataDir != nil && *dataDir != "" {
			saveList := new(saveDataList)
			if err := c.Bind(saveList); err != nil {
				return c.String(http.StatusBadRequest, "Invalid parameters.")
			}
			for _, file := range *saveList.Files {
				filePath := *dataDir + "/" + file
				globFiles, err := filepath.Glob(filePath)
				if err != nil {
					log.Println(err)
					returnCode = http.StatusForbidden
					break
				}
				for _, f := range globFiles {
					from, err := os.Open(f)
					if err != nil {
						log.Fatal(err)
					}
					defer from.Close()
					filename := filepath.Base(f)
					to, err := os.OpenFile(*saveList.Path+filename, os.O_RDWR|os.O_CREATE, 0666)
					if err != nil {
						log.Fatal(err)
					}
					defer to.Close()
					_, err = io.Copy(to, from)
					if err != nil {
						log.Fatal(err)
					}
				}

			}
		}
		return c.JSON(returnCode, "finished save.\n")
	})

	e.GET("/csv/*", func(c echo.Context) error {
		filename := *dataDir + "/" + c.ParamValues()[0]
		file, err := os.Open(filename)
		if err != nil {
			return err
		}
		defer file.Close()
		file.Seek(4, 0)

		hlen_bytes := make([]byte, 4)
		_, err = file.Read(hlen_bytes)
		if err != nil {
			return err
		}
		hlen := binary.LittleEndian.Uint32(hlen_bytes)

		file.Seek(int64(hlen), 1)

		return c.Stream(200, "application/octet-stream", file)
	})

	e.GET("/ml/start", func(c echo.Context) error {
		mlServerLock.Lock()
		defer mlServerLock.Unlock()

		log.Printf("Starting ML server with these parameters: %+v", mlSettings)
		if err := mlServer.Start(func(server ml.MLServer, event interface{}) {}); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}
		return c.String(http.StatusOK, "OK")
	})
	e.GET("/ml/settings/get", func(c echo.Context) error {
		settings := mlServer.Settings()
		acqSettings := acqUnit.Settings()
		params := MLParameters{
			LossThreshold: &mlLossThreshold,
		}
		if err := fillMLParameters(&params, &settings, &acqSettings); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		return c.JSON(http.StatusOK, params)
	})
	e.POST("/ml/settings/set", func(c echo.Context) error {
		params := new(MLParameters)
		if err := c.Bind(params); err != nil {
			return c.String(http.StatusBadRequest, "Invalid parameters.")
		}
		settings := mlServer.Settings()
		acqSettings := acqUnit.Settings()
		if err := updateMLServerSettings(&settings, params, &acqSettings); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}
		if err := mlServer.ChangeSettings(&settings); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}
		if params.LossThreshold != nil {
			mlLossThreshold = *params.LossThreshold
		}
		return c.String(http.StatusOK, "OK")
	})

	e.GET("/ml/stop", func(c echo.Context) error {
		mlServerLock.Lock()
		defer mlServerLock.Unlock()

		if err := mlServer.Stop(); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}
		mlServer.Stop()
		return c.String(http.StatusOK, "OK")
	})

	e.GET("/acquisition/start", func(c echo.Context) error {
		if err := acqUnit.Start(); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}
		return c.String(http.StatusOK, "OK")
	})
	e.GET("/acquisition/stop", func(c echo.Context) error {
		if err := acqUnit.Stop(); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}
		return c.String(http.StatusOK, "OK")
	})
	e.GET("/acquisition/config/get", func(c echo.Context) error {
		config := acqUnit.Config()
		return c.JSON(http.StatusOK, config)
	})
	e.GET("/acquisition/status", func(c echo.Context) error {
		status := acqUnit.Status()
		return c.JSON(http.StatusOK, status.Channels)
	})
	e.GET("/acquisition/settings/get", func(c echo.Context) error {
		status := acqUnit.Settings()
		return c.JSON(http.StatusOK, status)
	})
	e.POST("/acquisition/settings/set", func(c echo.Context) error {
		settings := new(UnitSettings)
		if err := c.Bind(settings); err != nil {
			return c.String(http.StatusBadRequest, "Invalid parameters.")
		}
		updated := acqUnit.Settings()
		if settings.Enabled != nil {
			updated.Enabled = *settings.Enabled
		}
		if settings.Comparing != nil {
			updated.Comparing = *settings.Comparing
		}
		if settings.Trigger != nil {
			updated.Trigger = *settings.Trigger
		}
		// Update settings by parameters.
		for name, setting := range settings.Channels {
			updatedChannel := updated.Channels[name]
			if setting.Enabled != nil {
				updatedChannel.Enabled = *setting.Enabled
			}
			if setting.SamplingRate != nil {
				updatedChannel.SamplingRate = *setting.SamplingRate
			}
			if setting.Range != nil {
				updatedChannel.Range = *setting.Range
			}
			if setting.Coupling != nil {
				updatedChannel.Coupling = *setting.Coupling
			}
			if setting.SignalInputType != nil {
				updatedChannel.SignalInputType = *setting.SignalInputType
			}
			if setting.Parameters != nil {
				updatedChannel.Parameters = setting.Parameters
			}
			updated.Channels[name] = updatedChannel
		}
		if err := acqUnit.ChangeSettings(&updated); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}
		return c.String(http.StatusOK, "OK")
	})
	e.GET("/monitoring/start", func(c echo.Context) error {
		if err := monServer.Start(); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}
		return c.String(http.StatusOK, "OK")
	})
	e.GET("/monitoring/stop", func(c echo.Context) error {
		if err := monServer.Stop(); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}
		return c.String(http.StatusOK, "OK")
	})
	e.GET("/monitoring/settings/get", func(c echo.Context) error {
		return c.JSON(http.StatusOK, monSettings)
	})
	e.POST("/monitoring/settings/set", func(c echo.Context) error {
		settings := new(MonitoringSettings)
		if err := c.Bind(settings); err != nil {
			return c.String(http.StatusBadRequest, "Invalid parameters.")
		}
		if settings.HorizontalPoints <= 0 {
			return c.String(http.StatusBadRequest, "Horizontal points must be greater than 0.")
		}
		// TODO: Update appropriate summarizer and channel
		acqSettings := acqUnit.Settings()
		samplingRate := acqSettings.Channels["ch1"].SamplingRate
		log.Printf("SamplingRate: %f, Duration: %f, HorizontalPoints: %d\n", samplingRate, settings.Duration, settings.HorizontalPoints)
		summarizePoints := samplingRate * settings.Duration / float32(settings.HorizontalPoints)
		if summarizePoints < 1 {
			return c.String(http.StatusBadRequest, "Invalid number of points")
		}
		summarizer.NumberOfPoints = int(summarizePoints)
		monSettings = *settings
		return c.String(http.StatusOK, "OK")
	})
	e.GET("/recording/settings/get", func(c echo.Context) error {
		return c.JSON(http.StatusOK, recordingSettings)
	})
	e.POST("/recording/settings/set", func(c echo.Context) error {
		params := new(RecordingSettings)
		if err := c.Bind(params); err != nil {
			return c.String(http.StatusBadRequest, "Invalid parameters.")
		}
		recordingSettings = *params
		// Update interval filter parameter.
		intervalFilter.SetInterval(time.Duration(float64(time.Second) * float64(recordingSettings.StoreEverySeconds)))
		return c.String(http.StatusOK, "OK")
	})
	e.GET("/status", func(c echo.Context) error {
		status := UnitStatus{
			IsAcquiring:  acqUnit.Status().Running,
			IsMLRunning:  mlServer.Running(),
			IsMonitoring: monServer.Running(),
		}
		return c.JSON(http.StatusOK, status)
	})

	// Start export storage service
	if err := RunExportStorageService(e); err != nil {
		e.Logger.Fatal(err)
	}

	// Start acquisition and monitoring
	log.Printf("Starting acquisition")
	if err := acqUnit.Start(); err != nil {
		log.Fatalf("Failed to start acquisition unit. %v", err)
	}
	log.Printf("Starting monitoring")
	if err := monServer.Start(); err != nil {
		log.Fatalf("Failed to start monitoring unit. %v", err)
	}
	// Start FFT filtering processor
	if !disableFFTFilter {
		fftFilter.Start()
	}
	// Start summarizer
	summarizer.Start()
	if enableDebugRecorder {
		recorder.Start()
	}
	if enableFrameRecorder {
		recorderPF.Start()
	}
	e.Logger.Fatal(e.Start(":2223"))
}

func clientFileServer(rootDirectory string) {
	e := echo.New()
	e.Use(middleware.CORS())

	e.Static("/", rootDirectory)

	e.Logger.Fatal(e.Start(":3000"))
}
func main() {
	flag.Usage = func() { flag.PrintDefaults() }
	clientDir := flag.String("client_dir", "", "directory to serve Web client files")
	dataDir := flag.String("data_dir", "", "directory to serve saved data files")
	mlConfigName := flag.String("ml_config", "", "name of ML configuration.")
	acqConfigFile := flag.String("acq_config_file", "acquisition.config.json", "Acquisition unit configuration file.")
	enableDebugRecorder := flag.Bool("record", false, "Enable debug RAW and FFT waveform recorder.")
	disableFFTFilter := flag.Bool("disable_fft", false, "Disable FFT filter for debugging.")
	enableFrameRecorder := flag.Bool("frame_record", false, "Enable RAW waveform per frame recorder.")
	flag.Parse()

	acqConfig, err := acquisition.LoadConfig(*acqConfigFile)
	if err != nil {
		panic(err)
	}
	log.Printf("Acquisition config: %+v", acqConfig)

	acqSettings, err := acquisition.LoadSettings("acquisition.settings.json")
	if err != nil {
		acqSettings = nil
		log.Printf("Failed to load acquisition settings: %s", err.Error())
	} else {
		log.Printf("Acquisition settings: %+v", acqSettings)
	}

	mlServerConfigs, err := ml.LoadConfigs("ml/mlservers.json")
	if err != nil {
		panic(err)
	}
	if *mlConfigName == "" {
		mlConfigName = &mlServerConfigs.Default
	}
	configIndex := -1
	for i, config := range mlServerConfigs.Configurations {
		if config.Name == *mlConfigName {
			configIndex = i
			break
		}
	}
	if configIndex < 0 {
		panic(fmt.Errorf("Could not find default ML server config %s", *mlConfigName))
	}
	mlServerConfig := mlServerConfigs.Configurations[configIndex]
	go apiServer(dataDir, acqConfig, acqSettings, &mlServerConfig, *enableDebugRecorder, *disableFFTFilter, *enableFrameRecorder)
	if len(*clientDir) > 0 {
		go clientFileServer(*clientDir)
	}
	// Listen
	http.ListenAndServe(":2222", nil)
}
