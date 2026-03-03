# my-linter-go-sandbox

## My Linter (Go)

Custom Go linter CLI:

1. Recursively scans `.go` files under a target directory.
2. Applies rules that implement a shared `Rule` interface.
3. Prints violations to standard error.

### Run

```bash
go run ./cmd/my-linter .
```

Auto-fix fixable rules:

```bash
go run ./cmd/my-linter --fix .
```

Or build binary:

```bash
go build -o my-linter ./cmd/my-linter
./my-linter .
```

### Exit code

- `0`: no violations
- `1`: violation(s) found
- `2`: execution failure (e.g., walk error)

### Rule architecture

Rules are defined by this interface in `internal/linter/engine.go`:

```go
type Rule interface {
	Name() string
	CheckFile(fset *token.FileSet, filePath string, file *ast.File) []Issue
}
```

To add a new rule:

1. Add a struct implementing `Rule` under `internal/rules`.
2. Register it in `cmd/my-linter/main.go` via `linter.NewEngine(...)`.

### Built-in rules

- `no-todo-comment`: flags TODO comments.
- `declaration-order`: checks top-level declaration order in a file:
  - `struct` declarations must come before plain functions.
  - `struct` names must be sorted a-z.
  - methods must be placed under their struct and sorted a-z.
  - plain function names must be sorted a-z.
  - supports auto-fix with `--fix`.

## LICENSE

This repository is an MIT License.  
See [the License](./LICENSE) file.
