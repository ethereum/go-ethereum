package main

import (
	"context"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// This helper adds a NEW admin to the whitelist precompile.
// It uses mode = 2 in the precompile:
//
//	mode = 2: add new admin (admin only, target address required)
//
// The caller must be an existing admin (the hard-coded admin key below).
// The new admin in this example is node2's miner address, so node2 can also
// manage the whitelist.
func main() {
	// Existing admin private key (same as used in whitelist_tx.go)
	privHex := "PRIVATE_KEY"
	priv, err := crypto.HexToECDSA(privHex)
	if err != nil {
		log.Fatalf("Failed to parse admin key: %v", err)
	}
	adminAddr := crypto.PubkeyToAddress(priv.PublicKey)

	// New admin address to add (example: node2 miner address)
	newAdmin := common.HexToAddress("0xab52b2c71f61cd9447a932c0cb55d1752571dab8")

	client, err := ethclient.Dial("http://localhost:8545")
	if err != nil {
		log.Fatalf("Failed to connect to node: %v", err)
	}
	defer client.Close()

	// Prepare input: mode=2, current admin address, new admin address
	input := []byte{2}
	input = append(input, adminAddr.Bytes()...)
	input = append(input, newAdmin.Bytes()...)

	nonce, err := client.PendingNonceAt(context.Background(), adminAddr)
	if err != nil {
		log.Fatalf("Failed to get nonce: %v", err)
	}
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatalf("Failed to get gas price: %v", err)
	}

	to := common.BytesToAddress([]byte{0x01, 0x00}) // address of whitelist precompile
	value := big.NewInt(0)
	var gasLimit uint64 = 100000

	tx := types.NewTransaction(nonce, to, value, gasLimit, gasPrice, input)

	chainID := big.NewInt(1234)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), priv)
	if err != nil {
		log.Fatalf("Failed to sign tx: %v", err)
	}

	if err := client.SendTransaction(context.Background(), signedTx); err != nil {
		log.Fatalf("Failed to send tx: %v", err)
	}

	log.Printf("Add-admin transaction sent: %s (new admin: %s)", signedTx.Hash().Hex(), newAdmin.Hex())
}
