package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/console"
	"github.com/ethereum/go-ethereum/internal/cli/flagset"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/mitchellh/cli"
)

// AttachCommand is the command to Connect to remote Bor IPC console
type AttachCommand struct {
	UI            cli.Ui
	Meta          *Meta
	Meta2         *Meta2
	ExecCMD       string
	Endpoint      string
	PreloadJSFlag string
	JSpathFlag    string
}

// MarkDown implements cli.MarkDown interface
func (c *AttachCommand) MarkDown() string {
	items := []string{
		"# Attach",
		"Connect to remote Bor IPC console.",
		c.Flags().MarkDown(),
	}

	return strings.Join(items, "\n\n")
}

// Help implements the cli.Command interface
func (c *AttachCommand) Help() string {
	return `Usage: bor attach <IPC FILE>

  Connect to remote Bor IPC console.`
}

// Synopsis implements the cli.Command interface
func (c *AttachCommand) Synopsis() string {
	return "Connect to Bor via IPC"
}

func (c *AttachCommand) Flags() *flagset.Flagset {
	f := flagset.NewFlagSet("attach")

	f.StringFlag(&flagset.StringFlag{
		Name:  "exec",
		Usage: "Command to run in remote console",
		Value: &c.ExecCMD,
	})

	f.StringFlag(&flagset.StringFlag{
		Name:  "preload",
		Usage: "Comma separated list of JavaScript files to preload into the console",
		Value: &c.PreloadJSFlag,
	})

	f.StringFlag(&flagset.StringFlag{
		Name:  "jspath",
		Usage: "JavaScript root path for `loadScript`",
		Value: &c.JSpathFlag,
	})

	return f
}

// Run implements the cli.Command interface
func (c *AttachCommand) Run(args []string) int {
	flags := c.Flags()

	//check if first arg is flag or IPC location
	if len(args) == 0 {
		args = append(args, "")
	}

	if args[0] != "" && strings.HasPrefix(args[0], "--") {
		if err := flags.Parse(args); err != nil {
			c.UI.Error(err.Error())
			return 1
		}
	} else {
		c.Endpoint = args[0]
		if err := flags.Parse(args[1:]); err != nil {
			c.UI.Error(err.Error())
			return 1
		}
	}

	if err := c.remoteConsole(); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	return 0
}

// remoteConsole will connect to a remote bor instance, attaching a JavaScript
// console to it.
// nolint: unparam
func (c *AttachCommand) remoteConsole() error {
	// Attach to a remotely running geth instance and start the JavaScript console
	path := node.DefaultDataDir()

	if c.Endpoint == "" {
		if c.Meta.dataDir != "" {
			path = c.Meta.dataDir
		}

		if path != "" {
			homeDir, _ := os.UserHomeDir()
			path = filepath.Join(homeDir, "/.bor/data")
		}

		c.Endpoint = fmt.Sprintf("%s/bor.ipc", path)
	}

	client, err := dialRPC(c.Endpoint)

	if err != nil {
		utils.Fatalf("Unable to attach to remote bor: %v", err)
	}

	config := console.Config{
		DataDir: path,
		DocRoot: c.JSpathFlag,
		Client:  client,
		Preload: c.makeConsolePreloads(),
	}

	console, err := console.New(config)
	if err != nil {
		utils.Fatalf("Failed to start the JavaScript console: %v", err)
	}

	defer func() {
		if err := console.Stop(false); err != nil {
			c.UI.Error(err.Error())
		}
	}()

	if c.ExecCMD != "" {
		console.Evaluate(c.ExecCMD)
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
func (c *AttachCommand) makeConsolePreloads() []string {
	// Skip preloading if there's nothing to preload
	if c.PreloadJSFlag == "" {
		return nil
	}
	// Otherwise resolve absolute paths and return them
	splitFlags := strings.Split(c.PreloadJSFlag, ",")
	preloads := make([]string, 0, len(splitFlags))

	for _, file := range splitFlags {
		preloads = append(preloads, strings.TrimSpace(file))
	}

	return preloads
}
