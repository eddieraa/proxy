package proxy

//Options options structs
type Options struct {
	fcts []FctService
}

//Option option func
type Option func(opts *Options)

func newOptions(opts ...Option) Options {
	options := Options{
		fcts: []FctService{},
	}
	for _, o := range opts {
		o(&options)
	}
	return options
}

//AddServiceOption adding service option
func AddServiceOption(fn FctService) Option {
	return func(opts *Options) {
		opts.fcts = append(opts.fcts, fn)
	}
}
