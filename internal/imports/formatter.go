package imports

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pkg/diff"
	"golang.org/x/mod/modfile"
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
	ErrEmptyPaths       = errors.New("empty paths")
	ErrNotGoFile        = errors.New("not go file")
	ErrGoGeneratedFile  = errors.New("go generated file")
	ErrGoModNotExist    = errors.New("go mod not exist")
	ErrGoModEmptyModule = errors.New("go mod empty module")
)

// stdPackages -> save std packages for later search
// moduleNames -> map path to its go.mod module name
// formattedPaths -> make sure we not format path more than 1 time
type Formatter struct {
	stdPackages      map[string]struct{}
	moduleNames      map[string]string
	formattedPaths   map[string]struct{}
	companyPrefix    string
	muModuleNames    sync.RWMutex
	muFormattedPaths sync.RWMutex
	isList           bool
	isWrite          bool
	isDiff           bool
	isVerbose        bool
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
	// ft.log("NewFormmater: stdPackages: %+v\n", ft.stdPackages)

	ft.moduleNames = make(map[string]string)
	ft.formattedPaths = make(map[string]struct{})

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
	ft.muFormattedPaths.RLock()
	if _, ok := ft.formattedPaths[path]; ok {
		ft.muFormattedPaths.RUnlock()
		return nil
	}
	ft.muFormattedPaths.RUnlock()

	// Return if not go file
	if !isGoFile(filepath.Base(path)) {
		return ErrNotGoFile
	}

	// Read file first
	pathBytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("os: failed to read file: [%s] %w", path, err)
	}

	// Get module name of path
	moduleName, err := ft.moduleName(path)
	if err != nil {
		return err
	}
	ft.log("formatFile: moduleName: %+v\n", moduleName)

	formattedBytes, err := ft.formatImports(path, pathBytes, moduleName)
	if err != nil {
		return err
	}

	if ft.isList {
		fmt.Println(path)
	}

	if ft.isWrite {
		ft.log("TODO: write to file\n")
	}

	if ft.isDiff {
		if err := diff.Text(path+" before", path+" after", pathBytes, formattedBytes, os.Stdout); err != nil {
			return fmt.Errorf("diff: failed to slices: %w", err)
		}
	}

	ft.muFormattedPaths.Lock()
	ft.formattedPaths[path] = struct{}{}
	ft.muFormattedPaths.Unlock()

	return nil
}

// Copy from goimports-reviser
func (ft *Formatter) formatImports(
	path string,
	pathBytes []byte,
	moduleName string,
) ([]byte, error) {
	// Parse ast
	fset := token.NewFileSet()

	parserMode := parser.Mode(0)
	parserMode |= parser.ParseComments

	astFile, err := parser.ParseFile(fset, path, pathBytes, parserMode)
	if err != nil {
		return nil, fmt.Errorf("parser: failed to parse file [%s]: %w", path, err)
	}

	// Ignore generated file
	if isGoGenerated(astFile) {
		return nil, ErrGoGeneratedFile
	}

	// Extract imports
	importSpecs := make([]ast.Spec, 0, len(astFile.Imports))
	for _, importSpec := range astFile.Imports {
		importSpecs = append(importSpecs, importSpec)
	}
	ft.mustLogImportSpecs("formatImports: importSpecs", importSpecs)

	groupedImportSpecs, err := ft.groupImportSpecs(
		importSpecs,
		moduleName,
	)
	if err != nil {
		return nil, err
	}

	formattedImportSpecs, err := ft.formatImportSpecs(
		importSpecs,
		groupedImportSpecs,
	)
	if err != nil {
		return nil, err
	}
	ft.mustLogImportSpecs("formatImports: formattedImportSpecs: ", formattedImportSpecs)

	// Combine multi import decl into one
	isExistFirstImportDecl := false
	decls := make([]ast.Decl, 0, len(astFile.Decls))

	for _, decl := range astFile.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			decls = append(decls, decl)
			continue
		}

		if genDecl.Tok != token.IMPORT {
			decls = append(decls, decl)
			continue
		}

		// Ignore second import decl and more
		if isExistFirstImportDecl {
			continue
		}

		// Ignore empty import
		if len(genDecl.Specs) == 0 {
			continue
		}

		// First import decl take all
		isExistFirstImportDecl = true
		genDecl.Specs = formattedImportSpecs
		decls = append(decls, genDecl)
	}

	// Update ast decls
	astFile.Decls = decls

	// Print formatted bytes from formatted ast
	var formattedBytes []byte
	formattedBuffer := bytes.NewBuffer(formattedBytes)
	if err := printer.Fprint(formattedBuffer, fset, astFile); err != nil {
		return nil, err
	}

	return formattedBuffer.Bytes(), nil
}

// Copy from goimports-reviser
// Group imports to std, third-party, company if exist, local
func (ft *Formatter) groupImportSpecs(
	importSpecs []ast.Spec,
	moduleName string,
) (map[string][]*ast.ImportSpec, error) {
	result := make(map[string][]*ast.ImportSpec)
	result[stdImport] = make([]*ast.ImportSpec, 0, 8)
	result[thirdPartyImport] = make([]*ast.ImportSpec, 0, 8)
	if ft.companyPrefix != "" {
		result[companyImport] = make([]*ast.ImportSpec, 0, 8)
	}
	result[localImport] = make([]*ast.ImportSpec, 0, 8)

	for _, importSpec := range importSpecs {
		importSpec, ok := importSpec.(*ast.ImportSpec)
		if !ok {
			continue
		}

		// "github.com/abc/xyz" -> github.com/abc/xyz
		importPath := strings.Trim(importSpec.Path.Value, `"`)

		if _, ok := ft.stdPackages[importPath]; ok {
			result[stdImport] = append(result[stdImport], importSpec)
			continue
		}

		if strings.HasPrefix(importPath, moduleName) {
			result[localImport] = append(result[localImport], importSpec)
			continue
		}

		if ft.companyPrefix != "" &&
			strings.HasPrefix(importPath, ft.companyPrefix) {
			result[companyImport] = append(result[companyImport], importSpec)
			continue
		}

		result[thirdPartyImport] = append(result[thirdPartyImport], importSpec)
	}

	ft.logImportSpecs("stdImport", result[stdImport])
	ft.logImportSpecs("thirdPartyImport", result[thirdPartyImport])
	if ft.companyPrefix != "" {
		ft.logImportSpecs("companyImport", result[companyImport])
	}
	ft.logImportSpecs("localImport", result[localImport])

	return result, nil
}

// Copy from goimports-reviser
func (ft *Formatter) formatImportSpecs(
	importSpecs []ast.Spec,
	groupedImportSpecs map[string][]*ast.ImportSpec,
) ([]ast.Spec, error) {
	result := make([]ast.Spec, 0, len(importSpecs))

	appendToResultFn := func(groupImportType string) {
		if specs, ok := groupedImportSpecs[groupImportType]; ok && len(specs) != 0 {
			if len(result) != 0 {
				result = append(result, &ast.ImportSpec{
					Path: &ast.BasicLit{
						Value: "",
						Kind:  token.STRING,
					},
				})
			}

			for _, spec := range specs {
				result = append(result, &ast.ImportSpec{
					Path: &ast.BasicLit{
						Value: ft.importNameAndPath(spec),
						Kind:  token.IMPORT,
					},
				})
			}
		}
	}

	appendToResultFn(stdImport)
	appendToResultFn(thirdPartyImport)
	if ft.companyPrefix != "" {
		appendToResultFn(companyImport)
	}
	appendToResultFn(localImport)

	return result, nil
}

// Copy from goimports-reviser
func (ft *Formatter) moduleName(path string) (string, error) {
	ft.muModuleNames.RLock()
	if pkgName, ok := ft.moduleNames[path]; ok {
		ft.muModuleNames.RUnlock()
		return pkgName, nil
	}
	ft.muModuleNames.RUnlock()

	// Copy from goimports-reviser
	// Check path/go.mod first
	// If not exist -> check ../go.mod
	// Assume path is dir path, maybe wrong but it is ok for now
	dirPath := filepath.Clean(path)
	var goModPath string
	for {
		goModPath = filepath.Join(dirPath, "go.mod")
		fileInfo, err := os.Stat(goModPath)
		if err == nil && !fileInfo.IsDir() {
			break
		}

		// Check ..
		if dirPath == filepath.Dir(dirPath) {
			// Reach root
			break
		}

		dirPath = filepath.Dir(dirPath)
	}

	if goModPath == "" {
		return "", ErrGoModNotExist
	}
	ft.log("moduleName: goModPath: %+v\n", goModPath)

	goModPathBytes, err := os.ReadFile(goModPath)
	if err != nil {
		return "", fmt.Errorf("os: failed to read file: [%s] %w", goModPath, err)
	}

	goModFile, err := modfile.Parse(goModPath, goModPathBytes, nil)
	if err != nil {
		return "", fmt.Errorf("modfile: failed to parse: [%s] %w", goModPath, err)
	}

	result := goModFile.Module.Mod.Path
	if result == "" {
		return "", ErrGoModEmptyModule
	}

	ft.muModuleNames.Lock()
	ft.moduleNames[path] = result
	ft.muModuleNames.Unlock()

	return result, nil
}

func (ft *Formatter) importNameAndPath(importSpec *ast.ImportSpec) string {
	if importSpec == nil {
		return ""
	}

	if importSpec.Name != nil {
		return importSpec.Name.String() + " " + importSpec.Path.Value
	}

	return importSpec.Path.Value
}
