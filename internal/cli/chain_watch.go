package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/internal/cli/flagset"
	"github.com/ethereum/go-ethereum/internal/cli/server/proto"
)

// ChainWatchCommand is the command to group the peers commands
type ChainWatchCommand struct {
	*Meta2
}

// Help implements the cli.Command interface
func (c *ChainWatchCommand) Help() string {
	return `Usage: bor chain watch

  This command is used to view the chainHead, reorg and fork events in real-time`
}

func (c *ChainWatchCommand) Flags() *flagset.Flagset {
	flags := c.NewFlagSet("chain watch")

	return flags
}

// Synopsis implements the cli.Command interface
func (c *ChainWatchCommand) Synopsis() string {
	return "Watch the chainHead, reorg and fork events in real-time"
}

// Run implements the cli.Command interface
func (c *ChainWatchCommand) Run(args []string) int {
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

	sub, err := borClt.ChainWatch(context.Background(), &proto.ChainWatchRequest{})
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signalCh
		sub.CloseSend()
	}()

	for {
		msg, err := sub.Recv()
		if err != nil {
			// if err == EOF if finished on the other side
			c.UI.Output(err.Error())
			break
		}
		c.UI.Output(formatHeadEvent(msg))
	}

	return 0
}

func formatHeadEvent(msg *proto.ChainWatchResponse) string {
	var out string
	if msg.Type == core.Chain2HeadCanonicalEvent {
		out = fmt.Sprintf("Block Added : %v", msg.Newchain)
	} else if msg.Type == core.Chain2HeadForkEvent {
		out = fmt.Sprintf("New Fork Block : %v", msg.Newchain)
	} else if msg.Type == core.Chain2HeadReorgEvent {
		out = fmt.Sprintf("Reorg Detected \nAdded : %v \nRemoved : %v", msg.Newchain, msg.Oldchain)
	}
	return out
}
