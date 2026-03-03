package linter

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
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

type FixableRule interface {
	Rule
	FixFile(fset *token.FileSet, filePath string, file *ast.File) (bool, error)
}

type Engine struct {
	rules []Rule
}

func NewEngine(rules ...Rule) *Engine {
	return &Engine{rules: rules}
}

func (e *Engine) Run(rootDir string) ([]Issue, error) {
	files, err := collectTargetFiles(rootDir)
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

	return sortIssues(issues), nil
}

func (e *Engine) Fix(rootDir string) ([]Issue, []string, error) {
	files, err := collectTargetFiles(rootDir)
	if err != nil {
		return nil, nil, err
	}

	var fixedFiles []string
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

		changed := false
		for _, rule := range e.rules {
			fixable, ok := rule.(FixableRule)
			if !ok {
				continue
			}
			ruleChanged, fixErr := fixable.FixFile(fset, path, file)
			if fixErr != nil {
				issues = append(issues, Issue{
					FilePath: path,
					Line:     1,
					Column:   1,
					RuleName: rule.Name(),
					Message:  "auto-fix failed: " + fixErr.Error(),
				})
				continue
			}
			if ruleChanged {
				changed = true
			}
		}

		if changed {
			var out bytes.Buffer
			if err := format.Node(&out, fset, file); err != nil {
				issues = append(issues, Issue{
					FilePath: path,
					Line:     1,
					Column:   1,
					RuleName: "format",
					Message:  "format failed: " + err.Error(),
				})
				continue
			}

			if err := os.WriteFile(path, out.Bytes(), 0o644); err != nil {
				issues = append(issues, Issue{
					FilePath: path,
					Line:     1,
					Column:   1,
					RuleName: "write",
					Message:  "write failed: " + err.Error(),
				})
				continue
			}

			fixedFiles = append(fixedFiles, path)

			// Re-parse the updated file, then run checks against the result.
			fset = token.NewFileSet()
			file, parseErr = parser.ParseFile(fset, path, nil, parser.ParseComments)
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
		}

		for _, rule := range e.rules {
			ruleIssues := rule.CheckFile(fset, path, file)
			issues = append(issues, ruleIssues...)
		}
	}

	return sortIssues(issues), fixedFiles, nil
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

func sortIssues(issues []Issue) []Issue {
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].FilePath != issues[j].FilePath {
			return issues[i].FilePath < issues[j].FilePath
		}
		if issues[i].Line != issues[j].Line {
			return issues[i].Line < issues[j].Line
		}
		return issues[i].Column < issues[j].Column
	})

	return issues
}
