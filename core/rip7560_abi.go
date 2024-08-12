package core

import (
	"errors"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"strings"
)

type AcceptAccountData struct {
	ValidAfter *big.Int
	ValidUntil *big.Int
}

type AcceptPaymasterData struct {
	ValidAfter *big.Int
	ValidUntil *big.Int
	Context    []byte
}

func abiEncodeValidateTransaction(tx *types.Rip7560AccountAbstractionTx, signingHash common.Hash) ([]byte, error) {
	jsonAbi, err := abi.JSON(strings.NewReader(ValidateTransactionAbi))
	if err != nil {
		return nil, err
	}

	txAbiEncoding, err := tx.AbiEncode()
	validateTransactionData, err := jsonAbi.Pack("validateTransaction", big.NewInt(0), signingHash, txAbiEncoding)
	return validateTransactionData, err
}

func abiEncodeValidatePaymasterTransaction(tx *types.Rip7560AccountAbstractionTx, signingHash common.Hash) ([]byte, error) {
	jsonAbi, err := abi.JSON(strings.NewReader(ValidatePaymasterTransactionAbi))
	txAbiEncoding, err := tx.AbiEncode()
	data, err := jsonAbi.Pack("validatePaymasterTransaction", big.NewInt(0), signingHash, txAbiEncoding)
	return data, err
}

func abiEncodePostPaymasterTransaction(context []byte) ([]byte, error) {
	jsonAbi, err := abi.JSON(strings.NewReader(PostPaymasterTransactionAbi))
	if err != nil {
		return nil, err
	}
	postOpData, err := jsonAbi.Pack("postPaymasterTransaction", true, big.NewInt(0), context)
	return postOpData, err
}

func abiDecodeAcceptAccount(input []byte) (*AcceptAccountData, error) {
	jsonAbi, err := abi.JSON(strings.NewReader(AcceptAccountAbi))
	if err != nil {
		return nil, err
	}
	methodSelector := new(big.Int).SetBytes(input[:4]).Uint64()
	if methodSelector != AcceptAccountMethodSig {
		if methodSelector == SigFailAccountMethodSig {
			return nil, errors.New("account signature error")
		}
		return nil, errors.New("account did not call the EntryPoint 'acceptAccount' callback")
	}
	acceptAccountData := &AcceptAccountData{}
	err = jsonAbi.UnpackIntoInterface(acceptAccountData, "acceptAccount", input[4:])
	return acceptAccountData, err
}

func abiDecodeAcceptPaymaster(input []byte) (*AcceptPaymasterData, error) {
	jsonAbi, err := abi.JSON(strings.NewReader(AcceptPaymasterAbi))
	if err != nil {
		return nil, err
	}
	methodSelector := new(big.Int).SetBytes(input[:4]).Uint64()
	if methodSelector != AcceptPaymasterMethodSig {
		return nil, errors.New("paymaster did not call the EntryPoint 'acceptPaymaster' callback")
	}
	acceptPaymasterData := &AcceptPaymasterData{}
	err = jsonAbi.UnpackIntoInterface(acceptPaymasterData, "acceptPaymaster", input[4:])
	if len(acceptPaymasterData.Context) > PaymasterMaxContextSize {
		return nil, errors.New("paymaster return data: context too large")
	}
	return acceptPaymasterData, err
}
