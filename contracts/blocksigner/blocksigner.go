// Copyright (c) 2018 Tomochain
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package blocksigner

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/blocksigner/contract"
	"math/big"
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
	blockSignerAddr, _, _, err := contract.DeployBlockSigner(transactOpts, contractBackend, big.NewInt(99))
	if err != nil {
		return blockSignerAddr, nil, err
	}

	blockSigner, err := NewBlockSigner(transactOpts, blockSignerAddr, contractBackend)
	if err != nil {
		return blockSignerAddr, nil, err
	}

	return blockSignerAddr, blockSigner, nil
}
