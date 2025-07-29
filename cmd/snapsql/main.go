package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
)

// CLI represents the command-line interface
var CLI struct {
	Generate GenerateCommand `cmd:"" help:"Generate intermediate files from SQL templates"`
	Validate ValidateCommand `cmd:"" help:"Validate SQL templates"`
	Version  VersionCommand  `cmd:"" help:"Show version information"`
}

// VersionCommand represents the version command
type VersionCommand struct{}

// Run executes the version command
func (cmd *VersionCommand) Run() error {
	fmt.Println("SnapSQL v0.1.0")
	return nil
}

// ValidateCommand represents the validate command
type ValidateCommand struct {
	Input string `help:"Input SQL file or directory" short:"i" default:"."`
}

// Run executes the validate command
func (cmd *ValidateCommand) Run() error {
	fmt.Println("Validating SQL templates...")
	// TODO: Implement validation
	return nil
}

func main() {
	ctx := kong.Parse(&CLI)
	err := ctx.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
