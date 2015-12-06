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

Example: TBD

#### Compatible (`*`)

The API signature has been changed, but the users do not have to change their usages.

Example: TBD

## Author

motemen <https://motemen.github.io/>
