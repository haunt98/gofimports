package main

import (
	"github.com/haunt98/gofimports/internal/cli"
	_ "go.uber.org/automaxprocs"
)

func main() {
	cli.NewApp().Run()
}
