package imports

import (
	"go/ast"
	"log"
)

// Wrap log.Printf with verbose flag
func (ft *Formatter) log(format string, v ...any) {
	if ft.isVerbose {
		log.Printf(format, v...)
	}
}

func (ft *Formatter) logImportSpecs(logPrefix string, importSpecs []*ast.ImportSpec) {
	if ft.isVerbose {
		for _, importSpec := range importSpecs {
			log.Printf("%s: importSpec: %+v %+v\n", logPrefix, importSpec.Name.String(), importSpec.Path.Value)
		}
	}
}

func (ft *Formatter) mustLogImportSpecs(logPrefix string, importSpecs []ast.Spec) {
	if ft.isVerbose {
		for _, importSpec := range importSpecs {
			importSpec, ok := importSpec.(*ast.ImportSpec)
			if !ok {
				continue
			}

			log.Printf("%s: importSpec: %+v %+v\n", logPrefix, importSpec.Name.String(), importSpec.Path.Value)
		}
	}
}
