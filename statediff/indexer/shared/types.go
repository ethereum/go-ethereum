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

package shared

import "github.com/ethereum/go-ethereum/common"
import "github.com/ethereum/go-ethereum/statediff/types"
import "github.com/ethereum/go-ethereum/statediff/indexer/models"

// Trie struct used to flag node as leaf or not
type TrieNode struct {
	Path    []byte
	LeafKey common.Hash
	Value   []byte
	Type    types.NodeType
}

// CIDPayload is a struct to hold all the CIDs and their associated meta data for indexing in Postgres
// Returned by IPLDPublisher
// Passed to CIDIndexer
type CIDPayload struct {
	HeaderCID       models.HeaderModel
	UncleCIDs       []models.UncleModel
	TransactionCIDs []models.TxModel
	ReceiptCIDs     map[common.Hash]models.ReceiptModel
	StateNodeCIDs   []models.StateNodeModel
	StateAccounts   map[string]models.StateAccountModel
	StorageNodeCIDs map[string][]models.StorageNodeModel
}
