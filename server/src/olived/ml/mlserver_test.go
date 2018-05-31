package ml

import "testing"

func TestLoadMLServerConfig(t *testing.T) {
	config, err := LoadMLServerConfig("mlservers.json")
	if err != nil {
		t.Fatalf("Failed to load. error:%v", err)
	}
	if config.Default != "Local" {
		t.Fatalf("Unexpected default value. expected=\"Local\", actual=\"%s\"", config.Default)
	}
	if len(config.Connections) != 2 {
		t.Fatalf("Unexpected number of connections. expected=2, actual=%d", len(config.Connections))
	}
}
