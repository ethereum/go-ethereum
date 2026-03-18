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

package t8ntool

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
	"github.com/ethereum/go-ethereum/eth/tracers/native"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/tests"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/database"
	"github.com/holiman/uint256"
	"github.com/urfave/cli/v2"
)

const (
	ErrorEVM              = 2
	ErrorConfig           = 3
	ErrorMissingBlockhash = 4

	ErrorJson = 10
	ErrorIO   = 11
	ErrorRlp  = 12

	stdinSelector = "stdin"
)

type NumberedError struct {
	errorCode int
	err       error
}

func NewError(errorCode int, err error) *NumberedError {
	return &NumberedError{errorCode, err}
}

func (n *NumberedError) Error() string {
	return fmt.Sprintf("ERROR(%d): %v", n.errorCode, n.err.Error())
}

func (n *NumberedError) ExitCode() int {
	return n.errorCode
}

// compile-time conformance test
var (
	_ cli.ExitCoder = (*NumberedError)(nil)
)

type input struct {
	Alloc types.GenesisAlloc            `json:"alloc,omitempty"`
	Env   *stEnv                        `json:"env,omitempty"`
	BT    map[common.Hash]hexutil.Bytes `json:"vkt,omitempty"`
	Txs   []*txWithKey                  `json:"txs,omitempty"`
	TxRlp string                        `json:"txsRlp,omitempty"`
}

func Transition(ctx *cli.Context) error {
	baseDir, err := createBasedir(ctx)
	if err != nil {
		return NewError(ErrorIO, fmt.Errorf("failed creating output basedir: %v", err))
	}
	// We need to load three things: alloc, env and transactions. May be either in
	// stdin input or in files.
	// Check if anything needs to be read from stdin
	var (
		prestate  Prestate
		txIt      txIterator // txs to apply
		allocStr  = ctx.String(InputAllocFlag.Name)
		btStr     = ctx.String(InputBTFlag.Name)
		envStr    = ctx.String(InputEnvFlag.Name)
		txStr     = ctx.String(InputTxsFlag.Name)
		inputData = &input{}
	)
	// Figure out the prestate alloc
	if allocStr == stdinSelector || btStr == stdinSelector || envStr == stdinSelector || txStr == stdinSelector {
		decoder := json.NewDecoder(os.Stdin)
		if err := decoder.Decode(inputData); err != nil {
			return NewError(ErrorJson, fmt.Errorf("failed unmarshalling stdin: %v", err))
		}
	}
	if allocStr != stdinSelector {
		if err := readFile(allocStr, "alloc", &inputData.Alloc); err != nil {
			return err
		}
	}
	prestate.Pre = inputData.Alloc

	if btStr != stdinSelector && btStr != "" {
		if err := readFile(btStr, "BT", &inputData.BT); err != nil {
			return err
		}
	}
	prestate.TreeLeaves = inputData.BT

	// Set the block environment
	if envStr != stdinSelector {
		var env stEnv
		if err := readFile(envStr, "env", &env); err != nil {
			return err
		}
		inputData.Env = &env
	}
	prestate.Env = *inputData.Env

	vmConfig := vm.Config{}
	// Construct the chainconfig
	var chainConfig *params.ChainConfig
	if cConf, extraEips, err := tests.GetChainConfig(ctx.String(ForknameFlag.Name)); err != nil {
		return NewError(ErrorConfig, fmt.Errorf("failed constructing chain configuration: %v", err))
	} else {
		chainConfig = cConf
		vmConfig.ExtraEips = extraEips
	}

	// Set the chain id
	chainConfig.ChainID = big.NewInt(ctx.Int64(ChainIDFlag.Name))

	if txIt, err = loadTransactions(txStr, inputData, chainConfig); err != nil {
		return err
	}
	if err := applyLondonChecks(&prestate.Env, chainConfig); err != nil {
		return err
	}
	if err := applyShanghaiChecks(&prestate.Env, chainConfig); err != nil {
		return err
	}
	if err := applyMergeChecks(&prestate.Env, chainConfig); err != nil {
		return err
	}
	if err := applyCancunChecks(&prestate.Env, chainConfig); err != nil {
		return err
	}

	// Configure tracer
	var tracer *tracers.Tracer
	if ctx.IsSet(TraceTracerFlag.Name) { // Custom tracing
		config := json.RawMessage(ctx.String(TraceTracerConfigFlag.Name))
		innerTracer, err := tracers.DefaultDirectory.New(ctx.String(TraceTracerFlag.Name),
			nil, config, chainConfig)
		if err != nil {
			return NewError(ErrorConfig, fmt.Errorf("failed instantiating tracer: %v", err))
		}
		tracer = newResultWriter(baseDir, innerTracer)
	} else if ctx.Bool(TraceFlag.Name) { // JSON opcode tracing
		logConfig := &logger.Config{
			DisableStack:     ctx.Bool(TraceDisableStackFlag.Name),
			EnableMemory:     ctx.Bool(TraceEnableMemoryFlag.Name),
			EnableReturnData: ctx.Bool(TraceEnableReturnDataFlag.Name),
		}
		if ctx.Bool(TraceEnableCallFramesFlag.Name) {
			tracer = newFileWriter(baseDir, func(out io.Writer) *tracing.Hooks {
				return logger.NewJSONLoggerWithCallFrames(logConfig, out)
			})
		} else {
			tracer = newFileWriter(baseDir, func(out io.Writer) *tracing.Hooks {
				return logger.NewJSONLogger(logConfig, out)
			})
		}
	}
	// Configure opcode counter
	var opcodeTracer *tracers.Tracer
	if ctx.IsSet(OpcodeCountFlag.Name) && ctx.String(OpcodeCountFlag.Name) != "" {
		opcodeTracer = native.NewOpcodeCounter()
		if tracer != nil {
			// If we have an existing tracer, multiplex with the opcode tracer
			mux, _ := native.NewMuxTracer([]string{"trace", "opcode"}, []*tracers.Tracer{tracer, opcodeTracer})
			vmConfig.Tracer = mux.Hooks
		} else {
			vmConfig.Tracer = opcodeTracer.Hooks
		}
	} else if tracer != nil {
		vmConfig.Tracer = tracer.Hooks
	}
	// Run the test and aggregate the result
	s, result, body, err := prestate.Apply(vmConfig, chainConfig, txIt, ctx.Int64(RewardFlag.Name))
	if err != nil {
		return err
	}
	// Write opcode counts if enabled
	if opcodeTracer != nil {
		fname := ctx.String(OpcodeCountFlag.Name)
		result, err := opcodeTracer.GetResult()
		if err != nil {
			return NewError(ErrorJson, fmt.Errorf("failed getting opcode counts: %v", err))
		}
		if err := saveFile(baseDir, fname, result); err != nil {
			return err
		}
	}
	// Dump the execution result
	var (
		collector = make(Alloc)
		btleaves  map[common.Hash]hexutil.Bytes
	)
	isBinary := chainConfig.IsVerkle(big.NewInt(int64(prestate.Env.Number)), prestate.Env.Timestamp)
	if !isBinary {
		s.DumpToCollector(collector, nil)
	} else {
		btleaves = make(map[common.Hash]hexutil.Bytes)
		if err := s.DumpBinTrieLeaves(btleaves); err != nil {
			return err
		}
	}

	return dispatchOutput(ctx, baseDir, result, collector, body, btleaves)
}

func applyLondonChecks(env *stEnv, chainConfig *params.ChainConfig) error {
	if !chainConfig.IsLondon(big.NewInt(int64(env.Number))) {
		return nil
	}
	// Sanity check, to not `panic` in state_transition
	if env.BaseFee != nil {
		// Already set, base fee has precedent over parent base fee.
		return nil
	}
	if env.ParentBaseFee == nil || env.Number == 0 {
		return NewError(ErrorConfig, errors.New("EIP-1559 config but missing 'parentBaseFee' in env section"))
	}
	env.BaseFee = eip1559.CalcBaseFee(chainConfig, &types.Header{
		Number:   new(big.Int).SetUint64(env.Number - 1),
		BaseFee:  env.ParentBaseFee,
		GasUsed:  env.ParentGasUsed,
		GasLimit: env.ParentGasLimit,
	})
	return nil
}

func applyShanghaiChecks(env *stEnv, chainConfig *params.ChainConfig) error {
	if !chainConfig.IsShanghai(big.NewInt(int64(env.Number)), env.Timestamp) {
		return nil
	}
	if env.Withdrawals == nil {
		return NewError(ErrorConfig, errors.New("Shanghai config but missing 'withdrawals' in env section"))
	}
	return nil
}

func applyMergeChecks(env *stEnv, chainConfig *params.ChainConfig) error {
	isMerged := chainConfig.TerminalTotalDifficulty != nil && chainConfig.TerminalTotalDifficulty.BitLen() == 0
	if !isMerged {
		// pre-merge: If difficulty was not provided by caller, we need to calculate it.
		if env.Difficulty != nil {
			// already set
			return nil
		}
		switch {
		case env.ParentDifficulty == nil:
			return NewError(ErrorConfig, errors.New("currentDifficulty was not provided, and cannot be calculated due to missing parentDifficulty"))
		case env.Number == 0:
			return NewError(ErrorConfig, errors.New("currentDifficulty needs to be provided for block number 0"))
		case env.Timestamp <= env.ParentTimestamp:
			return NewError(ErrorConfig, fmt.Errorf("currentDifficulty cannot be calculated -- currentTime (%d) needs to be after parent time (%d)",
				env.Timestamp, env.ParentTimestamp))
		}
		env.Difficulty = calcDifficulty(chainConfig, env.Number, env.Timestamp,
			env.ParentTimestamp, env.ParentDifficulty, env.ParentUncleHash)
		return nil
	}
	// post-merge:
	// - random must be supplied
	// - difficulty must be zero
	switch {
	case env.Random == nil:
		return NewError(ErrorConfig, errors.New("post-merge requires currentRandom to be defined in env"))
	case env.Difficulty != nil && env.Difficulty.BitLen() != 0:
		return NewError(ErrorConfig, errors.New("post-merge difficulty must be zero (or omitted) in env"))
	}
	env.Difficulty = nil
	return nil
}

func applyCancunChecks(env *stEnv, chainConfig *params.ChainConfig) error {
	if !chainConfig.IsCancun(big.NewInt(int64(env.Number)), env.Timestamp) {
		env.ParentBeaconBlockRoot = nil // un-set it if it has been set too early
		return nil
	}
	// Post-cancun
	// We require EIP-4788 beacon root to be set in the env
	if env.ParentBeaconBlockRoot == nil {
		return NewError(ErrorConfig, errors.New("post-cancun env requires parentBeaconBlockRoot to be set"))
	}
	return nil
}

type Alloc map[common.Address]types.Account

func (g Alloc) OnRoot(common.Hash) {}

func (g Alloc) OnAccount(addr *common.Address, dumpAccount state.DumpAccount) {
	if addr == nil {
		return
	}
	balance, _ := new(big.Int).SetString(dumpAccount.Balance, 0)
	var storage map[common.Hash]common.Hash
	if dumpAccount.Storage != nil {
		storage = make(map[common.Hash]common.Hash, len(dumpAccount.Storage))
		for k, v := range dumpAccount.Storage {
			storage[k] = common.HexToHash(v)
		}
	}
	genesisAccount := types.Account{
		Code:    dumpAccount.Code,
		Storage: storage,
		Balance: balance,
		Nonce:   dumpAccount.Nonce,
	}
	g[*addr] = genesisAccount
}

// saveFile marshals the object to the given file
func saveFile(baseDir, filename string, data interface{}) error {
	b, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		return NewError(ErrorJson, fmt.Errorf("failed marshalling output: %v", err))
	}
	location := filepath.Join(baseDir, filename)
	if err = os.WriteFile(location, b, 0644); err != nil {
		return NewError(ErrorIO, fmt.Errorf("failed writing output: %v", err))
	}
	log.Info("Wrote file", "file", location)
	return nil
}

// dispatchOutput writes the output data to either stderr or stdout, or to the specified
// files
func dispatchOutput(ctx *cli.Context, baseDir string, result *ExecutionResult, alloc Alloc, body hexutil.Bytes, bt map[common.Hash]hexutil.Bytes) error {
	stdOutObject := make(map[string]interface{})
	stdErrObject := make(map[string]interface{})
	dispatch := func(baseDir, fName, name string, obj interface{}) error {
		switch fName {
		case "stdout":
			stdOutObject[name] = obj
		case "stderr":
			stdErrObject[name] = obj
		case "":
			// don't save
		default: // save to file
			if err := saveFile(baseDir, fName, obj); err != nil {
				return err
			}
		}
		return nil
	}
	if err := dispatch(baseDir, ctx.String(OutputAllocFlag.Name), "alloc", alloc); err != nil {
		return err
	}
	if err := dispatch(baseDir, ctx.String(OutputResultFlag.Name), "result", result); err != nil {
		return err
	}
	if err := dispatch(baseDir, ctx.String(OutputBodyFlag.Name), "body", body); err != nil {
		return err
	}
	// Only write bt output if we actually have binary trie leaves
	if bt != nil {
		if err := dispatch(baseDir, ctx.String(OutputBTFlag.Name), "vkt", bt); err != nil {
			return err
		}
	}

	if len(stdOutObject) > 0 {
		b, err := json.MarshalIndent(stdOutObject, "", "  ")
		if err != nil {
			return NewError(ErrorJson, fmt.Errorf("failed marshalling output: %v", err))
		}
		os.Stdout.Write(b)
		os.Stdout.WriteString("\n")
	}
	if len(stdErrObject) > 0 {
		b, err := json.MarshalIndent(stdErrObject, "", "  ")
		if err != nil {
			return NewError(ErrorJson, fmt.Errorf("failed marshalling output: %v", err))
		}
		os.Stderr.Write(b)
		os.Stderr.WriteString("\n")
	}
	return nil
}

// BinKey computes the tree key given an address and an optional slot number.
func BinKey(ctx *cli.Context) error {
	if ctx.Args().Len() == 0 || ctx.Args().Len() > 2 {
		return errors.New("invalid number of arguments: expecting an address and an optional slot number")
	}

	addr, err := hexutil.Decode(ctx.Args().Get(0))
	if err != nil {
		return fmt.Errorf("error decoding address: %w", err)
	}

	if ctx.Args().Len() == 2 {
		slot, err := hexutil.Decode(ctx.Args().Get(1))
		if err != nil {
			return fmt.Errorf("error decoding slot: %w", err)
		}
		fmt.Printf("%#x\n", bintrie.GetBinaryTreeKeyStorageSlot(common.BytesToAddress(addr), slot))
	} else {
		fmt.Printf("%#x\n", bintrie.GetBinaryTreeKeyBasicData(common.BytesToAddress(addr)))
	}
	return nil
}

// BinKeys computes a set of tree keys given a genesis alloc.
func BinKeys(ctx *cli.Context) error {
	var allocStr = ctx.String(InputAllocFlag.Name)
	var alloc core.GenesisAlloc
	// Figure out the prestate alloc
	if allocStr == stdinSelector {
		decoder := json.NewDecoder(os.Stdin)
		if err := decoder.Decode(&alloc); err != nil {
			return NewError(ErrorJson, fmt.Errorf("failed unmarshaling stdin: %v", err))
		}
	}
	if allocStr != stdinSelector {
		if err := readFile(allocStr, "alloc", &alloc); err != nil {
			return err
		}
	}
	db := triedb.NewDatabase(rawdb.NewMemoryDatabase(), triedb.VerkleDefaults)
	defer db.Close()

	bt, err := genBinTrieFromAlloc(alloc, db)
	if err != nil {
		return fmt.Errorf("error generating bt: %w", err)
	}

	collector := make(map[common.Hash]hexutil.Bytes)
	it, err := bt.NodeIterator(nil)
	if err != nil {
		panic(err)
	}
	for it.Next(true) {
		if it.Leaf() {
			collector[common.BytesToHash(it.LeafKey())] = it.LeafBlob()
		}
	}

	output, err := json.MarshalIndent(collector, "", "")
	if err != nil {
		return fmt.Errorf("error outputting tree: %w", err)
	}

	fmt.Println(string(output))

	return nil
}

// BinTrieRoot computes the root of a Binary Trie from a genesis alloc.
func BinTrieRoot(ctx *cli.Context) error {
	var allocStr = ctx.String(InputAllocFlag.Name)
	var alloc core.GenesisAlloc
	if allocStr == stdinSelector {
		decoder := json.NewDecoder(os.Stdin)
		if err := decoder.Decode(&alloc); err != nil {
			return NewError(ErrorJson, fmt.Errorf("failed unmarshaling stdin: %v", err))
		}
	}
	if allocStr != stdinSelector {
		if err := readFile(allocStr, "alloc", &alloc); err != nil {
			return err
		}
	}
	db := triedb.NewDatabase(rawdb.NewMemoryDatabase(), triedb.VerkleDefaults)
	defer db.Close()

	bt, err := genBinTrieFromAlloc(alloc, db)
	if err != nil {
		return fmt.Errorf("error generating bt: %w", err)
	}
	fmt.Println(bt.Hash().Hex())

	return nil
}

// TODO(@CPerezz): Should this go to `bintrie` module?
func genBinTrieFromAlloc(alloc core.GenesisAlloc, db database.NodeDatabase) (*bintrie.BinaryTrie, error) {
	bt, err := bintrie.NewBinaryTrie(types.EmptyBinaryHash, db)
	if err != nil {
		return nil, err
	}
	for addr, acc := range alloc {
		for slot, value := range acc.Storage {
			err := bt.UpdateStorage(addr, slot.Bytes(), value.Big().Bytes())
			if err != nil {
				return nil, fmt.Errorf("error inserting storage: %w", err)
			}
		}
		account := &types.StateAccount{
			Balance:  uint256.MustFromBig(acc.Balance),
			Nonce:    acc.Nonce,
			CodeHash: crypto.Keccak256Hash(acc.Code).Bytes(),
			Root:     common.Hash{},
		}
		err := bt.UpdateAccount(addr, account, len(acc.Code))
		if err != nil {
			return nil, fmt.Errorf("error inserting account: %w", err)
		}
		err = bt.UpdateContractCode(addr, common.BytesToHash(account.CodeHash), acc.Code)
		if err != nil {
			return nil, fmt.Errorf("error inserting code: %w", err)
		}
	}
	return bt, nil
}

// BinaryCodeChunkKey computes the tree key of a code-chunk for a given address.
func BinaryCodeChunkKey(ctx *cli.Context) error {
	if ctx.Args().Len() == 0 || ctx.Args().Len() > 2 {
		return errors.New("invalid number of arguments: expecting an address and an code-chunk number")
	}

	addr, err := hexutil.Decode(ctx.Args().Get(0))
	if err != nil {
		return fmt.Errorf("error decoding address: %w", err)
	}
	chunkNumberBytes, err := hexutil.Decode(ctx.Args().Get(1))
	if err != nil {
		return fmt.Errorf("error decoding chunk number: %w", err)
	}
	var chunkNumber uint256.Int
	chunkNumber.SetBytes(chunkNumberBytes)

	fmt.Printf("%#x\n", bintrie.GetBinaryTreeKeyCodeChunk(common.BytesToAddress(addr), &chunkNumber))

	return nil
}

// BinaryCodeChunkCode returns the code chunkification for a given code.
func BinaryCodeChunkCode(ctx *cli.Context) error {
	if ctx.Args().Len() == 0 || ctx.Args().Len() > 1 {
		return errors.New("invalid number of arguments: expecting a bytecode")
	}

	bytecode, err := hexutil.Decode(ctx.Args().Get(0))
	if err != nil {
		return fmt.Errorf("error decoding address: %w", err)
	}

	chunkedCode := bintrie.ChunkifyCode(bytecode)
	fmt.Printf("%#x\n", chunkedCode)

	return nil
}
