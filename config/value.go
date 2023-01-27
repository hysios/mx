package config

import "github.com/stretchr/objx"

func NewMap(val interface{}) Map {
	return objx.New(val)
}
