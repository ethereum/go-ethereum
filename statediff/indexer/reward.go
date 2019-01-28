// VulcanizeDB
// Copyright © 2019 Vulcanize

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.

// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package indexer

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
)

func CalcEthBlockReward(header *types.Header, uncles []*types.Header, txs types.Transactions, receipts types.Receipts) *big.Int {
	staticBlockReward := staticRewardByBlockNumber(header.Number.Uint64())
	transactionFees := calcEthTransactionFees(txs, receipts)
	uncleInclusionRewards := calcEthUncleInclusionRewards(header, uncles)
	tmp := transactionFees.Add(transactionFees, uncleInclusionRewards)
	return tmp.Add(tmp, staticBlockReward)
}

func CalcUncleMinerReward(blockNumber, uncleBlockNumber uint64) *big.Int {
	staticBlockReward := staticRewardByBlockNumber(blockNumber)
	rewardDiv8 := staticBlockReward.Div(staticBlockReward, big.NewInt(8))
	mainBlock := new(big.Int).SetUint64(blockNumber)
	uncleBlock := new(big.Int).SetUint64(uncleBlockNumber)
	uncleBlockPlus8 := uncleBlock.Add(uncleBlock, big.NewInt(8))
	uncleBlockPlus8MinusMainBlock := uncleBlockPlus8.Sub(uncleBlockPlus8, mainBlock)
	return rewardDiv8.Mul(rewardDiv8, uncleBlockPlus8MinusMainBlock)
}

func staticRewardByBlockNumber(blockNumber uint64) *big.Int {
	staticBlockReward := new(big.Int)
	//https://blog.ethereum.org/2017/10/12/byzantium-hf-announcement/
	if blockNumber >= 7280000 {
		staticBlockReward.SetString("2000000000000000000", 10)
	} else if blockNumber >= 4370000 {
		staticBlockReward.SetString("3000000000000000000", 10)
	} else {
		staticBlockReward.SetString("5000000000000000000", 10)
	}
	return staticBlockReward
}

func calcEthTransactionFees(txs types.Transactions, receipts types.Receipts) *big.Int {
	transactionFees := new(big.Int)
	for i, transaction := range txs {
		receipt := receipts[i]
		gasPrice := big.NewInt(transaction.GasPrice().Int64())
		gasUsed := big.NewInt(int64(receipt.GasUsed))
		transactionFee := gasPrice.Mul(gasPrice, gasUsed)
		transactionFees = transactionFees.Add(transactionFees, transactionFee)
	}
	return transactionFees
}

func calcEthUncleInclusionRewards(header *types.Header, uncles []*types.Header) *big.Int {
	uncleInclusionRewards := new(big.Int)
	for range uncles {
		staticBlockReward := staticRewardByBlockNumber(header.Number.Uint64())
		staticBlockReward.Div(staticBlockReward, big.NewInt(32))
		uncleInclusionRewards.Add(uncleInclusionRewards, staticBlockReward)
	}
	return uncleInclusionRewards
}
