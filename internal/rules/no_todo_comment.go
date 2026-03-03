package rules

import (
	"go/ast"
	"go/token"
	"strings"

	"my-linter-go-sandbox/internal/linter"
)

type NoTodoCommentRule struct{}

func NewNoTodoCommentRule() *NoTodoCommentRule {
	return &NoTodoCommentRule{}
}

func (r *NoTodoCommentRule) Name() string {
	return "no-todo-comment"
}

func (r *NoTodoCommentRule) CheckFile(fset *token.FileSet, filePath string, file *ast.File) []linter.Issue {
	var issues []linter.Issue

	for _, cg := range file.Comments {
		for _, c := range cg.List {
			if !strings.Contains(strings.ToUpper(c.Text), "TODO") {
				continue
			}

			pos := fset.Position(c.Pos())
			issues = append(issues, linter.Issue{
				FilePath: filePath,
				Line:     pos.Line,
				Column:   pos.Column,
				RuleName: r.Name(),
				Message:  "TODO comment is not allowed",
			})
		}
	}

	return issues
}
