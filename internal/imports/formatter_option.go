package imports

type FormatterOptionFn func(*Formatter)

func FormatterWithList(isList bool) FormatterOptionFn {
	return func(ft *Formatter) {
		ft.isList = isList
	}
}

func FormatterWithWrite(isWrite bool) FormatterOptionFn {
	return func(ft *Formatter) {
		ft.isWrite = isWrite
	}
}

func FormatterWithDiff(isDiff bool) FormatterOptionFn {
	return func(ft *Formatter) {
		ft.isDiff = isDiff
	}
}

func FormatterWithVerbose(isVerbose bool) FormatterOptionFn {
	return func(ft *Formatter) {
		ft.isVerbose = isVerbose
	}
}

func FormatterWithCompanyPrefix(companyPrefix string) FormatterOptionFn {
	return func(ft *Formatter) {
		ft.companyPrefix = companyPrefix
	}
}
