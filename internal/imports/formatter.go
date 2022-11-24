package imports

type Formatter struct {
	isList  bool
	isWrite bool
	isDiff  bool
}

func NewFormmater(opts ...FormatterOptionFn) *Formatter {
	f := &Formatter{}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

// Accept a list of files or directories aka fsNames
func (f *Formatter) Format(fsNames ...string) error {
	return nil
}
