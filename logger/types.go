// Copyright 2015 The go-ethereum Authors
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

package logger

import (
	"math/big"
	"time"
)

type utctime8601 struct{}

func (utctime8601) MarshalJSON() ([]byte, error) {
	timestr := time.Now().UTC().Format(time.RFC3339Nano)
	// Bounds check
	if len(timestr) > 26 {
		timestr = timestr[:26]
	}
	return []byte(`"` + timestr + `Z"`), nil
}

type JsonLog interface {
	EventName() string
}

type LogEvent struct {
	// Guid string      `json:"guid"`
	Ts utctime8601 `json:"ts"`
	// Level string      `json:"level"`
}

type LogStarting struct {
	ClientString    string `json:"client_impl"`
	ProtocolVersion int    `json:"eth_version"`
	LogEvent
}

func (l *LogStarting) EventName() string {
	return "starting"
}

type P2PConnected struct {
	RemoteId            string `json:"remote_id"`
	RemoteAddress       string `json:"remote_addr"`
	RemoteVersionString string `json:"remote_version_string"`
	NumConnections      int    `json:"num_connections"`
	LogEvent
}

func (l *P2PConnected) EventName() string {
	return "p2p.connected"
}

type P2PDisconnected struct {
	NumConnections int    `json:"num_connections"`
	RemoteId       string `json:"remote_id"`
	LogEvent
}

func (l *P2PDisconnected) EventName() string {
	return "p2p.disconnected"
}

type EthMinerNewBlock struct {
	BlockHash     string   `json:"block_hash"`
	BlockNumber   *big.Int `json:"block_number"`
	ChainHeadHash string   `json:"chain_head_hash"`
	BlockPrevHash string   `json:"block_prev_hash"`
	LogEvent
}

func (l *EthMinerNewBlock) EventName() string {
	return "eth.miner.new_block"
}

type EthChainReceivedNewBlock struct {
	BlockHash     string   `json:"block_hash"`
	BlockNumber   *big.Int `json:"block_number"`
	ChainHeadHash string   `json:"chain_head_hash"`
	BlockPrevHash string   `json:"block_prev_hash"`
	RemoteId      string   `json:"remote_id"`
	LogEvent
}

func (l *EthChainReceivedNewBlock) EventName() string {
	return "eth.chain.received.new_block"
}

type EthChainNewHead struct {
	BlockHash     string   `json:"block_hash"`
	BlockNumber   *big.Int `json:"block_number"`
	ChainHeadHash string   `json:"chain_head_hash"`
	BlockPrevHash string   `json:"block_prev_hash"`
	LogEvent
}

func (l *EthChainNewHead) EventName() string {
	return "eth.chain.new_head"
}

type EthTxReceived struct {
	TxHash   string `json:"tx_hash"`
	RemoteId string `json:"remote_id"`
	LogEvent
}

func (l *EthTxReceived) EventName() string {
	return "eth.tx.received"
}

//
//
// The types below are legacy and need to be converted to new format or deleted
//
//

// type P2PConnecting struct {
// 	RemoteId       string `json:"remote_id"`
// 	RemoteEndpoint string `json:"remote_endpoint"`
// 	NumConnections int    `json:"num_connections"`
// 	LogEvent
// }

// func (l *P2PConnecting) EventName() string {
// 	return "p2p.connecting"
// }

// type P2PHandshaked struct {
// 	RemoteCapabilities []string `json:"remote_capabilities"`
// 	RemoteId           string   `json:"remote_id"`
// 	NumConnections     int      `json:"num_connections"`
// 	LogEvent
// }

// func (l *P2PHandshaked) EventName() string {
// 	return "p2p.handshaked"
// }

// type P2PDisconnecting struct {
// 	Reason         string `json:"reason"`
// 	RemoteId       string `json:"remote_id"`
// 	NumConnections int    `json:"num_connections"`
// 	LogEvent
// }

// func (l *P2PDisconnecting) EventName() string {
// 	return "p2p.disconnecting"
// }

// type P2PDisconnectingBadHandshake struct {
// 	Reason         string `json:"reason"`
// 	RemoteId       string `json:"remote_id"`
// 	NumConnections int    `json:"num_connections"`
// 	LogEvent
// }

// func (l *P2PDisconnectingBadHandshake) EventName() string {
// 	return "p2p.disconnecting.bad_handshake"
// }

// type P2PDisconnectingBadProtocol struct {
// 	Reason         string `json:"reason"`
// 	RemoteId       string `json:"remote_id"`
// 	NumConnections int    `json:"num_connections"`
// 	LogEvent
// }

// func (l *P2PDisconnectingBadProtocol) EventName() string {
// 	return "p2p.disconnecting.bad_protocol"
// }

// type P2PDisconnectingReputation struct {
// 	Reason         string `json:"reason"`
// 	RemoteId       string `json:"remote_id"`
// 	NumConnections int    `json:"num_connections"`
// 	LogEvent
// }

// func (l *P2PDisconnectingReputation) EventName() string {
// 	return "p2p.disconnecting.reputation"
// }

// type P2PDisconnectingDHT struct {
// 	Reason         string `json:"reason"`
// 	RemoteId       string `json:"remote_id"`
// 	NumConnections int    `json:"num_connections"`
// 	LogEvent
// }

// func (l *P2PDisconnectingDHT) EventName() string {
// 	return "p2p.disconnecting.dht"
// }

// type P2PEthDisconnectingBadBlock struct {
// 	Reason         string `json:"reason"`
// 	RemoteId       string `json:"remote_id"`
// 	NumConnections int    `json:"num_connections"`
// 	LogEvent
// }

// func (l *P2PEthDisconnectingBadBlock) EventName() string {
// 	return "p2p.eth.disconnecting.bad_block"
// }

// type P2PEthDisconnectingBadTx struct {
// 	Reason         string `json:"reason"`
// 	RemoteId       string `json:"remote_id"`
// 	NumConnections int    `json:"num_connections"`
// 	LogEvent
// }

// func (l *P2PEthDisconnectingBadTx) EventName() string {
// 	return "p2p.eth.disconnecting.bad_tx"
// }

// type EthNewBlockBroadcasted struct {
// 	BlockNumber     int    `json:"block_number"`
// 	HeadHash        string `json:"head_hash"`
// 	BlockHash       string `json:"block_hash"`
// 	BlockDifficulty int    `json:"block_difficulty"`
// 	BlockPrevHash   string `json:"block_prev_hash"`
// 	LogEvent
// }

// func (l *EthNewBlockBroadcasted) EventName() string {
// 	return "eth.newblock.broadcasted"
// }

// type EthNewBlockIsKnown struct {
// 	BlockNumber     int    `json:"block_number"`
// 	HeadHash        string `json:"head_hash"`
// 	BlockHash       string `json:"block_hash"`
// 	BlockDifficulty int    `json:"block_difficulty"`
// 	BlockPrevHash   string `json:"block_prev_hash"`
// 	LogEvent
// }

// func (l *EthNewBlockIsKnown) EventName() string {
// 	return "eth.newblock.is_known"
// }

// type EthNewBlockIsNew struct {
// 	BlockNumber     int    `json:"block_number"`
// 	HeadHash        string `json:"head_hash"`
// 	BlockHash       string `json:"block_hash"`
// 	BlockDifficulty int    `json:"block_difficulty"`
// 	BlockPrevHash   string `json:"block_prev_hash"`
// 	LogEvent
// }

// func (l *EthNewBlockIsNew) EventName() string {
// 	return "eth.newblock.is_new"
// }

// type EthNewBlockMissingParent struct {
// 	BlockNumber     int    `json:"block_number"`
// 	HeadHash        string `json:"head_hash"`
// 	BlockHash       string `json:"block_hash"`
// 	BlockDifficulty int    `json:"block_difficulty"`
// 	BlockPrevHash   string `json:"block_prev_hash"`
// 	LogEvent
// }

// func (l *EthNewBlockMissingParent) EventName() string {
// 	return "eth.newblock.missing_parent"
// }

// type EthNewBlockIsInvalid struct {
// 	BlockNumber     int    `json:"block_number"`
// 	HeadHash        string `json:"head_hash"`
// 	BlockHash       string `json:"block_hash"`
// 	BlockDifficulty int    `json:"block_difficulty"`
// 	BlockPrevHash   string `json:"block_prev_hash"`
// 	LogEvent
// }

// func (l *EthNewBlockIsInvalid) EventName() string {
// 	return "eth.newblock.is_invalid"
// }

// type EthNewBlockChainIsOlder struct {
// 	BlockNumber     int    `json:"block_number"`
// 	HeadHash        string `json:"head_hash"`
// 	BlockHash       string `json:"block_hash"`
// 	BlockDifficulty int    `json:"block_difficulty"`
// 	BlockPrevHash   string `json:"block_prev_hash"`
// 	LogEvent
// }

// func (l *EthNewBlockChainIsOlder) EventName() string {
// 	return "eth.newblock.chain.is_older"
// }

// type EthNewBlockChainIsCanonical struct {
// 	BlockNumber     int    `json:"block_number"`
// 	HeadHash        string `json:"head_hash"`
// 	BlockHash       string `json:"block_hash"`
// 	BlockDifficulty int    `json:"block_difficulty"`
// 	BlockPrevHash   string `json:"block_prev_hash"`
// 	LogEvent
// }

// func (l *EthNewBlockChainIsCanonical) EventName() string {
// 	return "eth.newblock.chain.is_cannonical"
// }

// type EthNewBlockChainNotCanonical struct {
// 	BlockNumber     int    `json:"block_number"`
// 	HeadHash        string `json:"head_hash"`
// 	BlockHash       string `json:"block_hash"`
// 	BlockDifficulty int    `json:"block_difficulty"`
// 	BlockPrevHash   string `json:"block_prev_hash"`
// 	LogEvent
// }

// func (l *EthNewBlockChainNotCanonical) EventName() string {
// 	return "eth.newblock.chain.not_cannonical"
// }

// type EthTxCreated struct {
// 	TxHash    string `json:"tx_hash"`
// 	TxSender  string `json:"tx_sender"`
// 	TxAddress string `json:"tx_address"`
// 	TxHexRLP  string `json:"tx_hexrlp"`
// 	TxNonce   int    `json:"tx_nonce"`
// 	LogEvent
// }

// func (l *EthTxCreated) EventName() string {
// 	return "eth.tx.created"
// }

// type EthTxBroadcasted struct {
// 	TxHash    string `json:"tx_hash"`
// 	TxSender  string `json:"tx_sender"`
// 	TxAddress string `json:"tx_address"`
// 	TxNonce   int    `json:"tx_nonce"`
// 	LogEvent
// }

// func (l *EthTxBroadcasted) EventName() string {
// 	return "eth.tx.broadcasted"
// }

// type EthTxValidated struct {
// 	TxHash    string `json:"tx_hash"`
// 	TxSender  string `json:"tx_sender"`
// 	TxAddress string `json:"tx_address"`
// 	TxNonce   int    `json:"tx_nonce"`
// 	LogEvent
// }

// func (l *EthTxValidated) EventName() string {
// 	return "eth.tx.validated"
// }

// type EthTxIsInvalid struct {
// 	TxHash    string `json:"tx_hash"`
// 	TxSender  string `json:"tx_sender"`
// 	TxAddress string `json:"tx_address"`
// 	Reason    string `json:"reason"`
// 	TxNonce   int    `json:"tx_nonce"`
// 	LogEvent
// }

// func (l *EthTxIsInvalid) EventName() string {
// 	return "eth.tx.is_invalid"
// }
