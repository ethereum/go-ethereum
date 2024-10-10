package eip7783

import "math/big"

/*
Implementation of EIP-7783:

def compute_gas_limit(blockNum: int, blockNumStart: int, initialGasLimit: int, r: int, gasLimitCap: int) -> int:

	if blockNum < blockNumStart:
	  return initialGasLimit
	else:
	  return min(gasLimitCap, initialGasLimit + r * (blockNum - blockNumStart))
*/
func CalcGasLimitEIP7783(blockNum, startBlockNum *big.Int, initialGasLimit, gasIncreaseRate, gasLimitCap uint64) uint64 {
	if blockNum.Cmp(startBlockNum) < 0 {
		return initialGasLimit
	} else {
		return min(gasLimitCap, initialGasLimit+gasIncreaseRate*(blockNum.Uint64()-startBlockNum.Uint64()))
	}
}
