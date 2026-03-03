package rules

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"my-linter-go-sandbox/internal/linter"
)

type DeclarationOrderRule struct{}

func NewDeclarationOrderRule() *DeclarationOrderRule {
	return &DeclarationOrderRule{}
}

func (r *DeclarationOrderRule) Name() string {
	return "declaration-order"
}

func (r *DeclarationOrderRule) CheckFile(fset *token.FileSet, filePath string, file *ast.File) []linter.Issue {
	var issues []linter.Issue

	const (
		phaseStructs = iota
		phaseFuncs
	)

	phase := phaseStructs
	lastStruct := ""
	currentStruct := ""
	lastMethodByStruct := map[string]string{}
	lastFunc := ""

	for _, decl := range file.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.TYPE {
			for _, spec := range gen.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if _, ok := typeSpec.Type.(*ast.StructType); !ok {
					continue
				}

				structName := typeSpec.Name.Name
				pos := fset.Position(typeSpec.Pos())

				if phase == phaseFuncs {
					issues = append(issues, linter.Issue{
						FilePath: filePath,
						Line:     pos.Line,
						Column:   pos.Column,
						RuleName: r.Name(),
						Message:  "struct declarations must come before function declarations",
					})
				}

				if lastStruct != "" && lessFold(structName, lastStruct) {
					issues = append(issues, linter.Issue{
						FilePath: filePath,
						Line:     pos.Line,
						Column:   pos.Column,
						RuleName: r.Name(),
						Message: fmt.Sprintf(
							"struct declarations must be sorted a-z: %q should not come after %q",
							structName,
							lastStruct,
						),
					})
				}

				lastStruct = structName
				currentStruct = structName
			}

			continue
		}

		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		pos := fset.Position(fn.Pos())

		if fn.Recv == nil {
			phase = phaseFuncs
			currentStruct = ""

			if lastFunc != "" && lessFold(fn.Name.Name, lastFunc) {
				issues = append(issues, linter.Issue{
					FilePath: filePath,
					Line:     pos.Line,
					Column:   pos.Column,
					RuleName: r.Name(),
					Message: fmt.Sprintf(
						"function declarations must be sorted a-z: %q should not come after %q",
						fn.Name.Name,
						lastFunc,
					),
				})
			}

			lastFunc = fn.Name.Name
			continue
		}

		receiver := receiverTypeName(fn.Recv)
		if receiver == "" {
			continue
		}

		if phase == phaseFuncs {
			issues = append(issues, linter.Issue{
				FilePath: filePath,
				Line:     pos.Line,
				Column:   pos.Column,
				RuleName: r.Name(),
				Message:  "methods must come before function declarations",
			})
		}

		if currentStruct == "" {
			issues = append(issues, linter.Issue{
				FilePath: filePath,
				Line:     pos.Line,
				Column:   pos.Column,
				RuleName: r.Name(),
				Message:  "a method must be declared after its struct declaration",
			})
		} else if receiver != currentStruct {
			issues = append(issues, linter.Issue{
				FilePath: filePath,
				Line:     pos.Line,
				Column:   pos.Column,
				RuleName: r.Name(),
				Message: fmt.Sprintf(
					"methods must be grouped under their struct: expected receiver %q, got %q",
					currentStruct,
					receiver,
				),
			})
		}

		lastMethod := lastMethodByStruct[receiver]
		if lastMethod != "" && lessFold(fn.Name.Name, lastMethod) {
			issues = append(issues, linter.Issue{
				FilePath: filePath,
				Line:     pos.Line,
				Column:   pos.Column,
				RuleName: r.Name(),
				Message: fmt.Sprintf(
					"methods of %q must be sorted a-z: %q should not come after %q",
					receiver,
					fn.Name.Name,
					lastMethod,
				),
			})
		}

		lastMethodByStruct[receiver] = fn.Name.Name
	}

	return issues
}

func receiverTypeName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}

	expr := recv.List[0].Type
	switch v := expr.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.StarExpr:
		if id, ok := v.X.(*ast.Ident); ok {
			return id.Name
		}
	}

	return ""
}

func lessFold(a, b string) bool {
	return strings.ToLower(a) < strings.ToLower(b)
}
