package main

import (
	"flag"
	"fmt"
	"os"

	"my-linter-go-sandbox/internal/linter"
	"my-linter-go-sandbox/internal/rules"
)

func main() {
	fix := flag.Bool("fix", false, "auto-fix fixable rule violations")
	flag.Parse()

	targetDir := "."
	if flag.NArg() > 0 {
		targetDir = flag.Arg(0)
	}

	engine := linter.NewEngine(
		rules.NewNoTodoCommentRule(),
		rules.NewDeclarationOrderRule(),
	)

	if *fix {
		issues, fixedFiles, err := engine.Fix(targetDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "linter failed: %v\n", err)
			os.Exit(2)
		}

		for _, path := range fixedFiles {
			fmt.Fprintf(os.Stderr, "fixed: %s\n", path)
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
