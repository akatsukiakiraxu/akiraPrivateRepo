package core

type RangeF32 struct {
	Min float32
	Max float32
}
type RangeF64 struct {
	Min float64
	Max float64
}

type Configurable interface {
	Configure(config map[string]interface{}) error
}
