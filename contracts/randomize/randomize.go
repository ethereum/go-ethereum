// Copyright (c) 2018 XDPoSChain
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

package randomize

import (
	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/contracts/randomize/contract"
)

type Randomize struct {
	*contract.XDCRandomizeSession
	contractBackend bind.ContractBackend
}

func NewRandomize(transactOpts *bind.TransactOpts, contractAddr common.Address, contractBackend bind.ContractBackend) (*Randomize, error) {
	randomize, err := contract.NewXDCRandomize(contractAddr, contractBackend)
	if err != nil {
		return nil, err
	}

	return &Randomize{
		&contract.XDCRandomizeSession{
			Contract:     randomize,
			TransactOpts: *transactOpts,
		},
		contractBackend,
	}, nil
}

func DeployRandomize(transactOpts *bind.TransactOpts, contractBackend bind.ContractBackend) (common.Address, *Randomize, error) {
	randomizeAddr, _, _, err := contract.DeployXDCRandomize(transactOpts, contractBackend)
	if err != nil {
		return randomizeAddr, nil, err
	}

	randomize, err := NewRandomize(transactOpts, randomizeAddr, contractBackend)
	if err != nil {
		return randomizeAddr, nil, err
	}

	return randomizeAddr, randomize, nil
}
