package main

import (
	"fmt"
	"os"

	"my-linter-go-sandbox/internal/linter"
	"my-linter-go-sandbox/internal/rules"
)

func main() {
	targetDir := "."
	if len(os.Args) > 1 {
		targetDir = os.Args[1]
	}

	engine := linter.NewEngine(
		rules.NewNoTodoCommentRule(),
	)

	issues, err := engine.Run(targetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "linter failed: %v\n", err)
		os.Exit(2)
	}

	if len(issues) == 0 {
		return
	}

	for _, issue := range issues {
		fmt.Fprintf(
			os.Stderr,
			"%s:%d:%d [%s] %s\n",
			issue.FilePath,
			issue.Line,
			issue.Column,
			issue.RuleName,
			issue.Message,
		)
	}

	os.Exit(1)
}
