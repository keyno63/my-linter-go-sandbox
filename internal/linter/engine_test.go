package linter_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"my-linter-go-sandbox/internal/linter"
	"my-linter-go-sandbox/internal/rules"
)

func TestEngineRun_FindsViolationInGoFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	file := filepath.Join(root, "main.go")
	src := "package main\n// TODO: fix me\nfunc main() {}\n"

	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	engine := linter.NewEngine(rules.NewNoTodoCommentRule())
	issues, err := engine.Run(root)
	if err != nil {
		t.Fatalf("run engine: %v", err)
	}

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	if issues[0].RuleName != "no-todo-comment" {
		t.Fatalf("unexpected rule name: %s", issues[0].RuleName)
	}
}

func TestEngineRun_IgnoresTestFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	file := filepath.Join(root, "main_test.go")
	src := "package main\n// TODO: allowed in ignored file\n"

	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	engine := linter.NewEngine(rules.NewNoTodoCommentRule())
	issues, err := engine.Run(root)
	if err != nil {
		t.Fatalf("run engine: %v", err)
	}

	if len(issues) != 0 {
		t.Fatalf("expected 0 issues, got %d", len(issues))
	}
}

func TestEngineFix_ReordersDeclarationOrder(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	file := filepath.Join(root, "main.go")
	src := "package main\nfunc Z() {}\ntype B struct{}\nfunc (b B) M() {}\ntype A struct{}\nfunc (a A) M2() {}\nfunc (a A) M1() {}\nfunc AFunc() {}\n"

	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	engine := linter.NewEngine(rules.NewDeclarationOrderRule())
	issues, fixedFiles, err := engine.Fix(root)
	if err != nil {
		t.Fatalf("fix engine: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected 0 issues, got %d", len(issues))
	}
	if len(fixedFiles) != 1 {
		t.Fatalf("expected 1 fixed file, got %d", len(fixedFiles))
	}

	got, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read fixed file: %v", err)
	}

	output := string(got)
	expected := []string{
		"type A struct{}",
		"func (a A) M1()",
		"func (a A) M2()",
		"type B struct{}",
		"func (b B) M()",
		"func AFunc()",
		"func Z()",
	}
	lastPos := -1
	for _, snippet := range expected {
		pos := strings.Index(output, snippet)
		if pos == -1 {
			t.Fatalf("expected snippet not found: %q\noutput:\n%s", snippet, output)
		}
		if pos < lastPos {
			t.Fatalf("snippet order is wrong: %q\noutput:\n%s", snippet, output)
		}
		lastPos = pos
	}
}
