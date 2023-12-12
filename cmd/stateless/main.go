package main

import (
	"fmt"
	"github.com/ethereum/go-ethereum/console/prompt"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/internal/debug"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/params"
	"github.com/urfave/cli/v2"
	"go.uber.org/automaxprocs/maxprocs"
	"os"
)

var (
	BlockWitnessFlag = &cli.StringFlag{
		Name:  "block-witness",
		Usage: "foo bar",
	}
)

var app = flags.NewApp("stateless block executor")

func init() {
	// Initialize the CLI app and start Geth
	app.Action = stateless
	app.Copyright = "Copyright 2013-2023 The go-ethereum Authors"

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

func stateless(ctx *cli.Context) error {
	var vmConfig vm.Config

	blockWitnessPath := ctx.String(BlockWitnessFlag.Name)
	if blockWitnessPath == "" {
		panic("block witness required")
	}

	f, err := os.Open(blockWitnessPath)
	if err != nil {
		panic(err)
	}

	var b []byte
	f.Read(b)
	block, witness, err := state.DecodeWitnessRLP(b)
	if err != nil {
		panic(err)
	}

	memoryDb := witness.PopulateMemoryDB()
	db, err := state.New(witness.Root(), state.NewDatabase(memoryDb), nil)
	if err != nil {
		panic(err)
	}
	chainConfig := params.MainnetChainConfig
	engine, err := ethconfig.CreateConsensusEngine(chainConfig, memoryDb)
	if err != nil {
		panic(err)
	}
	validator := core.NewBlockValidator(chainConfig, nil, engine)
	processor := core.NewStateProcessor(chainConfig, nil, engine)

	receipts, logs, usedGas, err := processor.ProcessStateless(witness, block, db, vmConfig)
	if err != nil {
		panic(err)
	}

	_ = logs

	if err := validator.ValidateState(block, db, receipts, usedGas); err != nil {
		panic(err)
	}

	return nil
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
