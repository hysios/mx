package mx

import "errors"

var (
	ErrServiceNotFound = errors.New("service not found")
	ErrSkipFile        = errors.New("skip file")
)
