package state

import (
	"fmt"
	"math/big"
)

type GasLimitErr struct {
	Message string
	Is, Max *big.Int
}

func IsGasLimitErr(err error) bool {
	_, ok := err.(*GasLimitErr)

	return ok
}
func (err *GasLimitErr) Error() string {
	return err.Message
}
func GasLimitError(is, max *big.Int) *GasLimitErr {
	return &GasLimitErr{Message: fmt.Sprintf("GasLimit error. Max %s, transaction would take it to %s", max, is), Is: is, Max: max}
}
