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
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
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
	Alloc core.GenesisAlloc `json:"alloc,omitempty"`
	Env   *stEnv            `json:"env,omitempty"`
	Txs   []*txWithKey      `json:"txs,omitempty"`
	TxRlp string            `json:"txsRlp,omitempty"`
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
	// Figure out the prestate alloc
	if allocStr == stdinSelector || envStr == stdinSelector || txStr == stdinSelector {
		decoder := json.NewDecoder(os.Stdin)
		if err := decoder.Decode(inputData); err != nil {
			return NewError(ErrorJson, fmt.Errorf("failed unmarshaling stdin: %v", err))
		}
	}
	if allocStr != stdinSelector {
		inFile, err := os.Open(allocStr)
		if err != nil {
			return NewError(ErrorIO, fmt.Errorf("failed reading alloc file: %v", err))
		}
		defer inFile.Close()
		decoder := json.NewDecoder(inFile)
		if err := decoder.Decode(&inputData.Alloc); err != nil {
			return NewError(ErrorJson, fmt.Errorf("failed unmarshaling alloc-file: %v", err))
		}
	}
	prestate.Pre = inputData.Alloc

	// Set the block environment
	if envStr != stdinSelector {
		inFile, err := os.Open(envStr)
		if err != nil {
			return NewError(ErrorIO, fmt.Errorf("failed reading env file: %v", err))
		}
		defer inFile.Close()
		decoder := json.NewDecoder(inFile)
		var env stEnv
		if err := decoder.Decode(&env); err != nil {
			return NewError(ErrorJson, fmt.Errorf("failed unmarshaling env-file: %v", err))
		}
		inputData.Env = &env
	}
	prestate.Env = *inputData.Env

	vmConfig := vm.Config{
		Tracer: tracer,
		Debug:  (tracer != nil),
	}
	// Construct the chainconfig
	var chainConfig *params.ChainConfig
	if cConf, extraEips, err := tests.GetChainConfig(ctx.String(ForknameFlag.Name)); err != nil {
		return NewError(ErrorVMConfig, fmt.Errorf("failed constructing chain configuration: %v", err))
	} else {
		chainConfig = cConf
		vmConfig.ExtraEips = extraEips
	}
	// Set the chain id
	chainConfig.ChainID = big.NewInt(ctx.Int64(ChainIDFlag.Name))

	var txsWithKeys []*txWithKey
	if txStr != stdinSelector {
		inFile, err := os.Open(txStr)
		if err != nil {
			return NewError(ErrorIO, fmt.Errorf("failed reading txs file: %v", err))
		}
		defer inFile.Close()
		decoder := json.NewDecoder(inFile)
		if strings.HasSuffix(txStr, ".rlp") {
			var body hexutil.Bytes
			if err := decoder.Decode(&body); err != nil {
				return err
			}
			var txs types.Transactions
			if err := rlp.DecodeBytes(body, &txs); err != nil {
				return err
			}
			for _, tx := range txs {
				txsWithKeys = append(txsWithKeys, &txWithKey{
					key: nil,
					tx:  tx,
				})
			}
		} else {
			if err := decoder.Decode(&txsWithKeys); err != nil {
				return NewError(ErrorJson, fmt.Errorf("failed unmarshaling txs-file: %v", err))
			}
		}
	} else {
		if len(inputData.TxRlp) > 0 {
			// Decode the body of already signed transactions
			body := common.FromHex(inputData.TxRlp)
			var txs types.Transactions
			if err := rlp.DecodeBytes(body, &txs); err != nil {
				return err
			}
			for _, tx := range txs {
				txsWithKeys = append(txsWithKeys, &txWithKey{
					key: nil,
					tx:  tx,
				})
			}
		} else {
			// JSON encoded transactions
			txsWithKeys = inputData.Txs
		}
	}
	// We may have to sign the transactions.
	signer := types.MakeSigner(chainConfig, big.NewInt(int64(prestate.Env.Number)))

	if txs, err = signUnsignedTransactions(txsWithKeys, signer); err != nil {
		return NewError(ErrorJson, fmt.Errorf("failed signing transactions: %v", err))
	}
	// Sanity check, to not `panic` in state_transition
	if chainConfig.IsLondon(big.NewInt(int64(prestate.Env.Number))) {
		if prestate.Env.BaseFee == nil {
			return NewError(ErrorVMConfig, errors.New("EIP-1559 config but missing 'currentBaseFee' in env section"))
		}
	}
	// Run the test and aggregate the result
	s, result, err := prestate.Apply(vmConfig, chainConfig, txs, ctx.Int64(RewardFlag.Name), getTracer)
	if err != nil {
		return err
	}
	body, _ := rlp.EncodeToBytes(txs)
	// Dump the excution result
	collector := make(Alloc)
	s.DumpToCollector(collector, nil)
	return dispatchOutput(ctx, baseDir, result, collector, body)
}

// txWithKey is a helper-struct, to allow us to use the types.Transaction along with
// a `secretKey`-field, for input
type txWithKey struct {
	key *ecdsa.PrivateKey
	tx  *types.Transaction
}

func (t *txWithKey) UnmarshalJSON(input []byte) error {
	// Read the secretKey, if present
	type sKey struct {
		Key *common.Hash `json:"secretKey"`
	}
	var key sKey
	if err := json.Unmarshal(input, &key); err != nil {
		return err
	}
	if key.Key != nil {
		k := key.Key.Hex()[2:]
		if ecdsaKey, err := crypto.HexToECDSA(k); err != nil {
			return err
		} else {
			t.key = ecdsaKey
		}
	}
	// Now, read the transaction itself
	var tx types.Transaction
	if err := json.Unmarshal(input, &tx); err != nil {
		return err
	}
	t.tx = &tx
	return nil
}

// signUnsignedTransactions converts the input txs to canonical transactions.
//
// The transactions can have two forms, either
//   1. unsigned or
//   2. signed
// For (1), r, s, v, need so be zero, and the `secretKey` needs to be set.
// If so, we sign it here and now, with the given `secretKey`
// If the condition above is not met, then it's considered a signed transaction.
//
// To manage this, we read the transactions twice, first trying to read the secretKeys,
// and secondly to read them with the standard tx json format
func signUnsignedTransactions(txs []*txWithKey, signer types.Signer) (types.Transactions, error) {
	var signedTxs []*types.Transaction
	for i, txWithKey := range txs {
		tx := txWithKey.tx
		key := txWithKey.key
		v, r, s := tx.RawSignatureValues()
		if key != nil && v.BitLen()+r.BitLen()+s.BitLen() == 0 {
			// This transaction needs to be signed
			signed, err := types.SignTx(tx, signer, key)
			if err != nil {
				return nil, NewError(ErrorJson, fmt.Errorf("tx %d: failed to sign tx: %v", i, err))
			}
			signedTxs = append(signedTxs, signed)
		} else {
			// Already signed
			signedTxs = append(signedTxs, tx)
		}
	}
	return signedTxs, nil
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
		Code:    dumpAccount.Code,
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
	location := path.Join(baseDir, filename)
	if err = ioutil.WriteFile(location, b, 0644); err != nil {
		return NewError(ErrorIO, fmt.Errorf("failed writing output: %v", err))
	}
	log.Info("Wrote file", "file", location)
	return nil
}

// dispatchOutput writes the output data to either stderr or stdout, or to the specified
// files
func dispatchOutput(ctx *cli.Context, baseDir string, result *ExecutionResult, alloc Alloc, body hexutil.Bytes) error {
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
	if len(stdOutObject) > 0 {
		b, err := json.MarshalIndent(stdOutObject, "", " ")
		if err != nil {
			return NewError(ErrorJson, fmt.Errorf("failed marshalling output: %v", err))
		}
		os.Stdout.Write(b)
		os.Stdout.Write([]byte("\n"))
	}
	if len(stdErrObject) > 0 {
		b, err := json.MarshalIndent(stdErrObject, "", " ")
		if err != nil {
			return NewError(ErrorJson, fmt.Errorf("failed marshalling output: %v", err))
		}
		os.Stderr.Write(b)
		os.Stderr.Write([]byte("\n"))
	}
	return nil
}
