package machine

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/tests"
	"gopkg.in/urfave/cli.v1"
)

var (
	TraceFlag = cli.BoolFlag{
		Name:  "trace",
		Usage: "Output full trace logs to files <txhash>.jsonl",
	}
	TraceDisableMemoryFlag = cli.BoolFlag{
		Name:  "trace.nomemory",
		Usage: "Disable full memory dump in traces",
	}
	TraceDisableStackFlag = cli.BoolFlag{
		Name:  "trace.nostack",
		Usage: "Disable stack output in traces",
	}
	OutputAllocFlag = cli.StringFlag{
		Name: "output.alloc",
		Usage: "Determines where to put the `alloc` of the post-state.\n" +
			"\t`stdout` - into the stdout output\n" +
			"\t`stderr` - into the stderr output\n" +
			"\t<file> - into the file <file> ",
		Value: "alloc.json",
	}
	OutputResultFlag = cli.StringFlag{
		Name: "output.result",
		Usage: "Determines where to put the `result` (stateroot, txroot etc) of the post-state.\n" +
			"\t`stdout` - into the stdout output\n" +
			"\t`stderr` - into the stderr output\n" +
			"\t<file> - into the file <file> ",
		Value: "result.json",
	}
	InputAllocFlag = cli.StringFlag{
		Name:  "input.alloc",
		Usage: "`stdin` or file name of where to find the prestate alloc to use.",
		Value: "alloc.json",
	}
	InputEnvFlag = cli.StringFlag{
		Name:  "input.env",
		Usage: "`stdin` or file name of where to find the prestate env to use.",
		Value: "env.json",
	}
	InputTxsFlag = cli.StringFlag{
		Name:  "input.txs",
		Usage: "`stdin` or file name of where to find the transactions to apply.",
		Value: "txs.json",
	}
	RewardFlag = cli.Int64Flag{
		Name:  "state.reward",
		Usage: "Mining reward. Set to -1 to disable",
		Value: 0,
	}
	ChainIDFlag = cli.Int64Flag{
		Name:  "state.chainid",
		Usage: "ChainID to use",
		Value: 1,
	}
	ForknameFlag = cli.StringFlag{
		Name: "state.fork",
		Usage: fmt.Sprintf("Name of ruleset to use."+
			"\n\tAvailable forknames:"+
			" \n\t  %v"+
			"\n\tAvailable extra eips: \n\t  %v"+
			"\n\tSyntax <forkname>(+ExtraEip)",
			strings.Join(tests.AvailableForks(), "\n\t  "),
			strings.Join(vm.ActivateableEips(), ",")),
		Value: "Istanbul",
	}
	VerbosityFlag = cli.IntFlag{
		Name:  "verbosity",
		Usage: "sets the verbosity level",
		Value: 3,
	}
)
