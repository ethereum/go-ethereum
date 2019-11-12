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
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/maticnetwork/bor/accounts"
	"github.com/maticnetwork/bor/cmd/utils"
	"github.com/maticnetwork/bor/common"
	"github.com/maticnetwork/bor/common/hexutil"
	"github.com/maticnetwork/bor/contracts/checkpointoracle"
	"github.com/maticnetwork/bor/contracts/checkpointoracle/contract"
	"github.com/maticnetwork/bor/crypto"
	"github.com/maticnetwork/bor/ethclient"
	"github.com/maticnetwork/bor/log"
	"github.com/maticnetwork/bor/params"
	"github.com/maticnetwork/bor/rpc"
	"gopkg.in/urfave/cli.v1"
)

var commandDeploy = cli.Command{
	Name:  "deploy",
	Usage: "Deploy a new checkpoint oracle contract",
	Flags: []cli.Flag{
		nodeURLFlag,
		clefURLFlag,
		signerFlag,
		signersFlag,
		thresholdFlag,
	},
	Action: utils.MigrateFlags(deploy),
}

var commandSign = cli.Command{
	Name:  "sign",
	Usage: "Sign the checkpoint with the specified key",
	Flags: []cli.Flag{
		nodeURLFlag,
		clefURLFlag,
		signerFlag,
		indexFlag,
		hashFlag,
		oracleFlag,
	},
	Action: utils.MigrateFlags(sign),
}

var commandPublish = cli.Command{
	Name:  "publish",
	Usage: "Publish a checkpoint into the oracle",
	Flags: []cli.Flag{
		nodeURLFlag,
		clefURLFlag,
		signerFlag,
		indexFlag,
		signaturesFlag,
	},
	Action: utils.MigrateFlags(publish),
}

// deploy deploys the checkpoint registrar contract.
//
// Note the network where the contract is deployed depends on
// the network where the connected node is located.
func deploy(ctx *cli.Context) error {
	// Gather all the addresses that should be permitted to sign
	var addrs []common.Address
	for _, account := range strings.Split(ctx.String(signersFlag.Name), ",") {
		if trimmed := strings.TrimSpace(account); !common.IsHexAddress(trimmed) {
			utils.Fatalf("Invalid account in --signers: '%s'", trimmed)
		}
		addrs = append(addrs, common.HexToAddress(account))
	}
	// Retrieve and validate the signing threshold
	needed := ctx.Int(thresholdFlag.Name)
	if needed == 0 || needed > len(addrs) {
		utils.Fatalf("Invalid signature threshold %d", needed)
	}
	// Print a summary to ensure the user understands what they're signing
	fmt.Printf("Deploying new checkpoint oracle:\n\n")
	for i, addr := range addrs {
		fmt.Printf("Admin %d => %s\n", i+1, addr.Hex())
	}
	fmt.Printf("\nSignatures needed to publish: %d\n", needed)

	// setup clef signer, create an abigen transactor and an RPC client
	transactor, client := newClefSigner(ctx), newClient(ctx)

	// Deploy the checkpoint oracle
	fmt.Println("Sending deploy request to Clef...")
	oracle, tx, _, err := contract.DeployCheckpointOracle(transactor, client, addrs, big.NewInt(int64(params.CheckpointFrequency)),
		big.NewInt(int64(params.CheckpointProcessConfirmations)), big.NewInt(int64(needed)))
	if err != nil {
		utils.Fatalf("Failed to deploy checkpoint oracle %v", err)
	}
	log.Info("Deployed checkpoint oracle", "address", oracle, "tx", tx.Hash().Hex())

	return nil
}

// sign creates the signature for specific checkpoint
// with local key. Only contract admins have the permission to
// sign checkpoint.
func sign(ctx *cli.Context) error {
	var (
		offline bool // The indicator whether we sign checkpoint by offline.
		chash   common.Hash
		cindex  uint64
		address common.Address

		node   *rpc.Client
		oracle *checkpointoracle.CheckpointOracle
	)
	if !ctx.GlobalIsSet(nodeURLFlag.Name) {
		// Offline mode signing
		offline = true
		if !ctx.IsSet(hashFlag.Name) {
			utils.Fatalf("Please specify the checkpoint hash (--hash) to sign in offline mode")
		}
		chash = common.HexToHash(ctx.String(hashFlag.Name))

		if !ctx.IsSet(indexFlag.Name) {
			utils.Fatalf("Please specify checkpoint index (--index) to sign in offline mode")
		}
		cindex = ctx.Uint64(indexFlag.Name)

		if !ctx.IsSet(oracleFlag.Name) {
			utils.Fatalf("Please specify oracle address (--oracle) to sign in offline mode")
		}
		address = common.HexToAddress(ctx.String(oracleFlag.Name))
	} else {
		// Interactive mode signing, retrieve the data from the remote node
		node = newRPCClient(ctx.GlobalString(nodeURLFlag.Name))

		checkpoint := getCheckpoint(ctx, node)
		chash, cindex, address = checkpoint.Hash(), checkpoint.SectionIndex, getContractAddr(node)

		// Check the validity of checkpoint
		reqCtx, cancelFn := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelFn()

		head, err := ethclient.NewClient(node).HeaderByNumber(reqCtx, nil)
		if err != nil {
			return err
		}
		num := head.Number.Uint64()
		if num < ((cindex+1)*params.CheckpointFrequency + params.CheckpointProcessConfirmations) {
			utils.Fatalf("Invalid future checkpoint")
		}
		_, oracle = newContract(node)
		latest, _, h, err := oracle.Contract().GetLatestCheckpoint(nil)
		if err != nil {
			return err
		}
		if cindex < latest {
			utils.Fatalf("Checkpoint is too old")
		}
		if cindex == latest && (latest != 0 || h.Uint64() != 0) {
			utils.Fatalf("Stale checkpoint, latest registered %d, given %d", latest, cindex)
		}
	}
	var (
		signature string
		signer    string
	)
	// isAdmin checks whether the specified signer is admin.
	isAdmin := func(addr common.Address) error {
		signers, err := oracle.Contract().GetAllAdmin(nil)
		if err != nil {
			return err
		}
		for _, s := range signers {
			if s == addr {
				return nil
			}
		}
		return fmt.Errorf("signer %v is not the admin", addr.Hex())
	}
	// Print to the user the data thy are about to sign
	fmt.Printf("Oracle     => %s\n", address.Hex())
	fmt.Printf("Index %4d => %s\n", cindex, chash.Hex())

	// Sign checkpoint in clef mode.
	signer = ctx.String(signerFlag.Name)

	if !offline {
		if err := isAdmin(common.HexToAddress(signer)); err != nil {
			return err
		}
	}
	clef := newRPCClient(ctx.String(clefURLFlag.Name))
	p := make(map[string]string)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, cindex)
	p["address"] = address.Hex()
	p["message"] = hexutil.Encode(append(buf, chash.Bytes()...))

	fmt.Println("Sending signing request to Clef...")
	if err := clef.Call(&signature, "account_signData", accounts.MimetypeDataWithValidator, signer, p); err != nil {
		utils.Fatalf("Failed to sign checkpoint, err %v", err)
	}
	fmt.Printf("Signer     => %s\n", signer)
	fmt.Printf("Signature  => %s\n", signature)
	return nil
}

// sighash calculates the hash of the data to sign for the checkpoint oracle.
func sighash(index uint64, oracle common.Address, hash common.Hash) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, index)

	data := append([]byte{0x19, 0x00}, append(oracle[:], append(buf, hash[:]...)...)...)
	return crypto.Keccak256(data)
}

// ecrecover calculates the sender address from a sighash and signature combo.
func ecrecover(sighash []byte, sig []byte) common.Address {
	sig[64] -= 27
	defer func() { sig[64] += 27 }()

	signer, err := crypto.SigToPub(sighash, sig)
	if err != nil {
		utils.Fatalf("Failed to recover sender from signature %x: %v", sig, err)
	}
	return crypto.PubkeyToAddress(*signer)
}

// publish registers the specified checkpoint which generated by connected node
// with a authorised private key.
func publish(ctx *cli.Context) error {
	// Print the checkpoint oracle's current status to make sure we're interacting
	// with the correct network and contract.
	status(ctx)

	// Gather the signatures from the CLI
	var sigs [][]byte
	for _, sig := range strings.Split(ctx.String(signaturesFlag.Name), ",") {
		trimmed := strings.TrimPrefix(strings.TrimSpace(sig), "0x")
		if len(trimmed) != 130 {
			utils.Fatalf("Invalid signature in --signature: '%s'", trimmed)
		} else {
			sigs = append(sigs, common.Hex2Bytes(trimmed))
		}
	}
	// Retrieve the checkpoint we want to sign to sort the signatures
	var (
		client       = newRPCClient(ctx.GlobalString(nodeURLFlag.Name))
		addr, oracle = newContract(client)
		checkpoint   = getCheckpoint(ctx, client)
		sighash      = sighash(checkpoint.SectionIndex, addr, checkpoint.Hash())
	)
	for i := 0; i < len(sigs); i++ {
		for j := i + 1; j < len(sigs); j++ {
			signerA := ecrecover(sighash, sigs[i])
			signerB := ecrecover(sighash, sigs[j])
			if bytes.Compare(signerA.Bytes(), signerB.Bytes()) > 0 {
				sigs[i], sigs[j] = sigs[j], sigs[i]
			}
		}
	}
	// Retrieve recent header info to protect replay attack
	reqCtx, cancelFn := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFn()

	head, err := ethclient.NewClient(client).HeaderByNumber(reqCtx, nil)
	if err != nil {
		return err
	}
	num := head.Number.Uint64()
	recent, err := ethclient.NewClient(client).HeaderByNumber(reqCtx, big.NewInt(int64(num-128)))
	if err != nil {
		return err
	}
	// Print a summary of the operation that's going to be performed
	fmt.Printf("Publishing %d => %s:\n\n", checkpoint.SectionIndex, checkpoint.Hash().Hex())
	for i, sig := range sigs {
		fmt.Printf("Signer %d => %s\n", i+1, ecrecover(sighash, sig).Hex())
	}
	fmt.Println()
	fmt.Printf("Sentry number => %d\nSentry hash   => %s\n", recent.Number, recent.Hash().Hex())

	// Publish the checkpoint into the oracle
	fmt.Println("Sending publish request to Clef...")
	tx, err := oracle.RegisterCheckpoint(newClefSigner(ctx), checkpoint.SectionIndex, checkpoint.Hash().Bytes(), recent.Number, recent.Hash(), sigs)
	if err != nil {
		utils.Fatalf("Register contract failed %v", err)
	}
	log.Info("Successfully registered checkpoint", "tx", tx.Hash().Hex())
	return nil
}
