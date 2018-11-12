package multisigwallet

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/multisigwallet/contract"
	"math/big"
)

type MultiSigWallet struct {
	*contract.MultiSigWalletSession
	contractBackend bind.ContractBackend
}

func NewMultiSigWallet(transactOpts *bind.TransactOpts, contractAddr common.Address, contractBackend bind.ContractBackend) (*MultiSigWallet, error) {
	blockSigner, err := contract.NewMultiSigWallet(contractAddr, contractBackend)
	if err != nil {
		return nil, err
	}

	return &MultiSigWallet{
		&contract.MultiSigWalletSession{
			Contract:     blockSigner,
			TransactOpts: *transactOpts,
		},
		contractBackend,
	}, nil
}

func DeployMultiSigWallet(transactOpts *bind.TransactOpts, contractBackend bind.ContractBackend, _owners []common.Address, _required *big.Int) (common.Address, *MultiSigWallet, error) {
	blockSignerAddr, _, _, err := contract.DeployMultiSigWallet(transactOpts, contractBackend, _owners, _required)
	if err != nil {
		return blockSignerAddr, nil, err
	}

	blockSigner, err := NewMultiSigWallet(transactOpts, blockSignerAddr, contractBackend)
	if err != nil {
		return blockSignerAddr, nil, err
	}

	return blockSignerAddr, blockSigner, nil
}