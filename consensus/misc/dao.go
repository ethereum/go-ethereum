// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package misc

import (
	"bytes"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

var (
	// ErrBadProDAOExtra is returned if a header doesn't support the DAO fork on a
	// pro-fork client.
	ErrBadProDAOExtra = errors.New("bad DAO pro-fork extra-data")

	// ErrBadNoDAOExtra is returned if a header does support the DAO fork on a no-
	// fork client.
	ErrBadNoDAOExtra = errors.New("bad DAO no-fork extra-data")
)

// VerifyDAOHeaderExtraData validates the extra-data field of a block header to
// ensure it conforms to DAO hard-fork rules.
//
// DAO hard-fork extension to the header validity:
//
//   - if the node is no-fork, do not accept blocks in the [fork, fork+10) range
//     with the fork specific extra-data set.
//   - if the node is pro-fork, require blocks in the specific range to have the
//     unique extra-data set.
//
// 블록헤더의 extra-data field 를 검증하여 DAO fork 규칙을 만족하는지 검사하는 함수입니다.
// 만약 현재 node 가 `no-fork` 라면, [fork, fork+10) 범위의 블록을 accept 하지 않습니다.
// 만약 현재 node 가 `pro-fork` 라면, 특정 범위의 블록들이 고유한 `extra-data set`을 갖는 것이 필요합니다.
func VerifyDAOHeaderExtraData(config *params.ChainConfig, header *types.Header) error {
	// Short circuit validation if the node doesn't care about the DAO fork
	if config.DAOForkBlock == nil {
		return nil
	}
	// Make sure the block is within the fork's modified extra-data range
	limit := new(big.Int).Add(config.DAOForkBlock, params.DAOForkExtraRange)
	if header.Number.Cmp(config.DAOForkBlock) < 0 || header.Number.Cmp(limit) >= 0 {
		return nil
	}
	// Depending on whether we support or oppose the fork, validate the extra-data contents
	if config.DAOForkSupport {
		if !bytes.Equal(header.Extra, params.DAOForkBlockExtra) {
			return ErrBadProDAOExtra
		}
	} else {
		if bytes.Equal(header.Extra, params.DAOForkBlockExtra) {
			return ErrBadNoDAOExtra
		}
	}
	// All ok, header has the same extra-data we expect
	return nil
}

// ApplyDAOHardFork modifies the state database according to the DAO hard-fork
// rules, transferring all balances of a set of DAO accounts to a single refund
// contract.
//
// EIP-779, TheDAO hard-fork 에 따라 DB의 상태를 변경하는 함수입니다.
// EIP-779에서도 설명하듯이 DAODrainList 의 계정들로부터 하나의 DAORefundContract 에
// 돈을 전송하게 됩니다.
func ApplyDAOHardFork(statedb *state.StateDB) {
	// Retrieve the contract to refund balances into
	// EIP-779, 돈을 받을 계정이 존재하지 않는다면 새로 하나 생성합니다.
	// 참고로, 계정주소는 "common.HexToAddress("0xbf4ed7b27f1d666546e30d74d50d173d20bca754")" 입니다.
	if !statedb.Exist(params.DAORefundContract) {
		statedb.CreateAccount(params.DAORefundContract)
	}

	// Move every DAO account and extra-balance account funds into the refund contract
	// 모든 `DAODrainList` 의 계정으로부터 `refund contract`에 돈을 보냅니다.
	for _, addr := range params.DAODrainList() {
		statedb.AddBalance(params.DAORefundContract, statedb.GetBalance(addr), tracing.BalanceIncreaseDaoContract)
		statedb.SetBalance(addr, new(uint256.Int), tracing.BalanceDecreaseDaoAccount)
	}
}
