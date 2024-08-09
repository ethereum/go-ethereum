package core

import (
	"errors"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"strings"
)

const AcceptAccountMethodSig = uint64(0x1256ebd1)   // acceptAccount(uint256,uint256)
const AcceptPaymasterMethodSig = uint64(0x03be8439) // acceptPaymaster(uint256,uint256,bytes)
const SigFailAccountMethodSig = uint64(0x7715fac2)  // sigFailAccount(uint256,uint256)
const PaymasterMaxContextSize = 65536

func abiEncodeValidateTransaction(tx *types.Rip7560AccountAbstractionTx, signingHash common.Hash) ([]byte, error) {
	jsondata := `[
	{"type":"function","name":"validateTransaction","inputs": [{"name": "version","type": "uint256"},{"name": "txHash","type": "bytes32"},{"name": "transaction","type": "bytes"}]}
	]`

	jsonAbi, err := abi.JSON(strings.NewReader(jsondata))
	if err != nil {
		return nil, err
	}

	txAbiEncoding, err := tx.AbiEncode()
	validateTransactionData, err := jsonAbi.Pack("validateTransaction", big.NewInt(0), signingHash, txAbiEncoding)
	return validateTransactionData, err
}

func abiEncodeValidatePaymasterTransaction(tx *types.Rip7560AccountAbstractionTx, signingHash common.Hash) ([]byte, error) {
	jsondata := `[
	{"type":"function","name":"validatePaymasterTransaction","inputs": [{"name": "version","type": "uint256"},{"name": "txHash","type": "bytes32"},{"name": "transaction","type": "bytes"}]}
	]`

	jsonAbi, err := abi.JSON(strings.NewReader(jsondata))
	txAbiEncoding, err := tx.AbiEncode()
	data, err := jsonAbi.Pack("validatePaymasterTransaction", big.NewInt(0), signingHash, txAbiEncoding)
	return data, err
}

func abiEncodePostPaymasterTransaction(context []byte) ([]byte, error) {
	jsondata := `[
			{"type":"function","name":"postPaymasterTransaction","inputs": [{"name": "success","type": "bool"},{"name": "actualGasCost","type": "uint256"},{"name": "context","type": "bytes"}]}
		]`
	jsonAbi, err := abi.JSON(strings.NewReader(jsondata))
	if err != nil {
		return nil, err
	}
	postOpData, err := jsonAbi.Pack("postPaymasterTransaction", true, big.NewInt(0), context)
	return postOpData, err
}

type AcceptAccountData struct {
	ValidAfter *big.Int
	ValidUntil *big.Int
}

type AcceptPaymasterData struct {
	ValidAfter *big.Int
	ValidUntil *big.Int
	Context    []byte
}

func abiDecodeAcceptAccount(input []byte) (*AcceptAccountData, error) {
	// this is not a true ABI of the "acceptAccount" function
	// this ABI swaps inputs and outputs as there is no suitable "abi.decode" function
	jsondata := `[
			{"type":"function","name":"acceptAccount","outputs": [{"name": "validAfter","type": "uint256"},{"name": "validUntil","type": "uint256"}]}
		]`
	jsonAbi, err := abi.JSON(strings.NewReader(jsondata))
	if err != nil {
		return nil, err
	}
	methodSelector := new(big.Int).SetBytes(input[:4]).Uint64()
	if methodSelector != AcceptAccountMethodSig {
		if methodSelector == SigFailAccountMethodSig {
			return nil, errors.New("account signature error")
		}
		return nil, errors.New("account did not return correct MAGIC_VALUE")
	}
	acceptAccountData := &AcceptAccountData{}
	err = jsonAbi.UnpackIntoInterface(acceptAccountData, "acceptAccount", input[4:])
	return acceptAccountData, err
}

func abiDecodeAcceptPaymaster(input []byte) (*AcceptPaymasterData, error) {
	jsondata := `[
			{"type":"function","name":"acceptPaymaster","outputs": [{"name": "validAfter","type": "uint256"},{"name": "validUntil","type": "uint256"},{"name": "context","type": "bytes"}]}
		]`
	jsonAbi, err := abi.JSON(strings.NewReader(jsondata))
	if err != nil {
		return nil, err
	}
	methodSelector := new(big.Int).SetBytes(input[:4]).Uint64()
	if methodSelector != AcceptPaymasterMethodSig {
		return nil, errors.New("paymaster did not return correct MAGIC_VALUE")
	}
	acceptPaymasterData := &AcceptPaymasterData{}
	err = jsonAbi.UnpackIntoInterface(acceptPaymasterData, "acceptPaymaster", input[4:])
	if len(acceptPaymasterData.Context) > PaymasterMaxContextSize {
		return nil, errors.New("paymaster return data: context too large")
	}
	return acceptPaymasterData, err
}
