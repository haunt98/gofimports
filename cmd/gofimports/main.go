package main

import (
	"context"

	"github.com/haunt98/gofimports/internal/cli"
)

func main() {
	app := cli.NewApp()
	app.Run(context.Background())
}
