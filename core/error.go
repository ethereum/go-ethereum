// Copyright 2014 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

var (
	BlockNumberErr  = errors.New("block number invalid")
	BlockFutureErr  = errors.New("block time is in the future")
	BlockEqualTSErr = errors.New("block time stamp equal to previous")
)

// Parent error. In case a parent is unknown this error will be thrown
// by the block manager
type ParentErr struct {
	Message string
}

func (err *ParentErr) Error() string {
	return err.Message
}

func ParentError(hash common.Hash) error {
	return &ParentErr{Message: fmt.Sprintf("Block's parent unknown %x", hash)}
}

func IsParentErr(err error) bool {
	_, ok := err.(*ParentErr)
	return ok
}

type UncleErr struct {
	Message string
}

func (err *UncleErr) Error() string {
	return err.Message
}

func UncleError(format string, v ...interface{}) error {
	return &UncleErr{Message: fmt.Sprintf(format, v...)}
}

func IsUncleErr(err error) bool {
	_, ok := err.(*UncleErr)
	return ok
}

// Block validation error. If any validation fails, this error will be thrown
type ValidationErr struct {
	Message string
}

func (err *ValidationErr) Error() string {
	return err.Message
}

func ValidationError(format string, v ...interface{}) *ValidationErr {
	return &ValidationErr{Message: fmt.Sprintf(format, v...)}
}

func IsValidationErr(err error) bool {
	_, ok := err.(*ValidationErr)
	return ok
}

type NonceErr struct {
	Message string
	Is, Exp uint64
}

func (err *NonceErr) Error() string {
	return err.Message
}

func NonceError(is, exp uint64) *NonceErr {
	return &NonceErr{Message: fmt.Sprintf("Transaction w/ invalid nonce. tx=%d  state=%d)", is, exp), Is: is, Exp: exp}
}

func IsNonceErr(err error) bool {
	_, ok := err.(*NonceErr)
	return ok
}

// BlockNonceErr indicates that a block's nonce is invalid.
type BlockNonceErr struct {
	Number *big.Int
	Hash   common.Hash
	Nonce  uint64
}

func (err *BlockNonceErr) Error() string {
	return fmt.Sprintf("block %d (%v) nonce is invalid (got %d)", err.Number, err.Hash, err.Nonce)
}

// IsBlockNonceErr returns true for invalid block nonce errors.
func IsBlockNonceErr(err error) bool {
	_, ok := err.(*BlockNonceErr)
	return ok
}

type InvalidTxErr struct {
	Message string
}

func (err *InvalidTxErr) Error() string {
	return err.Message
}

func InvalidTxError(err error) *InvalidTxErr {
	return &InvalidTxErr{fmt.Sprintf("%v", err)}
}

func IsInvalidTxErr(err error) bool {
	_, ok := err.(*InvalidTxErr)
	return ok
}

type TDError struct {
	a, b *big.Int
}

func (self *TDError) Error() string {
	return fmt.Sprintf("incoming chain has a lower or equal TD (%v <= %v)", self.a, self.b)
}
func IsTDError(e error) bool {
	_, ok := e.(*TDError)
	return ok
}

type KnownBlockError struct {
	number *big.Int
	hash   common.Hash
}

func (self *KnownBlockError) Error() string {
	return fmt.Sprintf("block %v already known (%x)", self.number, self.hash[0:4])
}
func IsKnownBlockErr(e error) bool {
	_, ok := e.(*KnownBlockError)
	return ok
}

type ValueTransferError struct {
	message string
}

func ValueTransferErr(str string, v ...interface{}) *ValueTransferError {
	return &ValueTransferError{fmt.Sprintf(str, v...)}
}

func (self *ValueTransferError) Error() string {
	return self.message
}
func IsValueTransferErr(e error) bool {
	_, ok := e.(*ValueTransferError)
	return ok
}
