package server

import "github.com/ethereum/go-ethereum/command/flagset"

func (c *Command) Flags() *flagset.Flagset {
	c.cliConfig = DefaultConfig()

	f := flagset.NewFlagSet("")

	f.BoolFlag(&flagset.BoolFlag{
		Name:  "debug",
		Value: c.cliConfig.Debug,
		Usage: "Path of the file to apply",
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "chain",
		Value: c.cliConfig.Chain,
		Usage: "Name of the chain to sync",
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "log-level",
		Value: c.cliConfig.LogLevel,
		Usage: "Set log level for the server",
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "data-dir",
		Value: c.cliConfig.DataDir,
		Usage: "Path of the data directory to store information",
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "config",
		Value: &c.configFile,
		Usage: "File for the config file",
	})

	// txpool options
	f.SliceStringFlag(&flagset.SliceStringFlag{
		Name:  "txpool.locals",
		Value: c.cliConfig.TxPool.Locals,
		Usage: "Comma separated accounts to treat as locals (no flush, priority inclusion)",
	})
	f.BoolFlag(&flagset.BoolFlag{
		Name:  "txpool.nolocals",
		Value: c.cliConfig.TxPool.NoLocals,
		Usage: "Disables price exemptions for locally submitted transactions",
	})

	// sealer options
	f.BoolFlag(&flagset.BoolFlag{
		Name:  "mine",
		Value: c.cliConfig.Sealer.Enabled,
		Usage: "",
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "miner.etherbase",
		Value: c.cliConfig.Sealer.Etherbase,
		Usage: "",
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "miner.extradata",
		Value: c.cliConfig.Sealer.ExtraData,
		Usage: "",
	})

	return f
}
