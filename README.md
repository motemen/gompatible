gompatible
==========

Gompatible is a tool to show Go package's API changes between two (git) revisions. The API changes are categorized into unchanged, added, removed, breaking, and compatible.

## Installation

    go get -u github.com/motemen/gompatible/cmd/gompat

## Usage

    gompat [-a] [-d] [-r] <rev1>[..[<rev2>]] [<import path>[/...]]

Extracts type information of target package (or the current directory if not specified) at two revisions _rev1_, _rev2_ and shows changes between them.

Flags:

    -a    show also unchanged APIs
    -d    run diff on multi-line changes
    -r    recurse into subdirectories
          (can be specified by "/..." suffix to the import path)

### Specifying revisions

- `<rev1>..<rev2>` ... Shows changes between revisions _rev1_ and _rev2_
- `<rev1>..` ... Shows changes between revisions _rev1_ and `HEAD` (same as `<rev1>..HEAD`)
- `<rev1>` .. Shows changes introduced by the commit _rev1_ (same as `<rev1>~1..<rev1>`)

## Example

~~~
% gompat 665374f1c86631cf73f4729095d4cf4313670 golang.org/x/tools/go/types
! func Eval(str string, pkg *Package, scope *Scope) (TypeAndValue, error)
. func Eval(fset *token.FileSet, pkg *Package, pos token.Pos, expr string) (tv TypeAndValue, err error)
- func EvalNode(fset *token.FileSet, node ast.Expr, pkg *Package, scope *Scope) (tv TypeAndValue, err error)
- func New(str string) Type
! func NewScope(parent *Scope, comment string) *Scope
. func NewScope(parent *Scope, pos, end token.Pos, comment string) *Scope
...
~~~

## Details

### API entities

API entities, in the sense of Gompatible recognizes,
are package-level and exported

- Functions and methods (`func`),
- Types (`type`), and
- Variables (`var`) and constants (`const`).

### API changes

The obvious API change kinds are:

- Unchanged (`=`)
  - The API entity kept unchanged
- Added (`+`)
  - The API entity was newly added
- Removed (`-`)
  - The API entity was removed

And below are the less-obvious ones:

#### Breaking (`!`)

The API signature has been changed and the users must change their usages.

Example:

~~~
// before
func F(n, m int) error

// after
func F(n, m int, b bool) error
~~~

~~~
// before
type T struct {
  Foo string
  Bar bool
}

// after
type T struct {
  Foo float64
  Bar bool
}
~~~

#### Compatible (`*`)

The API signature has been changed, but the users do not have to change their usages.

Example:

~~~
// before
func F(n, m int) error

// after
func F(n, m int, opt ...interface{}}) error
~~~

~~~
// before
func F(b *bytes.Buffer)

// after
func F(r io.Reader)
~~~

## Author

motemen <https://motemen.github.io/>
