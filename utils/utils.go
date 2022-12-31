package utils

type Registry[T any] struct {
	set map[string]Ctor[T]
}

type Ctor[T any] func() T

func (reg *Registry[T]) init() {
	if reg.set == nil {
		reg.set = make(map[string]Ctor[T])
	}
}

func (reg *Registry[T]) Register(name string, ctor Ctor[T]) {
	reg.init()

	reg.set[name] = ctor
}

func (reg *Registry[T]) Lookup(name string) (ctor Ctor[T], ok bool) {
	reg.init()

	ctor, ok = reg.set[name]
	return ctor, ok
}

// Range
func (reg *Registry[T]) Range(fn func(name string, ctor Ctor[T])) {
	reg.init()

	for name, ctor := range reg.set {
		fn(name, ctor)
	}
}
