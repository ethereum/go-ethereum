package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ethereum/go-ethereum/command/flagset"
	"github.com/ethereum/go-ethereum/command/server/proto"
	"github.com/ethereum/go-ethereum/core"
)

// ChainWatchCommand is the command to group the peers commands
type ChainWatchCommand struct {
	*Meta2
}

// Help implements the cli.Command interface
func (c *ChainWatchCommand) Help() string {
	return ``
}

func (c *ChainWatchCommand) Flags() *flagset.Flagset {
	flags := c.NewFlagSet("chain watch")

	return flags
}

// Synopsis implements the cli.Command interface
func (c *ChainWatchCommand) Synopsis() string {
	return ""
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
		if msg.Type == core.Chain2HeadCanonicalEvent {
			out := fmt.Sprintf("Block Added : %v", msg.Newchain)
			c.UI.Output(out)
		} else if msg.Type == core.Chain2HeadForkEvent {
			out := fmt.Sprintf("New Fork Block : %v", msg.Newchain)
			c.UI.Output(out)
		} else if msg.Type == core.Chain2HeadReorgEvent {
			c.UI.Output("Reorg Detected")

			out := fmt.Sprintf("Added : %v", msg.Newchain)
			c.UI.Output(out)

			out = fmt.Sprintf("Removed : %v", msg.Oldchain)
			c.UI.Output(out)
		}

		// fmt.Println(msg)
	}

	return 0
}
