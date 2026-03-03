package rules

import (
	"fmt"
	"go/ast"
	"go/token"
	"sort"
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

func (r *DeclarationOrderRule) FixFile(_ *token.FileSet, _ string, file *ast.File) (bool, error) {
	targetIndexes := make([]int, 0, len(file.Decls))
	targetDecls := make([]ast.Decl, 0, len(file.Decls))
	metas := make([]declMeta, 0, len(file.Decls))

	for i, decl := range file.Decls {
		meta, ok := classifyDecl(decl, i)
		if !ok {
			continue
		}
		targetIndexes = append(targetIndexes, i)
		targetDecls = append(targetDecls, decl)
		metas = append(metas, meta)
	}

	if len(metas) < 2 {
		return false, nil
	}

	structRanks := buildStructRanks(metas)
	sortedMetas := append([]declMeta(nil), metas...)
	sort.SliceStable(sortedMetas, func(i, j int) bool {
		return lessMeta(sortedMetas[i], sortedMetas[j], structRanks)
	})

	changed := false
	for i := range targetDecls {
		if targetDecls[i] != sortedMetas[i].decl {
			changed = true
			break
		}
	}
	if !changed {
		return false, nil
	}

	for i, idx := range targetIndexes {
		file.Decls[idx] = sortedMetas[i].decl
	}

	return true, nil
}

type declKind int

const (
	declKindStruct declKind = iota
	declKindMethod
	declKindFunc
)

type declMeta struct {
	decl  ast.Decl
	index int
	kind  declKind
	group string
	name  string
}

func classifyDecl(decl ast.Decl, index int) (declMeta, bool) {
	if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.TYPE {
		if len(gen.Specs) != 1 {
			return declMeta{}, false
		}
		typeSpec, ok := gen.Specs[0].(*ast.TypeSpec)
		if !ok {
			return declMeta{}, false
		}
		if _, ok := typeSpec.Type.(*ast.StructType); !ok {
			return declMeta{}, false
		}
		return declMeta{
			decl:  decl,
			index: index,
			kind:  declKindStruct,
			group: typeSpec.Name.Name,
			name:  typeSpec.Name.Name,
		}, true
	}

	fn, ok := decl.(*ast.FuncDecl)
	if !ok {
		return declMeta{}, false
	}

	if fn.Recv == nil {
		return declMeta{
			decl:  decl,
			index: index,
			kind:  declKindFunc,
			name:  fn.Name.Name,
		}, true
	}

	receiver := receiverTypeName(fn.Recv)
	if receiver == "" {
		return declMeta{}, false
	}

	return declMeta{
		decl:  decl,
		index: index,
		kind:  declKindMethod,
		group: receiver,
		name:  fn.Name.Name,
	}, true
}

func buildStructRanks(metas []declMeta) map[string]int {
	structNames := make([]string, 0, len(metas))
	seen := map[string]bool{}
	for _, meta := range metas {
		if meta.kind != declKindStruct {
			continue
		}
		if seen[meta.group] {
			continue
		}
		seen[meta.group] = true
		structNames = append(structNames, meta.group)
	}

	sort.SliceStable(structNames, func(i, j int) bool {
		li := strings.ToLower(structNames[i])
		lj := strings.ToLower(structNames[j])
		if li != lj {
			return li < lj
		}
		return structNames[i] < structNames[j]
	})

	ranks := make(map[string]int, len(structNames))
	for i, name := range structNames {
		ranks[name] = i
	}

	return ranks
}

func lessMeta(a, b declMeta, structRanks map[string]int) bool {
	ag, ap := metaRank(a, structRanks)
	bg, bp := metaRank(b, structRanks)

	if ag != bg {
		return ag < bg
	}
	if ap != bp {
		return ap < bp
	}
	if a.name != b.name {
		return lessFold(a.name, b.name)
	}
	return a.index < b.index
}

func metaRank(meta declMeta, structRanks map[string]int) (int, int) {
	switch meta.kind {
	case declKindStruct:
		if rank, ok := structRanks[meta.group]; ok {
			return rank, 0
		}
		return len(structRanks), 0
	case declKindMethod:
		if rank, ok := structRanks[meta.group]; ok {
			return rank, 1
		}
		return len(structRanks) + 2, 0
	case declKindFunc:
		return len(structRanks) + 1, 0
	default:
		return len(structRanks) + 3, 0
	}
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
