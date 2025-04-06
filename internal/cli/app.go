package cli

import (
	"context"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/make-go-great/color-go"
)

const (
	name  = "gofimports"
	usage = "goimports with my opinionated preferences"

	// Inspiration from gofmt flags
	flagCompanyPrefixName  = "company"
	flagCompanyPrefixUsage = "company prefix, split using comma (,), for example github.com/make-go-great,github.com/haunt98"

	flagListName  = "list"
	flagListUsage = "list files will be changed"

	flagWriteName  = "write"
	flagWriteUsage = "actually write changes to (source) files"

	flagDiffName  = "diff"
	flagDiffUsage = "show diff"

	flagVerboseName  = "verbose"
	flagVerboseUsage = "show verbose output, for debug only"

	flagProfilerName  = "profiler"
	flagProfilerUsage = "go profiler, for debug only"

	flagStockName  = "stock"
	flagStockUsage = "stock mode, only split standard pkg and non standard, ignore company flag"
)

var (
	flagListAliases  = []string{"l"}
	flagWriteAliases = []string{"w"}
	flagDiffAliases  = []string{"d"}
)

type App struct {
	cliApp *cli.Command
}

func NewApp() *App {
	a := &action{}

	// TODO: hide commands, show args usage
	cliApp := &cli.Command{
		Name:  name,
		Usage: usage,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  flagCompanyPrefixName,
				Usage: flagCompanyPrefixUsage,
			},
			&cli.BoolFlag{
				Name:    flagListName,
				Usage:   flagListUsage,
				Aliases: flagListAliases,
			},
			&cli.BoolFlag{
				Name:    flagWriteName,
				Usage:   flagWriteUsage,
				Aliases: flagWriteAliases,
			},
			&cli.BoolFlag{
				Name:    flagDiffName,
				Usage:   flagDiffUsage,
				Aliases: flagDiffAliases,
			},
			&cli.BoolFlag{
				Name:  flagVerboseName,
				Usage: flagVerboseUsage,
			},
			&cli.BoolFlag{
				Name:  flagProfilerName,
				Usage: flagProfilerUsage,
			},
			&cli.BoolFlag{
				Name:  flagStockName,
				Usage: flagStockUsage,
			},
		},
		Action: a.Run,
	}

	return &App{
		cliApp: cliApp,
	}
}

func (a *App) Run(ctx context.Context) {
	if err := a.cliApp.Run(ctx, os.Args); err != nil {
		color.PrintAppError(name, err.Error())
	}
}
