package imports

import (
	"log"

	"github.com/dave/dst"
)

// Wrap log.Printf with verbose flag
func (ft *Formatter) log(format string, v ...any) {
	if ft.isVerbose {
		log.Printf(format, v...)
	}
}

func (ft *Formatter) logDSTImportSpecs(logPrefix string, importSpecs []*dst.ImportSpec) {
	if ft.isVerbose {
		for _, importSpec := range importSpecs {
			log.Printf("%s: [%s] [%s] before %v after %v\n", logPrefix, importSpec.Name, importSpec.Path.Value, importSpec.Decs.Before, importSpec.Decs.After)
		}
	}
}
