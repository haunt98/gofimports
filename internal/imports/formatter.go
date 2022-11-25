package imports

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

const (
	// Use for group imports
	stdImport        = "std"
	thirdPartyImport = "third-party"
	companyImport    = "company"
	localImport      = "local"
)

var (
	ErrEmptyPaths      = errors.New("empty paths")
	ErrNotGoFile       = errors.New("not go file")
	ErrGoGeneratedFile = errors.New("go generated file")
)

type Formatter struct {
	stdPackages   map[string]struct{}
	companyPrefix string
	isList        bool
	isWrite       bool
	isDiff        bool
	isVerbose     bool
}

func NewFormmater(opts ...FormatterOptionFn) (*Formatter, error) {
	ft := &Formatter{}

	for _, opt := range opts {
		opt(ft)
	}

	stdPackages, err := packages.Load(nil, "std")
	if err != nil {
		return nil, fmt.Errorf("packages: failed to load std: %w", err)
	}

	ft.stdPackages = make(map[string]struct{})
	for _, stdPackage := range stdPackages {
		ft.stdPackages[stdPackage.PkgPath] = struct{}{}
	}
	ft.log("stdPackages: %+v\n", ft.stdPackages)

	return ft, nil
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
	// Return if not go file
	if !isGoFile(filepath.Base(path)) {
		return ErrNotGoFile
	}

	// Read file first
	pathBytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("os: failed to read file: [%s] %w", path, err)
	}

	// Parse ast
	pathASTFile, err := ft.wrapParseAST(path, pathBytes)
	if err != nil {
		return err
	}

	// Parse imports
	importsAST, err := ft.parseImports(pathASTFile)
	if err != nil {
		return err
	}

	// Return if empty imports
	if len(importsAST) == 0 {
		return nil
	}
	ft.log("importsAST: %+v\n", importsAST)

	groupImports, err := ft.groupImports(importsAST)
	if err != nil {
		return err
	}
	ft.log("groupImports: %+v\n", groupImports)

	return nil
}

func (ft *Formatter) wrapParseAST(path string, pathBytes []byte) (*ast.File, error) {
	fset := token.NewFileSet()

	parserMode := parser.Mode(0)
	parserMode |= parser.ParseComments

	pathASTFile, err := parser.ParseFile(fset, path, pathBytes, parserMode)
	if err != nil {
		return nil, fmt.Errorf("parser: failed to parse file [%s]: %w", path, err)
	}

	// Ignore generated file
	if isGoGenerated(pathASTFile) {
		return nil, ErrGoGeneratedFile
	}

	return pathASTFile, nil
}

// Copy from goimports-reviser
func (ft *Formatter) parseImports(pathASTFile *ast.File) (map[string]*ast.ImportSpec, error) {
	result := make(map[string]*ast.ImportSpec)

	for _, decl := range pathASTFile.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		if genDecl.Tok != token.IMPORT {
			continue
		}

		for _, spec := range genDecl.Specs {
			importSpec, ok := spec.(*ast.ImportSpec)
			if !ok {
				continue
			}

			var importNameAndPath string
			if importSpec.Name != nil {
				// Handle alias import
				// xyz "github.com/abc/xyz/123"
				importNameAndPath = importSpec.Name.String() + " " + importSpec.Path.Value
			} else {
				// Handle normal import
				// "github.com/abc/xyz"
				importNameAndPath = importSpec.Path.Value
			}

			result[importNameAndPath] = importSpec
		}
	}

	return result, nil
}

// Copy from goimports-reviser
// Group imports to std, third-party, company if exist, local
func (ft *Formatter) groupImports(importsAST map[string]*ast.ImportSpec) (map[string][]string, error) {
	result := make(map[string][]string)
	result[stdImport] = make([]string, 0, 8)
	result[thirdPartyImport] = make([]string, 0, 8)
	if ft.companyPrefix != "" {
		result[companyImport] = make([]string, 0, 8)
	}
	result[localImport] = make([]string, 0, 8)

	for importNameAndPath, importAST := range importsAST {
		// "github.com/abc/xyz" -> github.com/abc/xyz
		importPath := strings.Trim(importAST.Path.Value, "\"")

		if _, ok := ft.stdPackages[importPath]; ok {
			result[stdImport] = append(result[stdImport], importNameAndPath)
			continue
		}
	}

	return result, nil
}

// Wrap log.Printf with verbose flag
func (ft *Formatter) log(format string, v ...any) {
	if ft.isVerbose {
		log.Printf(format, v...)
	}
}
