// Copyright 2023 The go-ethereum Authors
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

package eth

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
)

// AdminAPI is the collection of Ethereum full node related APIs for node
// administration.
type AdminAPI struct {
	eth *Ethereum
}

// NewAdminAPI creates a new instance of AdminAPI.
func NewAdminAPI(eth *Ethereum) *AdminAPI {
	return &AdminAPI{eth: eth}
}

// ExportChain exports the current blockchain into a local file,
// or a range of blocks if first and last are non-nil.
func (api *AdminAPI) ExportChain(file string, first *uint64, last *uint64) (bool, error) {
	if first == nil && last != nil {
		return false, errors.New("last cannot be specified without first")
	}
	if first != nil && last == nil {
		head := api.eth.BlockChain().CurrentHeader().Number.Uint64()
		last = &head
	}
	if _, err := os.Stat(file); err == nil {
		// File already exists. Allowing overwrite could be a DoS vector,
		// since the 'file' may point to arbitrary paths on the drive.
		return false, errors.New("location would overwrite an existing file")
	}
	// Make sure we can create the file to export into
	out, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return false, err
	}
	defer out.Close()

	var writer io.Writer = out
	if strings.HasSuffix(file, ".gz") {
		writer = gzip.NewWriter(writer)
		defer writer.(*gzip.Writer).Close()
	}

	// Export the blockchain
	if first != nil {
		if err := api.eth.BlockChain().ExportN(writer, *first, *last); err != nil {
			return false, err
		}
	} else if err := api.eth.BlockChain().Export(writer); err != nil {
		return false, err
	}
	return true, nil
}

func hasAllBlocks(chain *core.BlockChain, bs []*types.Block) bool {
	for _, b := range bs {
		if !chain.HasBlock(b.Hash(), b.NumberU64()) {
			return false
		}
	}

	return true
}

// ImportChain imports a blockchain from a local file.
func (api *AdminAPI) ImportChain(file string) (bool, error) {
	// Make sure the can access the file to import
	in, err := os.Open(file)
	if err != nil {
		return false, err
	}
	defer in.Close()

	var reader io.Reader = in
	if strings.HasSuffix(file, ".gz") {
		if reader, err = gzip.NewReader(reader); err != nil {
			return false, err
		}
	}

	// Run actual the import in pre-configured batches
	stream := rlp.NewStream(reader, 0)

	blocks, index := make([]*types.Block, 0, 2500), 0
	for batch := 0; ; batch++ {
		// Load a batch of blocks from the input file
		for len(blocks) < cap(blocks) {
			block := new(types.Block)
			if err := stream.Decode(block); err == io.EOF {
				break
			} else if err != nil {
				return false, fmt.Errorf("block %d: failed to parse: %v", index, err)
			}
			// ignore the genesis block when importing blocks
			if block.NumberU64() == 0 {
				continue
			}
			blocks = append(blocks, block)
			index++
		}
		if len(blocks) == 0 {
			break
		}

		if hasAllBlocks(api.eth.BlockChain(), blocks) {
			blocks = blocks[:0]
			continue
		}
		// Import the batch and reset the buffer
		if _, err := api.eth.BlockChain().InsertChain(blocks); err != nil {
			return false, fmt.Errorf("batch %d: failed to insert: %v", batch, err)
		}
		blocks = blocks[:0]
	}
	return true, nil
}

// PauseForCloneArgs are the arguments of admin_pauseForClone.
type PauseForCloneArgs struct {
	Timeout   *string `json:"timeout"`   // How long the pause may last, as a duration such as "30s"
	AfterHead bool    `json:"afterHead"` // Whether to wait for a new head before pausing
}

// PauseForClone holds the chain writes and makes the database consistent on
// disk at the current head, for a copy of the data directory to be taken in the
// meantime. Other writers keep running, so the copy must capture every store
// reported at one instant, as a filesystem snapshot of the data directory does.
// It then opens at the head reported, without the rewind that follows an
// unclean shutdown.
//
// The timeout (20s by default, 2m at most) applies to getting the pause ready:
// the wait for a new head if asked, for the chain writes to stop, and making the
// database consistent. A pause ready only after the timeout is given up. Once
// ready, the pause ends with ResumeAfterClone, or by itself when the timeout
// passes again. Writing the journal and the syncs aren't interrupted, so they
// can overrun the timeout.
// One pause is held at a time. A copy is good only if ResumeAfterClone reports
// the pause intact, which says that the pause lasted, not that the copy holds
// every store.
//
// It's only served over IPC.
func (api *AdminAPI) PauseForClone(ctx context.Context, args *PauseForCloneArgs) (*ClonePauseResult, error) {
	if err := ipcOnly(ctx); err != nil {
		return nil, err
	}
	var (
		timeout   = defaultCloneTimeout
		afterHead bool
	)
	if args != nil {
		if args.Timeout != nil {
			var err error
			if timeout, err = time.ParseDuration(*args.Timeout); err != nil {
				return nil, err
			}
			if timeout <= 0 || timeout > maxCloneTimeout {
				return nil, fmt.Errorf("timeout %v out of range, at most %v", timeout, maxCloneTimeout)
			}
		}
		afterHead = args.AfterHead
	}
	return api.eth.clonePauser.pause(ctx, timeout, timeout, afterHead)
}

// ResumeAfterClone ends the pause for a clone with the given id, and tells
// whether it lasted until then: a copy taken during it is good only if so. It's
// only served over IPC.
func (api *AdminAPI) ResumeAfterClone(ctx context.Context, id string) (*CloneResumeResult, error) {
	if err := ipcOnly(ctx); err != nil {
		return nil, err
	}
	return api.eth.clonePauser.resume(id)
}

// ipcOnly refuses the calls that didn't come over IPC. In-process calls, the
// console's say, are served by the codec IPC uses, and report IPC too.
func ipcOnly(ctx context.Context) error {
	if transport := rpc.PeerInfoFromContext(ctx).Transport; transport != "ipc" {
		return fmt.Errorf("only served over IPC, not %s", transport)
	}
	return nil
}
