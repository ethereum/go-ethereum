package server

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ethereum/go-ethereum/log"
	"github.com/mitchellh/cli"
)

// Command is the command to start the sever
type Command struct {
	UI cli.Ui

	// cli configuration
	cliConfig *Config

	// final configuration
	config *Config

	configFile []string

	srv *Server
}

// Help implements the cli.Command interface
func (c *Command) Help() string {
	return `Usage: bor [options]
  
	Run the Bor server.
  ` + c.Flags().Help()
}

// Synopsis implements the cli.Command interface
func (c *Command) Synopsis() string {
	return "Run the Bor server"
}

// Run implements the cli.Command interface
func (c *Command) Run(args []string) int {
	flags := c.Flags()
	if err := flags.Parse(args); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	// read config file
	config := DefaultConfig()
	for _, configFile := range c.configFile {
		cfg, err := readConfigFile(configFile)
		if err != nil {
			c.UI.Error(err.Error())
			return 1
		}
		if err := config.Merge(cfg); err != nil {
			c.UI.Error(err.Error())
			return 1
		}
	}
	if err := config.Merge(c.cliConfig); err != nil {
		c.UI.Error(err.Error())
		return 1
	}
	c.config = config

	srv, err := NewServer(config)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}
	c.srv = srv

	return c.handleSignals()
}

func (c *Command) handleSignals() int {
	signalCh := make(chan os.Signal, 4)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	sig := <-signalCh

	c.UI.Output(fmt.Sprintf("Caught signal: %v", sig))
	c.UI.Output("Gracefully shutting down agent...")

	gracefulCh := make(chan struct{})
	go func() {
		c.srv.Stop()
		close(gracefulCh)
	}()

	for i := 10; i > 0; i-- {
		select {
		case <-signalCh:
			log.Warn("Already shutting down, interrupt more force stop.", "times", i-1)
		case <-gracefulCh:
			return 0
		}
	}
	return 1
}
