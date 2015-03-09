package ui

import "github.com/ethereum/go-ethereum/core/types"

type Interface interface {
	UnlockAccount(address []byte) bool
	ConfirmTransaction(tx *types.Transaction) bool
}
