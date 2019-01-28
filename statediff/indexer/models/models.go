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

package models

import "github.com/lib/pq"

// HeaderModel is the db model for eth.header_cids
type HeaderModel struct {
	ID              int64  `db:"id"`
	BlockNumber     string `db:"block_number"`
	BlockHash       string `db:"block_hash"`
	ParentHash      string `db:"parent_hash"`
	CID             string `db:"cid"`
	MhKey           string `db:"mh_key"`
	TotalDifficulty string `db:"td"`
	NodeID          int64  `db:"node_id"`
	Reward          string `db:"reward"`
	StateRoot       string `db:"state_root"`
	UncleRoot       string `db:"uncle_root"`
	TxRoot          string `db:"tx_root"`
	RctRoot         string `db:"receipt_root"`
	Bloom           []byte `db:"bloom"`
	Timestamp       uint64 `db:"timestamp"`
	TimesValidated  int64  `db:"times_validated"`
}

// UncleModel is the db model for eth.uncle_cids
type UncleModel struct {
	ID         int64  `db:"id"`
	HeaderID   int64  `db:"header_id"`
	BlockHash  string `db:"block_hash"`
	ParentHash string `db:"parent_hash"`
	CID        string `db:"cid"`
	MhKey      string `db:"mh_key"`
	Reward     string `db:"reward"`
}

// TxModel is the db model for eth.transaction_cids
type TxModel struct {
	ID         int64  `db:"id"`
	HeaderID   int64  `db:"header_id"`
	Index      int64  `db:"index"`
	TxHash     string `db:"tx_hash"`
	CID        string `db:"cid"`
	MhKey      string `db:"mh_key"`
	Dst        string `db:"dst"`
	Src        string `db:"src"`
	Data       []byte `db:"tx_data"`
	Deployment bool   `db:"deployment"`
}

// ReceiptModel is the db model for eth.receipt_cids
type ReceiptModel struct {
	ID           int64          `db:"id"`
	TxID         int64          `db:"tx_id"`
	CID          string         `db:"cid"`
	MhKey        string         `db:"mh_key"`
	Contract     string         `db:"contract"`
	ContractHash string         `db:"contract_hash"`
	LogContracts pq.StringArray `db:"log_contracts"`
	Topic0s      pq.StringArray `db:"topic0s"`
	Topic1s      pq.StringArray `db:"topic1s"`
	Topic2s      pq.StringArray `db:"topic2s"`
	Topic3s      pq.StringArray `db:"topic3s"`
}

// StateNodeModel is the db model for eth.state_cids
type StateNodeModel struct {
	ID       int64  `db:"id"`
	HeaderID int64  `db:"header_id"`
	Path     []byte `db:"state_path"`
	StateKey string `db:"state_leaf_key"`
	NodeType int    `db:"node_type"`
	CID      string `db:"cid"`
	MhKey    string `db:"mh_key"`
	Diff     bool   `db:"diff"`
}

// StorageNodeModel is the db model for eth.storage_cids
type StorageNodeModel struct {
	ID         int64  `db:"id"`
	StateID    int64  `db:"state_id"`
	Path       []byte `db:"storage_path"`
	StorageKey string `db:"storage_leaf_key"`
	NodeType   int    `db:"node_type"`
	CID        string `db:"cid"`
	MhKey      string `db:"mh_key"`
	Diff       bool   `db:"diff"`
}

// StorageNodeWithStateKeyModel is a db model for eth.storage_cids + eth.state_cids.state_key
type StorageNodeWithStateKeyModel struct {
	ID         int64  `db:"id"`
	StateID    int64  `db:"state_id"`
	Path       []byte `db:"storage_path"`
	StateKey   string `db:"state_leaf_key"`
	StorageKey string `db:"storage_leaf_key"`
	NodeType   int    `db:"node_type"`
	CID        string `db:"cid"`
	MhKey      string `db:"mh_key"`
	Diff       bool   `db:"diff"`
}

// StateAccountModel is a db model for an eth state account (decoded value of state leaf node)
type StateAccountModel struct {
	ID          int64  `db:"id"`
	StateID     int64  `db:"state_id"`
	Balance     string `db:"balance"`
	Nonce       uint64 `db:"nonce"`
	CodeHash    []byte `db:"code_hash"`
	StorageRoot string `db:"storage_root"`
}
