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
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/tests"
	"gopkg.in/urfave/cli.v1"
)

const (
	ErrorEVM              = 2
	ErrorVMConfig         = 3
	ErrorMissingBlockhash = 4

	ErrorJson = 10
	ErrorIO   = 11

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

func (n *NumberedError) Code() int {
	return n.errorCode
}

type input struct {
	Alloc core.GenesisAlloc  `json:"alloc,omitempty"`
	Env   *stEnv             `json:"env,omitempty"`
	Txs   types.Transactions `json:"txs,omitempty"`
}

func Main(ctx *cli.Context) error {
	// Configure the go-ethereum logger
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.Lvl(ctx.Int(VerbosityFlag.Name)))
	log.Root().SetHandler(glogger)

	var (
		err     error
		tracer  vm.Tracer
		baseDir = ""
	)
	var getTracer func(txIndex int, txHash common.Hash) (vm.Tracer, error)

	// If user specified a basedir, make sure it exists
	if ctx.IsSet(OutputBasedir.Name) {
		if base := ctx.String(OutputBasedir.Name); len(base) > 0 {
			err := os.MkdirAll(base, 0755) // //rw-r--r--
			if err != nil {
				return NewError(ErrorIO, fmt.Errorf("failed creating output basedir: %v", err))
			}
			baseDir = base
		}
	}
	if ctx.Bool(TraceFlag.Name) {
		// Configure the EVM logger
		logConfig := &vm.LogConfig{
			DisableStack:      ctx.Bool(TraceDisableStackFlag.Name),
			DisableMemory:     ctx.Bool(TraceDisableMemoryFlag.Name),
			DisableReturnData: ctx.Bool(TraceDisableReturnDataFlag.Name),
			Debug:             true,
		}
		var prevFile *os.File
		// This one closes the last file
		defer func() {
			if prevFile != nil {
				prevFile.Close()
			}
		}()
		getTracer = func(txIndex int, txHash common.Hash) (vm.Tracer, error) {
			if prevFile != nil {
				prevFile.Close()
			}
			traceFile, err := os.Create(path.Join(baseDir, fmt.Sprintf("trace-%d-%v.jsonl", txIndex, txHash.String())))
			if err != nil {
				return nil, NewError(ErrorIO, fmt.Errorf("failed creating trace-file: %v", err))
			}
			prevFile = traceFile
			return vm.NewJSONLogger(logConfig, traceFile), nil
		}
	} else {
		getTracer = func(txIndex int, txHash common.Hash) (tracer vm.Tracer, err error) {
			return nil, nil
		}
	}
	// We need to load three things: alloc, env and transactions. May be either in
	// stdin input or in files.
	// Check if anything needs to be read from stdin
	var (
		prestate Prestate
		txs      types.Transactions // txs to apply
		allocStr = ctx.String(InputAllocFlag.Name)

		envStr    = ctx.String(InputEnvFlag.Name)
		txStr     = ctx.String(InputTxsFlag.Name)
		inputData = &input{}
	)

	if allocStr == stdinSelector || envStr == stdinSelector || txStr == stdinSelector {
		decoder := json.NewDecoder(os.Stdin)
		decoder.Decode(inputData)
	}
	if allocStr != stdinSelector {
		inFile, err := os.Open(allocStr)
		if err != nil {
			return NewError(ErrorIO, fmt.Errorf("failed reading alloc file: %v", err))
		}
		defer inFile.Close()
		decoder := json.NewDecoder(inFile)
		if err := decoder.Decode(&inputData.Alloc); err != nil {
			return NewError(ErrorJson, fmt.Errorf("Failed unmarshaling alloc-file: %v", err))
		}
	}

	if envStr != stdinSelector {
		inFile, err := os.Open(envStr)
		if err != nil {
			return NewError(ErrorIO, fmt.Errorf("failed reading env file: %v", err))
		}
		defer inFile.Close()
		decoder := json.NewDecoder(inFile)
		var env stEnv
		if err := decoder.Decode(&env); err != nil {
			return NewError(ErrorJson, fmt.Errorf("Failed unmarshaling env-file: %v", err))
		}
		inputData.Env = &env
	}

	if txStr != stdinSelector {
		inFile, err := os.Open(txStr)
		if err != nil {
			return NewError(ErrorIO, fmt.Errorf("failed reading txs file: %v", err))
		}
		defer inFile.Close()
		decoder := json.NewDecoder(inFile)
		var txs types.Transactions
		if err := decoder.Decode(&txs); err != nil {
			return NewError(ErrorJson, fmt.Errorf("Failed unmarshaling txs-file: %v", err))
		}
		inputData.Txs = txs
	}

	prestate.Pre = inputData.Alloc
	prestate.Env = *inputData.Env
	txs = inputData.Txs

	// Iterate over all the tests, run them and aggregate the results
	vmConfig := vm.Config{
		Tracer: tracer,
		Debug:  (tracer != nil),
	}
	// Construct the chainconfig
	var chainConfig *params.ChainConfig
	if cConf, extraEips, err := tests.GetChainConfig(ctx.String(ForknameFlag.Name)); err != nil {
		return NewError(ErrorVMConfig, fmt.Errorf("Failed constructing chain configuration: %v", err))
	} else {
		chainConfig = cConf
		vmConfig.ExtraEips = extraEips
	}
	// Set the chain id
	chainConfig.ChainID = big.NewInt(ctx.Int64(ChainIDFlag.Name))

	// Run the test and aggregate the result
	state, result, err := prestate.Apply(vmConfig, chainConfig, txs, ctx.Int64(RewardFlag.Name), getTracer)
	if err != nil {
		return err
	}
	// Dump the excution result
	//postAlloc := state.DumpGenesisFormat(false, false, false)
	collector := make(Alloc)
	state.DumpToCollector(collector, false, false, false, nil, -1)
	return dispatchOutput(ctx, baseDir, result, collector)

}

type Alloc map[common.Address]core.GenesisAccount

func (g Alloc) OnRoot(common.Hash) {}

func (g Alloc) OnAccount(addr common.Address, dumpAccount state.DumpAccount) {
	balance, _ := new(big.Int).SetString(dumpAccount.Balance, 10)
	var storage map[common.Hash]common.Hash
	if dumpAccount.Storage != nil {
		storage = make(map[common.Hash]common.Hash)
		for k, v := range dumpAccount.Storage {
			storage[k] = common.HexToHash(v)
		}
	}
	genesisAccount := core.GenesisAccount{
		Code:    common.FromHex(dumpAccount.Code),
		Storage: storage,
		Balance: balance,
		Nonce:   dumpAccount.Nonce,
	}
	g[addr] = genesisAccount
}

// saveFile marshalls the object to the given file
func saveFile(baseDir, filename string, data interface{}) error {
	b, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		return NewError(ErrorJson, fmt.Errorf("failed marshalling output: %v", err))
	}
	if err = ioutil.WriteFile(path.Join(baseDir, filename), b, 0644); err != nil {
		return NewError(ErrorIO, fmt.Errorf("failed writing output: %v", err))
	}
	return nil
}

// dispatchOutput writes the output data to either stderr or stdout, or to the specified
// files
func dispatchOutput(ctx *cli.Context, baseDir string, result *ExecutionResult, alloc Alloc) error {
	stdOutObject := make(map[string]interface{})
	stdErrObject := make(map[string]interface{})
	dispatch := func(baseDir, fName, name string, obj interface{}) error {
		switch fName {
		case "stdout":
			stdOutObject[name] = obj
		case "stderr":
			stdErrObject[name] = obj
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
	if len(stdOutObject) > 0 {
		b, err := json.MarshalIndent(stdOutObject, "", " ")
		if err != nil {
			return NewError(ErrorJson, fmt.Errorf("failed marshalling output: %v", err))
		}
		os.Stdout.Write(b)
	}
	if len(stdErrObject) > 0 {
		b, err := json.MarshalIndent(stdErrObject, "", " ")
		if err != nil {
			return NewError(ErrorJson, fmt.Errorf("failed marshalling output: %v", err))
		}
		os.Stderr.Write(b)
	}
	return nil
}
