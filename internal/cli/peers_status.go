package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/internal/cli/flagset"
	"github.com/ethereum/go-ethereum/internal/cli/server/proto"
)

// PeersStatusCommand is the command to group the peers commands
type PeersStatusCommand struct {
	*Meta2
}

// Help implements the cli.Command interface
func (p *PeersStatusCommand) Help() string {
	return `Usage: bor peers status <peer id>

  Display the status of a peer by its id.

  ` + p.Flags().Help()
}

func (p *PeersStatusCommand) Flags() *flagset.Flagset {
	flags := p.NewFlagSet("peers status")

	return flags
}

// Synopsis implements the cli.Command interface
func (c *PeersStatusCommand) Synopsis() string {
	return "Display the status of a peer"
}

// Run implements the cli.Command interface
func (c *PeersStatusCommand) Run(args []string) int {
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

	req := &proto.PeersStatusRequest{
		Enode: args[0],
	}
	resp, err := borClt.PeersStatus(context.Background(), req)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	c.UI.Output(formatPeer(resp.Peer))
	return 0
}

func formatPeer(peer *proto.Peer) string {
	base := formatKV([]string{
		fmt.Sprintf("Name|%s", peer.Name),
		fmt.Sprintf("ID|%s", peer.Id),
		fmt.Sprintf("ENR|%s", peer.Enr),
		fmt.Sprintf("Capabilities|%s", strings.Join(peer.Caps, ",")),
		fmt.Sprintf("Enode|%s", peer.Enode),
		fmt.Sprintf("Static|%v", peer.Static),
		fmt.Sprintf("Trusted|%v", peer.Trusted),
	})
	return base
}
