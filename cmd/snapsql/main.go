package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
)

// Context represents the global context for commands
type Context struct {
	Config  string
	Verbose bool
	Quiet   bool
}

// CLI represents the command-line interface
var CLI struct {
	Config   string       `help:"Configuration file path" default:"snapsql.yaml"`
	Verbose  bool         `help:"Enable verbose output" short:"v"`
	Quiet    bool         `help:"Suppress output" short:"q"`
	Generate GenerateCmd  `cmd:"" help:"Generate intermediate files from SQL templates"`
	Validate ValidateCmd  `cmd:"" help:"Validate SQL templates"`
	Init     InitCmd      `cmd:"" help:"Initialize a new SnapSQL project"`
	Pull     PullCmd      `cmd:"" help:"Pull schema information from database"`
	Query    QueryCmd     `cmd:"" help:"Execute SQL queries"`
	Version  VersionCmd   `cmd:"" help:"Show version information"`
}

// VersionCmd represents the version command
type VersionCmd struct{}

// Run executes the version command
func (cmd *VersionCmd) Run() error {
	fmt.Println("SnapSQL v0.1.0")
	return nil
}

func main() {
	ctx := kong.Parse(&CLI)
	
	// Create context with config path
	appCtx := &Context{
		Config:  CLI.Config,
		Verbose: CLI.Verbose,
		Quiet:   CLI.Quiet,
	}
	
	err := ctx.Run(appCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
