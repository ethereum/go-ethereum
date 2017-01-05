package vm

import (
	"fmt"

	"github.com/ethereum/go-ethereum/params"
)

func makeStackFunc(pop, push int) stackValidationFunc {
	return func(stack *Stack) error {
		if err := stack.require(pop); err != nil {
			return err
		}

		if push > 0 && int64(stack.len()-pop+push) > params.StackLimit.Int64() {
			return fmt.Errorf("stack limit reached %d (%d)", stack.len(), params.StackLimit.Int64())
		}
		return nil
	}
}
