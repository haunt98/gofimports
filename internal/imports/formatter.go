package imports

import (
	"bytes"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
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
	ErrAlreadyFormatted = errors.New("already formatted")
	ErrGoModNotExist    = errors.New("go mod not exist")
	ErrGoModEmptyModule = errors.New("go mod empty module")
)

// stdPackages -> save std packages for later search
//
// moduleNames -> map path to its go.mod module name
//
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
				if ft.isIgnoreError(err) {
					continue
				}

				return err
			}
		}
	}

	return nil
}

// Copy from gofumpt
func (ft *Formatter) formatDir(path string) error {
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

		if err := ft.formatFile(path); err != nil {
			if ft.isIgnoreError(err) {
				return nil
			}

			return err
		}

		return nil
	}); err != nil {
		return fmt.Errorf("filepath: failed to walk dir: [%s] %w", path, err)
	}

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
	ft.log("formatFile: moduleName: [%s]\n", moduleName)

	formattedBytes, err := ft.formatImports(path, pathBytes, moduleName)
	if err != nil {
		return err
	}

	if bytes.Compare(pathBytes, formattedBytes) == 0 {
		return ErrAlreadyFormatted
	}

	if ft.isList {
		fmt.Println("Formatted: ", path)
	}

	if ft.isWrite {
		if err := os.WriteFile(path, formattedBytes, 0o600); err != nil {
			return fmt.Errorf("os: failed to write file: [%s] %w", path, err)
		}
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

// Copy from goimports, gofumpt, goimports-reviser
// First parse ast
//
// # Then group imports
//
// # Then format imports
//
// # Then update ast decls
//
// Then print ast
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

	dstFile, err := decorator.Parse(pathBytes)
	if err != nil {
		return nil, fmt.Errorf("decorator: failed to parse file [%s]: %w", path, err)
	}
	ft.logDSTImportSpecs("formatImports: dstImportSpecs", dstFile.Imports)

	groupedDSTImportSpecs, err := ft.groupDSTImportSpecs(
		dstFile.Imports,
		moduleName,
	)
	if err != nil {
		return nil, err
	}

	formattedDSTImportSpecs, err := ft.formatDSTImportSpecs(groupedDSTImportSpecs)
	if err != nil {
		return nil, err
	}
	ft.logDSTImportSpecs("formatImports: formattedDSTImportSpecs: ", formattedDSTImportSpecs)

	dstFile.Imports = formattedDSTImportSpecs

	var buf bytes.Buffer
	if err := decorator.Fprint(&buf, dstFile); err != nil {
		return nil, fmt.Errorf("decorator: failed to fprint [%s]: %w", path, err)
	}

	return buf.Bytes(), nil
}

func (ft *Formatter) groupDSTImportSpecs(
	importSpecs []*dst.ImportSpec,
	moduleName string,
) (map[string][]*dst.ImportSpec, error) {
	result := make(map[string][]*dst.ImportSpec)
	result[stdImport] = make([]*dst.ImportSpec, 0, 8)
	result[thirdPartyImport] = make([]*dst.ImportSpec, 0, 8)
	if ft.companyPrefix != "" {
		result[companyImport] = make([]*dst.ImportSpec, 0, 8)
	}
	result[localImport] = make([]*dst.ImportSpec, 0, 8)

	for _, importSpec := range importSpecs {
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

	ft.logDSTImportSpecs("groupDSTImportSpecs: stdImport", result[stdImport])
	ft.logDSTImportSpecs("groupDSTImportSpecs: thirdPartyImport", result[thirdPartyImport])
	if ft.companyPrefix != "" {
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

	result[len(result)-1].Decs.After = dst.NewLine

	return result, nil
}

// Copy from goimports-reviser
// Get module name from go.mod of path
// If current path doesn't have go.mod, recursive find its parent path
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

func (ft *Formatter) isIgnoreError(err error) bool {
	return errors.Is(err, ErrNotGoFile) ||
		errors.Is(err, ErrGoGeneratedFile) ||
		errors.Is(err, ErrAlreadyFormatted)
}
