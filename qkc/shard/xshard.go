// Copyright 2026-2027 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The QuarkChain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The QuarkChain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the QuarkChain library. If not, see <http://www.gnu.org/licenses/>.

package shard

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/bitutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

var errInvalidSentMessage = errors.New("invalid SentMessage event")

// XShardDeposits converts the cross-shard messages sent by a block of the chain,
// the SentMessage events of the outbox in its receipts, into deposits grouped
// by destination shard. Every other shard of the network gets a list, empty if
// the block sends it nothing, so that it knows the block has no deposits for it.
//
// The receipts are read from the chain database, so the block must already be
// inserted into the chain.
func XShardDeposits(chain *core.BlockChain, blockHash common.Hash) (map[uint32][]*types.DepositTx, error) {
	config := chain.Config().QuarkChain
	if config == nil {
		return nil, errors.New("not a QuarkChain shard chain")
	}
	header := chain.GetHeaderByHash(blockHash)
	if header == nil {
		return nil, fmt.Errorf("unknown block %x", blockHash)
	}
	receipts := chain.GetReceiptsByHash(blockHash)
	if receipts == nil && header.TxHash != types.EmptyTxsHash {
		return nil, fmt.Errorf("missing receipts of block %x", blockHash)
	}
	var logs []*types.Log
	for _, receipt := range receipts {
		logs = append(logs, receipt.Logs...)
	}
	return depositsFromLogs(config, blockHash, logs)
}

// depositsFromLogs converts the SentMessage events of the outbox among the logs
// of a block into deposits grouped by destination shard, as XShardDeposits.
func depositsFromLogs(config *params.QuarkChainConfig, blockHash common.Hash, logs []*types.Log) (map[uint32][]*types.DepositTx, error) {
	deposits := make(map[uint32][]*types.DepositTx)
	for _, shard := range config.ShardIDs {
		if shard != config.ShardID {
			deposits[shard] = []*types.DepositTx{}
		}
	}
	for _, log := range logs {
		if log.Address != params.XShardOutboxAddress || len(log.Topics) == 0 || log.Topics[0] != vm.XShardSentMessageTopic {
			continue
		}
		dstShard, deposit, err := unpackSentMessage(log)
		if err != nil {
			return nil, fmt.Errorf("log %d: %w", log.Index, err)
		}
		list, ok := deposits[dstShard]
		if !ok {
			return nil, fmt.Errorf("log %d: invalid destination shard %d", log.Index, dstShard)
		}
		deposit.SourceBlockHash, deposit.SourceLogIndex = blockHash, uint64(log.Index)
		deposits[dstShard] = append(list, deposit)
	}
	return deposits, nil
}

// unpackSentMessage decodes a SentMessage event of the outbox into a deposit,
// without its source, and the shard it goes to. Only the encoding the
// outbox emits is accepted:
//
//	topics: SentMessage, dstShard, from, to
//	data:   value, gasPrice, gasLimit, isMove, data offset (0xa0), data length, data padded to words
func unpackSentMessage(log *types.Log) (uint32, *types.DepositTx, error) {
	if len(log.Topics) != 4 {
		return 0, nil, errInvalidSentMessage
	}
	dstShard, from, toTopic := log.Topics[1], log.Topics[2], log.Topics[3]
	if bitutil.TestBytes(dstShard[:28]) || bitutil.TestBytes(from[:12]) || bitutil.TestBytes(toTopic[:12]) {
		return 0, nil, errInvalidSentMessage
	}
	data := log.Data
	if len(data) < 6*32 || bitutil.TestBytes(data[64:88]) || bitutil.TestBytes(data[96:127]) || data[127] > 1 {
		return 0, nil, errInvalidSentMessage
	}
	if offset := new(uint256.Int).SetBytes(data[128:160]); !offset.Eq(uint256.NewInt(5 * 32)) {
		return 0, nil, errInvalidSentMessage
	}
	// Bound the length by the data first, so that padding it cannot overflow.
	size := new(uint256.Int).SetBytes(data[160:192])
	if !size.IsUint64() || size.Uint64() > uint64(len(data)) || uint64(len(data)) != 6*32+(size.Uint64()+31)/32*32 {
		return 0, nil, errInvalidSentMessage
	}
	to := common.BytesToAddress(toTopic[12:])
	return binary.BigEndian.Uint32(dstShard[28:]), &types.DepositTx{
		Version:  types.DepositTxVersion,
		From:     common.BytesToAddress(from[12:]),
		To:       &to,
		Value:    new(uint256.Int).SetBytes(data[0:32]),
		GasPrice: new(uint256.Int).SetBytes(data[32:64]),
		Gas:      binary.BigEndian.Uint64(data[88:96]),
		IsMove:   data[127] == 1,
		Data:     common.CopyBytes(data[192 : 192+size.Uint64()]),
	}, nil
}
