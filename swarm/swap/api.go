// Copyright 2018 The go-ethereum Authors
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

package swap

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Wrapper for receiving pss messages when using the pss API
// providing access to sender of message
type APIMsg struct {
	Msg hexutil.Bytes
}

// Additional public methods accessible through API for pss
type API struct {
	*SwapProtocol
}

type SwapMetrics struct {
}

type Cheque struct {
}

func NewAPI(swap *SwapProtocol) *API {
	return &API{SwapProtocol: swap}
}

func (swapapi *API) Balance(ctx context.Context) (balance int, err error) {
	balance = 0
	err = nil
	return
}

func (swapapi *API) GetSwapMetrics() (*SwapMetrics, error) {
	return nil, nil
}

func (swapapi *API) IssueCheque(recipient *common.Address) (*Cheque, error) {
	return nil, nil
}

func (swapapi *API) RedeemCheque(cheque *Cheque) {
}
