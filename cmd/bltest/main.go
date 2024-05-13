// Copyright 2024 The go-ethereum Authors
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

//TODO only for manual testing of ethclient/lightclient; remove before merging to master
package main

import (
	"context"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/beacon/config"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient/lightclient"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"
)

var (
	verbosityFlag = &cli.IntFlag{
		Name:     "verbosity",
		Usage:    "Logging verbosity: 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=detail",
		Value:    3,
		Category: flags.LoggingCategory,
	}
	vmoduleFlag = &cli.StringFlag{
		Name:     "vmodule",
		Usage:    "Per-module verbosity: comma-separated list of <pattern>=<level> (e.g. eth/*=5,p2p=4)",
		Value:    "",
		Hidden:   true,
		Category: flags.LoggingCategory,
	}
)

func main() {
	app := flags.NewApp("beacon light syncer tool")
	app.Flags = []cli.Flag{
		utils.BeaconApiFlag,
		utils.BeaconApiHeaderFlag,
		utils.BeaconThresholdFlag,
		utils.BeaconNoFilterFlag,
		utils.BeaconConfigFlag,
		utils.BeaconGenesisRootFlag,
		utils.BeaconGenesisTimeFlag,
		utils.BeaconCheckpointFlag,
		utils.BltestApiFlag,
		//TODO datadir for optional permanent database
		utils.MainnetFlag,
		utils.SepoliaFlag,
		utils.GoerliFlag,
		verbosityFlag,
		vmoduleFlag,
	}
	app.Action = sync

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func sync(ctx *cli.Context) error {
	usecolor := (isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())) && os.Getenv("TERM") != "dumb"
	output := io.Writer(os.Stderr)
	if usecolor {
		output = colorable.NewColorable(os.Stderr)
	}
	verbosity := log.FromLegacyLevel(ctx.Int(verbosityFlag.Name))
	log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(output, verbosity, usecolor)))

	customHeaders := make(http.Header)
	for _, s := range ctx.StringSlice(utils.BeaconApiHeaderFlag.Name) { //TODO separate header flag for EL
		kv := strings.Split(s, ":")
		if len(kv) != 2 {
			utils.Fatalf("Invalid custom API header entry: %s", s)
		}
		customHeaders.Add(strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1]))
	}

	var opts []rpc.ClientOption
	if len(customHeaders) > 0 {
		opts = append(opts, rpc.WithHeaders(customHeaders))
	}
	rpcClient, err := rpc.DialOptions(context.Background(), ctx.String(utils.BltestApiFlag.Name), opts...)
	if err != nil {
		utils.Fatalf("Could not create RPC client: %v", err)
	}
	client := lightclient.NewClient(config.MakeLightClientConfig(ctx), memorydb.New(), rpcClient)
	client.Start()

	headCh := make(chan *types.Header, 1)
	client.SubscribeNewHead(context.Background(), headCh)

	// run until stopped
loop:
	for {
		select {
		case head := <-headCh:
			log.Info("SubscribeNewHead delivered new head", "number", head.Number, "hash", head.Hash(), "parentHash", head.ParentHash)
			ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
			if block, err := client.BlockByHash(ctx, head.ParentHash); err == nil {
				log.Info("BlockByHash", "hash", head.ParentHash, "block.Hash", block.Hash(), "block.Number", block.Number(), "len(block.Transactions)", len(block.Transactions()))
			} else {
				log.Error("BlockByHash", "hash", head.ParentHash, "error", err)
			}
			num := big.NewInt(10)
			num.Sub(head.Number, num)
			if block, err := client.BlockByNumber(ctx, num); err == nil {
				log.Info("BlockByNumber", "number", num, "block.Hash", block.Hash(), "block.Number", block.Number(), "len(block.Transactions)", len(block.Transactions()))
			} else {
				log.Error("BlockByNumber", "number", num, "error", err)
			}
			if tc, err := client.TransactionCount(ctx, head.Hash()); err == nil {
				log.Info("TransactionCount", "hash", head.Hash(), "count", tc)
			} else {
				log.Error("TransactionCount", "hash", head.Hash(), "error", err)
			}
			testState := func(addr common.Address) {
				if balance, err := client.BalanceAt(ctx, addr, big.NewInt(int64(rpc.LatestBlockNumber))); err == nil {
					log.Info("BalanceAt ", "address", addr, "balance", balance)
				} else {
					log.Error("BalanceAt ", "address", addr, "error", err)
				}
				if code, err := client.CodeAt(ctx, addr, big.NewInt(int64(rpc.LatestBlockNumber))); err == nil {
					log.Info("CodeAt ", "address", addr, "len(code)", len(code))
				} else {
					log.Error("CodeAt ", "address", addr, "error", err)
				}
				if storage, err := client.StorageAt(ctx, addr, common.Hash{}, big.NewInt(int64(rpc.LatestBlockNumber))); err == nil {
					log.Info("StorageAt ", "address", addr, "key", common.Hash{}, "storage", storage)
				} else {
					log.Error("StorageAt ", "address", addr, "key", common.Hash{}, "error", err)
				}
			}
			testState(common.Address{})
			testState(common.HexToAddress("0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2")) // WETH contract
		case <-ctx.Done():
			break loop
		}
	}

	client.Stop()
	return nil
}
