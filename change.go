package gompatible

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
	ShowBefore() string
	ShowAfter() string
	Kind() ChangeKind
}

func ShowChange(c Change) string {
	// TODO simplify packages
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
