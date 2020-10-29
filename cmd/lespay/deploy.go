// Copyright 2020 The go-ethereum Authors
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

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/external"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/lotterybook/contract"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"gopkg.in/urfave/cli.v1"
)

var commandDeploy = cli.Command{
	Name:  "deploy",
	Usage: "Deploy a new lottery contract",
	Flags: []cli.Flag{
		nodeURLFlag,
		clefURLFlag,
		signerFlag,
	},
	Action: utils.MigrateFlags(deploy),
}

// deploy deploys the lottery contract for les payment.
//
// Note the network where the contract is deployed depends on
// the network where the connected node is located.
func deploy(ctx *cli.Context) error {
	// Setup clef signer, create an abigen transactor
	clef, err := external.NewExternalSigner(ctx.String(clefURLFlag.Name))
	if err != nil {
		utils.Fatalf("Failed to create clef signer %v", err)
	}
	transactor := bind.NewClefTransactor(clef, accounts.Account{Address: common.HexToAddress(ctx.String(signerFlag.Name))})
	// Setup ethereum client for relaying transaction.
	client, err := ethclient.Dial(ctx.GlobalString(nodeURLFlag.Name))
	if err != nil {
		utils.Fatalf("Failed to connect to Ethereum node: %v", err)
	}
	// Deploy the lottery contract
	fmt.Println("Sending deploy request to Clef...")
	addr, tx, _, err := contract.DeployLotteryBook(transactor, client)
	if err != nil {
		utils.Fatalf("Failed to deploy lottery contract %v", err)
	}
	log.Info("Deployed lottery contract", "address", addr, "tx", tx.Hash().Hex())
	return nil
}
