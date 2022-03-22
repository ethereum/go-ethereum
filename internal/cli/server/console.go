package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/console"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/mitchellh/cli"
)

// ConsoleCommand is the command to Connect to remote Bor IPC console
type ConsoleCommand struct {
	// cli configuration
	cliConfig *Config

	// final configuration
	config *Config

	configFile    []string
	UI            cli.Ui
	ExecCMD       string
	Endpoint      string
	PreloadJSFlag string
	JSpathFlag    string
	srv           *Server
}

// Help implements the cli.Command interface
func (c *ConsoleCommand) Help() string {
	return `Usage: bor console

  Connect to local Bor IPC console.`
}

// Synopsis implements the cli.Command interface
func (c *ConsoleCommand) Synopsis() string {
	return "Connect to Bor console"
}

// Run implements the cli.Command interface
func (c *ConsoleCommand) Run(args []string) int {
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

	c.localConsole()

	return 0
}

// localConsole starts a new geth node, attaching a JavaScript console to it at the
// same time.
func (c *ConsoleCommand) localConsole() error {
	// Create and start the node based on the CLI flags
	stack := c.srv.node
	stack.Start()
	defer stack.Close()

	path := node.DefaultDataDir()

	if c.Endpoint == "" {
		if c.config.DataDir != "" {
			path = c.config.DataDir
		}
		if path != "" {
			homeDir, _ := os.UserHomeDir()
			path = filepath.Join(homeDir, "/.bor/data")
		}
		c.Endpoint = fmt.Sprintf("%s/bor.ipc", path)
	}

	// Attach to the newly started node and start the JavaScript console
	client, err := stack.Attach()
	if err != nil {
		utils.Fatalf("Failed to attach to the inproc geth: %v", err)
	}
	config := console.Config{
		DataDir: path,
		DocRoot: c.JSpathFlag,
		Client:  client,
		Preload: c.MakeConsolePreloads(),
	}

	console, err := console.New(config)
	if err != nil {
		utils.Fatalf("Failed to start the JavaScript console: %v", err)
	}
	defer console.Stop(false)

	// If only a short execution was requested, evaluate and return
	if script := c.ExecCMD; script != "" {
		console.Evaluate(script)
		return nil
	}
	// Otherwise print the welcome screen and enter interactive mode
	console.Welcome()
	console.Interactive()

	return nil
}

// dialRPC returns a RPC client which connects to the given endpoint.
// The check for empty endpoint implements the defaulting logic
// for "geth attach" with no argument.
func dialRPC(endpoint string) (*rpc.Client, error) {
	if endpoint == "" {
		endpoint = node.DefaultIPCEndpoint("bor")
	} else if strings.HasPrefix(endpoint, "rpc:") || strings.HasPrefix(endpoint, "ipc:") {
		// Backwards compatibility with geth < 1.5 which required
		// these prefixes.
		endpoint = endpoint[4:]
	}
	return rpc.Dial(endpoint)
}

// MakeConsolePreloads retrieves the absolute paths for the console JavaScript
// scripts to preload before starting.
func (c *ConsoleCommand) MakeConsolePreloads() []string {
	// Skip preloading if there's nothing to preload
	if c.PreloadJSFlag == "" {
		return nil
	}
	// Otherwise resolve absolute paths and return them
	var preloads []string

	for _, file := range strings.Split(c.PreloadJSFlag, ",") {
		preloads = append(preloads, strings.TrimSpace(file))
	}
	return preloads
}
