package cli

import (
	"fmt"

	"github.com/haunt98/gofimports/internal/imports"
	"github.com/urfave/cli/v2"
)

type action struct {
	flags struct {
		list  bool
		write bool
		diff  bool
	}
}

func (a *action) RunHelp(c *cli.Context) error {
	return cli.ShowAppHelp(c)
}

func (a *action) getFlags(c *cli.Context) {
	a.flags.list = c.Bool(flagListName)
	a.flags.write = c.Bool(flagWriteName)
	a.flags.diff = c.Bool(flagDiffName)
}

func (a *action) Run(c *cli.Context) error {
	a.getFlags(c)

	// No flag is set
	if !a.flags.list &&
		!a.flags.write &&
		!a.flags.diff {
		return a.RunHelp(c)
	}

	// Empty args
	if c.Args().Len() == 0 {
		return a.RunHelp(c)
	}

	f := imports.NewFormmater(
		imports.FormatterWithList(a.flags.list),
		imports.FormatterWithWrite(a.flags.write),
		imports.FormatterWithDiff(a.flags.diff),
	)

	args := c.Args().Slice()
	if err := f.Format(args...); err != nil {
		return fmt.Errorf("imports formatter: failed to format %v: %w", args, err)
	}

	return nil
}
