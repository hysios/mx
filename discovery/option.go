package discovery

type LookupOption struct {
	Namespace   string
	ServiceType string
}

type LookupOptionFunc func(*LookupOption)

func WithNamespace(ns string) LookupOptionFunc {
	return func(opt *LookupOption) {
		opt.Namespace = ns
	}
}

func WithServiceType(serviceType string) LookupOptionFunc {
	return func(opt *LookupOption) {
		opt.ServiceType = serviceType
	}
}

func (option *LookupOption) MatchServiceType(serviceType string) bool {
	if option.ServiceType == "" {
		return true
	}
	return option.ServiceType == serviceType
}
