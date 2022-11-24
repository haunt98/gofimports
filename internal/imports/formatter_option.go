package imports

type FormatterOptionFn func(*Formatter)

func FormatterWithList() FormatterOptionFn {
	return func(f *Formatter) {
		f.isList = true
	}
}

func FormatterWithWrite() FormatterOptionFn {
	return func(f *Formatter) {
		f.isWrite = true
	}
}

func FormatterWithDiff() FormatterOptionFn {
	return func(f *Formatter) {
		f.isDiff = true
	}
}
