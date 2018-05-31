package ml

import (
	"encoding/json"
	"log"

	"olived/core"
)

type WSServer struct {
	ws *core.WSServer
	ml MLServer
}

// NewWSServer
// Wraps a ML server and redirect events from the ML server to WebSocket clients.
func NewWSServer(mlServer MLServer, urlPattern string) *WSServer {
	s := &WSServer{
		ws: core.NewWSServer(urlPattern, func(c *core.WSClient) {}),
		ml: mlServer,
	}
	s.ws.Listen()
	return s
}

func (s *WSServer) Running() bool {
	return s.ml.Running()
}
func (s *WSServer) Settings() MLServerSettings {
	return s.ml.Settings()
}
func (s *WSServer) DefaultSettings() MLServerSettings {
	return s.ml.DefaultSettings()
}
func (s *WSServer) Configure(config *MLServerConfig) error {
	return s.ml.Configure(config)
}
func (s *WSServer) ChangeSettings(settings *MLServerSettings) error {
	return s.ml.ChangeSettings(settings)
}
func (s *WSServer) Start(handler MLServerEventHandler) error {
	serverHandler := func(server MLServer, event interface{}) {
		handler(server, event)
		if json, err := json.Marshal(event); err == nil {
			s.ws.Write(string(json))
		} else {
			log.Printf("Error: marshaling failed. %s", err.Error())
		}
	}
	return s.ml.Start(serverHandler)
}
func (s *WSServer) Stop() error {
	return s.ml.Stop()
}
func (s *WSServer) Write(input MLServerInput) error {
	return s.ml.Write(input)
}
