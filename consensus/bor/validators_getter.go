package bor

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/bor/valset"
)

//go:generate mockgen -destination=./validators_getter_mock.go -package=bor . ValidatorsGetter
type ValidatorsGetter interface {
	GetCurrentValidators(headerHash common.Hash, blockNumber uint64) ([]*valset.Validator, error)
}
