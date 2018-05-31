package ml

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"olived/core"
)

type oselmPythonServer struct {
	command    *exec.Cmd
	settings   MLServerSettings
	pythonPath string
	scriptPath string
	useSSH     bool
	sshPath    string
	sshArgs    []string

	address    string
	port       int
	connection net.Conn

	lock          sync.RWMutex
	commandDoneCh chan struct{}
	inputCh       chan MLServerInput
	stopCh        chan struct{}
	doneCh        chan struct{}
}

func oselmDefaultSettings() MLServerSettings {
	return MLServerSettings{
		InputType:          Raw,
		TrainingCount:      256,
		InputDataSize:      768,
		InputDataOffset:    0,
		ActivationFunction: ReLU,
		LossFunction:       MeanSquaredError,
	}
}

func newOSELMPythonServer() MLServer {
	return &oselmPythonServer{
		lock:     sync.RWMutex{},
		settings: oselmDefaultSettings(),
	}
}

// Communication goroutine
func communicator(server *oselmPythonServer, inputCh chan MLServerInput, handler MLServerEventHandler, stopCh chan struct{}, doneCh chan struct{}) {
	defer func() { close(doneCh) }() // Close done_chan just before exiting this function.

	readBuffer := make([]byte, 12)
	reader := bytes.NewReader(readBuffer)

	stopWriterCh := make(chan struct{})
	doneWriterCh := make(chan struct{})

	go func() {
		defer func() { doneWriterCh <- struct{}{} }()
		for {
			select {
			case input, ok := <-inputCh: // Input from ADC.
				if !ok {
					return
				}
				server.connection.SetWriteDeadline(time.Now().Add(1 * time.Second))
				if err := binary.Write(server.connection, binary.LittleEndian, input); err != nil {
					log.Printf("Failed to write data length=%d", len(input))
				}
			case <-stopWriterCh:
				return
			}
		}
	}()
	for {
		select {
		case <-stopCh: // Requested to stop.
			// Stop writer goroutine and wait it.
			log.Printf("communicator requested to stop.")
			stopWriterCh <- struct{}{}
			<-doneWriterCh
			log.Printf("communicator has stopped.")
			server.connection.Close()
			return
		default:
			// Read from the ML server.
			server.connection.SetReadDeadline(time.Now().Add(1 * time.Second))
			if n, err := server.connection.Read(readBuffer); err == nil && n > 0 {
				reader.Reset(readBuffer)

				var magic uint32
				var packetType uint32
				binary.Read(reader, binary.LittleEndian, &magic)
				binary.Read(reader, binary.LittleEndian, &packetType)

				//log.Printf("Responded %08x, %08x", magic, packetType)
				if magic == 0x0115eaa0 { // Check macgic number
					switch packetType {
					case 0: // training progress
						var count uint32
						binary.Read(reader, binary.LittleEndian, &count)
						event := MLTrainingEvent{
							Type:         MLEventTrainingProgress,
							TrainedCount: int(count),
						}
						handler(server, &event)
					case 1: // training done
						var count uint32
						binary.Read(reader, binary.LittleEndian, &count)
						event := MLTrainingEvent{
							Type:         MLEventTrainingDone,
							TrainedCount: int(count),
						}
						handler(server, &event)
					case 2: // inference result
						var lossValue float32
						binary.Read(reader, binary.LittleEndian, &lossValue)
						event := MLInferenceEvent{
							Type:      MLEventInferenceResult,
							LossValue: lossValue,
						}
						handler(server, &event)
					}
				}
			}
		}
	}

}

func (server *oselmPythonServer) Configure(config *MLServerConfig) error {
	if pythonPath, ok := config.Settings["python_path"]; ok {
		server.pythonPath = pythonPath.(string)
	} else {
		return fmt.Errorf("python_path must be set in config")
	}

	if scriptPath, ok := config.Settings["script_path"]; ok {
		server.scriptPath = scriptPath.(string)
	} else {
		return fmt.Errorf("script_path must be set in config")
	}
	if useSSH, ok := config.Settings["use_ssh"]; ok {
		server.useSSH = useSSH.(bool)
	}
	if server.useSSH {
		if sshArgs, ok := config.Settings["ssh_args"]; ok {
			server.sshArgs = core.ToStringSlice(sshArgs.([]interface{}))
		} else {
			server.sshArgs = make([]string, 0)
		}
		if sshPath, ok := config.Settings["ssh_path"]; ok {
			server.sshPath = sshPath.(string)
		} else {
			server.sshPath = "ssh"
		}
	}

	server.port = config.Connection.Port
	server.address = config.Connection.Address

	return nil
}

func (server *oselmPythonServer) ChangeSettings(settings *MLServerSettings) error {
	if server.Running() {
		return fmt.Errorf("cannot change settings while running")
	}

	server.lock.Lock()
	defer server.lock.Unlock()
	server.settings = *settings

	return nil
}

func (server *oselmPythonServer) Start(handler MLServerEventHandler) error {
	if server.command != nil {
		return fmt.Errorf("Server has already started")
	}
	if server.pythonPath == "" {
		return fmt.Errorf("PythonPath is not set")
	}
	if server.scriptPath == "" {
		return fmt.Errorf("ScriptPath is not set")
	}

	var commandPath string
	var launchArgs []string
	if server.useSSH {
		commandPath = server.sshPath
		launchArgs = make([]string, len(server.sshArgs)+3)
		copy(launchArgs, server.sshArgs)
		launchArgs[len(launchArgs)-3] = server.address
		launchArgs[len(launchArgs)-2] = server.pythonPath
		launchArgs[len(launchArgs)-1] = server.scriptPath
	} else {
		commandPath = server.pythonPath
		launchArgs = []string{server.scriptPath}
	}

	scriptArgs := []string{
		"--inputs", strconv.Itoa(server.settings.InputDataSize),
		"--units", strconv.Itoa(server.settings.TrainingCount),
		"--activation", string(server.settings.ActivationFunction),
		"--loss", string(server.settings.LossFunction),
		"--port", strconv.Itoa(server.port),
		"--quantize-x-count", "16",
		"--quantize-y-count", "16",
		"--quantize-y-min", "-10",
		"--quantize-y-max", "10",
	}
	args := make([]string, len(launchArgs)+len(scriptArgs))
	copy(args[:len(launchArgs)], launchArgs)
	copy(args[len(launchArgs):], scriptArgs)

	log.Printf("Server command: %s %v", commandPath, args)

	command := exec.Command(commandPath, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	err := command.Start()
	if err != nil {
		return fmt.Errorf("Failed to start process. %s", err.Error())
	}

	server.commandDoneCh = make(chan struct{})
	go func() {
		command.Wait() // Wait the process in another goroutine to avoid the process gets defunct after exiting this process.
		server.commandDoneCh <- struct{}{}
	}()

	serverAddress := fmt.Sprintf("%s:%d", server.address, server.port)

	var conn net.Conn
	for i := 0; i < 10; i++ {
		conn, err = net.Dial("tcp", serverAddress)
		if err == nil {
			break
		} else {
			time.Sleep(500 * time.Millisecond)
		}
	}
	if err != nil {
		command.Process.Kill()
		return fmt.Errorf("Failed to connect to the server at %s - %s", serverAddress, err.Error())
	}
	log.Printf("Connected to OS-ELM server at %s", serverAddress)

	server.lock.Lock()
	server.command = command
	server.inputCh = make(chan MLServerInput)
	server.stopCh = make(chan struct{})
	server.doneCh = make(chan struct{})
	server.connection = conn
	server.lock.Unlock()

	// Run communication goroutine
	go communicator(server, server.inputCh, handler, server.stopCh, server.doneCh)

	return nil
}

func (server *oselmPythonServer) Stop() error {
	if server.command == nil {
		return fmt.Errorf("Server is not started")
	}

	server.lock.Lock()
	defer server.lock.Unlock()

	var err error

	// Stop the communication goroutine.
	server.stopCh <- struct{}{}

	select {
	case <-server.commandDoneCh:
		log.Printf("OS-ELM server stopped successfully.")
	case <-time.After(5 * time.Second):
		log.Printf("Failed to stop OS-ELM server gracefully. Try to kill it.")
		err = server.command.Process.Kill()
	}

	// Wait for completion of the communication goroutine
	<-server.doneCh

	server.command = nil

	if err != nil {
		return fmt.Errorf("Failed to stop the server process")
	}
	return nil
}

func (server *oselmPythonServer) Write(input MLServerInput) error {
	defer server.lock.RUnlock()
	server.lock.RLock()
	if server.command == nil {
		return fmt.Errorf("Not running")
	}
	server.inputCh <- input
	return nil
}

func (server *oselmPythonServer) Running() bool {
	server.lock.RLock()
	defer server.lock.RUnlock()
	return server.command != nil
}

func (server *oselmPythonServer) DefaultSettings() MLServerSettings {
	return oselmDefaultSettings()
}
func (server *oselmPythonServer) Settings() MLServerSettings {
	return server.settings
}
