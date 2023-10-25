package main

import (
	"encoding/json"
	"flag"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func main() {
	keystorePath := flag.String("keystore", "", "Keystore path")
	txnPath := flag.String("txn", "", "Unsigned transaction")
	chainIDArg := flag.String("chainid", "", "Chain ID")
	signer := flag.String("signer", "", "Signer")
	password := flag.String("password", "", "Password")

	flag.Parse()

	ks := keystore.NewKeyStore(*keystorePath, keystore.StandardScryptN, keystore.StandardScryptP)

	account, err := ks.Find(accounts.Account{Address: common.HexToAddress(*signer)})
	if err != nil {
		panic(err)
	}

	txnData, err := os.ReadFile(*txnPath)
	if err != nil {
		panic(err)
	}

	var txn types.Transaction
	if err := txn.UnmarshalJSON(txnData); err != nil {
		panic(err)
	}

	chainID, ok := new(big.Int).SetString(*chainIDArg, 10)
	if !ok {
		panic("invalid chain ID")
	}

	signedTxn, err := ks.SignTxWithPassphrase(account, *password, &txn, chainID)
	if err != nil {
		panic(err)
	}
	txs := []*types.Transaction{signedTxn}
	txsData, err := json.Marshal(txs)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile("txs.json", txsData, 0644)
	if err != nil {
		panic(err)
	}
}
