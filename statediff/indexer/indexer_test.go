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

package indexer_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-ipfs-blockstore"
	"github.com/ipfs/go-ipfs-ds-help"

	ind "github.com/ethereum/go-ethereum/statediff/indexer"
	"github.com/ethereum/go-ethereum/statediff/indexer/mocks"
	eth "github.com/ethereum/go-ethereum/statediff/indexer/models"
	"github.com/ethereum/go-ethereum/statediff/indexer/postgres"
	"github.com/ethereum/go-ethereum/statediff/indexer/shared"
)

var (
	db        *postgres.DB
	err       error
	indexer   *ind.StateDiffIndexer
	ipfsPgGet = `SELECT data FROM public.blocks
					WHERE key = $1`
)

func expectTrue(t *testing.T, value bool) {
	if !value {
		t.Fatalf("Assertion failed")
	}
}

func setup(t *testing.T) {
	db, err = shared.SetupDB()
	if err != nil {
		t.Fatal(err)
	}
	indexer = ind.NewStateDiffIndexer(params.MainnetChainConfig, db)
	var tx *ind.BlockTx
	tx, err = indexer.PushBlock(
		mocks.MockBlock,
		mocks.MockReceipts,
		mocks.MockBlock.Difficulty())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Close()
	for _, node := range mocks.StateDiffs {
		err = indexer.PushStateNode(tx, node)
		if err != nil {
			t.Fatal(err)
		}
	}

	shared.ExpectEqual(t, tx.BlockNumber, mocks.BlockNumber.Uint64())
}

func tearDown(t *testing.T) {
	ind.TearDownDB(t, db)
}

func TestPublishAndIndexer(t *testing.T) {
	t.Run("Publish and index header IPLDs in a single tx", func(t *testing.T) {
		setup(t)
		defer tearDown(t)
		pgStr := `SELECT cid, td, reward, id
				FROM eth.header_cids
				WHERE block_number = $1`
		// check header was properly indexed
		type res struct {
			CID    string
			TD     string
			Reward string
			ID     int
		}
		header := new(res)
		err = db.QueryRowx(pgStr, 1).StructScan(header)
		if err != nil {
			t.Fatal(err)
		}
		shared.ExpectEqual(t, header.CID, mocks.HeaderCID.String())
		shared.ExpectEqual(t, header.TD, mocks.MockBlock.Difficulty().String())
		shared.ExpectEqual(t, header.Reward, "5000000000000011250")
		dc, err := cid.Decode(header.CID)
		if err != nil {
			t.Fatal(err)
		}
		mhKey := dshelp.MultihashToDsKey(dc.Hash())
		prefixedKey := blockstore.BlockPrefix.String() + mhKey.String()
		var data []byte
		err = db.Get(&data, ipfsPgGet, prefixedKey)
		if err != nil {
			t.Fatal(err)
		}
		shared.ExpectEqual(t, data, mocks.MockHeaderRlp)
	})

	t.Run("Publish and index transaction IPLDs in a single tx", func(t *testing.T) {
		setup(t)
		defer tearDown(t)
		// check that txs were properly indexed
		trxs := make([]string, 0)
		pgStr := `SELECT transaction_cids.cid FROM eth.transaction_cids INNER JOIN eth.header_cids ON (transaction_cids.header_id = header_cids.id)
				WHERE header_cids.block_number = $1`
		err = db.Select(&trxs, pgStr, 1)
		if err != nil {
			t.Fatal(err)
		}
		shared.ExpectEqual(t, len(trxs), 3)
		expectTrue(t, shared.ListContainsString(trxs, mocks.Trx1CID.String()))
		expectTrue(t, shared.ListContainsString(trxs, mocks.Trx2CID.String()))
		expectTrue(t, shared.ListContainsString(trxs, mocks.Trx3CID.String()))
		// and published
		for _, c := range trxs {
			dc, err := cid.Decode(c)
			if err != nil {
				t.Fatal(err)
			}
			mhKey := dshelp.MultihashToDsKey(dc.Hash())
			prefixedKey := blockstore.BlockPrefix.String() + mhKey.String()
			var data []byte
			err = db.Get(&data, ipfsPgGet, prefixedKey)
			if err != nil {
				t.Fatal(err)
			}
			switch c {
			case mocks.Trx1CID.String():
				shared.ExpectEqual(t, data, mocks.MockTransactions.GetRlp(0))
			case mocks.Trx2CID.String():
				shared.ExpectEqual(t, data, mocks.MockTransactions.GetRlp(1))
			case mocks.Trx3CID.String():
				shared.ExpectEqual(t, data, mocks.MockTransactions.GetRlp(2))
			}
		}
	})

	t.Run("Publish and index receipt IPLDs in a single tx", func(t *testing.T) {
		setup(t)
		defer tearDown(t)
		// check receipts were properly indexed
		rcts := make([]string, 0)
		pgStr := `SELECT receipt_cids.cid FROM eth.receipt_cids, eth.transaction_cids, eth.header_cids
				WHERE receipt_cids.tx_id = transaction_cids.id
				AND transaction_cids.header_id = header_cids.id
				AND header_cids.block_number = $1`
		err = db.Select(&rcts, pgStr, 1)
		if err != nil {
			t.Fatal(err)
		}
		shared.ExpectEqual(t, len(rcts), 3)
		expectTrue(t, shared.ListContainsString(rcts, mocks.Rct1CID.String()))
		expectTrue(t, shared.ListContainsString(rcts, mocks.Rct2CID.String()))
		expectTrue(t, shared.ListContainsString(rcts, mocks.Rct3CID.String()))
		// and published
		for _, c := range rcts {
			dc, err := cid.Decode(c)
			if err != nil {
				t.Fatal(err)
			}
			mhKey := dshelp.MultihashToDsKey(dc.Hash())
			prefixedKey := blockstore.BlockPrefix.String() + mhKey.String()
			var data []byte
			err = db.Get(&data, ipfsPgGet, prefixedKey)
			if err != nil {
				t.Fatal(err)
			}
			switch c {
			case mocks.Rct1CID.String():
				shared.ExpectEqual(t, data, mocks.MockReceipts.GetRlp(0))
			case mocks.Rct2CID.String():
				shared.ExpectEqual(t, data, mocks.MockReceipts.GetRlp(1))
			case mocks.Rct3CID.String():
				shared.ExpectEqual(t, data, mocks.MockReceipts.GetRlp(2))
			}
		}
	})

	t.Run("Publish and index state IPLDs in a single tx", func(t *testing.T) {
		setup(t)
		defer tearDown(t)
		// check that state nodes were properly indexed and published
		stateNodes := make([]eth.StateNodeModel, 0)
		pgStr := `SELECT state_cids.id, state_cids.cid, state_cids.state_leaf_key, state_cids.node_type, state_cids.state_path, state_cids.header_id
				FROM eth.state_cids INNER JOIN eth.header_cids ON (state_cids.header_id = header_cids.id)
				WHERE header_cids.block_number = $1`
		err = db.Select(&stateNodes, pgStr, 1)
		if err != nil {
			t.Fatal(err)
		}
		shared.ExpectEqual(t, len(stateNodes), 2)
		for _, stateNode := range stateNodes {
			var data []byte
			dc, err := cid.Decode(stateNode.CID)
			if err != nil {
				t.Fatal(err)
			}
			mhKey := dshelp.MultihashToDsKey(dc.Hash())
			prefixedKey := blockstore.BlockPrefix.String() + mhKey.String()
			err = db.Get(&data, ipfsPgGet, prefixedKey)
			if err != nil {
				t.Fatal(err)
			}
			pgStr = `SELECT * from eth.state_accounts WHERE state_id = $1`
			var account eth.StateAccountModel
			err = db.Get(&account, pgStr, stateNode.ID)
			if err != nil {
				t.Fatal(err)
			}
			if stateNode.CID == mocks.State1CID.String() {
				shared.ExpectEqual(t, stateNode.NodeType, 2)
				shared.ExpectEqual(t, stateNode.StateKey, common.BytesToHash(mocks.ContractLeafKey).Hex())
				shared.ExpectEqual(t, stateNode.Path, []byte{'\x06'})
				shared.ExpectEqual(t, data, mocks.ContractLeafNode)
				shared.ExpectEqual(t, account, eth.StateAccountModel{
					ID:          account.ID,
					StateID:     stateNode.ID,
					Balance:     "0",
					CodeHash:    mocks.ContractCodeHash.Bytes(),
					StorageRoot: mocks.ContractRoot,
					Nonce:       1,
				})
			}
			if stateNode.CID == mocks.State2CID.String() {
				shared.ExpectEqual(t, stateNode.NodeType, 2)
				shared.ExpectEqual(t, stateNode.StateKey, common.BytesToHash(mocks.AccountLeafKey).Hex())
				shared.ExpectEqual(t, stateNode.Path, []byte{'\x0c'})
				shared.ExpectEqual(t, data, mocks.AccountLeafNode)
				shared.ExpectEqual(t, account, eth.StateAccountModel{
					ID:          account.ID,
					StateID:     stateNode.ID,
					Balance:     "1000",
					CodeHash:    mocks.AccountCodeHash.Bytes(),
					StorageRoot: mocks.AccountRoot,
					Nonce:       0,
				})
			}
		}
		pgStr = `SELECT * from eth.state_accounts WHERE state_id = $1`
	})

	t.Run("Publish and index storage IPLDs in a single tx", func(t *testing.T) {
		setup(t)
		defer tearDown(t)
		// check that storage nodes were properly indexed
		storageNodes := make([]eth.StorageNodeWithStateKeyModel, 0)
		pgStr := `SELECT storage_cids.cid, state_cids.state_leaf_key, storage_cids.storage_leaf_key, storage_cids.node_type, storage_cids.storage_path
				FROM eth.storage_cids, eth.state_cids, eth.header_cids
				WHERE storage_cids.state_id = state_cids.id
				AND state_cids.header_id = header_cids.id
				AND header_cids.block_number = $1`
		err = db.Select(&storageNodes, pgStr, 1)
		if err != nil {
			t.Fatal(err)
		}
		shared.ExpectEqual(t, len(storageNodes), 1)
		shared.ExpectEqual(t, storageNodes[0], eth.StorageNodeWithStateKeyModel{
			CID:        mocks.StorageCID.String(),
			NodeType:   2,
			StorageKey: common.BytesToHash(mocks.StorageLeafKey).Hex(),
			StateKey:   common.BytesToHash(mocks.ContractLeafKey).Hex(),
			Path:       []byte{},
		})
		var data []byte
		dc, err := cid.Decode(storageNodes[0].CID)
		if err != nil {
			t.Fatal(err)
		}
		mhKey := dshelp.MultihashToDsKey(dc.Hash())
		prefixedKey := blockstore.BlockPrefix.String() + mhKey.String()
		err = db.Get(&data, ipfsPgGet, prefixedKey)
		if err != nil {
			t.Fatal(err)
		}
		shared.ExpectEqual(t, data, mocks.StorageLeafNode)
	})
}
