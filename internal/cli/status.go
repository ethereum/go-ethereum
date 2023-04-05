package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/internal/cli/flagset"
	"github.com/ethereum/go-ethereum/internal/cli/server/proto"
)

// StatusCommand is the command to output the status of the client
type StatusCommand struct {
	*Meta2

	wait bool
}

func (c *StatusCommand) Flags() *flagset.Flagset {
	flags := c.NewFlagSet("status")

	flags.BoolFlag(&flagset.BoolFlag{
		Name:    "w",
		Value:   &c.wait,
		Usage:   "wait for Bor node to be available",
		Default: false,
	})

	return flags
}

// MarkDown implements cli.MarkDown interface
func (p *StatusCommand) MarkDown() string {
	items := []string{
		"# Status",
		"The ```status``` command outputs the status of the client.",
	}

	return strings.Join(items, "\n\n")
}

// Help implements the cli.Command interface
func (p *StatusCommand) Help() string {
	return `Usage: bor status

  Output the status of the client`
}

// Synopsis implements the cli.Command interface
func (c *StatusCommand) Synopsis() string {
	return "Output the status of the client"
}

// Run implements the cli.Command interface
func (c *StatusCommand) Run(args []string) int {
	flags := c.Flags()
	if err := flags.Parse(args); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	borClt, err := c.BorConn()
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	status, err := borClt.Status(context.Background(), &proto.StatusRequest{Wait: c.wait})
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	c.UI.Output(printStatus(status))

	return 0
}

func printStatus(status *proto.StatusResponse) string {
	printHeader := func(h *proto.Header) string {
		return formatKV([]string{
			fmt.Sprintf("Hash|%s", h.Hash),
			fmt.Sprintf("Number|%d", h.Number),
		})
	}

	forks := make([]string, len(status.Forks)+1)
	forks[0] = "Name|Block|Enabled"

	for i, d := range status.Forks {
		forks[i+1] = fmt.Sprintf("%s|%d|%v", d.Name, d.Block, !d.Disabled)
	}

	full := []string{
		"General",
		formatKV([]string{
			fmt.Sprintf("Num peers|%d", status.NumPeers),
			fmt.Sprintf("Sync mode|%s", status.SyncMode),
		}),
		"\nCurrent Header",
		printHeader(status.CurrentHeader),
		"\nCurrent Block",
		printHeader(status.CurrentBlock),
		"\nSyncing",
		formatKV([]string{
			fmt.Sprintf("Current block|%d", status.Syncing.CurrentBlock),
			fmt.Sprintf("Highest block|%d", status.Syncing.HighestBlock),
			fmt.Sprintf("Starting block|%d", status.Syncing.StartingBlock),
		}),
		"\nForks",
		formatList(forks),
	}

	return strings.Join(full, "\n")
}
