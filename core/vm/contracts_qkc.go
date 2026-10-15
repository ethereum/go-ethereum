// Copyright 2026 The quarkchain Authors
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

package vm

import (
	"bytes"
	"encoding/binary"
	"errors"
	"maps"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/bitutil"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// PrecompiledContractsQuarkChain contains the precompiled contracts of
// QuarkChain shard chains, enabled on top of the Ethereum ones.
var PrecompiledContractsQuarkChain = PrecompiledContracts{
	params.XShardOutboxAddress: &xShardOutbox{},
}

// PrecompiledAddressesQuarkChain contains the addresses of the QuarkChain
// precompiled contracts.
var PrecompiledAddressesQuarkChain = slices.Collect(maps.Keys(PrecompiledContractsQuarkChain))

// withQuarkChainPrecompiles returns a copy of the given precompiled contracts
// extended with the QuarkChain ones.
func withQuarkChainPrecompiles(contracts PrecompiledContracts) PrecompiledContracts {
	contracts = maps.Clone(contracts)
	maps.Copy(contracts, PrecompiledContractsQuarkChain)
	return contracts
}

// statefulPrecompiledContract is a precompiled contract that needs the context
// of the call into it, e.g. to read the caller or the call value, or to modify
// the state. A CALL runs the contract bound to its context. Any other kind of
// call, or a system call, moves no value into the contract and runs the
// unbound contract.
type statefulPrecompiledContract interface {
	PrecompiledContract

	// convertToStateful returns the contract bound to a CALL from caller into addr. The call
	// value is already transferred to addr when the bound contract runs.
	convertToStateful(evm *EVM, caller, addr common.Address, value *uint256.Int) PrecompiledContract
}

var (
	errXShardCallOnly          = errors.New("xshard: outbox only accepts CALL")
	errXShardSenderNotOrigin   = errors.New("xshard: sender is not the transaction origin")
	errXShardInvalidInput      = errors.New("xshard: invalid input")
	errXShardInvalidShard      = errors.New("xshard: invalid destination shard")
	errXShardInvalidGasLimit   = errors.New("xshard: invalid gas limit")
	errXShardInsufficientValue = errors.New("xshard: call value below value plus gas cost")
	errXShardNotMove           = errors.New("xshard: only moves are enabled")
	errXShardInvalidMove       = errors.New("xshard: move with data or a gas limit other than the send gas")
)

var (
	// xShardSendSelector is the selector of the outbox's only method:
	//
	//	send(uint32 dstShard, address to, uint256 value, uint64 gasLimit, bool isMove, bytes data)
	xShardSendSelector = crypto.Keccak256([]byte("send(uint32,address,uint256,uint64,bool,bytes)"))[:4]

	// XShardSentMessageTopic is the topic of the event the outbox emits for
	// every message:
	//
	//	SentMessage(uint32 indexed dstShard, address indexed from, address indexed to,
	//	            uint256 value, uint256 gasPrice, uint64 gasLimit, bool isMove, bytes data)
	XShardSentMessageTopic = crypto.Keccak256Hash([]byte("SentMessage(uint32,address,address,uint256,uint256,uint64,bool,bytes)"))
)

// xShardMessage is a cross-shard message, as given to the outbox.
type xShardMessage struct {
	dstShard uint32
	to       common.Address
	value    *uint256.Int
	gasLimit uint64
	isMove   bool // only transfers value to the address, running no code
	data     []byte
}

// decodeXShardSend decodes the ABI encoded input of send. Only the canonical
// encoding is accepted.
func decodeXShardSend(input []byte) (*xShardMessage, error) {
	if len(input) < 4 || !bytes.Equal(input[:4], xShardSendSelector) {
		return nil, errXShardInvalidInput
	}
	// The six head words, then the data length and the data padded to words.
	args := input[4:]
	if len(args) < 7*32 {
		return nil, errXShardInvalidInput
	}
	if bitutil.TestBytes(args[0:28]) || bitutil.TestBytes(args[32:44]) || bitutil.TestBytes(args[96:120]) {
		return nil, errXShardInvalidInput
	}
	if bitutil.TestBytes(args[128:159]) || args[159] > 1 {
		return nil, errXShardInvalidInput
	}
	if offset := new(uint256.Int).SetBytes(args[160:192]); !offset.Eq(uint256.NewInt(6 * 32)) {
		return nil, errXShardInvalidInput
	}
	size := new(uint256.Int).SetBytes(args[192:224])
	if !size.IsUint64() || size.Uint64() > uint64(len(args)) || uint64(len(args)) != 7*32+toWordSize(size.Uint64())*32 {
		return nil, errXShardInvalidInput
	}
	return &xShardMessage{
		dstShard: binary.BigEndian.Uint32(args[28:32]),
		to:       common.BytesToAddress(args[44:64]),
		value:    new(uint256.Int).SetBytes(args[64:96]),
		gasLimit: binary.BigEndian.Uint64(args[120:128]),
		isMove:   args[159] == 1,
		data:     common.CopyBytes(args[224 : 224+size.Uint64()]),
	}, nil
}

// xShardSentMessageLog returns the SentMessage event of a message.
func xShardSentMessageLog(addr, from common.Address, msg *xShardMessage, gasPrice *uint256.Int) *types.Log {
	var dstShard, gasLimit, isMove, offset, size common.Hash
	binary.BigEndian.PutUint32(dstShard[28:], msg.dstShard)
	binary.BigEndian.PutUint64(gasLimit[24:], msg.gasLimit)
	if msg.isMove {
		isMove[31] = 1
	}
	offset[31] = 5 * 32
	binary.BigEndian.PutUint64(size[24:], uint64(len(msg.data)))

	value, price := msg.value.Bytes32(), gasPrice.Bytes32()
	data := make([]byte, 0, 6*32+toWordSize(uint64(len(msg.data)))*32)
	data = append(data, value[:]...)
	data = append(data, price[:]...)
	data = append(data, gasLimit[:]...)
	data = append(data, isMove[:]...)
	data = append(data, offset[:]...)
	data = append(data, size[:]...)
	data = append(data, common.RightPadBytes(msg.data, int(toWordSize(uint64(len(msg.data)))*32))...)

	return &types.Log{
		Address: addr,
		Topics: []common.Hash{
			XShardSentMessageTopic,
			dstShard,
			common.BytesToHash(from.Bytes()),
			common.BytesToHash(msg.to.Bytes()),
		},
		Data: data,
	}
}

// xShardOutbox is the source side of cross-shard messaging. It takes a message
// from an EOA, burns the value and the gas the destination may use, and emits
// the message as an event, for the shard's CL to relay to the destination.
type xShardOutbox struct{}

func (c *xShardOutbox) RequiredGas(input []byte) uint64 {
	// Only the event is paid for here, priced as LOG4 with the input bounding
	// its data. The message itself is paid for on the destination shard.
	return params.LogGas + 4*params.LogTopicGas + uint64(len(input))*params.LogDataGas
}

func (c *xShardOutbox) Run(input []byte) ([]byte, error) {
	return nil, errXShardCallOnly
}

func (c *xShardOutbox) Name() string {
	return "XSHARD_OUTBOX"
}

func (c *xShardOutbox) convertToStateful(evm *EVM, caller, addr common.Address, value *uint256.Int) PrecompiledContract {
	return &xShardOutboxCall{evm: evm, caller: caller, addr: addr, value: value}
}

// xShardOutboxCall is the outbox bound to a CALL into it.
type xShardOutboxCall struct {
	xShardOutbox
	evm    *EVM
	caller common.Address
	addr   common.Address // The address of the precompile
	value  *uint256.Int
}

func (c *xShardOutboxCall) Run(input []byte) ([]byte, error) {
	evm := c.evm
	if evm.readOnly {
		return nil, ErrWriteProtection
	}
	// The destination credits the message to the sender's address, so only an
	// EOA, which controls its address on every shard, may send.
	if c.caller != evm.Origin {
		return nil, errXShardSenderNotOrigin
	}
	toMsg, err := decodeXShardSend(input)
	if err != nil {
		return nil, err
	}
	if !evm.chainConfig.QuarkChain.IsXShardDestination(toMsg.dstShard) {
		return nil, errXShardInvalidShard
	}
	// The destination charges XShardSendGas out of the gas limit.
	if toMsg.gasLimit < params.XShardSendGas || toMsg.gasLimit > params.MaxCrossShardTxGas {
		return nil, errXShardInvalidGasLimit
	}
	// Only moves are enabled for now. A move only transfers the value to the
	// address on the destination, which runs no code and charges nothing but
	// XShardSendGas for it.
	if !toMsg.isMove {
		return nil, errXShardNotMove
	}
	if len(toMsg.data) != 0 || toMsg.gasLimit != params.XShardSendGas {
		return nil, errXShardInvalidMove
	}
	// The destination runs the message at the gas price of this transaction,
	// so the sender pays for the value and all of the gas up front.
	gasPrice := evm.GasPrice
	cost, overflow := new(uint256.Int).MulOverflow(uint256.NewInt(toMsg.gasLimit), gasPrice)
	if !overflow {
		_, overflow = cost.AddOverflow(cost, toMsg.value)
	}
	if overflow || cost.Gt(c.value) {
		return nil, errXShardInsufficientValue
	}
	// Burn the cost transferred to the precompile and return the rest of the call value.
	evm.StateDB.SubBalance(c.addr, cost, tracing.BalanceDecreaseXShardSend)
	if refund := new(uint256.Int).Sub(c.value, cost); !refund.IsZero() {
		evm.Context.Transfer(evm.StateDB, c.addr, c.caller, refund, &evm.chainRules)
	}
	evm.StateDB.AddLog(xShardSentMessageLog(c.addr, c.caller, toMsg, gasPrice))
	return nil, nil
}
