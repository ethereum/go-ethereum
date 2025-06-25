package export

import (
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/internal/ethapi/override"
)

type RPCTransaction = ethapi.RPCTransaction
type SignTransactionResult = ethapi.SignTransactionResult
type TransactionArgs = ethapi.TransactionArgs
type StateOverride = override.StateOverride
type BlockOverrides = override.BlockOverrides
type ChainContextBackend = ethapi.ChainContextBackend

var NewRPCTransaction = ethapi.NewRPCTransaction
var AccessList = ethapi.AccessList
var DoEstimateGas = ethapi.DoEstimateGas
var DoEstimateGasAfterCalls = ethapi.DoEstimateGasAfterCalls
var DoCall = ethapi.DoCall
var NewRPCPendingTransaction = ethapi.NewRPCPendingTransaction
