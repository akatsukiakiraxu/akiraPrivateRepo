package ml

import (
	"fmt"
)

// ML Server Types
type MLServerType string

const (
	Mock        MLServerType = "mock"
	OSELMPython              = "oselm_python"
)

func NewMLServer(config *MLServerConfig) (MLServer, error) {
	var server MLServer
	switch config.Type {
	case OSELMPython:
		server = newOSELMPythonServer()
	case Mock:
		server = newMockServer()
	default:
		return nil, fmt.Errorf("Unknown ML Server - %s", string(config.Type))
	}

	if err := server.Configure(config); err != nil {
		return nil, err
	}

	return server, nil
}
