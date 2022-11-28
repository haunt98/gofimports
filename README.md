# gofimports

Group Go imports with my opinionated preferences.

First is standard.
Then third party, then company if exist.
The last is local.

Also main selling point of this is group imports not sort imports.
So please run `gofumpt` or `gofmt` after running this tool.

Under the hood, this tool get all imports, then group them into 4 groups (std, third party, company, local).
Remember, no sort here.
Then insert empty import (empty path) between each group to get final imports
Then update Go ast decls import with final imports.

There is side effect of course, because we do not create empty line but we add empty import, so there is trailing space in that line (Go indent that empty impoty).
That why I suggest you need to re-format after.

## Install

With Go version `>= 1.16`:

```sh
go install github.com/haunt98/gofimports/cmd/gofimports@latest
```

## Usage

```sh
# Format ./internal
# with print impacted file (-l),
# write to file (-w),
# print diff (-d)
# company is github.com/make-go-great
gofimports -l -company github.com/make-go-great -w -d ./internal
```

Example result:

```go
import (
	"fmt"

	"github.com/urfave/cli/v2"
	"github.com/pkg/diff"

	"github.com/make-go-great/color-go"

	"github.com/haunt98/gofimports/internal/imports"
)
```

## Roadmap

- [ ] Diff with color
- [ ] Add profiling
- [ ] Improve performance

## Thanks

- [goimports](https://pkg.go.dev/golang.org/x/tools/cmd/goimports)
- [mvdan/gofumpt](https://github.com/mvdan/gofumpt)
- [incu6us/goimports-reviser](https://github.com/incu6us/goimports-reviser)
