package server

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/mitchellh/cli"
)

// Command is the command to start the sever
type Command struct {
	UI cli.Ui

	// cli configuration
	cliConfig *Config

	configFile string

	// bor node
	node *node.Node
}

// Help implements the cli.Command interface
func (c *Command) Help() string {
	return ""
}

// Synopsis implements the cli.Command interface
func (c *Command) Synopsis() string {
	return ""
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
	if c.configFile != "" {
		c, err := readConfigFile(c.configFile)
		if err != nil {
			panic(err)
		}
		config.Merge(c)
	}
	config.Merge(c.cliConfig)

	// start the logger
	setupLogger(*config.LogLevel)

	// create the node
	nodeCfg, err := config.buildNode()
	if err != nil {
		panic(err)
	}
	stack, err := node.New(nodeCfg)
	if err != nil {
		panic(err)
	}
	c.node = stack

	// register the ethereum backend
	ethCfg, err := config.buildEth()
	if err != nil {
		panic(err)
	}
	backend, err := eth.New(stack, ethCfg)
	if err != nil {
		panic(err)
	}
	c.node.RegisterAPIs(tracers.APIs(backend.APIBackend))

	// start the node
	if err := c.node.Start(); err != nil {
		panic(err)
	}
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
		c.node.Close()
		close(gracefulCh)
	}()

	select {
	case <-signalCh:
		return 1
	case <-time.After(5 * time.Second):
		return 1
	case <-gracefulCh:
		return 0
	}
}

func setupLogger(logLevel string) {
	output := io.Writer(os.Stderr)
	usecolor := (isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())) && os.Getenv("TERM") != "dumb"
	if usecolor {
		output = colorable.NewColorableStderr()
	}
	ostream := log.StreamHandler(output, log.TerminalFormat(usecolor))
	glogger := log.NewGlogHandler(ostream)

	// logging
	lvl, err := log.LvlFromString(strings.ToLower(logLevel))
	if err == nil {
		glogger.Verbosity(lvl)
	} else {
		glogger.Verbosity(log.LvlInfo)
	}
	log.Root().SetHandler(glogger)
}
