package cli

import (
	"context"

	"github.com/ethereum/go-ethereum/internal/cli/flagset"
	"github.com/ethereum/go-ethereum/internal/cli/server/proto"
)

// PeersAddCommand is the command to group the peers commands
type PeersAddCommand struct {
	*Meta2

	trusted bool
}

// Help implements the cli.Command interface
func (p *PeersAddCommand) Help() string {
	return `Usage: bor peers add <enode>

  Joins the local client to another remote peer.

  ` + p.Flags().Help()
}

func (p *PeersAddCommand) Flags() *flagset.Flagset {
	flags := p.NewFlagSet("peers add")

	flags.BoolFlag(&flagset.BoolFlag{
		Name:  "trusted",
		Usage: "Add the peer as a trusted",
		Value: &p.trusted,
	})

	return flags
}

// Synopsis implements the cli.Command interface
func (c *PeersAddCommand) Synopsis() string {
	return "Join the client to a remote peer"
}

// Run implements the cli.Command interface
func (c *PeersAddCommand) Run(args []string) int {
	flags := c.Flags()
	if err := flags.Parse(args); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	args = flags.Args()
	if len(args) != 1 {
		c.UI.Error("No enode address provided")
		return 1
	}

	borClt, err := c.BorConn()
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	req := &proto.PeersAddRequest{
		Enode:   args[0],
		Trusted: c.trusted,
	}
	if _, err := borClt.PeersAdd(context.Background(), req); err != nil {
		c.UI.Error(err.Error())
		return 1
	}
	return 0
}
