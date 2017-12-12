package ethapi

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/les/status"
	"github.com/ethereum/go-ethereum/log"
)

// StatusBackend exposes Ethereum internals to support custom semantics in status-go bindings
type StatusBackend struct {
	eapi  *PublicEthereumAPI        // Wrapper around the Ethereum object to access metadata
	bcapi *PublicBlockChainAPI      // Wrapper around the blockchain to access chain data
	txapi *PublicTransactionPoolAPI // Wrapper around the transaction pool to access transaction data

	am *status.AccountManager
}

var (
	ErrStatusBackendNotInited = errors.New("StatusIM backend is not properly inited")
)

// NewStatusBackend creates a new backend using an existing Ethereum object.
func NewStatusBackend(apiBackend Backend) *StatusBackend {
	log.Info("StatusIM: backend service inited")
	return &StatusBackend{
		eapi:  NewPublicEthereumAPI(apiBackend),
		bcapi: NewPublicBlockChainAPI(apiBackend),
		txapi: NewPublicTransactionPoolAPI(apiBackend, new(AddrLocker)),
		am:    status.NewAccountManager(apiBackend.AccountManager()),
	}
}

// SetAccountsFilterHandler sets a callback that is triggered when account list is requested
func (b *StatusBackend) SetAccountsFilterHandler(fn status.AccountsFilterHandler) {
	b.am.SetAccountsFilterHandler(fn)
}

// AccountManager returns reference to account manager
func (b *StatusBackend) AccountManager() *status.AccountManager {
	return b.am
}

// SendTransaction wraps call to PublicTransactionPoolAPI.SendTransactionWithPassphrase
func (b *StatusBackend) SendTransaction(ctx context.Context, args status.SendTxArgs, passphrase string) (common.Hash, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if estimatedGas, err := b.EstimateGas(ctx, args); err == nil {
		if estimatedGas.ToInt().Cmp(big.NewInt(defaultGas)) == 1 { // gas > defaultGas
			args.Gas = estimatedGas
		}
	}

	return b.txapi.SendTransactionWithPassphrase(ctx, SendTxArgs(args), passphrase)
}

// EstimateGas uses underlying blockchain API to obtain gas for a given tx arguments
func (b *StatusBackend) EstimateGas(ctx context.Context, args status.SendTxArgs) (*hexutil.Big, error) {
	if args.Gas != nil {
		return args.Gas, nil
	}

	var gasPrice hexutil.Big
	if args.GasPrice != nil {
		gasPrice = *args.GasPrice
	}

	var value hexutil.Big
	if args.Value != nil {
		value = *args.Value
	}

	callArgs := CallArgs{
		From:     args.From,
		To:       args.To,
		GasPrice: gasPrice,
		Value:    value,
		Data:     args.Data,
	}

	return b.bcapi.EstimateGas(ctx, callArgs)
}
