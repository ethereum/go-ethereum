// Copyright 2025 The go-ethereum Authors
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
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/eth/tracers/native"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/internal/testrand"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

var (
	proofGenerateCommand = &cli.Command{
		Name:      "proofgen",
		Usage:     "Generates tests for state proof verification",
		ArgsUsage: "<RPC endpoint URL>",
		Action:    generateProofTests,
		Flags: []cli.Flag{
			proofTestFileFlag,
			proofTestResultOutputFlag,
			proofTestStatesFlag,
			proofTestStartBlockFlag,
			proofTestEndBlockFlag,
		},
	}

	proofTestFileFlag = &cli.StringFlag{
		Name:     "proof-tests",
		Usage:    "JSON file containing proof test queries",
		Value:    "proof_tests.json",
		Category: flags.TestingCategory,
	}
	proofTestResultOutputFlag = &cli.StringFlag{
		Name:     "proof-output",
		Usage:    "Folder containing detailed trace output files",
		Value:    "",
		Category: flags.TestingCategory,
	}
	proofTestStatesFlag = &cli.Int64Flag{
		Name:     "proof-states",
		Usage:    "Number of states to generate proof against",
		Value:    10000,
		Category: flags.TestingCategory,
	}
	proofTestInvalidOutputFlag = &cli.StringFlag{
		Name:     "proof-invalid",
		Usage:    "Folder containing the mismatched state proof output files",
		Value:    "",
		Category: flags.TestingCategory,
	}
	proofTestStartBlockFlag = &cli.Uint64Flag{
		Name:     "proof-start",
		Usage:    "The number of starting block for proof verification (included)",
		Category: flags.TestingCategory,
	}
	proofTestEndBlockFlag = &cli.Uint64Flag{
		Name:     "proof-end",
		Usage:    "The number of ending block for proof verification (excluded)",
		Category: flags.TestingCategory,
	}
)

type proofGenerator func(cli *client, startBlock uint64, endBlock uint64, number int) ([]uint64, [][]common.Address, [][][]string, error)

func genAccountProof(cli *client, startBlock uint64, endBlock uint64, number int) ([]uint64, [][]common.Address, [][][]string, error) {
	var (
		blockNumbers     []uint64
		accountAddresses [][]common.Address
		storageKeys      [][][]string
		nAccounts        int
		ctx              = context.Background()
		start            = time.Now()
	)
	chainID, err := cli.Eth.ChainID(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	signer := types.LatestSignerForChainID(chainID)

	for {
		if nAccounts >= number {
			break
		}
		blockNumber := uint64(rand.Intn(int(endBlock-startBlock))) + startBlock

		block, err := cli.Eth.BlockByNumber(context.Background(), big.NewInt(int64(blockNumber)))
		if err != nil {
			continue
		}
		var (
			addresses []common.Address
			keys      [][]string
			gather    = func(address common.Address) {
				addresses = append(addresses, address)
				keys = append(keys, nil)
				nAccounts++
			}
		)
		for _, tx := range block.Transactions() {
			if nAccounts >= number {
				break
			}
			sender, err := signer.Sender(tx)
			if err != nil {
				log.Error("Failed to resolve the sender address", "hash", tx.Hash(), "err", err)
				continue
			}
			gather(sender)

			if tx.To() != nil {
				gather(*tx.To())
			}
		}
		blockNumbers = append(blockNumbers, blockNumber)
		accountAddresses = append(accountAddresses, addresses)
		storageKeys = append(storageKeys, keys)
	}
	log.Info("Generated tests for account proof", "blocks", len(blockNumbers), "accounts", nAccounts, "elapsed", common.PrettyDuration(time.Since(start)))
	return blockNumbers, accountAddresses, storageKeys, nil
}

func genNonExistentAccountProof(cli *client, startBlock uint64, endBlock uint64, number int) ([]uint64, [][]common.Address, [][][]string, error) {
	var (
		blockNumbers     []uint64
		accountAddresses [][]common.Address
		storageKeys      [][][]string
		total            int
	)
	for i := 0; i < number/5; i++ {
		var (
			addresses   []common.Address
			keys        [][]string
			blockNumber = uint64(rand.Intn(int(endBlock-startBlock))) + startBlock
		)
		for j := 0; j < 5; j++ {
			addresses = append(addresses, testrand.Address())
			keys = append(keys, nil)
		}
		total += len(addresses)
		blockNumbers = append(blockNumbers, blockNumber)
		accountAddresses = append(accountAddresses, addresses)
		storageKeys = append(storageKeys, keys)
	}
	log.Info("Generated tests for non-existing account proof", "blocks", len(blockNumbers), "accounts", total)
	return blockNumbers, accountAddresses, storageKeys, nil
}

func genStorageProof(cli *client, startBlock uint64, endBlock uint64, number int) ([]uint64, [][]common.Address, [][][]string, error) {
	var (
		blockNumbers     []uint64
		accountAddresses [][]common.Address
		storageKeys      [][][]string

		nAccounts int
		nStorages int
		start     = time.Now()
	)
	for {
		if nAccounts+nStorages >= number {
			break
		}
		blockNumber := uint64(rand.Intn(int(endBlock-startBlock))) + startBlock

		block, err := cli.Eth.BlockByNumber(context.Background(), big.NewInt(int64(blockNumber)))
		if err != nil {
			continue
		}
		var (
			addresses     []common.Address
			slots         [][]string
			tracer        = "prestateTracer"
			configBlob, _ = json.Marshal(native.PrestateTracerConfig{
				DiffMode:       false,
				DisableCode:    true,
				DisableStorage: false,
			})
		)
		for _, tx := range block.Transactions() {
			if nAccounts+nStorages >= number {
				break
			}
			if tx.To() == nil {
				continue
			}
			ret, err := cli.Geth.TraceTransaction(context.Background(), tx.Hash(), &tracers.TraceConfig{
				Tracer:       &tracer,
				TracerConfig: configBlob,
			})
			if err != nil {
				log.Error("Failed to trace the transaction", "blockNumber", blockNumber, "hash", tx.Hash(), "err", err)
				continue
			}
			blob, err := json.Marshal(ret)
			if err != nil {
				log.Error("Failed to marshal data", "err", err)
				continue
			}
			var accounts map[common.Address]*types.Account
			if err := json.Unmarshal(blob, &accounts); err != nil {
				log.Error("Failed to decode trace result", "blockNumber", blockNumber, "hash", tx.Hash(), "err", err)
				continue
			}
			for addr, account := range accounts {
				if len(account.Storage) == 0 {
					continue
				}
				addresses = append(addresses, addr)
				nAccounts += 1

				var keys []string
				for k := range account.Storage {
					keys = append(keys, k.Hex())
				}
				nStorages += len(keys)

				var emptyKeys []string
				for i := 0; i < 3; i++ {
					emptyKeys = append(emptyKeys, testrand.Hash().Hex())
				}
				nStorages += len(emptyKeys)

				slots = append(slots, append(keys, emptyKeys...))
			}
		}
		blockNumbers = append(blockNumbers, blockNumber)
		accountAddresses = append(accountAddresses, addresses)
		storageKeys = append(storageKeys, slots)
	}
	log.Info("Generated tests for storage proof", "blocks", len(blockNumbers), "accounts", nAccounts, "storages", nStorages, "elapsed", common.PrettyDuration(time.Since(start)))
	return blockNumbers, accountAddresses, storageKeys, nil
}

func genProofRequests(cli *client, startBlock, endBlock uint64, states int) (*proofTest, error) {
	var (
		blockNumbers     []uint64
		accountAddresses [][]common.Address
		storageKeys      [][][]string
	)
	ratio := []float64{0.2, 0.1, 0.7}
	for i, fn := range []proofGenerator{genAccountProof, genNonExistentAccountProof, genStorageProof} {
		numbers, addresses, keys, err := fn(cli, startBlock, endBlock, int(float64(states)*ratio[i]))
		if err != nil {
			return nil, err
		}
		blockNumbers = append(blockNumbers, numbers...)
		accountAddresses = append(accountAddresses, addresses...)
		storageKeys = append(storageKeys, keys...)
	}
	return &proofTest{
		BlockNumbers: blockNumbers,
		Addresses:    accountAddresses,
		StorageKeys:  storageKeys,
	}, nil
}

func generateProofTests(clictx *cli.Context) error {
	var (
		client     = makeClient(clictx)
		ctx        = context.Background()
		states     = clictx.Int(proofTestStatesFlag.Name)
		outputFile = clictx.String(proofTestFileFlag.Name)
		outputDir  = clictx.String(proofTestResultOutputFlag.Name)
		startBlock = clictx.Uint64(proofTestStartBlockFlag.Name)
		endBlock   = clictx.Uint64(proofTestEndBlockFlag.Name)
	)
	head, err := client.Eth.BlockNumber(ctx)
	if err != nil {
		exit(err)
	}
	if startBlock > head || endBlock > head {
		return fmt.Errorf("chain is out of proof range, head %d, start: %d, limit: %d", head, startBlock, endBlock)
	}
	if endBlock == 0 {
		endBlock = head
	}
	log.Info("Generating proof states", "startBlock", startBlock, "endBlock", endBlock, "states", states)

	test, err := genProofRequests(client, startBlock, endBlock, states)
	if err != nil {
		exit(err)
	}
	for i, blockNumber := range test.BlockNumbers {
		var hashes []common.Hash
		for j := 0; j < len(test.Addresses[i]); j++ {
			res, err := client.Geth.GetProof(ctx, test.Addresses[i][j], test.StorageKeys[i][j], big.NewInt(int64(blockNumber)))
			if err != nil {
				log.Error("Failed to prove the state", "number", blockNumber, "address", test.Addresses[i][j], "slots", len(test.StorageKeys[i][j]), "err", err)
				continue
			}
			blob, err := json.Marshal(res)
			if err != nil {
				return err
			}
			hashes = append(hashes, crypto.Keccak256Hash(blob))

			writeStateProof(outputDir, blockNumber, test.Addresses[i][j], res)
		}
		test.Results = append(test.Results, hashes)
	}
	writeJSON(outputFile, test)
	return nil
}

func writeStateProof(dir string, blockNumber uint64, address common.Address, result any) {
	if dir == "" {
		return
	}
	// Ensure the directory exists
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		exit(fmt.Errorf("failed to create directories: %w", err))
	}
	fname := fmt.Sprintf("%d-%x", blockNumber, address)
	name := filepath.Join(dir, fname)
	file, err := os.Create(name)
	if err != nil {
		exit(fmt.Errorf("error creating %s: %v", name, err))
		return
	}
	defer file.Close()

	data, _ := json.MarshalIndent(result, "", "    ")
	_, err = file.Write(data)
	if err != nil {
		exit(fmt.Errorf("error writing %s: %v", name, err))
		return
	}
}
