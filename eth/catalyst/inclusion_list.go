// Copyright 2026 The go-ethereum Authors
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

package catalyst

import (
	"bytes"
	"context"
	"errors"
	"slices"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/log"
)

// inclusionListCacheLimit is the number of payloads whose inclusion list is
// kept, so a later engine call can still give the verdict on them.
const inclusionListCacheLimit = 128

// maxInclusionListBytes is MAX_TRANSACTIONS_BYTES_PER_INCLUSION_LIST, the size
// limit of the transactions on an inclusion list.
const maxInclusionListBytes = 8192

// retainedInclusionList is the inclusion list a payload was handed with, and
// the verdict on it once the payload was found valid.
type retainedInclusionList struct {
	txs       [][]byte
	satisfied *bool
}

// retainInclusionList stores the inclusion list of a payload. A payload sent
// again with the same list keeps its verdict.
func (api *ConsensusAPI) retainInclusionList(hash common.Hash, list []hexutil.Bytes) {
	txs := make([][]byte, len(list))
	for i, tx := range list {
		txs[i] = tx
	}
	if prev, ok := api.inclusionLists.Get(hash); ok && slices.EqualFunc(prev.txs, txs, bytes.Equal) {
		return
	}
	api.inclusionLists.Add(hash, &retainedInclusionList{txs: txs})
}

// inclusionListStatus adds the inclusion list verdict to the status of the
// payload with the given hash. The verdict is only given for valid payloads
// whose inclusion list is known.
func (api *ConsensusAPI) inclusionListStatus(status engine.PayloadStatusV1, hash common.Hash) engine.PayloadStatusV2 {
	res := engine.PayloadStatusV2{PayloadStatusV1: status}
	if status.Status != engine.VALID {
		return res
	}
	list, ok := api.inclusionLists.Get(hash)
	if !ok {
		return res
	}
	if list.satisfied == nil {
		satisfied, err := api.checkInclusionList(hash, list.txs)
		if err != nil {
			log.Warn("Failed to check inclusion list", "hash", hash, "err", err)
			return res
		}
		list = &retainedInclusionList{txs: list.txs, satisfied: &satisfied}
		api.inclusionLists.Add(hash, list)
	}
	res.InclusionListSatisfied = list.satisfied
	return res
}

// checkInclusionList reports whether the block with the given hash satisfies
// the inclusion list.
func (api *ConsensusAPI) checkInclusionList(hash common.Hash, txs [][]byte) (bool, error) {
	chain := api.eth.BlockChain()
	block := chain.GetBlockByHash(hash)
	if block == nil {
		return false, errors.New("unknown block")
	}
	execution, state, err := chain.BlockGasUsed(block)
	if err != nil {
		return false, err
	}
	return chain.CheckInclusionList(block, execution, state, txs)
}

// GetInclusionListV1 returns transactions from the local pool for an inclusion
// list (EIP-7805). It takes the next transaction of each account that can pay
// the next block's base fee, longest waiting first, up to the size limit. A later
// transaction of the same account can only be included after the next one, so
// it would take up room without adding anything.
func (api *ConsensusAPI) GetInclusionListV1() ([]hexutil.Bytes, error) {
	// Blob transactions can't go on an inclusion list, the pool leaves them out
	// by default.
	var (
		baseFee    = api.eth.APIBackend.BaseFee(context.Background())
		pending, _ = api.eth.TxPool().Pending(txpool.PendingFilter{})
		next       = make([]*txpool.LazyTransaction, 0, len(pending))
	)
	for _, txs := range pending {
		if baseFee != nil && txs[0].GasFeeCap.CmpBig(baseFee) < 0 {
			continue
		}
		next = append(next, txs[0])
	}
	slices.SortFunc(next, func(a, b *txpool.LazyTransaction) int {
		return a.Time.Compare(b.Time)
	})
	var (
		list = make([]hexutil.Bytes, 0)
		size int
	)
	for _, lazy := range next {
		tx := lazy.Resolve()
		if tx == nil {
			continue
		}
		enc, err := tx.MarshalBinary()
		if err != nil || size+len(enc) > maxInclusionListBytes {
			continue
		}
		list = append(list, enc)
		size += len(enc)
	}
	return list, nil
}
