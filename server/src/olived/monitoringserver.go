package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"sort"
	"sync/atomic"

	"olived/acquisition"
	"olived/core"
)

type MonitoringServer struct {
	running  bool
	server   *core.WSServer
	settings acquisition.UnitSettings

	scheduleSendChannelData int32
}

func NewMonitoringServer() *MonitoringServer {
	s := MonitoringServer{}
	s.server = core.NewWSServer("/monitoring/summary", func(c *core.WSClient) {
		log.Printf("Monitoring client connected.")
		atomic.StoreInt32(&s.scheduleSendChannelData, 1)
	})
	return &s
}

func (s *MonitoringServer) Type() acquisition.ProcessorType {
	return acquisition.Monitor
}

func (s *MonitoringServer) Running() bool {
	return s.running
}

func (s *MonitoringServer) Start() error {
	if s.running {
		return fmt.Errorf("Already running")
	}

	s.server.Listen()

	s.running = true

	return nil
}

func (s *MonitoringServer) Stop() error {
	if !s.running {
		return fmt.Errorf("Not running")
	}

	s.running = false
	return nil
}

// sendChannelData - Sends channel data monitoring packet which contains current acquisition settings.
func (s *MonitoringServer) sendChannelData() {
	keys := make([]string, 0, len(s.settings.Channels))
	for key := range s.settings.Channels {
		keys = append(keys, key)
	}
	sort.Sort(sort.StringSlice(keys))
	buffer := new(bytes.Buffer)
	binary.Write(buffer, binary.LittleEndian, uint16(3)) // Data type = 3 (channel data)
	binary.Write(buffer, binary.LittleEndian, uint16(0)) // Flags = 0
	binary.Write(buffer, binary.LittleEndian, uint32(0)) // Length
	binary.Write(buffer, binary.LittleEndian, uint16(len(keys)))
	binary.Write(buffer, binary.LittleEndian, uint16(len(keys)))
	binary.Write(buffer, binary.LittleEndian, uint16(0))
	for _, key := range keys {
		binary.Write(buffer, binary.LittleEndian, uint16(0)) // TODO: Caclulate correct channel flags
		binary.Write(buffer, binary.LittleEndian, uint16(len(key)))
		binary.Write(buffer, binary.LittleEndian, []byte(key))
	}
	data := buffer.Bytes()
	binary.LittleEndian.PutUint32(data[4:8], uint32(len(data)-8)) // Update Length field.
	s.server.Write(data)
}

func (s *MonitoringServer) Started(unit acquisition.Unit) {}
func (s *MonitoringServer) Stopped(unit acquisition.Unit) {}
func (s *MonitoringServer) SettingsChanging(unit acquisition.Unit, settings *acquisition.UnitSettings) error {
	return nil
}
func (s *MonitoringServer) SettingsChanged(unit acquisition.Unit, settings *acquisition.UnitSettings) {
	s.settings = *settings
	s.sendChannelData()
}

func (s *MonitoringServer) DataArrived(unit acquisition.Unit, data acquisition.AcquiredData) {
	if !s.running {
		return
	}
	if atomic.CompareAndSwapInt32(&s.scheduleSendChannelData, 1, 0) {
		s.sendChannelData()
	}
	switch data.Type() {
	case acquisition.TimeSeriesSummary:
		s.server.Write([]byte(data.RawData()))
	case acquisition.FFT:
		s.server.Write([]byte(data.RawData()))
	}
}
