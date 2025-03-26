package imports

import "strings"

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
		ft.companyPrefixes = make(map[string]struct{})
		for prefix := range strings.SplitSeq(companyPrefix, ",") {
			prefix = strings.TrimSpace(prefix)
			if prefix == "" {
				continue
			}
			ft.companyPrefixes[prefix] = struct{}{}
		}
	}
}

func FormatterWithStock(isStock bool) FormatterOptionFn {
	return func(ft *Formatter) {
		ft.isStock = isStock
	}
}
