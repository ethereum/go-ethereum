package core

import (
	"errors"
	"fmt"
	"math/big"
)

var (
	BlockNumberErr = errors.New("block number invalid")
	BlockFutureErr = errors.New("block time is in the future")
)

// Parent error. In case a parent is unknown this error will be thrown
// by the block manager
type ParentErr struct {
	Message string
}

func (err *ParentErr) Error() string {
	return err.Message
}

func ParentError(hash []byte) error {
	return &ParentErr{Message: fmt.Sprintf("Block's parent unkown %x", hash)}
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
	return &NonceErr{Message: fmt.Sprintf("Nonce err. Is %d, expected %d", is, exp), Is: is, Exp: exp}
}

func IsNonceErr(err error) bool {
	_, ok := err.(*NonceErr)

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
	hash   []byte
}

func (self *KnownBlockError) Error() string {
	return fmt.Sprintf("block %v already known (%x)", self.number, self.hash[0:4])
}
func IsKnownBlockErr(e error) bool {
	_, ok := e.(*KnownBlockError)
	return ok
}
