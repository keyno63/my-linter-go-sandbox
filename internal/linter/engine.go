package linter

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

type Issue struct {
	FilePath string
	Line     int
	Column   int
	RuleName string
	Message  string
}

type Rule interface {
	Name() string
	CheckFile(fset *token.FileSet, filePath string, file *ast.File) []Issue
}

type Engine struct {
	rules []Rule
}

func NewEngine(rules ...Rule) *Engine {
	return &Engine{rules: rules}
}

func (e *Engine) Run(rootDir string) ([]Issue, error) {
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

	var issues []Issue
	for _, path := range files {
		fset := token.NewFileSet()
		file, parseErr := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if parseErr != nil {
			issues = append(issues, Issue{
				FilePath: path,
				Line:     1,
				Column:   1,
				RuleName: "parse",
				Message:  parseErr.Error(),
			})
			continue
		}

		for _, rule := range e.rules {
			ruleIssues := rule.CheckFile(fset, path, file)
			issues = append(issues, ruleIssues...)
		}
	}

	sort.Slice(issues, func(i, j int) bool {
		if issues[i].FilePath != issues[j].FilePath {
			return issues[i].FilePath < issues[j].FilePath
		}
		if issues[i].Line != issues[j].Line {
			return issues[i].Line < issues[j].Line
		}
		return issues[i].Column < issues[j].Column
	})

	return issues, nil
}
