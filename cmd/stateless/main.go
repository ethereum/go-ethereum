package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/console/prompt"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/internal/debug"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/params"
	"github.com/urfave/cli/v2"
	"go.uber.org/automaxprocs/maxprocs"
)

var (
	BlockWitnessFlag = &cli.StringFlag{
		Name:  "block-witness",
		Usage: "foo bar",
	}
	ChainConfigFlag = &cli.StringFlag{
		Name:  "chain-config",
		Usage: "path to a genesis file to source a chain configuration from",
	}

	BlockWitness1Flag = &cli.StringFlag{
		Name:  "witness1",
		Usage: "path to a file containing an rlp-encoded block witness",
	}
	BlockWitness2Flag = &cli.StringFlag{
		Name:  "witness2",
		Usage: "path to a file containing an rlp-encoded block witness",
	}

	LogFileFlag = &cli.StringFlag{
		Name:  "logfile",
		Usage: "if present, generate debug trace (just evm traces in the future).  store trace to the file",
	}

	WitnessDiffCommand = &cli.Command{
		Action:    witnessCmp,
		Name:      "cmp",
		Usage:     "outputs whether two block witnesses are equal",
		ArgsUsage: "cmp --witness1 /path/to/bw1.rlp --witness2 /path/to/bw2.rlp",
		Flags: []cli.Flag{
			BlockWitness1Flag,
			BlockWitness2Flag,
		},
		Description: ``,
	}
	PPCommand = &cli.Command{
		Action:    pp,
		Name:      "pp",
		Usage:     "",
		ArgsUsage: "pp --block-witness /path/to/witness.rlp",
		Flags: []cli.Flag{
			BlockWitnessFlag,
		},
		Description: `pretty-print a block witness`,
	}
	ExecCommand = &cli.Command{
		Action: execCmd,
		Name:   "exec",
		Usage:  "",
		ArgsUsage: "exec --block-witness /path/to/bw.rlp --chain-config /path/to/chainconfig.json" +
			"--log-file /path/to/logfile.txt",
		Flags: []cli.Flag{
			BlockWitnessFlag,
			ChainConfigFlag,
			LogFileFlag,
		},
		Description: `statelessly execute and verify a block`,
	}
	ServerCommand = &cli.Command{
		Action:      server,
		Name:        "server",
		Usage:       "",
		ArgsUsage:   "server --chain-config /path/to/chain-config.json",
		Flags:       []cli.Flag{ChainConfigFlag},
		Description: `Runs an HTTP server which provides an API endpoint for stateless block verification`,
	}
)

var app = flags.NewApp("stateless block execution/verification utilities")

func init() {
	app.Copyright = "Copyright 2013-2024 The go-ethereum Authors"
	app.Commands = []*cli.Command{
		WitnessDiffCommand,
		PPCommand,
		ExecCommand,
		ServerCommand,
	}

	app.Flags = []cli.Flag{
		BlockWitnessFlag,
	}

	app.Before = func(ctx *cli.Context) error {
		maxprocs.Set() // Automatically set GOMAXPROCS to match Linux container CPU quota.
		if err := debug.Setup(ctx); err != nil {
			return err
		}
		return nil
	}
	app.After = func(ctx *cli.Context) error {
		debug.Exit()
		prompt.Stdin.Close() // Resets terminal mode.
		return nil
	}
}

func loadChainConfig(chainConfigPath string) *params.ChainConfig {
	var chainConfig *params.ChainConfig

	if chainConfigPath != "" {
		configBytes, err := os.ReadFile(chainConfigPath)
		if err != nil {
			panic(err)
		}
		dec := json.NewDecoder(bytes.NewBuffer(configBytes))
		err = dec.Decode(&chainConfig)
		if err != nil {
			panic(err)
		}
	} else {
		panic("chain config must be specified")
	}
	return chainConfig
}

func execCmd(ctx *cli.Context) error {
	var logWriter *bufio.Writer
	blockWitnessPath := ctx.String(BlockWitnessFlag.Name)
	if blockWitnessPath == "" {
		panic("block witness required")
	}

	logFile := ctx.String(LogFileFlag.Name)
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY, 0744)
		if err != nil {
			return err
		}
		logWriter = bufio.NewWriter(f)
		if err != nil {
			panic(err)
		}
		defer logWriter.Flush()
	}

	b, err := os.ReadFile(blockWitnessPath)
	if err != nil {
		panic(err)
	}

	witness, err := state.DecodeWitnessRLP(b)
	if err != nil {
		panic(err)
	}

	chainConfig := loadChainConfig(ctx.String(ChainConfigFlag.Name))

	localRoot, err := utils.StatelessExecute(os.Stdout, chainConfig, witness)
	if err != nil {
		panic(err)
	}
	if localRoot != witness.Block.Root() {
		return fmt.Errorf("state root mismatch (local: %x, remote: %x)", localRoot, witness.Block.Root())
	}
	return nil
}

func pp(ctx *cli.Context) error {
	witnessPath := ctx.String(BlockWitnessFlag.Name)
	b, err := os.ReadFile(witnessPath)
	if err != nil {
		return err
	}
	w, err := state.DecodeWitnessRLP(b)
	if err != nil {
		panic(err)
	}

	fmt.Println(w.PrettyPrint())
	return nil
}

func witnessCmp(ctx *cli.Context) error {
	witness1Path := ctx.String(BlockWitness1Flag.Name)
	witness2Path := ctx.String(BlockWitness2Flag.Name)

	b1, err := os.ReadFile(witness1Path)
	if err != nil {
		return err
	}

	b2, err := os.ReadFile(witness2Path)
	if err != nil {
		return err
	}

	w1, err := state.DecodeWitnessRLP(b1)
	if err != nil {
		panic(err)
	}

	w2, err := state.DecodeWitnessRLP(b2)
	if err != nil {
		panic(err)
	}

	w1Hash := w1.Hash()
	w2Hash := w2.Hash()
	if w1Hash != w2Hash {
		fmt.Printf("witness 1 hash (%x) != witness 2 hash (%x)\n", w1Hash, w2Hash)
	}
	return nil
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
