package main

import (
	_ "go.uber.org/automaxprocs"

	"github.com/haunt98/gofimports/internal/cli"
)

func main() {
	cli.NewApp().Run()
}
