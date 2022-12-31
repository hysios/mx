package registry

type LookupOption struct {
	Namespace string
}

type LookupOptionFunc func(*LookupOption)

func WithNamespace(ns string) LookupOptionFunc {
	return func(opt *LookupOption) {
		opt.Namespace = ns
	}
}
