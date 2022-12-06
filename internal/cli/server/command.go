package server

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/mitchellh/cli"

	"github.com/ethereum/go-ethereum/log"
)

// Command is the command to start the sever
type Command struct {
	UI cli.Ui

	// cli configuration
	cliConfig *Config

	// final configuration
	config *Config

	configFile string

	srv *Server
}

// MarkDown implements cli.MarkDown interface
func (c *Command) MarkDown() string {
	items := []string{
		"# Server",
		"The ```bor server``` command runs the Bor client.",
		c.Flags().MarkDown(),
	}

	return strings.Join(items, "\n\n")
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

func (c *Command) extractFlags(args []string) error {
	config := *DefaultConfig()

	flags := c.Flags()
	if err := flags.Parse(args); err != nil {
		c.UI.Error(err.Error())
		c.config = &config

		return err
	}

	// TODO: Check if this can be removed or not
	// read cli flags
	if err := config.Merge(c.cliConfig); err != nil {
		c.UI.Error(err.Error())
		c.config = &config

		return err
	}
	// read if config file is provided, this will overwrite the cli flags, if provided
	if c.configFile != "" {
		log.Warn("Config File provided, this will overwrite the cli flags.", "configFile:", c.configFile)
		cfg, err := readConfigFile(c.configFile)
		if err != nil {
			c.UI.Error(err.Error())
			c.config = &config

			return err
		}
		if err := config.Merge(cfg); err != nil {
			c.UI.Error(err.Error())
			c.config = &config

			return err
		}
	}

	c.config = &config

	return nil
}

// Run implements the cli.Command interface
func (c *Command) Run(args []string) int {
	err := c.extractFlags(args)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	srv, err := NewServer(c.config)
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

// GetConfig returns the user specified config
func (c *Command) GetConfig() *Config {
	return c.cliConfig
}
