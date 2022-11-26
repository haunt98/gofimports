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
	"sort"
	"strings"
	"sync"

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
	ft.log("stdPackages: %+v\n", ft.stdPackages)

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

	// TODO: Find dir go.mod package name
	pkgName, err := ft.moduleName(path)
	if err != nil {
		return err
	}
	ft.log("pkgName: %+v\n", pkgName)

	groupImports, err := ft.groupImports(importsAST)
	if err != nil {
		return err
	}
	ft.log("groupImports: %+v\n", groupImports)

	ft.muFormattedPaths.Lock()
	ft.formattedPaths[path] = struct{}{}
	ft.muFormattedPaths.Unlock()

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

	// TODO: not sure if this match gofumpt output, but at lease it is sorted
	sort.Strings(result[stdImport])
	sort.Strings(result[thirdPartyImport])
	if ft.companyPrefix != "" {
		sort.Strings(result[companyImport])
	}
	sort.Strings(result[localImport])

	return result, nil
}

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
	ft.log("goModPath: %+v\n", goModPath)

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

// Wrap log.Printf with verbose flag
func (ft *Formatter) log(format string, v ...any) {
	if ft.isVerbose {
		log.Printf(format, v...)
	}
}
