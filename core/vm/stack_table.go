package vm

import (
	"fmt"

	"github.com/ethereum/go-ethereum/params"
)

func makeStackFunc(pop, diff int) stackValidationFunc {
	return func(stack *Stack) error {
		if err := stack.require(pop); err != nil {
			return err
		}

		if int64(stack.len()+diff) > params.StackLimit.Int64() {
			return fmt.Errorf("stack limit reached %d (%d)", stack.len(), params.StackLimit)
		}
		return nil
	}
}

func makeDupStackFunc(n int) stackValidationFunc {
	return makeStackFunc(n, 1)
}

func makeSwapStackFunc(n int) stackValidationFunc {
	return makeStackFunc(n, 0)
}
