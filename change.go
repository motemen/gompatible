package gompatible

import (
	"golang.org/x/tools/go/types"
)

// ChangeKind represents the kind of a change of an API between two revisions.
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
		return "Unchanged"
	case ChangeAdded:
		return "Added"
	case ChangeRemoved:
		return "Removed"
	case ChangeCompatible:
		return "Compatible"
	case ChangeBreaking:
		return "Breaking"
	}

	return ""
}

// A Change represents a change of an API between two revisions.
type Change interface {
	TypesObject() types.Object
	ShowBefore() string
	ShowAfter() string
	Kind() ChangeKind
}

// ShowChange returns a string represnetation of an API change.
func ShowChange(c Change) string {
	switch c.Kind() {
	case ChangeAdded:
		return "+ " + c.ShowAfter()
	case ChangeRemoved:
		return "- " + c.ShowBefore()
	case ChangeUnchanged:
		return "= " + c.ShowBefore()
	case ChangeCompatible:
		return "* " + c.ShowBefore() + " -> " + c.ShowAfter()
	case ChangeBreaking:
		fallthrough
	default:
		return "! " + c.ShowBefore() + " -> " + c.ShowAfter()
	}
}
