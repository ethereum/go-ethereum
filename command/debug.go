package main

import (
	"fmt"

	"github.com/ethereum/go-ethereum/command/flagset"
)

type DebugCommand struct {
	*Meta2
}

// Help implements the cli.Command interface
func (d *DebugCommand) Help() string {
	return `Usage: bor debug

  Debug`
}

func (d *DebugCommand) Flags() *flagset.Flagset {
	return d.NewFlagSet("debug")
}

// Synopsis implements the cli.Command interface
func (d *DebugCommand) Synopsis() string {
	return "Debug"
}

// Run implements the cli.Command interface
func (d *DebugCommand) Run(args []string) int {
	flags := d.Flags()
	if err := flags.Parse(args); err != nil {
		d.UI.Error(err.Error())
		return 1
	}

	clt, err := d.BorConn()
	if err != nil {
		d.UI.Error(err.Error())
		return 1
	}
	fmt.Println(clt)
	return 0
}
