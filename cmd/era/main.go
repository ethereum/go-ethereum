// Copyright 2023 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/era"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/urfave/cli/v2"
)

var app = flags.NewApp("go-ethereum era tool")

var (
	dirFlag = &cli.StringFlag{
		Name:  "dir",
		Usage: "directory storing all relevant era1 files",
		Value: "eras",
	}
	networkFlag = &cli.StringFlag{
		Name:  "network",
		Usage: "network name associated with era1 files",
		Value: "mainnet",
	}
	eraSizeFlag = &cli.IntFlag{
		Name:  "size",
		Usage: "number of blocks per era",
		Value: era.MaxEra1Size,
	}
	txsFlag = &cli.BoolFlag{
		Name:  "txs",
		Usage: "print full transaction values",
	}
)

var (
	blockCommand = &cli.Command{
		Name:      "block",
		Usage:     "get block data",
		ArgsUsage: "<number>",
		Action:    block,
		Flags: []cli.Flag{
			txsFlag,
		},
	}
	infoCommand = &cli.Command{
		Name:      "info",
		ArgsUsage: "<epoch>",
		Usage:     "get epoch information",
		Action:    info,
	}
	verifyCommand = &cli.Command{
		Name:      "verify",
		ArgsUsage: "<expected>",
		Usage:     "verifies each era1 against expected accumulator root",
		Action:    verify,
	}
)

func init() {
	app.Commands = []*cli.Command{
		blockCommand,
		infoCommand,
		verifyCommand,
	}
	app.Flags = []cli.Flag{
		dirFlag,
		networkFlag,
		eraSizeFlag,
	}
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

// block prints the specified block from an era1 store.
func block(ctx *cli.Context) error {
	num, err := strconv.ParseUint(ctx.Args().First(), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid block number: %w", err)
	}
	e, err := open(ctx, num/uint64(ctx.Int(eraSizeFlag.Name)))
	if err != nil {
		return fmt.Errorf("error opening era1: %w", err)
	}
	defer e.Close()
	// Read block with number.
	block, err := e.GetBlockByNumber(num)
	if err != nil {
		return fmt.Errorf("error reading block %d: %w", num, err)
	}
	// Convert block to JSON and print.
	val := ethapi.RPCMarshalBlock(block, ctx.Bool(txsFlag.Name), ctx.Bool(txsFlag.Name), params.MainnetChainConfig)
	b, err := json.MarshalIndent(val, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling json: %w", err)
	}
	fmt.Println(string(b))
	return nil
}

// info prints some high-level information about the era1 file.
func info(ctx *cli.Context) error {
	epoch, err := strconv.ParseUint(ctx.Args().First(), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid epoch number: %w", err)
	}
	e, err := open(ctx, epoch)
	if err != nil {
		return err
	}
	defer e.Close()
	acc, err := e.Accumulator()
	if err != nil {
		return fmt.Errorf("error reading accumulator: %w", err)
	}
	td, err := e.InitialTD()
	if err != nil {
		return fmt.Errorf("error reading total difficulty: %w", err)
	}
	info := struct {
		Accumulator     common.Hash `json:"accumulator"`
		TotalDifficulty *big.Int    `json:"totalDifficulty"`
		StartBlock      uint64      `json:"startBlock"`
		Count           uint64      `json:"count"`
	}{
		acc, td, e.Start(), e.Count(),
	}
	b, _ := json.MarshalIndent(info, "", "  ")
	fmt.Println(string(b))
	return nil
}

// open opens an era1 file at a certain epoch.
func open(ctx *cli.Context, epoch uint64) (*era.Era, error) {
	var (
		dir     = ctx.String(dirFlag.Name)
		network = ctx.String(networkFlag.Name)
	)
	entries, err := era.ReadDir(dir, network)
	if err != nil {
		return nil, fmt.Errorf("error reading era dir: %w", err)
	}
	if epoch >= uint64(len(entries)) {
		return nil, fmt.Errorf("epoch out-of-bounds: last %d, want %d", len(entries)-1, epoch)
	}
	return era.Open(path.Join(dir, entries[epoch]))
}

// verify checks each era1 file in a directory to ensure it is well-formed and
// that the accumulator matches the expected value.
func verify(ctx *cli.Context) error {
	if ctx.Args().Len() != 1 {
		return errors.New("missing accumulators file")
	}

	roots, err := readHashes(ctx.Args().First())
	if err != nil {
		return fmt.Errorf("unable to read expected roots file: %w", err)
	}

	var (
		dir      = ctx.String(dirFlag.Name)
		network  = ctx.String(networkFlag.Name)
		start    = time.Now()
		reported = time.Now()
	)

	entries, err := era.ReadDir(dir, network)
	if err != nil {
		return fmt.Errorf("error reading %s: %w", dir, err)
	}

	if len(entries) != len(roots) {
		return errors.New("number of era1 files should match the number of accumulator hashes")
	}

	// Verify each epoch matches the expected root.
	for i, want := range roots {
		// Wrap in function so defers don't stack.
		err := func() error {
			name := entries[i]
			e, err := era.Open(path.Join(dir, name))
			if err != nil {
				return fmt.Errorf("error opening era1 file %s: %w", name, err)
			}
			defer e.Close()
			// Read accumulator and check against expected.
			if got, err := e.Accumulator(); err != nil {
				return fmt.Errorf("error retrieving accumulator for %s: %w", name, err)
			} else if got != want {
				return fmt.Errorf("invalid root %s: got %s, want %s", name, got, want)
			}
			// Recompute accumulator.
			if err := checkAccumulator(e); err != nil {
				return fmt.Errorf("error verify era1 file %s: %w", name, err)
			}
			// Give the user some feedback that something is happening.
			if time.Since(reported) >= 8*time.Second {
				fmt.Printf("Verifying Era1 files \t\t verified=%d,\t elapsed=%s\n", i, common.PrettyDuration(time.Since(start)))
				reported = time.Now()
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}

	return nil
}

// checkAccumulator verifies the accumulator matches the data in the Era.
func checkAccumulator(e *era.Era) error {
	var (
		err    error
		want   common.Hash
		td     *big.Int
		tds    = make([]*big.Int, 0)
		hashes = make([]common.Hash, 0)
	)
	if want, err = e.Accumulator(); err != nil {
		return fmt.Errorf("error reading accumulator: %w", err)
	}
	if td, err = e.InitialTD(); err != nil {
		return fmt.Errorf("error reading total difficulty: %w", err)
	}
	it, err := era.NewIterator(e)
	if err != nil {
		return fmt.Errorf("error making era iterator: %w", err)
	}
	// To fully verify an era the following attributes must be checked:
	//   1) the block index is constructed correctly
	//   2) the tx root matches the value in the block
	//   3) the receipts root matches the value in the block
	//   4) the starting total difficulty value is correct
	//   5) the accumulator is correct by recomputing it locally, which verifies
	//      the blocks are all correct (via hash)
	//
	// The attributes 1), 2), and 3) are checked for each block. 4) and 5) require
	// accumulation across the entire set and are verified at the end.
	for it.Next() {
		// 1) next() walks the block index, so we're able to implicitly verify it.
		if it.Error() != nil {
			return fmt.Errorf("error reading block %d: %w", it.Number(), err)
		}
		block, receipts, err := it.BlockAndReceipts()
		if it.Error() != nil {
			return fmt.Errorf("error reading block %d: %w", it.Number(), err)
		}
		// 2) recompute tx root and verify against header.
		tr := types.DeriveSha(block.Transactions(), trie.NewStackTrie(nil))
		if tr != block.TxHash() {
			return fmt.Errorf("tx root in block %d mismatch: want %s, got %s", block.NumberU64(), block.TxHash(), tr)
		}
		// 3) recompute receipt root and check value against block.
		rr := types.DeriveSha(receipts, trie.NewStackTrie(nil))
		if rr != block.ReceiptHash() {
			return fmt.Errorf("receipt root in block %d mismatch: want %s, got %s", block.NumberU64(), block.ReceiptHash(), rr)
		}
		hashes = append(hashes, block.Hash())
		td.Add(td, block.Difficulty())
		tds = append(tds, new(big.Int).Set(td))
	}
	// 4+5) Verify accumulator and total difficulty.
	got, err := era.ComputeAccumulator(hashes, tds)
	if err != nil {
		return fmt.Errorf("error computing accumulator: %w", err)
	}
	if got != want {
		return fmt.Errorf("expected accumulator root does not match calculated: got %s, want %s", got, want)
	}
	return nil
}

// readHashes reads a file of newline-delimited hashes.
func readHashes(f string) ([]common.Hash, error) {
	b, err := os.ReadFile(f)
	if err != nil {
		return nil, errors.New("unable to open accumulators file")
	}
	s := strings.Split(string(b), "\n")
	// Remove empty last element, if present.
	if s[len(s)-1] == "" {
		s = s[:len(s)-1]
	}
	// Convert to hashes.
	r := make([]common.Hash, len(s))
	for i := range s {
		r[i] = common.HexToHash(s[i])
	}
	return r, nil
}
