package cli

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/urfave/cli/v2"

	"github.com/haunt98/gofimports/internal/imports"
)

type action struct {
	flags struct {
		companyPrefix string
		list          bool
		write         bool
		diff          bool
		verbose       bool
		profiler      bool
	}
}

func (a *action) RunHelp(c *cli.Context) error {
	return cli.ShowAppHelp(c)
}

func (a *action) getFlags(c *cli.Context) {
	a.flags.companyPrefix = c.String(flagCompanyPrefixName)
	a.flags.list = c.Bool(flagListName)
	a.flags.write = c.Bool(flagWriteName)
	a.flags.diff = c.Bool(flagDiffName)
	a.flags.verbose = c.Bool(flagVerboseName)
	a.flags.profiler = c.Bool(flagProfilerName)

	if a.flags.verbose {
		fmt.Printf("flags: %+v\n", a.flags)
	}
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

	if a.flags.profiler {
		f, err := os.Create("cpu.prof")
		if err != nil {
			return fmt.Errorf("os: failed to create: %w", err)
		}
		defer f.Close()

		if err := pprof.StartCPUProfile(f); err != nil {
			return fmt.Errorf("pprof: failed to start cpu profile: %w", err)
		}
		defer pprof.StopCPUProfile()
	}

	ft, err := imports.NewFormmater(
		imports.FormatterWithList(a.flags.list),
		imports.FormatterWithWrite(a.flags.write),
		imports.FormatterWithDiff(a.flags.diff),
		imports.FormatterWithVerbose(a.flags.verbose),
		imports.FormatterWithCompanyPrefix(a.flags.companyPrefix),
	)
	if err != nil {
		return fmt.Errorf("imports: failed to new formatter: %w", err)
	}

	args := c.Args().Slice()
	if err := ft.Format(args...); err != nil {
		return fmt.Errorf("imports formatter: failed to format %v: %w", args, err)
	}

	if a.flags.profiler {
		f, err := os.Create("mem.prof")
		if err != nil {
			return fmt.Errorf("os: failed to create: %w", err)
		}
		defer f.Close()

		runtime.GC()
		if err := pprof.WriteHeapProfile(f); err != nil {
			return fmt.Errorf("pprof: failed to write heap profile: %w", err)
		}
	}

	return nil
}
