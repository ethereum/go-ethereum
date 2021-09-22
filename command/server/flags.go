package server

import "github.com/ethereum/go-ethereum/command/flagset"

func (c *Command) Flags() *flagset.Flagset {
	c.cliConfig = &Config{}

	f := flagset.NewFlagSet("server")

	f.BoolFlag(&flagset.BoolFlag{
		Name:  "debug",
		Value: c.cliConfig.Debug,
		Usage: "Path of the file to apply",
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

	return f
}
