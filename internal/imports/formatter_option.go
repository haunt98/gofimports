package imports

type FormatterOptionFn func(*Formatter)

func FormatterWithList(isList bool) FormatterOptionFn {
	return func(f *Formatter) {
		f.isList = isList
	}
}

func FormatterWithWrite(isWrite bool) FormatterOptionFn {
	return func(f *Formatter) {
		f.isWrite = isWrite
	}
}

func FormatterWithDiff(isDiff bool) FormatterOptionFn {
	return func(f *Formatter) {
		f.isDiff = isDiff
	}
}
