package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"slices"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

const outputPath = "internal/imports/data/std_packages.txt"

func main() {
	stdPackages, err := packages.Load(nil, "std")
	if err != nil {
		log.Printf("packages: failed to load std: %v", err)
		return
	}

	pkgPaths := make([]string, 0, len(stdPackages))
	for _, stdPackage := range stdPackages {
		if strings.HasPrefix(stdPackage.PkgPath, "vendor/") {
			continue
		}

		parts := strings.Split(stdPackage.PkgPath, "/")
		if slices.Contains(parts, "internal") {
			continue
		}

		pkgPaths = append(pkgPaths, stdPackage.PkgPath)
	}

	sort.Strings(pkgPaths)

	f, err := os.Create(outputPath)
	if err != nil {
		log.Printf("os: failed to create file: [%s] %v", outputPath, err)
		return
	}
	defer f.Close()

	w := bufio.NewWriter(f)

	for _, pkgPath := range pkgPaths {
		if _, err := fmt.Fprintln(w, pkgPath); err != nil {
			log.Printf("bufio: failed to write: %v", err)
			return
		}
	}

	if err := w.Flush(); err != nil {
		log.Printf("bufio: failed to flush: %v", err)
		return
	}
}
