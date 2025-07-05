// Copyright 2025 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package miner

import (
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/holiman/uint256"
)

type BlobPrediction struct {
	id          engine.PredictionID // prediction is identified by its id
	transaction []*types.Transaction
	lock        sync.Mutex
	stop        chan struct{} // closed before resolving
}

func (prediction *BlobPrediction) Convert() []*engine.BlobPredictionToStage {
	prediction.lock.Lock()
	defer prediction.lock.Unlock()

	var res []*engine.BlobPredictionToStage
	for _, tx := range prediction.transaction {
		for blobIdx, blob := range tx.BlobTxSidecar().Blobs {
			r := engine.BlobPredictionToStage{
				TxHash: tx.Hash(),
			}
			r.Blob = blob[:]
			r.BlobIndex = uint(blobIdx)
			r.KzgCommitment = tx.BlobTxSidecar().Commitments[blobIdx][:]
			proofs := tx.BlobTxSidecar().CellProofsAt(blobIdx)
			for _, proof := range proofs {
				r.CellProofs = append(r.CellProofs, proof[:])
			}
			res = append(res, &r)
		}
	}

	return res
}

// Refer to fillTransaction
// parent:
// len: maximum # of blobs
// header: For gas fee calculation
// [Main Difference from fillTransactions]
// - No prio/normal transaction
func (miner *Miner) fillBlobs(blobId engine.PredictionID, max uint8, W uint8, timestamp uint64) ([]*types.Transaction, error) {
	miner.confMu.RLock()
	tip := miner.config.GasPrice
	miner.confMu.RUnlock()

	// Refer to `prepareWork`
	// We don't need to reuse prepareWork itself as we need only gas-related fields
	parent := miner.chain.CurrentBlock()

	// Assume that other miners also have similar tip(minimum gas price) value
	filter := txpool.PendingFilter{
		MinTip: uint256.MustFromBig(tip),
	}
	// Predict base fee under the assumption that market will go down as much as possible
	predictedBaseFee := eip1559.CalcBaseFee(miner.chainConfig, parent)
	multiplier := new(big.Int).Exp(big.NewInt(875), new(big.Int).SetUint64(uint64(W)), nil)
	divisor := new(big.Int).Exp(big.NewInt(1000), new(big.Int).SetUint64(uint64(W)), nil)

	predictedBaseFee.Mul(predictedBaseFee, multiplier)
	predictedBaseFee.Div(predictedBaseFee, divisor)
	if predictedBaseFee.Cmp(new(big.Int).SetUint64(1)) <= 0 {
		predictedBaseFee = new(big.Int).SetUint64(1)
	}
	// Predict blob fee under the assumption that market will go down as much as possible
	// todo(healthykim): EIP-7918
	excessBlobGas := eip4844.CalcExcessBlobGas(miner.chainConfig, parent, timestamp)
	// CalcBlobFeeWithoutHeader is new function to calculate blob fee without loading all header info
	predictedBlobFee := eip4844.CalcBlobFeeWithoutHeader(miner.chainConfig, excessBlobGas, timestamp)

	exp := new(big.Int).SetUint64(uint64(W))
	multiplier = new(big.Int).Exp(big.NewInt(1000), exp, nil)
	divisor = new(big.Int).Exp(big.NewInt(1125), exp, nil)

	predictedBlobFee = new(big.Int).Mul(predictedBlobFee, multiplier)
	predictedBlobFee.Div(predictedBlobFee, divisor)
	if predictedBlobFee.Cmp(new(big.Int).SetUint64(1)) <= 0 {
		predictedBlobFee = new(big.Int).SetUint64(1)
	}

	// Refer to fillTransaction
	filter.BaseFee = uint256.MustFromBig(predictedBaseFee)
	filter.BlobFee = uint256.MustFromBig(predictedBlobFee)
	filter.OnlyPlainTxs, filter.OnlyBlobTxs = false, true
	pendingBlobTxs := miner.txpool.Pending(filter)

	// Fill the block with all available pending transactions.
	res := make([]*types.Transaction, 0, max)

	if len(pendingBlobTxs) > 0 {
		// Refer to newTransactionsByPriceAndNonce
		signer := types.MakeSigner(miner.chainConfig, parent.Number, parent.Time)
		blobTxs := newTransactionsByPriceAndNonce(signer, pendingBlobTxs, predictedBaseFee)
		for {
			bltx, _ := blobTxs.Peek()
			// Resolve to check if we have full blob data
			// Can be changed
			btx := bltx.Resolve()

			if btx != nil {
				res = append(res, btx)
			}

			blobTxs.Pop()

			if len(res) >= int(max) || len(blobTxs.heads) == 0 {
				break
			}
		}
	}
	return res, nil
}

func (miner *Miner) predictBlobs(blobId engine.PredictionID, max uint8, W uint8, timestamp uint64) (*BlobPrediction, error) {
	prediction := &BlobPrediction{
		id:          blobId,
		transaction: make([]*types.Transaction, 0, max),
		stop:        make(chan struct{}),
	}

	go func() {
		timer := time.NewTimer(0)
		defer timer.Stop()

		// Prediction environment changes for every new slot
		endTimer := time.NewTimer(time.Second * 12)

		for {
			select {
			case <-timer.C:
				res, err := miner.fillBlobs(blobId, max, W, timestamp)
				if err == nil {
					// update prediction
					prediction.lock.Lock()
					prediction.transaction = res
					prediction.lock.Unlock()
				} else {
					log.Error("Error while generating prediction", "id", prediction.id, "err", err)
				}
				timer.Reset(miner.config.Recommit)
			case <-prediction.stop: //todo(healthykim) where
				log.Info("Stopping work on prediction", "id", prediction.id, "reason", "delivery")
				return
			case <-endTimer.C:
				log.Info("Stopping work on prediction", "id", prediction.id, "reason", "timeout")
				return
			}
		}
	}()

	return prediction, nil
}
