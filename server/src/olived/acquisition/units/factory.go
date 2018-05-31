package units

import (
	"fmt"

	"olived/acquisition/units/adcdb"
	"olived/acquisition/units/discovery"
	"olived/acquisition/units/mock"

	parent "olived/acquisition"
)

func NewUnit(config *parent.UnitConfig) (parent.Unit, error) {
	switch config.Type {
	case "mock":
		return mock.NewMockUnit(config)
	case "adcdb":
		return adcdb.NewADCDBUnit(config)
	case "discovery":
		return discovery.NewDiscoveryUnit(config)
	default:
		return nil, fmt.Errorf("unknown unit type %s", config.Type)
	}
}
