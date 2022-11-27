# gofimports

Group Go imports with my opinionated preferences.

First is standard.
Then third party, then company if exist.
The last is local.

Also main selling point of this is group imports not sort imports.
So please run `gofumpt` or `gofmt` after this.

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

## Roadmap

- [ ] Diff with color
- [ ] Add profiling
- [ ] Improve performance

## Thanks

- [goimports](https://pkg.go.dev/golang.org/x/tools/cmd/goimports)
- [mvdan/gofumpt](https://github.com/mvdan/gofumpt)
- [incu6us/goimports-reviser](https://github.com/incu6us/goimports-reviser)
