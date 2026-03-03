package rules_test

import (
	"bytes"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
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

func TestDeclarationOrderRule_FixFile_ReordersDeclarations(t *testing.T) {
	t.Parallel()

	src := `package p
func Zeta() {}
type Banana struct{}
func (b Banana) MethodB() {}
type Apple struct{}
func (a Apple) MethodB() {}
func (a Apple) MethodA() {}
func Alpha() {}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse file: %v", err)
	}

	rule := rules.NewDeclarationOrderRule()
	changed, err := rule.FixFile(fset, "sample.go", file)
	if err != nil {
		t.Fatalf("fix file: %v", err)
	}
	if !changed {
		t.Fatalf("expected file to change")
	}

	var out bytes.Buffer
	if err := format.Node(&out, fset, file); err != nil {
		t.Fatalf("format node: %v", err)
	}

	got := out.String()
	expectedSnippets := []string{
		"type Apple struct{}",
		"func (a Apple) MethodA()",
		"func (a Apple) MethodB()",
		"type Banana struct{}",
		"func (b Banana) MethodB()",
		"func Alpha()",
		"func Zeta()",
	}
	lastPos := -1
	for _, snippet := range expectedSnippets {
		pos := strings.Index(got, snippet)
		if pos == -1 {
			t.Fatalf("expected snippet not found: %q\noutput:\n%s", snippet, got)
		}
		if pos < lastPos {
			t.Fatalf("snippet order is wrong: %q\noutput:\n%s", snippet, got)
		}
		lastPos = pos
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
