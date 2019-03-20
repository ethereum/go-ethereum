// Copyright 2016 The go-ethereum Authors
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

// Command bzzhash computes a swarm tree hash.
package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/ens"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"gopkg.in/urfave/cli.v1"
)

var hashCommand = cli.Command{
	Action:             hash,
	CustomHelpTemplate: helpTemplate,
	Name:               "hash",
	Usage:              "print the swarm hash of a file or directory",
	ArgsUsage:          "<file>",
	Description:        "Prints the swarm hash of file or directory",
	Subcommands: []cli.Command{
		{
			CustomHelpTemplate: helpTemplate,
			Name:               "ens",
			Usage:              "converts a swarm hash to an ens EIP1577 compatible CIDv1 hash",
			ArgsUsage:          "<ref>",
			Description:        "",
			Subcommands: []cli.Command{
				{
					Action:             encodeEipHash,
					CustomHelpTemplate: helpTemplate,
					Name:               "contenthash",
					Usage:              "converts a swarm hash to an ens EIP1577 compatible CIDv1 hash",
					ArgsUsage:          "<ref>",
					Description:        "",
				},
				{
					Action:             ensNodeHash,
					CustomHelpTemplate: helpTemplate,
					Name:               "node",
					Usage:              "converts an ens name to an ENS node hash",
					ArgsUsage:          "<ref>",
					Description:        "",
				},
			},
		},
	}}

func hash(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		utils.Fatalf("Usage: swarm hash <file name>")
	}
	f, err := os.Open(args[0])
	if err != nil {
		utils.Fatalf("Error opening file " + args[1])
	}
	defer f.Close()

	stat, _ := f.Stat()
	fileStore := storage.NewFileStore(&storage.FakeChunkStore{}, storage.NewFileStoreParams())
	addr, _, err := fileStore.Store(context.TODO(), f, stat.Size(), false)
	if err != nil {
		utils.Fatalf("%v\n", err)
	} else {
		fmt.Printf("%v\n", addr)
	}
}
func ensNodeHash(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		utils.Fatalf("Usage: swarm hash ens node <ens name>")
	}
	ensName := args[0]

	hash := ens.EnsNode(ensName)

	stringHex := hex.EncodeToString(hash[:])
	fmt.Println(stringHex)
}
func encodeEipHash(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		utils.Fatalf("Usage: swarm hash ens <swarm hash>")
	}
	swarmHash := args[0]

	hash := common.HexToHash(swarmHash)
	ensHash, err := ens.EncodeSwarmHash(hash)
	if err != nil {
		utils.Fatalf("error converting swarm hash", err)
	}

	stringHex := hex.EncodeToString(ensHash)
	fmt.Println(stringHex)
}
