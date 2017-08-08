// Copyright 2014 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

// Package utils contains internal helper functions for go-ethereum commands.
package utils

import (
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/internal/debug"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	importBatchSize = 2500
)

// Fatalf formats a message to standard error and exits the program.
// The message is also printed to standard output if standard error
// is redirected to a different file.
func Fatalf(format string, args ...interface{}) {
	w := io.MultiWriter(os.Stdout, os.Stderr)
	if runtime.GOOS == "windows" {
		// The SameFile check below doesn't work on Windows.
		// stdout is unlikely to get redirected though, so just print there.
		w = os.Stdout
	} else {
		outf, _ := os.Stdout.Stat()
		errf, _ := os.Stderr.Stat()
		if outf != nil && errf != nil && os.SameFile(outf, errf) {
			w = os.Stderr
		}
	}
	fmt.Fprintf(w, "Fatal: "+format+"\n", args...)
	os.Exit(1)
}

func StartNode(stack *node.Node) {
	if err := stack.Start(); err != nil {
		Fatalf("Error starting protocol stack: %v", err)
	}
	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, os.Interrupt)
		defer signal.Stop(sigc)
		<-sigc
		log.Info("Got interrupt, shutting down...")
		go stack.Stop()
		for i := 10; i > 0; i-- {
			<-sigc
			if i > 1 {
				log.Warn("Already shutting down, interrupt more to panic.", "times", i-1)
			}
		}
		debug.Exit() // ensure trace and CPU profile data is flushed.
		debug.LoudPanic("boom")
	}()
}

func ImportChain(chain *core.BlockChain, fn string) error {
	// Watch for Ctrl-C while the import is running.
	// If a signal is received, the import will stop at the next batch.
	interrupt := make(chan os.Signal, 1)
	stop := make(chan struct{})
	signal.Notify(interrupt, os.Interrupt)
	defer signal.Stop(interrupt)
	defer close(interrupt)
	go func() {
		if _, ok := <-interrupt; ok {
			log.Info("Interrupted during import, stopping at next batch")
		}
		close(stop)
	}()
	checkInterrupt := func() bool {
		select {
		case <-stop:
			return true
		default:
			return false
		}
	}

	log.Info("Importing blockchain", "file", fn)
	fh, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer fh.Close()

	var reader io.Reader = fh
	if strings.HasSuffix(fn, ".gz") {
		if reader, err = gzip.NewReader(reader); err != nil {
			return err
		}
	}

	stream := rlp.NewStream(reader, 0)

	// Run actual the import.
	blocks := make(types.Blocks, importBatchSize)
	n := 0
	for batch := 0; ; batch++ {
		// Load a batch of RLP blocks.
		if checkInterrupt() {
			return fmt.Errorf("interrupted")
		}
		i := 0
		for ; i < importBatchSize; i++ {
			var b types.Block
			if err := stream.Decode(&b); err == io.EOF {
				break
			} else if err != nil {
				return fmt.Errorf("at block %d: %v", n, err)
			}
			// don't import first block
			if b.NumberU64() == 0 {
				i--
				continue
			}
			blocks[i] = &b
			n++
		}
		if i == 0 {
			break
		}
		// Import the batch.
		if checkInterrupt() {
			return fmt.Errorf("interrupted")
		}
		if hasAllBlocks(chain, blocks[:i]) {
			log.Info("Skipping batch as all blocks present", "batch", batch, "first", blocks[0].Hash(), "last", blocks[i-1].Hash())
			continue
		}

		if _, err := chain.InsertChain(blocks[:i]); err != nil {
			return fmt.Errorf("invalid block %d: %v", n, err)
		}
	}
	return nil
}

func hasAllBlocks(chain *core.BlockChain, bs []*types.Block) bool {
	for _, b := range bs {
		if !chain.HasBlock(b.Hash()) {
			return false
		}
	}
	return true
}

func ExportChain(blockchain *core.BlockChain, chainDb ethdb.Database, fn string, first uint64, last uint64) error {
	log.Info("Exporting blockchain", "file", fn)
	// TODO verify mode perms
	fh, err := os.OpenFile(fn, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer fh.Close()

	var writer io.Writer = fh
	if strings.HasSuffix(fn, ".gz") {
		writer = gzip.NewWriter(writer)
		defer writer.(*gzip.Writer).Close()
	}

	var z *zip.Writer
	if strings.HasSuffix(fn, ".zip") {
		z = zip.NewWriter(fh)
		writer, err = z.Create("blocks.dat")
		if err != nil {
			return err
		}
	}

	log.Info("Writing blocks", "first", first, "last", last)
	if err := blockchain.ExportN(writer, first, last); err != nil {
		return err
	}

	if z != nil {
		log.Info("Writing state", "blockNumber", last)
		writer, err := z.Create("state.dat")
		if err != nil {
			return err
		}
		if err := exportState(blockchain, chainDb, writer, last); err != nil {
			return err
		}

		log.Info("Writing receipts", "blockNumber", last)
		writer, err = z.Create("receipts.dat")
		if err != nil {
			return err
		}
		if err := exportReceipts(blockchain, chainDb, writer, last); err != nil {
			return err
		}

		err = z.Close()
		if err != nil {
			return err
		}
	}

	log.Info("Exported blockchain to", "file", fn)
	return nil
}

func exportState(blockchain *core.BlockChain, chainDb ethdb.Database, writer io.Writer, blocknum uint64) error {
	sdb, err := blockchain.StateAt(blockchain.GetHeaderByNumber(blocknum).Root)
	if err != nil {
		return err
	}

	iter := state.NewNodeIterator(sdb)
	nodes, hashnodes := 0, 0
	for iter.Next() {
		nodes += 1
		if nodes%100000 == 0 {
			log.Info("Exported nodes", "count", nodes, "hashnodes", hashnodes)
		}
		if iter.Hash != (common.Hash{}) {
			hashnodes += 1
			entry, err := chainDb.Get(iter.Hash.Bytes())
			if err != nil {
				return err
			}
			writer.Write(entry)
		}
	}

	return nil
}

func exportReceipts(blockchain *core.BlockChain, chainDb ethdb.Database, writer io.Writer, blocknum uint64) error {
	count := 0
	var i uint64
	for i = 0; i <= blocknum; i++ {
		receipts := core.GetBlockReceipts(chainDb, blockchain.GetHeaderByNumber(i).Hash(), i)
		count += len(receipts)
		for _, receipt := range receipts {
			receipt.EncodeRLP(writer)
		}
		if i%100000 == 0 {
			log.Info("Exported receipts", "count", count, "blocknumber", i)
		}
	}
	return nil
}
