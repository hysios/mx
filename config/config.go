package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/stretchr/objx"
)

type (
	Map   = objx.Map
	Value = objx.Value
)

type Config struct {
	defaults  Map
	providers []ConfigProvider
}

// NewConfig returns a new config.
func NewConfig(defaults map[string]interface{}, providers ...ConfigProvider) *Config {
	return &Config{
		defaults:  objx.New(defaults),
		providers: providers,
	}
}

func (c *Config) reverseProviders() []ConfigProvider {
	var providers = make([]ConfigProvider, len(c.providers))
	for i, p := range c.providers {
		providers[len(c.providers)-i-1] = p
	}

	return providers
}

// Get returns the value of the given selector.
func (c *Config) Get(selector string) (val *Value, ok bool) {
	for _, p := range c.reverseProviders() {
		if val, ok = p.LookupPath(selector); ok {
			return
		}
	}

	val = c.defaults.Get(selector)
	return
}

// Set sets the value of the given selector.
func (c *Config) Set(selector string, val interface{}) (old interface{}, err error) {
	defer func() {
		if _err := recover(); _err != nil {
			switch x := _err.(type) {
			case error:
				err = x
			case string:
				err = errors.New(x)
			default:
				err = fmt.Errorf("%v", x)
			}
		}
	}()

	for _, p := range c.reverseProviders() {
		old = p.Set(selector, val)
		break
	}

	return
}

// Update updates the config with the given values.
func (c *Config) DefaultsUpdate(vals map[string]interface{}) Map {
	return c.defaults.MergeHere(objx.New(vals))
}

func (c *Config) Update(vals map[string]interface{}) Map {
	var m = Map{}
	for _, p := range c.reverseProviders() {
		m.MergeHere(p.Update(vals))
	}

	return m
}

func (c *Config) All() Map {
	var m = Map{}
	for _, p := range c.reverseProviders() {
		m.MergeHere(p.Data())
	}

	return m
}

// Str returns the string value of the given selector.
func (c *Config) Str(selector string) string {
	val, ok := c.Get(selector)
	if !ok {
		return ""
	}
	return val.Str()
}

// Int returns the int value of the given selector.
func (c *Config) Int(selector string) int {
	val, ok := c.Get(selector)
	if !ok {
		return 0
	}
	return val.Int()
}

// Uint
func (c *Config) Uint(selector string) uint {
	val, ok := c.Get(selector)
	if !ok {
		return 0
	}
	return val.Uint()
}

// Int32
func (c *Config) Int32(selector string) int32 {
	val, ok := c.Get(selector)
	if !ok {
		return 0
	}
	return val.Int32()
}

// Uint32
func (c *Config) Uint32(selector string) uint32 {
	val, ok := c.Get(selector)
	if !ok {
		return 0
	}
	return val.Uint32()
}

// Int64
func (c *Config) Int64(selector string) int64 {
	val, ok := c.Get(selector)
	if !ok {
		return 0
	}
	return val.Int64()
}

// Uint64
func (c *Config) Uint64(selector string) uint64 {
	val, ok := c.Get(selector)
	if !ok {
		return 0
	}
	return val.Uint64()
}

// Bool returns the bool value of the given selector.
func (c *Config) Bool(selector string) bool {
	val, ok := c.Get(selector)
	if !ok {
		return false
	}
	return val.Bool()
}

// Float returns the float value of the given selector.
func (c *Config) Float(selector string) float32 {
	val, ok := c.Get(selector)
	if !ok {
		return 0
	}
	return val.Float32()
}

// Float64 returns the float64 value of the given selector.
func (c *Config) Float64(selector string) float64 {
	val, ok := c.Get(selector)
	if !ok {
		return 0
	}
	return val.Float64()
}

// Bytes returns the bytes value of the given selector.
func (c *Config) Bytes(selector string) []byte {
	val, ok := c.Get(selector)
	if !ok {
		return nil
	}
	bytes, ok := val.Data().([]byte)
	if !ok {
		return nil
	}
	return bytes
}

// Duration returns the time duration value of the given selector.
func (c *Config) Duration(selector string) time.Duration {
	val, ok := c.Get(selector)
	if !ok {
		return 0
	}

	return time.Duration(val.Int64())
}

// Time returns the time value of the given selector.
func (c *Config) Time(selector string) time.Time {
	val, ok := c.Get(selector)
	if !ok {
		return time.Time{}
	}

	switch x := val.Data().(type) {
	case string:
		t, err := time.Parse(time.RFC3339, x)
		if err != nil {
			return time.Time{}
		}
		return t
	case time.Time:
		return x
	case int64:
		return time.UnixMilli(x)
	case int:
		return time.UnixMilli(int64(x))
	default:
		return time.Time{}
	}
}

// IntSlice returns the int slice value of the given selector.
func (c *Config) IntSlice(selector string) []int {
	val, ok := c.Get(selector)
	if !ok {
		return nil
	}
	return val.IntSlice()
}

// StringSlice returns the string slice value of the given selector.
func (c *Config) StringSlice(selector string) []string {
	val, ok := c.Get(selector)
	if !ok {
		return nil
	}
	return val.StrSlice()
}

// BoolSlice returns the bool slice value of the given selector.
func (c *Config) BoolSlice(selector string) []bool {
	val, ok := c.Get(selector)
	if !ok {
		return nil
	}
	return val.BoolSlice()
}

// FloatSlice returns the float slice value of the given selector.
func (c *Config) FloatSlice(selector string) []float32 {
	val, ok := c.Get(selector)
	if !ok {
		return nil
	}

	return val.Float32Slice()
}

// Float64Slice returns the float64 slice value of the given selector.
func (c *Config) Float64Slice(selector string) []float64 {
	val, ok := c.Get(selector)
	if !ok {
		return nil
	}
	return val.Float64Slice()
}

// Map returns the map value of the given selector.
func (c *Config) Map(selector string) map[string]interface{} {
	val, ok := c.Get(selector)
	if !ok {
		return nil
	}
	return map[string]interface{}(val.ObjxMap())
}

// Slice returns the slice value of the given selector.
func (c *Config) Slice(selector string) []interface{} {
	val, ok := c.Get(selector)
	if !ok {
		return nil
	}
	return val.InterSlice()
}
