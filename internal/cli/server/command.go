package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/maticnetwork/heimdall/cmd/heimdalld/service"
	"github.com/mitchellh/cli"
	"github.com/pelletier/go-toml"

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
		log.Warn("Config File provided, this will overwrite the cli flags", "path", c.configFile)
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

	// nolint: nestif
	// check for log-level and verbosity here
	if c.configFile != "" {
		data, _ := toml.LoadFile(c.configFile)
		if data.Has("verbosity") && data.Has("log-level") {
			log.Warn("Config contains both, verbosity and log-level, log-level will be deprecated soon. Use verbosity only.", "using", data.Get("verbosity"))
		} else if !data.Has("verbosity") && data.Has("log-level") {
			log.Warn("Config contains log-level only, note that log-level will be deprecated soon. Use verbosity instead.", "using", data.Get("log-level"))
			config.Verbosity = VerbosityStringToInt(strings.ToLower(data.Get("log-level").(string)))
		}
	} else {
		tempFlag := 0
		for _, val := range args {
			if (strings.HasPrefix(val, "-verbosity") || strings.HasPrefix(val, "--verbosity")) && config.LogLevel != "" {
				tempFlag = 1
				break
			}
		}
		if tempFlag == 1 {
			log.Warn("Both, verbosity and log-level flags are provided, log-level will be deprecated soon. Use verbosity only.", "using", config.Verbosity)
		} else if tempFlag == 0 && config.LogLevel != "" {
			log.Warn("Only log-level flag is provided, note that log-level will be deprecated soon. Use verbosity instead.", "using", config.LogLevel)
			config.Verbosity = VerbosityStringToInt(strings.ToLower(config.LogLevel))
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

	if c.config.Heimdall.RunHeimdall {
		shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		go func() {
			service.NewHeimdallService(shutdownCtx, c.getHeimdallArgs())
		}()
	}

	srv, err := NewServer(c.config, WithGRPCAddress())
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

func (c *Command) getHeimdallArgs() []string {
	heimdallArgs := strings.Split(c.config.Heimdall.RunHeimdallArgs, ",")
	return append([]string{"start"}, heimdallArgs...)
}
