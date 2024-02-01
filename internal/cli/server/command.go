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
		c.Flags(nil).MarkDown(),
	}

	return strings.Join(items, "\n\n")
}

// Help implements the cli.Command interface
func (c *Command) Help() string {
	return `Usage: bor [options]

	Run the Bor server.
  ` + c.Flags(nil).Help()
}

// Synopsis implements the cli.Command interface
func (c *Command) Synopsis() string {
	return "Run the Bor server"
}

// checkConfigFlag checks if the config flag is set or not. If set,
// it returns the value else an empty string.
func checkConfigFlag(args []string) string {
	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Check for single or double dashes
		if strings.HasPrefix(arg, "-config") || strings.HasPrefix(arg, "--config") {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				return parts[1]
			}

			// If there's no equal sign, check the next argument
			if i+1 < len(args) {
				return args[i+1]
			}
		}
	}

	return ""
}

func (c *Command) extractFlags(args []string) error {
	// Check if config file is provided or not
	configFilePath := checkConfigFlag(args)

	if configFilePath != "" {
		log.Info("Reading config file", "path", configFilePath)

		// Parse the config file
		cfg, err := readConfigFile(configFilePath)
		if err != nil {
			c.UI.Error(err.Error())

			return err
		}

		log.Warn("Config set via config file will be overridden by cli flags")

		// Initialise a flagset based on the config created above
		flags := c.Flags(cfg)

		// Check for explicit cli args
		cmd := Command{} // use a new variable to keep the original config intact

		cliFlags := cmd.Flags(nil)
		if err := cliFlags.Parse(args); err != nil {
			c.UI.Error(err.Error())

			return err
		}

		// Get the list of flags set explicitly
		names, values := cliFlags.Visit()

		// Set these flags using the flagset created earlier
		flags.UpdateValue(names, values)
	} else {
		flags := c.Flags(nil)

		if err := flags.Parse(args); err != nil {
			c.UI.Error(err.Error())

			return err
		}
	}

	// nolint: nestif
	// check for log-level and verbosity here
	if configFilePath != "" {
		data, _ := toml.LoadFile(configFilePath)
		if data.Has("verbosity") && data.Has("log-level") {
			log.Warn("Config contains both, verbosity and log-level, log-level will be deprecated soon. Use verbosity only.", "using", data.Get("verbosity"))
		} else if !data.Has("verbosity") && data.Has("log-level") {
			log.Warn("Config contains log-level only, note that log-level will be deprecated soon. Use verbosity instead.", "using", data.Get("log-level"))
			c.cliConfig.Verbosity = VerbosityStringToInt(strings.ToLower(data.Get("log-level").(string)))
		}
	} else {
		tempFlag := 0

		for _, val := range args {
			if (strings.HasPrefix(val, "-verbosity") || strings.HasPrefix(val, "--verbosity")) && c.cliConfig.LogLevel != "" {
				tempFlag = 1
				break
			}
		}

		if tempFlag == 1 {
			log.Warn("Both, verbosity and log-level flags are provided, log-level will be deprecated soon. Use verbosity only.", "using", c.cliConfig.Verbosity)
		} else if tempFlag == 0 && c.cliConfig.LogLevel != "" {
			log.Warn("Only log-level flag is provided, note that log-level will be deprecated soon. Use verbosity instead.", "using", c.cliConfig.LogLevel)
			c.cliConfig.Verbosity = VerbosityStringToInt(strings.ToLower(c.cliConfig.LogLevel))
		}
	}

	c.config = c.cliConfig

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
