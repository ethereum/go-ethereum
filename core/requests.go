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

package core

import (
	"github.com/ethereum/go-ethereum/access"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"golang.org/x/net/context"
)

// BlockRequest is the ODR request type for block bodies
type BlockRequest struct {
	access.Request
	ctx       context.Context
	blockHash common.Hash
	data      []byte
}

func (req *BlockRequest) Ctx() context.Context { return req.ctx }

func (req *BlockRequest) GetRlp() []byte {
	return req.data
}

func (req *BlockRequest) StoreResult(db access.Database) {
	WriteBodyRlp(db, req.blockHash, req.GetRlp())
}

// ReceiptsRequest is the ODR request type for block receipts by block hash
type ReceiptsRequest struct {
	access.Request
	ctx       context.Context
	blockHash common.Hash
	data      types.Receipts
}

func (req *ReceiptsRequest) Ctx() context.Context { return req.ctx }

func (req *ReceiptsRequest) GetReceipts() types.Receipts {
	return req.data
}

func (req *ReceiptsRequest) StoreResult(db access.Database) {
	PutBlockReceipts(db, req.blockHash, req.GetReceipts())
}
