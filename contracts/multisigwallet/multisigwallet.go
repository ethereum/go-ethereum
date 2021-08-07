// Copyright (c) 2018 XDCchain
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

package multisigwallet

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/multisigwallet/contract"
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
