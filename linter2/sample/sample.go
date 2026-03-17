package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// This feature tells you whether fmt.Println is used.

func main() {
	targetDir := "."
	if len(os.Args) < 1 {
		fmt.Println("usage: go run sample.go")
		os.Exit(1)
	}
	if len(os.Args) >= 2 {
		targetDir = os.Args[1]
	}

	targetFiles, err := collectTargetFiles(targetDir)
	if err != nil {
		fmt.Printf("failed to collect files: %s\n", err.Error())
		os.Exit(1)
	}
	for _, filename := range targetFiles {
		fmt.Println(filename)
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, filename, nil, 0)
		if err != nil {
			panic(err)
		}

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			xIdent, ok := selector.X.(*ast.Ident)
			if !ok {
				return true
			}

			if xIdent.Name == "fmt" && selector.Sel.Name == "Println" {
				pos := fset.Position(call.Pos())
				// print `target filename:10:2: avoid using fmt.Println`
				fmt.Printf(
					"%s:%d:%d: avoid using fmt.Println\n",
					pos.Filename,
					pos.Line,
					pos.Column,
				)
			}

			return true
		})

	}
}

func collectTargetFiles(rootDir string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}
