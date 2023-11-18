// Copyright 2023 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"
)

var app = &cli.App{
	Usage:  "go-ethereum beacon light sync tool",
	Action: run,
}

var (
	EngineApiFlag = &cli.StringFlag{
		Name:  "engine",
		Usage: "url to execution client engine api",
	}
	JwtSecretFlag = &cli.StringFlag{
		Name:  "jwtsecret",
		Usage: "path to jwt secret used to communicate with execution client",
	}
	LightClientServerFlag = &cli.StringFlag{
		Name:  "server",
		Usage: "server which provides the beacon light client apis",
	}
	TrustedBlockRootFlag = &cli.StringFlag{
		Name:  "trusted-root",
		Usage: "root of trusted block within the weak-subjectivity window",
	}
	VerbosityFlag = &cli.IntFlag{
		Name:  "verbosity",
		Usage: "Logging verbosity: 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=detail",
		Value: 3,
	}
)

func init() {
	app.Flags = flags.Merge([]cli.Flag{
		EngineApiFlag,
		JwtSecretFlag,
		LightClientServerFlag,
		TrustedBlockRootFlag,
		VerbosityFlag,
	})
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx *cli.Context) error {
	// Setup logger.
	var (
		usecolor  = (isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())) && os.Getenv("TERM") != "dumb"
		output    = io.Writer(os.Stdout)
		verbosity = log.FromLegacyLevel(ctx.Int(VerbosityFlag.Name))
	)
	if usecolor {
		output = colorable.NewColorable(os.Stdout)
	}
	log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(output, verbosity, usecolor)))

	// Start light client.
	var (
		root   = ctx.String(TrustedBlockRootFlag.Name)
		engine = makeRPCClient(ctx)
		server = ctx.String(LightClientServerFlag.Name)
	)
	chain, err := bootstrap(context.Background(), server, common.HexToHash(root))
	if err != nil {
		return fmt.Errorf("failed to bootstrap: %v", err)
	}
	go chain.Start()

	headCh := make(chan ChainHeadEvent)
	chain.SubscribeChainHeadEvent(headCh)

	// Send new head events to engine api.
	for {
		select {
		case head := <-headCh:
			if err := sendUpdate(engine, head.Data, chain.Finalized().Hash()); err != nil {
				log.Error("unable to send update to execution client", "err", err)
			}
		}
	}
}

// sendUpdate passes the execution payload to execution client and affirms it
// with a forck choice updated.
func sendUpdate(engine *rpc.Client, ep *engine.ExecutableData, finalized common.Hash) error {
	if _, err := callNewPayloadV2(engine, ep); err != nil {
		return fmt.Errorf("failed to send new payload: %w", err)
	}
	if _, err := callForkchoiceUpdatedV1(engine, ep.BlockHash, finalized); err != nil {
		return fmt.Errorf("failed to send forkchoice updated: %w", err)
	}
	return nil
}

func makeRPCClient(ctx *cli.Context) *rpc.Client {
	if !ctx.IsSet(EngineApiFlag.Name) {
		log.Warn("No engine API target specified, performing a dry run")
		return nil
	}
	if !ctx.IsSet(JwtSecretFlag.Name) {
		exit(fmt.Errorf("JWT secret parameter missing")) // TODO use default if datadir is specified
	}
	engineApiUrl, jwtFileName := ctx.String(EngineApiFlag.Name), ctx.String(JwtSecretFlag.Name)
	var jwtSecret [32]byte
	if jwt, err := os.ReadFile(jwtFileName); err != nil {
		utils.Fatalf("Error loading or generating JWT secret: %v", err)
	} else {
		copy(jwtSecret[:], common.FromHex(string(jwt)))
	}
	auth := node.NewJWTAuth(jwtSecret)
	cl, err := rpc.DialOptions(context.Background(), engineApiUrl, rpc.WithHTTPAuth(auth))
	if err != nil {
		utils.Fatalf("Could not create RPC client: %v", err)
	}
	return cl
}

func callNewPayloadV2(client *rpc.Client, ep *engine.ExecutableData) (string, error) {
	var resp engine.PayloadStatusV1
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	err := client.CallContext(ctx, &resp, "engine_newPayloadV2", ep)
	cancel()
	return resp.Status, err
}

func callForkchoiceUpdatedV1(client *rpc.Client, headHash, finalizedHash common.Hash) (string, error) {
	var resp engine.ForkChoiceResponse
	update := engine.ForkchoiceStateV1{
		HeadBlockHash:      headHash,
		SafeBlockHash:      common.Hash{},
		FinalizedBlockHash: common.Hash{},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	err := client.CallContext(ctx, &resp, "engine_forkchoiceUpdatedV1", update, nil)
	cancel()
	return resp.PayloadStatus.Status, err
}

func exit(err interface{}) {
	if err == nil {
		os.Exit(0)
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
