package viper

import (
	"github.com/hysios/mx/config"
	"github.com/spf13/viper"
)

type ViperProvider struct {
	v    *viper.Viper
	vals config.Map
}

// NewViperProvider creates a new ViperProvider instance.
func NewViperProvider(v *viper.Viper) *ViperProvider {
	return &ViperProvider{v: v}
}

// init initializes the ViperProvider.
func (vp *ViperProvider) init() {
	vp.vals = config.NewMap(vp.v.AllSettings())
}

// LookupPath retrieves a value from the Viper configuration.
func (vp *ViperProvider) LookupPath(selector string) (val *config.Value, ok bool) {
	if vp.vals == nil {
		vp.init()
	}

	v := vp.vals.Get(selector)
	if v == nil {
		return nil, false
	}
	return v, true
}

// Set sets a value in the Viper configuration.
func (vp *ViperProvider) Set(selector string, value interface{}) (old interface{}, err error) {
	if vp.vals == nil {
		vp.init()
	}

	vp.vals.Set(selector, value)
	return vp.vals.Get(selector), nil
}

// Update updates the Viper configuration with a map of values.
func (vp *ViperProvider) Update(vals map[string]interface{}) config.Map {
	if vp.vals == nil {
		vp.init()
	}
	vp.vals = vp.vals.MergeHere(vals)
	return vp.vals
}

// Data returns the data of the provider.
func (vp *ViperProvider) Data() config.Map {
	if vp.vals == nil {
		vp.init()
	}
	return vp.vals
}
