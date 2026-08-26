package imports

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/pkg/diff"
	"github.com/sourcegraph/conc/pool"
	"golang.org/x/mod/modfile"
)

//go:embed data/std_packages.txt
var stdPackages []byte

const (
	// Use for group imports
	stdImport        = "std"
	thirdPartyImport = "third-party"
	companyImport    = "company"
	localImport      = "local"
)

const defaultMaxGoroutines = 32

var (
	ErrEmptyPaths       = errors.New("empty paths")
	ErrNotGoFile        = errors.New("not go file")
	ErrGoGeneratedFile  = errors.New("go generated file")
	ErrAlreadyFormatted = errors.New("already formatted")
	ErrEmptyImport      = errors.New("empty import")
	ErrGoModNotExist    = errors.New("go mod not exist")
	ErrGoModEmptyModule = errors.New("go mod empty module")
	ErrNotDSTGenDecl    = errors.New("not dst.GenDecl")
)

// stdPackages -> save std packages for later search.
// moduleNames -> map path to its go.mod module name.
// formattedPaths -> make sure we not format path more than 1 time.
type Formatter struct {
	stdPackages      map[string]struct{}
	moduleNames      map[string]string
	formattedPaths   map[string]struct{}
	companyPrefixes  map[string]struct{}
	p                *pool.ErrorPool
	muFormattedPaths sync.RWMutex
	isList           bool
	isWrite          bool
	isDiff           bool
	isVerbose        bool
	isStock          bool
}

func NewFormmater(opts ...FormatterOptionFn) (*Formatter, error) {
	ft := &Formatter{}

	for _, opt := range opts {
		opt(ft)
	}

	ft.stdPackages = make(map[string]struct{})

	for pkgPath := range strings.SplitSeq(strings.TrimSpace(string(stdPackages)), "\n") {
		pkgPath = strings.TrimSpace(pkgPath)
		if pkgPath == "" {
			continue
		}

		ft.stdPackages[pkgPath] = struct{}{}
	}

	ft.moduleNames = make(map[string]string)
	ft.formattedPaths = make(map[string]struct{})

	ft.p = pool.New().
		WithErrors().
		WithMaxGoroutines(defaultMaxGoroutines)

	return ft, nil
}

// Accept a list of files or directories
func (ft *Formatter) Format(paths ...string) error {
	if len(paths) == 0 {
		return ErrEmptyPaths
	}

	// Logic switch case copy from goimports, gofumpt
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}

		dirPath := filepath.Clean(path)
		var goModPath string
		var foundGoMod bool
		for {
			goModPath = filepath.Join(dirPath, "go.mod")
			fileInfo, err := os.Stat(goModPath)
			if err == nil &&
				!fileInfo.IsDir() {
				foundGoMod = true
				break
			}

			// Check ..
			if dirPath == filepath.Dir(dirPath) {
				// Reach root
				break
			}

			dirPath = filepath.Dir(dirPath)
		}

		if !foundGoMod {
			ft.log("Format: go.mod not found for path: [%s], skip", path)
			continue
		}

		goModPathBytes, err := os.ReadFile(goModPath)
		if err != nil {
			return fmt.Errorf("os: failed to read file: [%s] %w", goModPath, err)
		}

		goModFile, err := modfile.Parse(goModPath, goModPathBytes, nil)
		if err != nil {
			return fmt.Errorf("modfile: failed to parse: [%s] %w", goModPath, err)
		}

		moduleName := goModFile.Module.Mod.Path

		switch dir, err := os.Stat(path); {
		case err != nil:
			return fmt.Errorf("os: failed to stat: [%s] %w", path, err)
		case dir.IsDir():
			if err := ft.formatDir(path, moduleName); err != nil {
				return err
			}
		default:
			ft.p.Go(func() error {
				if err := ft.formatFile(path, moduleName); err != nil {
					if ft.isIgnoreError(err) {
						return nil
					}

					return err
				}

				return nil
			})
		}
	}

	if err := ft.p.Wait(); err != nil {
		return err
	}

	return nil
}

// Copy from gofumpt
func (ft *Formatter) formatDir(path, moduleName string) error {
	if err := filepath.WalkDir(path, func(path string, dirEntry fs.DirEntry, err error) error {
		if filepath.Base(path) == "vendor" {
			return filepath.SkipDir
		}

		if err != nil {
			return err
		}

		if dirEntry.IsDir() {
			return nil
		}

		ft.p.Go(func() error {
			if err := ft.formatFile(path, moduleName); err != nil {
				if ft.isIgnoreError(err) {
					return nil
				}

				return err
			}

			return nil
		})

		return nil
	}); err != nil {
		return fmt.Errorf("filepath: failed to walk dir: [%s] %w", path, err)
	}

	return nil
}

func (ft *Formatter) formatFile(path, moduleName string) error {
	ft.muFormattedPaths.Lock()
	defer ft.muFormattedPaths.Unlock()

	if _, ok := ft.formattedPaths[path]; ok {
		return nil
	}

	ft.formattedPaths[path] = struct{}{}

	// Return if not go file
	if !isGoFile(filepath.Base(path)) {
		return ErrNotGoFile
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("os: failed to open file: [%s] %w", path, err)
	}
	defer f.Close()

	fileInfo, err := f.Stat()
	if err != nil {
		return fmt.Errorf("os: failed to stat: [%s] %w", path, err)
	}

	pathBytes, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("io: failed to read all: [%s] %w", path, err)
	}

	formattedBytes, err := ft.formatImports(path, pathBytes, moduleName)
	if err != nil {
		return err
	}

	if bytes.Equal(pathBytes, formattedBytes) {
		return ErrAlreadyFormatted
	}

	if ft.isList {
		fmt.Println("Formatted: ", path)
	}

	if ft.isWrite {
		if err := os.WriteFile(path, formattedBytes, fileInfo.Mode().Perm()); err != nil {
			return fmt.Errorf("os: failed to write file: [%s] %w", path, err)
		}
	}

	if ft.isDiff {
		if err := diff.Text(path+" before", path+" after", pathBytes, formattedBytes, os.Stdout); err != nil {
			return fmt.Errorf("diff: failed to slices: %w", err)
		}
	}

	return nil
}

// Copy from goimports, gofumpt, goimports-reviser.
// First parse ast.
// Then group imports.
// Then format imports.
// Then print to bytes.
func (ft *Formatter) formatImports(
	path string,
	pathBytes []byte,
	moduleName string,
) ([]byte, error) {
	// Parse ast
	fset := token.NewFileSet()

	// Copy from gofumpt
	parserMode := parser.Mode(0)
	parserMode |= parser.ParseComments
	parserMode |= parser.SkipObjectResolution

	astFile, err := parser.ParseFile(fset, path, pathBytes, parserMode)
	if err != nil {
		return nil, fmt.Errorf("parser: failed to parse file [%s]: %w", path, err)
	}

	// Ignore generated file
	if isGoGenerated(astFile) {
		return nil, ErrGoGeneratedFile
	}

	dec := decorator.NewDecorator(fset)
	dstFile, err := dec.DecorateFile(astFile)
	if err != nil {
		return nil, fmt.Errorf("decorator: failed to parse file [%s]: %w", path, err)
	}
	if len(dstFile.Imports) == 0 {
		return nil, ErrEmptyImport
	}
	ft.logDSTImportSpecs("formatImports: dstImportSpecs", dstFile.Imports)

	groupedDSTImportSpecs, err := ft.groupDSTImportSpecs(dstFile.Imports, moduleName)
	if err != nil {
		return nil, err
	}

	formattedDSTImportSpecs, err := ft.formatDSTImportSpecs(groupedDSTImportSpecs)
	if err != nil {
		return nil, err
	}
	ft.logDSTImportSpecs("formatImports: formattedDSTImportSpecs: ", formattedDSTImportSpecs)

	// First update
	dstFile.Imports = formattedDSTImportSpecs

	// Find all import block
	// `import (...)`
	// Buffer only 1 because *.go normally has 1 import block
	importDeclIdxes := make([]int, 0, 1)
	for i, decl := range dstFile.Decls {
		genDecl, ok := decl.(*dst.GenDecl)
		if !ok ||
			genDecl.Tok != token.IMPORT {
			continue
		}

		importDeclIdxes = append(importDeclIdxes, i)
	}

	if len(importDeclIdxes) == 0 {
		return nil, ErrNotDSTGenDecl
	}

	// Merge all import specs into the first import block
	firstImportGenDecl, ok := dstFile.Decls[importDeclIdxes[0]].(*dst.GenDecl)
	if !ok {
		return nil, ErrNotDSTGenDecl
	}

	firstImportGenDecl.Specs = make([]dst.Spec, 0, len(formattedDSTImportSpecs))
	for _, importSpec := range formattedDSTImportSpecs {
		firstImportGenDecl.Specs = append(firstImportGenDecl.Specs, importSpec)
	}

	// Make sure first block has Lparen and Rparen if there are multiple import specs
	if len(firstImportGenDecl.Specs) > 1 &&
		!firstImportGenDecl.Lparen {
		firstImportGenDecl.Lparen = true
		firstImportGenDecl.Rparen = true
	}

	// Drop the remaining import blocks
	if len(importDeclIdxes) > 1 {
		removedIdxes := make(map[int]struct{}, len(importDeclIdxes)-1)
		for _, idx := range importDeclIdxes[1:] {
			removedIdxes[idx] = struct{}{}
		}

		filteredDecls := make([]dst.Decl, 0, len(dstFile.Decls)-len(removedIdxes))
		for i, decl := range dstFile.Decls {
			if _, ok := removedIdxes[i]; ok {
				continue
			}

			filteredDecls = append(filteredDecls, decl)
		}

		dstFile.Decls = filteredDecls
	}

	var b bytes.Buffer

	if err := decorator.Fprint(&b, dstFile); err != nil {
		return nil, fmt.Errorf("decorator: failed to fprint [%s]: %w", path, err)
	}

	return b.Bytes(), nil
}

func (ft *Formatter) groupDSTImportSpecs(importSpecs []*dst.ImportSpec, moduleName string) (map[string][]*dst.ImportSpec, error) {
	result := make(map[string][]*dst.ImportSpec)
	result[stdImport] = make([]*dst.ImportSpec, 0, 8)
	result[thirdPartyImport] = make([]*dst.ImportSpec, 0, 8)
	if !ft.isStock {
		// Only split company, local imports if not stock
		// Otherwise everything is third party imports
		if len(ft.companyPrefixes) != 0 {
			result[companyImport] = make([]*dst.ImportSpec, 0, 8)
		}
		result[localImport] = make([]*dst.ImportSpec, 0, 8)
	}

	for _, importSpec := range importSpecs {
		// "github.com/abc/xyz" -> github.com/abc/xyz
		importPath := strings.Trim(importSpec.Path.Value, `"`)

		if _, ok := ft.stdPackages[importPath]; ok {
			result[stdImport] = append(result[stdImport], importSpec)
			continue
		}

		if !ft.isStock {
			// Local if module itself or a subpackage of it
			if importPath == moduleName ||
				strings.HasPrefix(importPath, moduleName+"/") {
				result[localImport] = append(result[localImport], importSpec)
				continue
			}

			if len(ft.companyPrefixes) != 0 {
				existImport := false
				for companyPrefix := range ft.companyPrefixes {
					if strings.HasPrefix(importPath, companyPrefix) {
						result[companyImport] = append(result[companyImport], importSpec)
						existImport = true
						break
					}
				}

				if existImport {
					continue
				}
			}
		}

		result[thirdPartyImport] = append(result[thirdPartyImport], importSpec)
	}

	ft.logDSTImportSpecs("groupDSTImportSpecs: stdImport", result[stdImport])
	ft.logDSTImportSpecs("groupDSTImportSpecs: thirdPartyImport", result[thirdPartyImport])
	if len(ft.companyPrefixes) != 0 {
		ft.logDSTImportSpecs("groupDSTImportSpecs: companyImport", result[companyImport])
	}
	ft.logDSTImportSpecs("groupDSTImportSpecs: localImport", result[localImport])

	return result, nil
}

func (ft *Formatter) formatDSTImportSpecs(groupedImportSpecs map[string][]*dst.ImportSpec,
) ([]*dst.ImportSpec, error) {
	result := make([]*dst.ImportSpec, 0, 32)

	appendToResultFn := func(groupImportType string) {
		importSpecs, ok := groupedImportSpecs[groupImportType]
		if !ok || len(importSpecs) == 0 {
			return
		}

		for _, importSpec := range importSpecs {
			importSpec.Decs.Before = dst.NewLine
			importSpec.Decs.After = dst.NewLine
		}

		importSpecs[len(importSpecs)-1].Decs.After = dst.EmptyLine

		result = append(result, importSpecs...)
	}

	appendToResultFn(stdImport)
	appendToResultFn(thirdPartyImport)
	appendToResultFn(companyImport)
	appendToResultFn(localImport)

	if len(result) == 0 {
		return result, nil
	}

	result[len(result)-1].Decs.After = dst.NewLine

	return result, nil
}

func (ft *Formatter) isIgnoreError(err error) bool {
	return errors.Is(err, ErrNotGoFile) ||
		errors.Is(err, ErrGoGeneratedFile) ||
		errors.Is(err, ErrAlreadyFormatted) ||
		errors.Is(err, ErrEmptyImport)
}
