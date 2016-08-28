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

package chequebook

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

const Version = "1.0"

var errNoChequebook = errors.New("no chequebook")

type Api struct {
	chequebookf func() *Chequebook
}

func NewApi(ch func() *Chequebook) *Api {
	return &Api{ch}
}

func (self *Api) Balance() (string, error) {
	ch := self.chequebookf()
	if ch == nil {
		return "", errNoChequebook
	}
	return ch.Balance().String(), nil
}

func (self *Api) Issue(beneficiary common.Address, amount *big.Int) (cheque *Cheque, err error) {
	ch := self.chequebookf()
	if ch == nil {
		return nil, errNoChequebook
	}
	return ch.Issue(beneficiary, amount)
}

func (self *Api) Cash(cheque *Cheque) (txhash string, err error) {
	ch := self.chequebookf()
	if ch == nil {
		return "", errNoChequebook
	}
	return ch.Cash(cheque)
}

func (self *Api) Deposit(amount *big.Int) (txhash string, err error) {
	ch := self.chequebookf()
	if ch == nil {
		return "", errNoChequebook
	}
	return ch.Deposit(amount)
}
