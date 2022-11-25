package cli

import (
	"os"

	"github.com/make-go-great/color-go"
	"github.com/urfave/cli/v2"
)

const (
	name  = "gofimports"
	usage = "goimports with my opinionated preferences"

	// Inspiration from gofmt flags
	flagListName  = "list"
	flagListUsage = "list files will be changed"

	flagWriteName  = "write"
	flagWriteUsage = "actually write changes to (source) files"

	flagDiffName  = "diff"
	flagDiffUsage = "show diff"

	flagVerboseName  = "verbose"
	flagVerboseUsage = "show verbose output, for debug only"

	flagCompanyPrefixName  = "company"
	flagCompanyPrefixUsage = "company prefix, for example github.com/haunt98"
)

var (
	flagListAliases  = []string{"l"}
	flagWriteAliases = []string{"w"}
	flagDiffAliases  = []string{"d"}
)

type App struct {
	cliApp *cli.App
}

func NewApp() *App {
	a := &action{}

	// TODO: hide commands, show args usage
	cliApp := &cli.App{
		Name:  name,
		Usage: usage,
		Flags: []cli.Flag{
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
			&cli.StringFlag{
				Name:  flagCompanyPrefixName,
				Usage: flagCompanyPrefixUsage,
			},
		},
		Action: a.Run,
	}

	return &App{
		cliApp: cliApp,
	}
}

func (a *App) Run() {
	if err := a.cliApp.Run(os.Args); err != nil {
		color.PrintAppError(name, err.Error())
	}
}
