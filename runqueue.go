package mx

import "fmt"

type (
	step int
)

const (
	_ step = iota
	Init
	Setup
)

type runqueue struct {
	cur  step
	runs map[step][]func()
}

func (h *runqueue) call(s step, fn func()) error {
	if h.runs == nil {
		h.runs = make(map[step][]func())
	}

	h.runs[s] = append(h.runs[s], fn)
	if h.cur >= s {
		return h.do(s)
	}
	return nil
}

func (h *runqueue) recoved(fn func()) (err error) {
	defer func() {
		if _err := recover(); _err != nil {
			switch x := _err.(type) {
			case error:
				err = x
			case nil:
			default:
				err = fmt.Errorf("%v", x)
			}
		}
	}()

	fn()
	return nil
}

func (h *runqueue) do(step step) error {
	h.cur = step

	var pops []int
	defer func() {
		// remove multi call
		c := 0
		for _, i := range pops {
			h.runs[step] = append(h.runs[step][:i-c], h.runs[step][i-c+1:]...)
			c++
		}
	}()

	for i, fn := range h.runs[step] {
		if err := h.recoved(fn); err != nil {
			return err
		}
		pops = append(pops, i)
	}

	return nil
}
