// Copyright 2018 The go-ethereum Authors
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
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"gopkg.in/urfave/cli.v1"
)

var commandQueryAdmin = cli.Command{
	Name:  "queryadmin",
	Usage: "Fetch the admin list of specified registrar contract",
	Description: `
Fetch the admin list of the specified registrar contract which are regarded
as trusted signers to register checkpoint.
`,
	Flags: []cli.Flag{
		clientURLFlag,
	},
	Action: utils.MigrateFlags(queryAdmin),
}

var commandQueryCheckpoint = cli.Command{
	Name:  "querycheckpoint",
	Usage: "Fetch the specified checkpoint in the registrar contract",
	Description: `
Fetch the registered checkpoint with the specified index.
`,
	Flags: []cli.Flag{
		checkpointIndexFlag,
		clientURLFlag,
	},
	Action: utils.MigrateFlags(queryCheckpoint),
}

var commandPendingProposal = cli.Command{
	Name:  "queryproposal",
	Usage: "Fetch the detail of the inflight new checkpoint proposal",
	Description: `
Get detailed data of the new checkpoint proposal currently in progress, 
including the trusted signer address who has been approved and the corresponding 
checkpoint hash
`,
	Flags: []cli.Flag{
		clientURLFlag,
	},
	Action: utils.MigrateFlags(queryProposal),
}

// queryAdmin fetches the admin list of specified registrar contract.
func queryAdmin(ctx *cli.Context) error {
	contract := setupContract(setupDialContext(ctx))
	adminList, err := contract.Contract().GetAllAdmin(nil)
	if err != nil {
		return err
	}
	fmt.Println("Total admin number", len(adminList))
	for i, admin := range adminList {
		fmt.Printf("Admin %d => %s\n", i+1, admin.Hex())
	}
	return nil
}

// queryCheckpoint fetches the checkpoint hash with specified index from
// registrar contract.
func queryCheckpoint(ctx *cli.Context) error {
	contract := setupContract(setupDialContext(ctx))
	if ctx.GlobalIsSet(checkpointIndexFlag.Name) {
		index := ctx.GlobalInt64(checkpointIndexFlag.Name)
		checkpoint, height, err := contract.Contract().GetCheckpoint(nil, big.NewInt(index))
		if err != nil {
			return err
		}
		fmt.Printf("Checkpoint(registered at height #%d) %d => %s\n", height, index, common.Hash(checkpoint).Hex())
	} else {
		index, checkpoint, height, err := contract.Contract().GetLatestCheckpoint(nil)
		if err != nil {
			return err
		}
		fmt.Printf("Latest checkpoint(registered at height #%d) %d => %s\n", height, index, common.Hash(checkpoint).Hex())
	}
	return nil
}

// queryProposal fetches the detail of inflight new checkpoint proposal
// with specified contract address.
func queryProposal(ctx *cli.Context) error {
	contract := setupContract(setupDialContext(ctx))
	index, addr, hashes, err := contract.Contract().GetPending(nil)
	if err != nil {
		return err
	}
	if len(addr) != len(hashes) {
		return errors.New("trusted signer number is not match with corresponding hash")
	}
	fmt.Printf("Pending checkpoint proposal(index #%d)\n", index)
	for i, a := range addr {
		fmt.Printf("Signer(%s) => checkpoint hash(%s)\n", a.Hex(), common.Hash(hashes[i]).Hex())
	}
	return nil
}
