package imports

import (
	"go/ast"
	"regexp"
	"strings"
)

var reGoGenerated = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.$`)

// Copy from https://github.com/mvdan/gofumpt
func isGoFile(name string) bool {
	// Hidden files are ignored
	if strings.HasPrefix(name, ".") {
		return false
	}

	return strings.HasSuffix(name, ".go")
}

// Copy from https://github.com/mvdan/gofumpt
func isGoGenerated(file *ast.File) bool {
	for _, cg := range file.Comments {
		// Ignore if package ... is on top
		if cg.Pos() > file.Package {
			return false
		}

		for _, line := range cg.List {
			if reGoGenerated.MatchString(line.Text) {
				return true
			}
		}
	}

	return false
}
