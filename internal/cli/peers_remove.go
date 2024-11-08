package cli

import (
	"context"
	"strings"

	"github.com/ethereum/go-ethereum/internal/cli/flagset"
	"github.com/ethereum/go-ethereum/internal/cli/server/proto"
)

// PeersRemoveCommand is the command to group the peers commands
type PeersRemoveCommand struct {
	*Meta2

	trusted bool
}

// MarkDown implements cli.MarkDown interface
func (p *PeersRemoveCommand) MarkDown() string {
	items := []string{
		"# Peers remove",
		"The ```peers remove <enode>``` command disconnects the local client from a connected peer if exists.",
		p.Flags().MarkDown(),
	}

	return strings.Join(items, "\n\n")
}

// Help implements the cli.Command interface
func (p *PeersRemoveCommand) Help() string {
	return `Usage: bor peers remove <enode>

  Disconnects the local client from a connected peer if exists.

  ` + p.Flags().Help()
}

func (p *PeersRemoveCommand) Flags() *flagset.Flagset {
	flags := p.NewFlagSet("peers remove")

	flags.BoolFlag(&flagset.BoolFlag{
		Name:  "trusted",
		Usage: "Add the peer as a trusted",
		Value: &p.trusted,
	})

	return flags
}

// Synopsis implements the cli.Command interface
func (p *PeersRemoveCommand) Synopsis() string {
	return "Disconnects a peer from the client"
}

// Run implements the cli.Command interface
func (p *PeersRemoveCommand) Run(args []string) int {
	flags := p.Flags()
	if err := flags.Parse(args); err != nil {
		p.UI.Error(err.Error())
		return 1
	}

	args = flags.Args()
	if len(args) != 1 {
		p.UI.Error("No enode address provided")
		return 1
	}

	borClt, err := p.BorConn()
	if err != nil {
		p.UI.Error(err.Error())
		return 1
	}

	req := &proto.PeersRemoveRequest{
		Enode:   args[0],
		Trusted: p.trusted,
	}
	if _, err := borClt.PeersRemove(context.Background(), req); err != nil {
		p.UI.Error(err.Error())
		return 1
	}

	return 0
}
