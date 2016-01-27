package gompatible

import (
	"fmt"
	"regexp"
	"strings"

	"go/ast"
	"go/types"
)

type ValueChange struct {
	Before *Value
	After  *Value
}

func (vc ValueChange) TypesObject() types.Object {
	return vc.Before.Types
}

var rxAfterEqualSign = regexp.MustCompile(` =.*$`)

// ref: src/cmd/doc/pkg.go
func (v Value) showSpec() string {
	for _, spec := range v.Doc.Decl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		for i, ident := range valueSpec.Names {
			if ident.Name != v.Name {
				continue
			}

			typ := ""
			if valueSpec.Type != nil {
				typ = " " + v.Package.showASTNode(valueSpec.Type)
			}

			val := ""
			if i < len(valueSpec.Values) && valueSpec.Values[i] != nil {
				val = fmt.Sprintf(" = %s", v.Package.showASTNode(valueSpec.Values[i]))
			}

			return fmt.Sprintf("%s %s%s%s", v.Doc.Decl.Tok, ident.Name, typ, val)
		}
	}

	panic(fmt.Sprintf("BUG: Could not find name %q from decl %#v", v.Name, v.Doc.Decl))
}

func (vc ValueChange) ShowBefore() string {
	v := vc.Before
	if v == nil || v.Doc == nil {
		return ""
	}

	s := v.showSpec()
	lines := strings.Split(s, "\n")
	if len(lines) == 1 {
		return lines[0]
	} else {
		return rxAfterEqualSign.ReplaceAllLiteralString(lines[0], " = ...")
	}
}

func (vc ValueChange) ShowAfter() string {
	v := vc.After
	if v == nil || v.Doc == nil {
		return ""
	}

	s := v.showSpec()
	lines := strings.Split(s, "\n")
	if len(lines) == 1 {
		return lines[0]
	} else {
		return rxAfterEqualSign.ReplaceAllLiteralString(lines[0], " = ...")
	}
}

func (vc ValueChange) Kind() ChangeKind {
	switch {
	case vc.Before == nil && vc.After == nil:
		// might not happen
		return ChangeUnchanged

	case vc.Before == nil:
		return ChangeAdded

	case vc.After == nil:
		return ChangeRemoved
	}

	// i) const -> var:   compatible
	// i) var -> const:   breaking (or weak compatible)
	// i) const -> const: identical
	// i) var -> var:     identical
	// ii) types identical:  identical
	// ii) types compatible: compatible
	// ii) types breaking:   breaking

	var k ChangeKind
	if vc.Before.IsConst == false && vc.After.IsConst == true {
		// var -> const: breaking (or weak compatible)
		return ChangeBreaking
	} else if vc.Before.IsConst == true && vc.After.IsConst == false {
		// const -> var: compatible
		k = ChangeCompatible
	} else {
		k = ChangeUnchanged
	}

	switch compareTypes(vc.Before.Types.Type(), vc.After.Types.Type()) {
	case compIncompatible:
		return ChangeBreaking
	case compCompatible:
		return ChangeCompatible
	case compIdentical:
		return k
	}

	panic("could not happen here")
}
