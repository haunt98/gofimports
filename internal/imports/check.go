package imports

import (
	"go/ast"
	"strings"
)

// Copy from https://github.com/mvdan/gofumpt
func isGoFile(name string) bool {
	// Hidden files are ignored
	if strings.HasPrefix(name, ".") {
		return false
	}

	return strings.HasSuffix(name, ".go")
}

// Copy from https://github.com/mvdan/gofumpt
// Copy from https://github.com/incu6us/goimports-reviser
func isGoGenerated(file *ast.File) bool {
	for _, cg := range file.Comments {
		// Ignore if package ... is on top
		if cg.Pos() > file.Package {
			return false
		}

		for _, line := range cg.List {
			if strings.Contains(line.Text, "// Code generated") {
				return true
			}
		}
	}

	return false
}
