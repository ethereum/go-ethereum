package health

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/ethclient"
)

func checkBlockNumber(ec *ethclient.Client, blockNumber *big.Int) error {
	_, err := ec.BlockByNumber(context.TODO(), blockNumber)
	if err != nil {
		return fmt.Errorf("no known block with number %v (%x hex)", blockNumber.Int64(), blockNumber.Int64())
	}
	return nil
}
