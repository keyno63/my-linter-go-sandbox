package linter_test

import (
	"os"
	"path/filepath"
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
