// Copyright 2016 The go-ethereum Authors
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

package light

import (
	"bytes"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/net/context"
)

var sha3_nil = crypto.Keccak256Hash(nil)

var (
	ErrNoTrustedCht = errors.New("No trusted canonical hash trie")
	ErrNoHeader     = errors.New("Header not found")

	ChtFrequency  = uint64(4096)
	trustedChtKey = []byte("TrustedCHT")
)

type ChtNode struct {
	Hash common.Hash
	Td   *big.Int
}

type TrustedCht struct {
	Number uint64
	Root   common.Hash
}

func GetTrustedCht(db ethdb.Database) TrustedCht {
	data, _ := db.Get(trustedChtKey)
	var res TrustedCht
	if err := rlp.DecodeBytes(data, &res); err != nil {
		return TrustedCht{0, common.Hash{}}
	}
	return res
}

func WriteTrustedCht(db ethdb.Database, cht TrustedCht) {
	data, _ := rlp.EncodeToBytes(cht)
	db.Put(trustedChtKey, data)
}

func DeleteTrustedCht(db ethdb.Database) {
	db.Delete(trustedChtKey)
}

func GetHeaderByNumber(ctx context.Context, odr OdrBackend, number uint64) (*types.Header, error) {
	db := odr.Database()
	hash := core.GetCanonicalHash(db, number)
	if (hash != common.Hash{}) {
		// if there is a canonical hash, there is a header too
		header := core.GetHeader(db, hash, number)
		if header == nil {
			panic("Canonical hash present but header not found")
		}
		return header, nil
	}

	cht := GetTrustedCht(db)
	if number >= cht.Number*ChtFrequency {
		return nil, ErrNoTrustedCht
	}

	r := &ChtRequest{ChtRoot: cht.Root, ChtNum: cht.Number, BlockNum: number}
	if err := odr.Retrieve(ctx, r); err != nil {
		return nil, err
	} else {
		return r.Header, nil
	}
}

func GetCanonicalHash(ctx context.Context, odr OdrBackend, number uint64) (common.Hash, error) {
	hash := core.GetCanonicalHash(odr.Database(), number)
	if (hash != common.Hash{}) {
		return hash, nil
	}
	header, err := GetHeaderByNumber(ctx, odr, number)
	if header != nil {
		return header.Hash(), nil
	}
	return common.Hash{}, err
}

// retrieveContractCode tries to retrieve the contract code of the given account
// with the given hash from the network (id points to the storage trie belonging
// to the same account)
func retrieveContractCode(ctx context.Context, odr OdrBackend, id *TrieID, hash common.Hash) ([]byte, error) {
	if hash == sha3_nil {
		return nil, nil
	}
	res, _ := odr.Database().Get(hash[:])
	if res != nil {
		return res, nil
	}
	r := &CodeRequest{Id: id, Hash: hash}
	if err := odr.Retrieve(ctx, r); err != nil {
		return nil, err
	} else {
		return r.Data, nil
	}
}

// GetBodyRLP retrieves the block body (transactions and uncles) in RLP encoding.
func GetBodyRLP(ctx context.Context, odr OdrBackend, hash common.Hash, number uint64) (rlp.RawValue, error) {
	if data := core.GetBodyRLP(odr.Database(), hash, number); data != nil {
		return data, nil
	}
	r := &BlockRequest{Hash: hash, Number: number}
	if err := odr.Retrieve(ctx, r); err != nil {
		return nil, err
	} else {
		return r.Rlp, nil
	}
}

// GetBody retrieves the block body (transactons, uncles) corresponding to the
// hash.
func GetBody(ctx context.Context, odr OdrBackend, hash common.Hash, number uint64) (*types.Body, error) {
	data, err := GetBodyRLP(ctx, odr, hash, number)
	if err != nil {
		return nil, err
	}
	body := new(types.Body)
	if err := rlp.Decode(bytes.NewReader(data), body); err != nil {
		glog.V(logger.Error).Infof("invalid block body RLP for hash %x: %v", hash, err)
		return nil, err
	}
	return body, nil
}

// GetBlock retrieves an entire block corresponding to the hash, assembling it
// back from the stored header and body.
func GetBlock(ctx context.Context, odr OdrBackend, hash common.Hash, number uint64) (*types.Block, error) {
	// Retrieve the block header and body contents
	header := core.GetHeader(odr.Database(), hash, number)
	if header == nil {
		return nil, ErrNoHeader
	}
	body, err := GetBody(ctx, odr, hash, number)
	if err != nil {
		return nil, err
	}
	// Reassemble the block and return
	return types.NewBlockWithHeader(header).WithBody(body.Transactions, body.Uncles), nil
}

// GetBlockReceipts retrieves the receipts generated by the transactions included
// in a block given by its hash.
func GetBlockReceipts(ctx context.Context, odr OdrBackend, hash common.Hash, number uint64) (types.Receipts, error) {
	receipts := core.GetBlockReceipts(odr.Database(), hash, number)
	if receipts != nil {
		return receipts, nil
	}
	r := &ReceiptsRequest{Hash: hash, Number: number}
	if err := odr.Retrieve(ctx, r); err != nil {
		return nil, err
	} else {
		return r.Receipts, nil
	}
}
