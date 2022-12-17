package errors

import (
	"errors"
	"fmt"
)

func New(err error) error {
	return &traceableError{
		err:   err,
		stack: recordStack(),
	}
}

func Wrap(p any) error {
	switch x := p.(type) {
	case error:
		return New(x)
	case string:
		return New(errors.New(x))
	default:
		return New(errors.New(fmt.Sprint(x)))
	}
}

type traceableError struct {
	err   error
	stack *stack
}

func (err traceableError) Error() string {
	return err.stack.Format()
}
