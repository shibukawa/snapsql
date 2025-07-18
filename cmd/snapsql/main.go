package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/fatih/color"
)

// CLI represents the command line interface structure
type CLI struct {
	Generate GenerateCmd `cmd:"" help:"Generate intermediate files or runtime code"`
	Validate ValidateCmd `cmd:"" help:"Validate SQL templates"`
	Pull     PullCmd     `cmd:"" help:"Extract database schema information"`
	Init     InitCmd     `cmd:"" help:"Initialize a new SnapSQL project"`

	Config  string `help:"Configuration file path" default:"./snapsql.yaml" type:"path"`
	Verbose bool   `short:"v" help:"Enable verbose output"`
	Quiet   bool   `short:"q" help:"Enable quiet mode"`
	NoColor bool   `help:"Disable colored output"`
	Version bool   `help:"Show version information"`
}

func main() {
	var cli CLI

	// Handle version flag before parsing commands
	for _, arg := range os.Args[1:] {
		if arg == "--version" {
			fmt.Println("snapsql version 0.1.0")
			return
		}
	}

	ctx := kong.Parse(&cli,
		kong.Name("snapsql"),
		kong.Description("SnapSQL - SQL template engine with 2-way SQL format"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
	)

	// Handle version flag
	if cli.Version {
		fmt.Println("snapsql version 0.1.0")
		return
	}

	// Configure color output
	if cli.NoColor {
		color.NoColor = true
	}

	// Execute the selected command
	err := ctx.Run(&Context{
		Config:  cli.Config,
		Verbose: cli.Verbose,
		Quiet:   cli.Quiet,
	})

	if err != nil {
		if !cli.Quiet {
			color.Red("Error: %v", err)
		}
		os.Exit(1)
	}
}

// Context holds the application context and configuration
type Context struct {
	Config  string
	Verbose bool
	Quiet   bool
}
