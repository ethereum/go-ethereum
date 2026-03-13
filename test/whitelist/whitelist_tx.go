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

func main() {
	// Admin private key
	privHex := "badb9f5dec5b628a70ce52d143f5ac75e6ef5fda9afedfdd423bb539552b40cc"
	priv, err := crypto.HexToECDSA(privHex)
	if err != nil {
		log.Fatalf("Failed to parse admin key: %v", err)
	}
	adminAddr := crypto.PubkeyToAddress(priv.PublicKey)

	// Target address to whitelist (node1 - Clique signer)
	targetAddr := common.HexToAddress("0xca6b49ee60cdd276ab503fbd6fb80a3cfbc06ffc")

	client, err := ethclient.Dial("http://localhost:8545")
	if err != nil {
		log.Fatalf("Failed to connect to node: %v", err)
	}

	// Prepare input: mode=1, admin address, target address
	input := []byte{1}
	input = append(input, adminAddr.Bytes()...)
	input = append(input, targetAddr.Bytes()...)

	nonce, err := client.PendingNonceAt(context.Background(), adminAddr)
	if err != nil {
		log.Fatalf("Failed to get nonce: %v", err)
	}
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatalf("Failed to get gas price: %v", err)
	}

	to := common.BytesToAddress([]byte{0x01, 0x00})
	var value = big.NewInt(0)
	var gasLimit uint64 = 100000

	tx := types.NewTransaction(nonce, to, value, gasLimit, gasPrice, input)

	chainID := big.NewInt(1234)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), priv)
	if err != nil {
		log.Fatalf("Failed to sign tx: %v", err)
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Fatalf("Failed to send tx: %v", err)
	}

	log.Printf("Whitelisting transaction sent: %s", signedTx.Hash().Hex())
}
