package blocksigner

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/blocksigner/contract"
)

type BlockSigner struct {
	*contract.BlockSignerSession
	contractBackend bind.ContractBackend
}

func NewBlockSigner(transactOpts *bind.TransactOpts, contractAddr common.Address, contractBackend bind.ContractBackend) (*BlockSigner, error) {
	blockSigner, err := contract.NewBlockSigner(contractAddr, contractBackend)
	if err != nil {
		return nil, err
	}

	return &BlockSigner{
		&contract.BlockSignerSession{
			Contract:     blockSigner,
			TransactOpts: *transactOpts,
		},
		contractBackend,
	}, nil
}

func DeployBlockSigner(transactOpts *bind.TransactOpts, contractBackend bind.ContractBackend) (common.Address, *BlockSigner, error) {
	blockSignerAddr, _, _, err := contract.DeployBlockSigner(transactOpts, contractBackend)
	if err != nil {
		return blockSignerAddr, nil, err
	}

	blockSigner, err := NewBlockSigner(transactOpts, blockSignerAddr, contractBackend)
	if err != nil {
		return blockSignerAddr, nil, err
	}

	return blockSignerAddr, blockSigner, nil
}
