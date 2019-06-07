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
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/contracts/registrar"
	"github.com/ethereum/go-ethereum/contracts/registrar/contract"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"gopkg.in/urfave/cli.v1"
)

var commandDeployContract = cli.Command{
	Name:  "deploy",
	Usage: "Deploy a registrar contract with specified trusted signers.",
	Flags: []cli.Flag{
		signerFlag,
		thresholdFlag,
		nodeURLFlag,
		keyFileFlag,
		utils.PasswordFileFlag,
	},
	Action: utils.MigrateFlags(deployContract),
}

var commandSignCheckpoint = cli.Command{
	Name:  "sign",
	Usage: "Sign the checkpoint with the specified key",
	Flags: []cli.Flag{
		nodeURLFlag,
		clefURLFlag,
		indexFlag,
		hashFlag,
		addressFlag,
		keyFileFlag,
		signerFlag,
		utils.PasswordFileFlag,
	},
	Action: utils.MigrateFlags(signCheckpoint),
}

var commandRegisterCheckpoint = cli.Command{
	Name:  "register",
	Usage: "Register specified checkpoint into contract",
	Flags: []cli.Flag{
		nodeURLFlag,
		indexFlag,
		signerFlag,
		signatureFlag,
		keyFileFlag,
		utils.PasswordFileFlag,
	},
	Action: utils.MigrateFlags(registerCheckpoint),
}

// deployContract deploys the checkpoint registrar contract.
//
// Note the network where the contract is deployed depends on
// the network where the connected node is located.
func deployContract(ctx *cli.Context) error {
	var addrs []common.Address
	signers := strings.Split(ctx.GlobalString(signerFlag.Name), ",")
	for _, account := range signers {
		if trimmed := strings.TrimSpace(account); !common.IsHexAddress(trimmed) {
			utils.Fatalf("Invalid account in --signer: %s", trimmed)
		} else {
			addrs = append(addrs, common.HexToAddress(account))
		}
	}

	t := ctx.GlobalInt64(thresholdFlag.Name)
	if t == 0 || int(t) > len(signers) {
		utils.Fatalf("Invalid signature threshold %d", t)
	}
	addr, tx, _, err := contract.DeployContract(bind.NewKeyedTransactor(getKey(ctx).PrivateKey), newClient(ctx), addrs, big.NewInt(int64(params.CheckpointFrequency)),
		big.NewInt(int64(params.CheckpointProcessConfirmations)), big.NewInt(t))
	if err != nil {
		utils.Fatalf("Failed to deploy registrar contract %v", err)
	}
	log.Info("Deployed registrar contract successfully", "address", addr, "tx", tx.Hash())
	return nil
}

// signCheckpoint creates the signature for specific checkpoint
// with local key. Only contract admins have the permission to
// sign checkpoint.
func signCheckpoint(ctx *cli.Context) error {
	var (
		offline bool // The indicator whether we sign checkpoint by offline.
		chash   common.Hash
		cindex  uint64
		address common.Address

		node *rpc.Client
		c    *registrar.Registrar
	)
	if !ctx.GlobalIsSet(nodeURLFlag.Name) {
		// Offline mode signing
		offline = true
		if !ctx.GlobalIsSet(hashFlag.Name) {
			utils.Fatalf("Please specify checkpoint hash for offline mode signing")
		}
		chash = common.HexToHash(ctx.GlobalString(hashFlag.Name))
		if !ctx.GlobalIsSet(indexFlag.Name) {
			utils.Fatalf("Please specify checkpoint index for offline mode signing")
		}
		cindex = ctx.GlobalUint64(indexFlag.Name)
		if !ctx.GlobalIsSet(addressFlag.Name) {
			utils.Fatalf("Please specify contract address for offline mode signing")
		}
		address = common.HexToAddress(ctx.GlobalString(addressFlag.Name))
	} else {
		// Interactive mode signing
		node = newRPCClient(ctx.GlobalString(nodeURLFlag.Name))
		checkpoint := getCheckpoint(ctx, node)

		chash = checkpoint.Hash()
		cindex = checkpoint.SectionIndex
		address = getContractAddr(node)

		reqCtx, cancelFn := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelFn()

		// Check the validity of checkpoint.
		head, err := ethclient.NewClient(node).HeaderByNumber(reqCtx, nil)
		if err != nil {
			return err
		}
		num := head.Number.Uint64()
		if num < ((cindex+1)*params.CheckpointFrequency + params.CheckpointProcessConfirmations) {
			utils.Fatalf("Invalid future checkpoint")
		}
		c = newContract(node)
		latest, _, h, err := c.Contract().GetLatestCheckpoint(nil)
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
		signers, err := c.Contract().GetAllAdmin(nil)
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
	switch {
	case ctx.GlobalIsSet(clefURLFlag.Name):
		// Sign checkpoint in clef mode.
		signer = ctx.GlobalString(signerFlag.Name)

		if !offline {
			if err := isAdmin(common.HexToAddress(signer)); err != nil {
				return err
			}
		}
		clef := newRPCClient(ctx.GlobalString(clefURLFlag.Name))
		p := make(map[string]string)
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, cindex)
		p["address"] = address.Hex()
		p["message"] = hexutil.Encode(append(buf, chash.Bytes()...))
		if err := clef.Call(&signature, "account_signData", "data/validator", signer, p); err != nil {
			utils.Fatalf("Failed to sign checkpoint, err %v", err)
		}
	case ctx.GlobalIsSet(keyFileFlag.Name):
		// Sign checkpoint in raw private key file mode.
		key := getKey(ctx)
		signer = key.Address.Hex()

		if !offline {
			if err := isAdmin(key.Address); err != nil {
				return err
			}
		}
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, cindex)
		data := append([]byte{0x19, 0x00}, append(address[:], append(buf, chash.Bytes()...)...)...)
		sig, err := crypto.Sign(crypto.Keccak256(data), key.PrivateKey)
		if err != nil {
			utils.Fatalf("Failed to sign checkpoint, err %v", err)
		}
		sig[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
		signature = common.Bytes2Hex(sig)
	default:
		utils.Fatalf("Please specify clef URL or private key file path to sign checkpoint")
	}
	log.Info("Successfully signed checkpoint", "index", cindex, "hash", chash.Hex(), "address", address.Hex(), "signer", signer, "signature", signature)
	return nil
}

type Signer struct {
	addr common.Address
	sig  []byte
}
type Signers []Signer

func (s Signers) Len() int           { return len(s) }
func (s Signers) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s Signers) Less(i, j int) bool { return bytes.Compare(s[i].addr.Bytes(), s[j].addr.Bytes()) < 0 }

// registerCheckpoint registers the specified checkpoint which generated by connected
// node with a authorised private key.
func registerCheckpoint(ctx *cli.Context) error {
	var (
		addrs   []common.Address
		sigs    [][]byte
		signers Signers
	)
	signerStrs := strings.Split(ctx.GlobalString(signerFlag.Name), ",")
	for _, account := range signerStrs {
		if trimmed := strings.TrimSpace(account); !common.IsHexAddress(trimmed) {
			utils.Fatalf("Invalid account in --signer: %s", trimmed)
		} else {
			addrs = append(addrs, common.HexToAddress(account))
		}
	}
	sigStrs := strings.Split(ctx.GlobalString(signatureFlag.Name), ",")
	for _, sig := range sigStrs {
		trimmed := strings.TrimPrefix(strings.TrimSpace(sig), "0x")
		if len(trimmed) != 130 {
			utils.Fatalf("Invalid signature in --signature: %s", trimmed)
		} else {
			sigs = append(sigs, common.Hex2Bytes(trimmed))
		}
	}
	if len(addrs) != len(sigs) {
		utils.Fatalf("The length of signer and corresponding signature mismatch")
	}
	for i := 0; i < len(addrs); i++ {
		signers = append(signers, Signer{addr: addrs[i], sig: sigs[i]})
	}
	sort.Sort(signers)
	sigs = sigs[:0]
	for i := 0; i < len(signers); i++ {
		sigs = append(sigs, signers[i].sig)
	}

	reqCtx, cancelFn := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFn()

	// Retrieve recent header info to protect replay attack.
	node := newRPCClient(ctx.GlobalString(nodeURLFlag.Name))
	head, err := ethclient.NewClient(node).HeaderByNumber(reqCtx, nil)
	if err != nil {
		return err
	}
	num := head.Number.Uint64()
	recent, err := ethclient.NewClient(node).HeaderByNumber(reqCtx, big.NewInt(int64(num-128)))
	if err != nil {
		return err
	}
	var (
		c          = newContract(node)
		key        = getKey(ctx)
		checkpoint = getCheckpoint(ctx, node)
	)
	tx, err := c.RegisterCheckpoint(key.PrivateKey, checkpoint.SectionIndex, checkpoint.Hash().Bytes(), recent.Number, recent.Hash(), sigs)
	if err != nil {
		utils.Fatalf("Register contract failed %v", err)
	}
	log.Info("Successfully registered checkpoint", "index", checkpoint.SectionIndex, "hash", checkpoint.Hash(), "signumber", len(signers), "txhash", tx.Hash())
	return nil
}
