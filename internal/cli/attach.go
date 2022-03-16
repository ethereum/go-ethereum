package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/console"
	"github.com/ethereum/go-ethereum/node"
	"github.com/mitchellh/cli"
)

// VersionCommand is the command to show the version of the agent
type AttachCommand struct {
	UI    cli.Ui
	Meta  *Meta
	Meta2 *Meta2
}

// Help implements the cli.Command interface
func (c *AttachCommand) Help() string {
	return `Usage: bor attach <IPC FILE>

  Connect to Bor IPC console.`
}

// Synopsis implements the cli.Command interface
func (c *AttachCommand) Synopsis() string {
	return "Connect to Bor via IPC"
}

// Run implements the cli.Command interface
func (c *AttachCommand) Run(args []string) int {

	c.remoteConsole(args)

	return 0
}

// remoteConsole will connect to a remote geth instance, attaching a JavaScript
// console to it.
func (c *AttachCommand) remoteConsole(args []string) error {
	// Attach to a remotely running geth instance and start the JavaScript console
	endpoint := args[0]
	path := node.DefaultDataDir()

	if endpoint == "" {
		if c.Meta.dataDir != "" {
			path = c.Meta.dataDir
		}
		if path != "" {
			homeDir, _ := os.UserHomeDir()
			path = filepath.Join(homeDir, "/.bor/data")
		}
		endpoint = fmt.Sprintf("%s/geth.ipc", path)
	}
	client, err := dialRPC(endpoint)
	if err != nil {
		utils.Fatalf("Unable to attach to remote geth: %v", err)
	}
	config := console.Config{
		DataDir: path,
		DocRoot: utils.JSpathFlag.Name,
		Client:  client,
	}

	console, err := console.New(config)
	if err != nil {
		utils.Fatalf("Failed to start the JavaScript console: %v", err)
	}
	defer console.Stop(false)

	if len(args) > 1 {
		if script := args[1]; script == "exec" {
			console.Evaluate(args[2])
			return nil
		}
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
