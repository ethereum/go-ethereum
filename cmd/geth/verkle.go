// Copyright 2020 The go-ethereum Authors
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

package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
	tutils "github.com/ethereum/go-ethereum/trie/utils"
	"github.com/gballet/go-verkle"
	"github.com/holiman/uint256"
	"github.com/shirou/gopsutil/mem"
	cli "github.com/urfave/cli/v2"
)

var (
	verkleCommand = &cli.Command{
		Name:        "verkle",
		Usage:       "A set of experimental verkle tree management commands",
		Category:    "MISCELLANEOUS COMMANDS",
		Description: "",
		Subcommands: []*cli.Command{
			{
				Name:      "to-verkle",
				Usage:     "use the snapshot to compute a translation of a MPT into a verkle tree",
				ArgsUsage: "<root>",
				Action:    convertToVerkle,
				Flags: utils.GroupFlags([]cli.Flag{
					utils.VerkleConversionInsertRangeStartFlag,
					utils.VerkleConversionInsertRangeSizeFlag,
				}, utils.NetworkFlags, utils.DatabasePathFlags),
				Description: `
geth snapshot to-verkle <state-root>
This command takes a snapshot and inserts its values in a fresh verkle tree.

The argument is interpreted as the root hash. If none is provided, the latest
block is used.
 `,
			},
			{
				Name:      "verify-verkle",
				Usage:     "verify the translation of a MPT into a verkle tree",
				ArgsUsage: "<root>",
				Action:    verifyVerkle,
				Flags: utils.GroupFlags([]cli.Flag{
					utils.VerkleConversionInsertRangeStartFlag,
					utils.VerkleConversionInsertRangeSizeFlag,
				}, utils.NetworkFlags, utils.DatabasePathFlags),
				Description: `
geth snapshot to-verkle <state-root>
This command takes a snapshot and inserts its values in a fresh verkle tree.

The argument is interpreted as the root hash. If none is provided, the latest
block is used.
 `,
			},
		},
	}
)

// checkAccount iterates the snap data layers, and looks up the given account
// across all layers.
func checkAccount(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return errors.New("need <address|hash> arg")
	}
	var (
		hash common.Hash
		addr common.Address
	)
	switch arg := ctx.Args().First(); len(arg) {
	case 40, 42:
		addr = common.HexToAddress(arg)
		hash = crypto.Keccak256Hash(addr.Bytes())
	case 64, 66:
		hash = common.HexToHash(arg)
	default:
		return errors.New("malformed address or hash")
	}
	stack, _ := makeConfigNode(ctx)
	defer stack.Close()
	chaindb := utils.MakeChainDatabase(ctx, stack, true)
	defer chaindb.Close()
	start := time.Now()
	log.Info("Checking difflayer journal", "address", addr, "hash", hash)
	if err := snapshot.CheckJournalAccount(chaindb, hash); err != nil {
		return err
	}
	log.Info("Checked the snapshot journalled storage", "time", common.PrettyDuration(time.Since(start)))
	return nil
}

func convertToVerkle(ctx *cli.Context) error {
	stack, _ := makeConfigNode(ctx)
	defer stack.Close()

	chaindb := utils.MakeChainDatabase(ctx, stack, false)
	if chaindb == nil {
		return errors.New("nil chaindb")
	}
	headBlock := rawdb.ReadHeadBlock(chaindb)
	if headBlock == nil {
		log.Error("Failed to load head block")
		return errors.New("no head block")
	}
	if ctx.NArg() > 1 {
		log.Error("Too many arguments given")
		return errors.New("too many arguments")
	}
	var (
		root common.Hash
		err  error
	)
	if ctx.NArg() == 1 {
		root, err = parseRoot(ctx.Args().First())
		if err != nil {
			log.Error("Failed to resolve state root", "error", err)
			return err
		}
		log.Info("Start traversing the state", "root", root)
	} else {
		root = headBlock.Root()
		log.Info("Start traversing the state", "root", root, "number", headBlock.NumberU64())
	}

	var (
		accounts   int
		lastReport time.Time
		start      = time.Now()
		rangeStart = ctx.Uint64(utils.VerkleConversionInsertRangeStartFlag.Name)
		rangeEnd   = rangeStart + ctx.Uint64(utils.VerkleConversionInsertRangeSizeFlag.Name)
		wg         sync.WaitGroup
		flushError error
	)

	if rangeEnd > 256 {
		rangeEnd = 256
	}

	flushCh := make(chan verkle.VerkleNode)
	saveverkle := func(node verkle.VerkleNode) {
		flushCh <- node
	}
	var flushWg sync.WaitGroup
	flushWg.Add(1)
	go func() {
		for node := range flushCh {
			comm := node.ComputeCommitment()
			s, err := node.Serialize()
			if err != nil {
				panic(err)
			}
			commB := comm.Bytes()
			if err := chaindb.Put(commB[:], s); err != nil {
				flushError = err
				break
			}
		}
		flushWg.Done()
	}()

	snaptree, err := snapshot.New(chaindb, trie.NewDatabase(chaindb), 256, root, false, false, false)
	if err != nil {
		return err
	}
	accIt, err := snaptree.AccountIterator(root, common.Hash{})
	if err != nil {
		return err
	}
	defer accIt.Release()

	type treeHugger struct {
		node *verkle.LeafNode
		stem []byte
	}
	treeHuggers := make([]chan *treeHugger, runtime.NumCPU())
	subRoots := make([]*verkle.InternalNode, runtime.NumCPU())
	rootPerCPU := (256 + runtime.NumCPU() - 1) / runtime.NumCPU()
	for i := range treeHuggers {
		treeHuggers[i] = make(chan *treeHugger, 128)
		subRoots[i] = verkle.New().(*verkle.InternalNode)

		// save references for the goroutine to capture
		hugger := treeHuggers[i]
		root := subRoots[i]
		wg.Add(1)

		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for hug := range hugger {
				select {
				case <-ticker.C:
					// Check the memory usage every 10 seconds. If it
					// goes above a given watermark, flush the tree's
					// lower nodes to disk.
					v, _ := mem.VirtualMemory()

					// Compute flushing depth
					// 1 if > 80%, 2 if 80% < && > 60%...
					// don't bother cleaning up above 60%
					depth := 5 - uint8(v.UsedPercent/20)

					if depth < 3 {
						root.FlushAtDepth(depth, saveverkle)
					}
				default:
				}

				if uint64(hug.stem[0]) < rangeStart && uint64(hug.stem[0]) > rangeEnd {
					// skip stem outside the insertion range
					continue
				}

				hug.node.ComputeCommitment()
				hashed := hug.node.ToHashedNode()
				flushCh <- hug.node
				root.InsertStem(hug.stem, hashed, chaindb.Get)
			}
			wg.Done()
		}()
	}

	// Process all accounts sequentially
	for accIt.Next() {
		accounts += 1
		acc, err := snapshot.FullAccount(accIt.Account())
		if err != nil {
			log.Error("Invalid account encountered during traversal", "error", err)
			return err
		}

		// Store the basic account data
		var (
			nonce, balance, version [32]byte
			newValues               = make([][]byte, 256)
		)
		newValues[0] = version[:]
		newValues[1] = balance[:]
		newValues[2] = nonce[:]
		newValues[4] = version[:] // memory-saving trick: by default, an account has 0 size
		binary.LittleEndian.PutUint64(nonce[:8], acc.Nonce)
		for i, b := range acc.Balance.Bytes() {
			balance[len(acc.Balance.Bytes())-1-i] = b
		}
		addr := rawdb.ReadPreimage(chaindb, accIt.Hash())
		if addr == nil {
			return fmt.Errorf("could not find preimage for address %x %v %v", accIt.Hash(), acc, accIt.Error())
		}
		stem := tutils.GetTreeKeyVersion(addr)

		// Store the account code if present
		if !bytes.Equal(acc.CodeHash, emptyCode) {
			code := rawdb.ReadCode(chaindb, common.BytesToHash(acc.CodeHash))
			chunks := trie.ChunkifyCode(code)

			for i := 0; i < 128 && i < len(chunks)/32; i++ {
				newValues[128+i] = chunks[32*i : 32*(i+1)]
			}

			for i := 128; i < len(chunks)/32; {
				values := make([][]byte, 256)
				chunkkey := tutils.GetTreeKeyCodeChunk(addr, uint256.NewInt(uint64(i)))
				j := i
				for ; (j-i) < 256 && j < len(chunks)/32; j++ {
					values[(j-128)%256] = chunks[32*j : 32*(j+1)]
				}
				i = j

				// Otherwise, store the previous group in the tree with a
				// stem insertion.
				treeHuggers[int(chunkkey[0])/rootPerCPU] <- &treeHugger{stem: chunkkey[:31], node: verkle.NewLeafNode(chunkkey[:31], values)}
			}

			// Write the code size in the account header group
			var size [32]byte
			newValues[4] = size[:]
			binary.LittleEndian.PutUint64(size[:8], uint64(len(code)))
		}

		// Save every slot into the tree
		if !bytes.Equal(acc.Root, emptyRoot[:]) {
			var (
				laststem [31]byte
				values   = make([][]byte, 256)
			)
			copy(laststem[:], stem)

			storageIt, err := snaptree.StorageIterator(root, accIt.Hash(), common.Hash{})
			if err != nil {
				log.Error("Failed to open storage trie", "root", acc.Root, "error", err)
				return err
			}
			for storageIt.Next() {
				slotnr := rawdb.ReadPreimage(chaindb, storageIt.Hash())
				if slotnr == nil {
					return fmt.Errorf("could not find preimage for slot %x", storageIt.Hash())
				}
				slotkey := tutils.GetTreeKeyStorageSlot(addr, uint256.NewInt(0).SetBytes(slotnr))

				var value [32]byte
				copy(value[:len(storageIt.Slot())-1], storageIt.Slot())

				// if the slot belongs to the header group, store it there
				if bytes.Equal(slotkey[:31], stem) {
					newValues[int(slotkey[31])] = value[:]
					continue
				}

				// if the slot belongs to the same group as the previous
				// one, add it to the current group of values.
				if bytes.Equal(laststem[:], slotkey[:31]) {
					values[slotkey[31]] = value[:]
					continue
				}

				// flush the previous group, iff it's not the header group
				if !bytes.Equal(stem[:31], laststem[:]) {
					treeHuggers[int(laststem[0])/rootPerCPU] <- &treeHugger{stem: laststem[:], node: verkle.NewLeafNode(laststem[:], values)}
				}
			}
			if !bytes.Equal(laststem[:31], stem[:31]) {
				treeHuggers[int(laststem[0])/rootPerCPU] <- &treeHugger{stem: laststem[:], node: verkle.NewLeafNode(laststem[:], values)}
			}
			storageIt.Release()
			if storageIt.Error() != nil {
				log.Error("Failed to traverse storage trie", "root", acc.Root, "error", storageIt.Error())
				return storageIt.Error()
			}
		}
		// Finish with storing the complete account header group inside the tree.
		treeHuggers[int(stem[0])/rootPerCPU] <- &treeHugger{stem: stem[:], node: verkle.NewLeafNode(stem[:31], newValues)}

		if time.Since(lastReport) > time.Second*8 {
			log.Info("Traversing state", "accounts", accounts, "elapsed", common.PrettyDuration(time.Since(start)))
			lastReport = time.Now()
		}
	}
	if accIt.Error() != nil {
		log.Error("Failed to compute commitment", "root", root, "error", accIt.Error())
		return accIt.Error()
	}
	log.Info("Wrote all leaves", "accounts", accounts, "elapsed", common.PrettyDuration(time.Since(start)))
	for _, hugger := range treeHuggers {
		close(hugger)
	}
	wg.Wait()
	if flushError != nil {
		log.Error("Error encountered by the flusing goroutine", "error", flushError)
	}

	vRoot := verkle.MergeTrees(subRoots)
	vRoot.ComputeCommitment()
	vRoot.(*verkle.InternalNode).Flush(saveverkle)
	close(flushCh)
	flushWg.Wait()

	if rangeStart != 0 || rangeEnd != 256 {
		children := vRoot.(*verkle.InternalNode).Children()
		// Print partial subtree root commitments, as only a partial tree has been built
		log.Info("Conversion complete", "accounts", accounts, "elapsed", common.PrettyDuration(time.Since(start)))
		for i := rangeStart; i < rangeEnd; i++ {
			log.Info("Root commitment at depth 1", "offset", i, "commitment", fmt.Sprintf("%x", children[i].ComputeCommitment().Bytes()))
		}
	} else {
		log.Info("Conversion complete", "root commitment", fmt.Sprintf("%x", vRoot.ComputeCommitment().Bytes()), "accounts", accounts, "elapsed", common.PrettyDuration(time.Since(start)))
	}
	return nil
}

var zero [32]byte

// recurse into each child to ensure they can be loaded from the db. The tree isn't rebuilt
// (only its nodes are loaded) so there is no need to flush them, the garbage collector should
// take care of that for us.
func checkChildren(root verkle.VerkleNode, resolver verkle.NodeResolverFn) error {
	switch node := root.(type) {
	case *verkle.InternalNode:
		for i, child := range node.Children() {
			childC := child.ComputeCommitment().Bytes()

			childS, err := resolver(childC[:])
			if bytes.Equal(childC[:], zero[:]) {
				continue
			}
			if err != nil {
				return fmt.Errorf("could not find child %x in db: %w", childC, err)
			}
			// depth is set to 0, the tree isn't rebuilt so it's not a problem
			childN, err := verkle.ParseNode(childS, 0, childC[:])
			if err != nil {
				return fmt.Errorf("decode error child %x in db: %w", child.ComputeCommitment().Bytes(), err)
			}
			if err := checkChildren(childN, resolver); err != nil {
				return fmt.Errorf("%x%w", i, err) // write the path to the erroring node
			}
		}
	case *verkle.LeafNode:
		// sanity check: ensure at least one value is non-zero

		for i := 0; i < verkle.NodeWidth; i++ {
			if len(node.Value(i)) != 0 {
				return nil
			}
		}
		return fmt.Errorf("Both balance and nonce are 0")
	case verkle.Empty:
		// nothing to do
	default:
		return fmt.Errorf("unsupported type encountered %v", root)
	}

	return nil
}

func verifyVerkle(ctx *cli.Context) error {
	stack, _ := makeConfigNode(ctx)
	defer stack.Close()

	chaindb := utils.MakeChainDatabase(ctx, stack, true)
	headBlock := rawdb.ReadHeadBlock(chaindb)
	if headBlock == nil {
		log.Error("Failed to load head block")
		return errors.New("no head block")
	}
	if ctx.NArg() > 1 {
		log.Error("Too many arguments given")
		return errors.New("too many arguments")
	}
	var (
		rootC common.Hash
		err   error
	)
	if ctx.NArg() == 1 {
		rootC, err = parseRoot(ctx.Args().First())
		if err != nil {
			log.Error("Failed to resolve state root", "error", err)
			return err
		}
		log.Info("Rebuilding the tree", "root", rootC)
	} else {
		rootC = headBlock.Root()
		log.Info("Rebuilding the tree", "root", rootC, "number", headBlock.NumberU64())
	}

	var (
		//start      = time.Now()
		rangeStart = ctx.Uint64(utils.VerkleConversionInsertRangeStartFlag.Name)
		rangeEnd   = rangeStart + ctx.Uint64(utils.VerkleConversionInsertRangeSizeFlag.Name)
	)

	if rangeEnd > 256 {
		rangeEnd = 256
	}

	serializedRoot, err := chaindb.Get(rootC[:])
	if err != nil {
		return err
	}
	root, err := verkle.ParseNode(serializedRoot, 0, rootC[:])
	if err != nil {
		return err
	}

	if err := checkChildren(root, chaindb.Get); err != nil {
		log.Error("Could not rebuild the tree from the database", "err", err)
		return err
	}

	log.Info("Tree was rebuilt from the database")
	return nil
}
