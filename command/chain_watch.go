package main

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/command/flagset"
	"github.com/ethereum/go-ethereum/command/server/proto"
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
		panic(err)
	}

	for {
		msg, err := sub.Recv()
		if err != nil {
			// if err == EOF if finished on the other side
			panic(err)
		}
		if msg.Type == "head" {
			fmt.Println("Block Added : ", msg.Newchain)
		} else if msg.Type == "fork" {
			fmt.Println("New Fork Block :", msg.Newchain)
		} else if msg.Type == "reorg" {
			fmt.Println("Reorg Detected")
			fmt.Println("Added :", msg.Newchain)
			fmt.Println("Removed :", msg.Oldchain)
		}

		// fmt.Println(msg)
	}

	return 0
}
