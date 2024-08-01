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
func NewViperProvider() *ViperProvider {
	return &ViperProvider{v: viper.GetViper()}
}

// LookupPath retrieves a value from the Viper configuration.
func (vp *ViperProvider) LookupPath(selector string) (val *config.Value, ok bool) {
	v := vp.v.Get(selector)
	if v == nil {
		return nil, false
	}
	return &config.Value{v}, true
}
