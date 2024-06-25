package main

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/internal/debug"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/urfave/cli/v2"
)

var app = flags.NewApp("the evm command line interface")

var (
	RefTestFlag = &cli.StringFlag{
		Name:  "test",
		Usage: "Path to EOF validation reference test.",
	}
	HexFlag = &cli.StringFlag{
		Name:  "hex",
		Usage: "single container data parse and validation",
	}
)

var eofParserCommand = &cli.Command{
	Name:    "eofparser",
	Aliases: []string{"eof"},
	Usage:   "parses hex eof container and returns validation errors (if any)",
	Action:  eofParser,
	Flags: []cli.Flag{
		HexFlag,
		RefTestFlag,
	},
}

var eofDumpCommand = &cli.Command{
	Name:   "eofdump",
	Usage:  "parses hex eof container",
	Action: eofDump,
	Flags: []cli.Flag{
		HexFlag,
	},
}

func init() {
	app.Commands = []*cli.Command{
		eofParserCommand,
		eofDumpCommand,
	}
	app.Before = func(ctx *cli.Context) error {
		flags.MigrateGlobalFlags(ctx)
		return debug.Setup(ctx)
	}
	app.After = func(ctx *cli.Context) error {
		debug.Exit()
		return nil
	}
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
