package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/internal/cli/flagset"
	"github.com/ethereum/go-ethereum/internal/cli/server/proto"
)

// DebugBlockCommand is the command to group the peers commands
type DebugBlockCommand struct {
	*Meta2

	output string
}

func (p *DebugBlockCommand) MarkDown() string {
	items := []string{
		"# Debug trace",
		"The ```bor debug block <number>``` command will create an archive containing traces of a bor block.",
		p.Flags().MarkDown(),
	}

	return strings.Join(items, "\n\n")
}

// Help implements the cli.Command interface
func (c *DebugBlockCommand) Help() string {
	return `Usage: bor debug block <number>

  This command is used get traces of a bor block`
}

func (c *DebugBlockCommand) Flags() *flagset.Flagset {
	flags := c.NewFlagSet("trace")

	flags.StringFlag(&flagset.StringFlag{
		Name:  "output",
		Value: &c.output,
		Usage: "Output directory",
	})

	return flags
}

// Synopsis implements the cli.Command interface
func (c *DebugBlockCommand) Synopsis() string {
	return "Get trace of a bor block"
}

// Run implements the cli.Command interface
func (c *DebugBlockCommand) Run(args []string) int {
	flags := c.Flags()

	var number *int64 = nil

	// parse the block number (if available)
	if len(args)%2 != 0 {
		num, err := strconv.ParseInt(args[0], 10, 64)
		if err == nil {
			number = &num
		}

		args = args[1:]
	}
	// parse output directory
	if err := flags.Parse(args); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	borClt, err := c.BorConn()
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	dEnv := &debugEnv{
		output: c.output,
		prefix: "bor-block-trace-",
	}
	if err := dEnv.init(); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	c.UI.Output("Starting block tracer...")
	c.UI.Output("")

	// create a debug block request
	var debugRequest *proto.DebugBlockRequest = &proto.DebugBlockRequest{}
	if number != nil {
		debugRequest.Number = *number
	} else {
		debugRequest.Number = -1
	}

	// send the request
	// receives a grpc stream of debug block response
	stream, err := borClt.DebugBlock(context.Background(), debugRequest)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	if err := dEnv.writeFromStream("block.json", stream); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	if err := dEnv.finish(); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	if c.output != "" {
		c.UI.Output(fmt.Sprintf("Created debug directory: %s", dEnv.dst))
	} else {
		c.UI.Output(fmt.Sprintf("Created block trace archive: %s", dEnv.tarName()))
	}

	return 0
}
