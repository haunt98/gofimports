package cli

import (
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

	if !a.flags.write &&
		!a.flags.diff {
		if err := a.RunHelp(c); err != nil {
			return err
		}

		return nil
	}

	return nil
}
