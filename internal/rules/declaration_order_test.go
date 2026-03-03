package rules_test

import (
	"go/parser"
	"go/token"
	"testing"

	"my-linter-go-sandbox/internal/linter"
	"my-linter-go-sandbox/internal/rules"
)

func TestDeclarationOrderRule_NoIssueWhenOrdered(t *testing.T) {
	t.Parallel()

	src := `package p
type Apple struct{}
func (a Apple) Alpha() {}
func (a Apple) Beta() {}
type Banana struct{}
func (b *Banana) Create() {}
func AFunc() {}
func BFunc() {}
`

	rule := rules.NewDeclarationOrderRule()
	issues := runRule(t, rule, src)

	if len(issues) != 0 {
		t.Fatalf("expected 0 issues, got %d", len(issues))
	}
}

func TestDeclarationOrderRule_FindsStructSortViolation(t *testing.T) {
	t.Parallel()

	src := `package p
type Banana struct{}
type Apple struct{}
`

	rule := rules.NewDeclarationOrderRule()
	issues := runRule(t, rule, src)

	if len(issues) == 0 {
		t.Fatalf("expected issues, got 0")
	}
}

func TestDeclarationOrderRule_FindsMethodGroupingViolation(t *testing.T) {
	t.Parallel()

	src := `package p
type Apple struct{}
type Banana struct{}
func (a Apple) Alpha() {}
`

	rule := rules.NewDeclarationOrderRule()
	issues := runRule(t, rule, src)

	if len(issues) == 0 {
		t.Fatalf("expected issues, got 0")
	}
}

func TestDeclarationOrderRule_FindsFunctionSortViolation(t *testing.T) {
	t.Parallel()

	src := `package p
type Apple struct{}
func BFunc() {}
func AFunc() {}
`

	rule := rules.NewDeclarationOrderRule()
	issues := runRule(t, rule, src)

	if len(issues) == 0 {
		t.Fatalf("expected issues, got 0")
	}
}

func runRule(t *testing.T, rule *rules.DeclarationOrderRule, src string) []linter.Issue {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse file: %v", err)
	}

	return rule.CheckFile(fset, "sample.go", file)
}
