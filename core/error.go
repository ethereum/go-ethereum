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

func UncleError(str string) error {
	return &UncleErr{Message: str}
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

type OutOfGasErr struct {
	Message string
}

func OutOfGasError() *OutOfGasErr {
	return &OutOfGasErr{Message: "Out of gas"}
}
func (self *OutOfGasErr) Error() string {
	return self.Message
}

func IsOutOfGasErr(err error) bool {
	_, ok := err.(*OutOfGasErr)

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
