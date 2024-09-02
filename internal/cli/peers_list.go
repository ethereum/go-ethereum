package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/internal/cli/flagset"
	"github.com/ethereum/go-ethereum/internal/cli/server/proto"
)

// PeersListCommand is the command to group the peers commands
type PeersListCommand struct {
	*Meta2
}

// MarkDown implements cli.MarkDown interface
func (p *PeersListCommand) MarkDown() string {
	items := []string{
		"# Peers add",
		"The ```peers list``` command lists the connected peers.",
		p.Flags().MarkDown(),
	}

	return strings.Join(items, "\n\n")
}

// Help implements the cli.Command interface
func (p *PeersListCommand) Help() string {
	return `Usage: bor peers list

  Lists the connected peers

  ` + p.Flags().Help()
}

func (p *PeersListCommand) Flags() *flagset.Flagset {
	flags := p.NewFlagSet("peers list")

	return flags
}

// Synopsis implements the cli.Command interface
func (p *PeersListCommand) Synopsis() string {
	return ""
}

// Run implements the cli.Command interface
func (p *PeersListCommand) Run(args []string) int {
	flags := p.Flags()
	if err := flags.Parse(args); err != nil {
		p.UI.Error(err.Error())
		return 1
	}

	borClt, err := p.BorConn()
	if err != nil {
		p.UI.Error(err.Error())
		return 1
	}

	req := &proto.PeersListRequest{}
	resp, err := borClt.PeersList(context.Background(), req)

	if err != nil {
		p.UI.Error(err.Error())
		return 1
	}

	p.UI.Output(formatPeers(resp.Peers))

	return 0
}

func formatPeers(peers []*proto.Peer) string {
	if len(peers) == 0 {
		return "No peers found"
	}

	rows := make([]string, len(peers)+1)
	rows[0] = "ID|Enode|Name|Caps|Static|Trusted"

	for i, d := range peers {
		enode := strings.TrimPrefix(d.Enode, "enode://")

		rows[i+1] = fmt.Sprintf("%s|%s|%s|%s|%v|%v",
			d.Id,
			enode[:10],
			d.Name,
			strings.Join(d.Caps, ","),
			d.Static,
			d.Trusted)
	}

	return formatList(rows)
}
