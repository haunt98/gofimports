package imports

import (
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
)

var ErrEmptyPaths = errors.New("empty paths")

type Formatter struct {
	isList  bool
	isWrite bool
	isDiff  bool
}

func NewFormmater(opts ...FormatterOptionFn) *Formatter {
	ft := &Formatter{}

	for _, opt := range opts {
		opt(ft)
	}

	return ft
}

// Accept a list of files or directories
func (ft *Formatter) Format(paths ...string) error {
	if len(paths) == 0 {
		return ErrEmptyPaths
	}

	// Logic switch case copy from goimports, gofumpt
	for _, path := range paths {
		switch dir, err := os.Stat(path); {
		case err != nil:
			return fmt.Errorf("os: failed to stat: [%s] %w", path, err)
		case dir.IsDir():
			if err := ft.formatDir(path); err != nil {
				return err
			}
		default:
			if err := ft.formatFile(path); err != nil {
				return err
			}
		}
	}

	return nil
}

func (ft *Formatter) formatDir(path string) error {
	return nil
}

func (ft *Formatter) formatFile(path string) error {
	// Check go file
	if !isGoFile(filepath.Base(path)) {
		return nil
	}

	pathBytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("os: failed to read file: [%s] %w", path, err)
	}

	// Parse ast
	fset := token.NewFileSet()

	parserMode := parser.Mode(0)
	parserMode |= parser.ParseComments

	pathASTFile, err := parser.ParseFile(fset, path, pathBytes, parserMode)
	if err != nil {
		return fmt.Errorf("parser: failed to parse file [%s]: %w", path, err)
	}

	// Ignore generated file
	if isGoGenerated(pathASTFile) {
		return nil
	}

	// TODO: fix imports

	return nil
}
