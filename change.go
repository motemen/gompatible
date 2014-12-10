package gompatible

import (
	"golang.org/x/tools/go/types"
)

type ChangeKind int

const (
	ChangeUnchanged ChangeKind = iota
	ChangeAdded
	ChangeRemoved
	ChangeCompatible
	ChangeBreaking
)

func (ck ChangeKind) String() string {
	switch ck {
	case ChangeUnchanged:
		return "ChangeUnchanged"
	case ChangeAdded:
		return "ChangeAdded"
	case ChangeRemoved:
		return "ChangeRemoved"
	case ChangeCompatible:
		return "ChangeCompatible"
	case ChangeBreaking:
		return "ChangeBreaking"
	}

	return ""
}

type Change interface {
	ObjectBefore() types.Object
	ObjectAfter() types.Object
	Kind() ChangeKind
}

func ShowChange(c Change) string {
	// TODO use types.ObjectString()
	switch c.Kind() {
	case ChangeAdded:
		return "+ " + c.ObjectAfter().String()
	case ChangeRemoved:
		return "- " + c.ObjectBefore().String()
	case ChangeUnchanged:
		return "= " + c.ObjectBefore().String()
	case ChangeCompatible:
		return "* " + c.ObjectBefore().String() + " -> " + c.ObjectAfter().String()
	case ChangeBreaking:
		fallthrough
	default:
		return "! " + c.ObjectBefore().String() + " -> " + c.ObjectAfter().String()
	}
}
