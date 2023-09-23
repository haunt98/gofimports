# gofimports

[![Go](https://github.com/haunt98/gofimports/workflows/Go/badge.svg?branch=main)](https://github.com/haunt98/gofimports/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/haunt98/gofimports.svg)](https://pkg.go.dev/github.com/haunt98/gofimports)
[![Latest Version](https://img.shields.io/github/v/tag/haunt98/gofimports)](https://github.com/haunt98/gofimports/tags)

Group Go imports with my opinionated preferences.

First is **standard**. Then **third party**, then **company** if exist. The last
is **local**.

Also main selling point of this tool is to handle imports only. So please run
`gofumpt` or `gofmt` to format you files after running this tool.

## Install

With Go version `>= 1.16`:

```sh
go install github.com/haunt98/gofimports/cmd/gofimports@latest
```

## Usage

```sh
# Format ./internal with:
# - print impacted file (-l),
# - write to file (-w),
# - print diff (-d)
# - company prefix, split using comma (,)
gofimports -l -w -d --company github.com/make-go-great,github.com/haunt98 ./internal

# Format ./internal with:
# - write
# - stock mode, only split standard and non standard
gofimports -w --stock ./internal
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
- [x] Add profiling
- [ ] Improve performance

## Thanks

- [goimports](https://pkg.go.dev/golang.org/x/tools/cmd/goimports)
- [mvdan/gofumpt](https://github.com/mvdan/gofumpt)
- [incu6us/goimports-reviser](https://github.com/incu6us/goimports-reviser)
- [dave/dst](https://github.com/dave/dst)
