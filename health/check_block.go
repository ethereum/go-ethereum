package health

import (
	"context"
	"math/big"
)

// checkBlockNumber confirms this node is aware of a specific block.
func checkBlockNumber(ec ethClient, blockNumber *big.Int) error {
	_, err := ec.BlockByNumber(context.TODO(), blockNumber)
	if err != nil {
		return err
	}
	return nil
}
