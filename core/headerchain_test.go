// Copyright 2020 The go-ethereum Authors
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
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

func verifyUnbrokenCanonchain(hc *HeaderChain) error {
	h := hc.CurrentHeader()
	for {
		canonHash := rawdb.ReadCanonicalHash(hc.chainDb, h.Number.Uint64())
		if exp := h.Hash(); canonHash != exp {
			return fmt.Errorf("Canon hash chain broken, block %d got %x, expected %x",
				h.Number, canonHash[:8], exp[:8])
		}
		// Verify that we have the TD
		if td := rawdb.ReadTd(hc.chainDb, canonHash, h.Number.Uint64()); td == nil {
			return fmt.Errorf("Canon TD missing at block %d", h.Number)
		}
		if h.Number.Uint64() == 0 {
			break
		}
		h = hc.GetHeader(h.ParentHash, h.Number.Uint64()-1)
	}
	return nil
}

func testInsert(t *testing.T, hc *HeaderChain, chain []*types.Header, expInsert, expCanon, expSide int) error {
	t.Helper()
	gotInsert, gotCanon, gotSide := 0, 0, 0

	_, err := hc.InsertHeaderChain(chain, func(header *types.Header) error {
		status, err := hc.WriteHeader(header)
		if err != nil{
			return err
		}
		gotInsert++
		switch status {
		case CanonStatTy:
			gotCanon++
		default:
			gotSide++
		}
		return nil

	}, time.Now())

	if gotInsert != expInsert {
		t.Errorf("wrong number of callback invocations, got %d, exp %d", gotInsert, expInsert)
	}
	if gotCanon != expCanon {
		t.Errorf("wrong number of canon headers, got %d, exp %d", gotCanon, expCanon)
	}
	if gotSide != expSide {
		t.Errorf("wrong number of side headers, got %d, exp %d", gotSide, expSide)
	}
	// Always verify that the header chain is unbroken
	if err := verifyUnbrokenCanonchain(hc); err != nil {
		t.Fatal(err)
		return err
	}
	return err
}

func TestHeaderInsertion(t *testing.T) {
	var (
		db      = rawdb.NewMemoryDatabase()
		genesis = new(Genesis).MustCommit(db)
	)

	hc, err := NewHeaderChain(db, params.AllEthashProtocolChanges, ethash.NewFaker(), func() bool { return false })
	if err != nil {
		t.Fatal(err)
	}
	// chain A: G->A1->A2...A128
	chainA := makeHeaderChain(genesis.Header(), 128, ethash.NewFaker(), db, 10)
	// chain B: G->A1->B2...B128
	chainB := makeHeaderChain(chainA[0], 128, ethash.NewFaker(), db, 10)
	log.Root().SetHandler(log.StdoutHandler)

	// Inserting 64 headers on an empty chain, expecting
	// 64 callbacks, 64 canon-status, 0 sidestatus,
	if err := testInsert(t, hc, chainA[:64], 64, 64, 0); err != nil {
		t.Fatal(err)
	}

	// Inserting 64 indentical headers, expecting
	// 0 callbacks, 0 canon-status, 0 sidestatus,
	if err := testInsert(t, hc, chainA[:64], 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	// Inserting the same some old, some new headers
	// 32 callbacks, 32 canon, 0 side
	if err := testInsert(t, hc, chainA[32:96], 32, 32, 0); err != nil {
		t.Fatal(err)
	}
	// Inserting side blocks, but not overtaking the canon chain
	if err := testInsert(t, hc, chainB[0:32], 32, 0, 32); err != nil {
		t.Fatal(err)
	}
	// Inserting more side blocks, but we don't have the parent
	if err := testInsert(t, hc, chainB[34:36], 0, 0, 0); !errors.Is(err, consensus.ErrUnknownAncestor) {
		t.Fatal(fmt.Errorf("Expected %v, got %v", consensus.ErrUnknownAncestor, err))
	}
	// Inserting more sideblocks, overtaking the canon chain
	if err := testInsert(t, hc, chainB[32:97], 65, 65, 0); err != nil {
		t.Fatal(err)
	}
	// Inserting more A-headers, taking back the canonicality
	if err := testInsert(t, hc, chainA[90:100], 4, 4, 0); err != nil {
		t.Fatal(err)
	}
	// And B becomes canon again
	if err := testInsert(t, hc, chainB[97:107], 10, 10, 0); err != nil {
		t.Fatal(err)
	}
	// And B becomes even longer
	if err := testInsert(t, hc, chainB[107:128], 21, 21, 0); err != nil {
		t.Fatal(err)
	}
}
