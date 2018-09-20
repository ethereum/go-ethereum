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
	"fmt"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"gopkg.in/urfave/cli.v1"
)

var commandQueryAdmin = cli.Command{
	Name:  "queryadmin",
	Usage: "Fetch the admin list of specified registrar contract",
	Flags: []cli.Flag{
		nodeURLFlag,
	},
	Action: utils.MigrateFlags(queryAdmin),
}

var commandQueryCheckpoint = cli.Command{
	Name:  "querycheckpoint",
	Usage: "Fetch the latest registered checkpoint in the registrar contract",
	Flags: []cli.Flag{
		nodeURLFlag,
	},
	Action: utils.MigrateFlags(queryCheckpoint),
}

// queryAdmin fetches the admin list of specified registrar contract.
func queryAdmin(ctx *cli.Context) error {
	contract := newContract(newRPCClient(ctx.GlobalString(nodeURLFlag.Name)))
	admins, err := contract.Contract().GetAllAdmin(nil)
	if err != nil {
		return err
	}
	fmt.Println("Total admin number", len(admins))
	for i, admin := range admins {
		fmt.Printf("Admin %d => %s\n", i+1, admin.Hex())
	}
	return nil
}

// queryCheckpoint fetches the checkpoint hash with specified index from
// registrar contract.
func queryCheckpoint(ctx *cli.Context) error {
	contract := newContract(newRPCClient(ctx.GlobalString(nodeURLFlag.Name)))
	index, checkpoint, height, err := contract.Contract().GetLatestCheckpoint(nil)
	if err != nil {
		return err
	}
	fmt.Printf("Latest checkpoint(registered at height #%d) %d => %s\n", height, index, common.Hash(checkpoint).Hex())
	return nil
}
