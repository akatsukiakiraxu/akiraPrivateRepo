package ml

import (
	"encoding/json"
	"io/ioutil"
)

// LoadConfig - Load acquisition unit configuration from file.
func LoadConfigs(path string) (*MLServerConfigs, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	obj := new(MLServerConfigs)
	if err = json.Unmarshal(b, obj); err != nil {
		return nil, err
	}

	return obj, nil
}
