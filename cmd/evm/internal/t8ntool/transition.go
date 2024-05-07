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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/tests"
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

type tracerFn func(baseDir string) func(txIndex int, txHash common.Hash) (*tracers.Tracer, io.WriteCloser, error)

type input struct {
	Alloc types.GenesisAlloc `json:"alloc,omitempty"`
	Env   *stEnv             `json:"env,omitempty"`
	Txs   []*txWithKey       `json:"txs,omitempty"`
	TxRlp hexutil.Bytes      `json:"txsRlp,omitempty"`
}

func (i *input) prestate() Prestate {
	return Prestate{
		Pre: i.Alloc,
		Env: *i.Env,
	}
}

func (i *input) txIterator(chainConfig *params.ChainConfig) (txIterator, error) {
	if len(i.TxRlp) > 0 {
		// Decode the body of already signed transactions
		return newRlpTxIterator(i.TxRlp), nil
	}
	// We may have to sign the transactions.
	signer := types.LatestSignerForChainID(chainConfig.ChainID)
	txs, err := signUnsignedTransactions(i.Txs, signer)
	return newSliceTxIterator(txs), err
}

type stateReq struct {
	Fork    string `json:"fork,omitempty"`
	ChainID int64  `json:"chainid,omitempty"`
	Reward  int64  `json:"reward,omitempty"`
}

type transitionRequest struct {
	Input   input    `json:"input,omitempty"`
	State   stateReq `json:"state,omitempty"`
	BaseDir string   `json:"baseDir,omitempty"`
}

type transitionRequestOutput struct {
	Result *ExecutionResult `json:"result"`
	Alloc  Alloc            `json:"alloc"`
	Body   hexutil.Bytes    `json:"body"`
}

func (r *transitionRequest) process(tracerGenFn tracerFn) (*transitionRequestOutput, error) {
	baseDir, err := createBasedirFromString(r.BaseDir)
	if err != nil {
		return nil, fmt.Errorf("failed creating output basedir: %v", err)
	}

	prestate := r.Input.prestate()
	vmConfig := vm.Config{}
	// Construct the chainconfig
	var chainConfig *params.ChainConfig
	if cConf, extraEips, err := tests.GetChainConfig(r.State.Fork); err != nil {
		return nil, fmt.Errorf("failed constructing chain configuration: %v", err)
	} else {
		chainConfig = cConf
		vmConfig.ExtraEips = extraEips
	}
	// Set the chain id
	chainConfig.ChainID = big.NewInt(r.State.ChainID)

	txIt, err := r.Input.txIterator(chainConfig)
	if err != nil {
		return nil, fmt.Errorf("failed loading transactions: %v", err)
	}
	if err := applyLondonChecks(&prestate.Env, chainConfig); err != nil {
		return nil, fmt.Errorf("failed applying London checks: %v", err)
	}
	if err := applyShanghaiChecks(&prestate.Env, chainConfig); err != nil {
		return nil, fmt.Errorf("failed applying Shanghai checks: %v", err)
	}
	if err := applyMergeChecks(&prestate.Env, chainConfig); err != nil {
		return nil, fmt.Errorf("failed applying Merge checks: %v", err)
	}
	if err := applyCancunChecks(&prestate.Env, chainConfig); err != nil {
		return nil, fmt.Errorf("failed applying Cancun checks: %v", err)
	}
	// Run the test and aggregate the result
	s, result, body, err := prestate.Apply(vmConfig, chainConfig, txIt, r.State.Reward, tracerGenFn(baseDir))
	if err != nil {
		return nil, fmt.Errorf("failed applying prestate: %v", err)
	}
	// Dump the execution result
	collector := make(Alloc)
	s.DumpToCollector(collector, nil)
	return &transitionRequestOutput{
		Result: result,
		Alloc:  collector,
		Body:   body,
	}, nil
}

func tracerGenerator(ctx *cli.Context) tracerFn {
	if ctx.Bool(TraceFlag.Name) { // JSON opcode tracing
		// Configure the EVM logger
		logConfig := &logger.Config{
			DisableStack:     ctx.Bool(TraceDisableStackFlag.Name),
			EnableMemory:     ctx.Bool(TraceEnableMemoryFlag.Name),
			EnableReturnData: ctx.Bool(TraceEnableReturnDataFlag.Name),
			Debug:            true,
		}
		enableCallFrames := ctx.Bool(TraceEnableCallFramesFlag.Name)
		return func(baseDir string) func(txIndex int, txHash common.Hash) (*tracers.Tracer, io.WriteCloser, error) {
			return func(txIndex int, txHash common.Hash) (*tracers.Tracer, io.WriteCloser, error) {
				traceFile, err := os.Create(filepath.Join(baseDir, fmt.Sprintf("trace-%d-%v.jsonl", txIndex, txHash.String())))
				if err != nil {
					return nil, nil, NewError(ErrorIO, fmt.Errorf("failed creating trace-file: %v", err))
				}
				var l *tracing.Hooks
				if enableCallFrames {
					l = logger.NewJSONLoggerWithCallFrames(logConfig, traceFile)
				} else {
					l = logger.NewJSONLogger(logConfig, traceFile)
				}
				tracer := &tracers.Tracer{
					Hooks: l,
					// jsonLogger streams out result to file.
					GetResult: func() (json.RawMessage, error) { return nil, nil },
					Stop:      func(err error) {},
				}
				return tracer, traceFile, nil
			}
		}
	} else if ctx.IsSet(TraceTracerFlag.Name) {
		var config json.RawMessage
		if ctx.IsSet(TraceTracerConfigFlag.Name) {
			config = []byte(ctx.String(TraceTracerConfigFlag.Name))
		}
		tracerStr := ctx.String(TraceTracerFlag.Name)
		return func(baseDir string) func(txIndex int, txHash common.Hash) (*tracers.Tracer, io.WriteCloser, error) {
			return func(txIndex int, txHash common.Hash) (*tracers.Tracer, io.WriteCloser, error) {
				traceFile, err := os.Create(filepath.Join(baseDir, fmt.Sprintf("trace-%d-%v.json", txIndex, txHash.String())))
				if err != nil {
					return nil, nil, NewError(ErrorIO, fmt.Errorf("failed creating trace-file: %v", err))
				}
				tracer, err := tracers.DefaultDirectory.New(tracerStr, nil, config)
				if err != nil {
					return nil, nil, NewError(ErrorConfig, fmt.Errorf("failed instantiating tracer: %w", err))
				}
				return tracer, traceFile, nil
			}
		}
	}
	// Default to no tracing
	return func(baseDir string) func(txIndex int, txHash common.Hash) (*tracers.Tracer, io.WriteCloser, error) {
		return func(txIndex int, txHash common.Hash) (*tracers.Tracer, io.WriteCloser, error) {
			return nil, nil, nil
		}
	}
}

func Transition(ctx *cli.Context) error {
	// We need to load three things: alloc, env and transactions. May be either in
	// stdin input or in files.
	// Check if anything needs to be read from stdin
	var (
		allocStr = ctx.String(InputAllocFlag.Name)
		envStr   = ctx.String(InputEnvFlag.Name)
		txStr    = ctx.String(InputTxsFlag.Name)
	)
	request := transitionRequest{
		BaseDir: ctx.String(OutputBasedir.Name),
		State: stateReq{
			Fork:    ctx.String(ForknameFlag.Name),
			ChainID: ctx.Int64(ChainIDFlag.Name),
			Reward:  ctx.Int64(RewardFlag.Name),
		},
	}
	// Figure out the prestate alloc
	if allocStr == stdinSelector || envStr == stdinSelector || txStr == stdinSelector {
		decoder := json.NewDecoder(os.Stdin)
		if err := decoder.Decode(&request.Input); err != nil {
			return NewError(ErrorJson, fmt.Errorf("failed unmarshalling stdin: %v", err))
		}
	}
	if allocStr != stdinSelector {
		if err := readFile(allocStr, "alloc", &request.Input.Alloc); err != nil {
			return err
		}
	}

	// Set the block environment
	if envStr != stdinSelector {
		var env stEnv
		if err := readFile(envStr, "env", &env); err != nil {
			return err
		}
		request.Input.Env = &env
	}

	// Load the transactions from file if needed
	if txStr != stdinSelector {
		data, err := os.ReadFile(txStr)
		if err != nil {
			return NewError(ErrorIO, fmt.Errorf("failed reading txs file: %v", err))
		}
		if strings.HasSuffix(txStr, ".rlp") { // A file containing an rlp list
			err = json.Unmarshal(data, &request.Input.TxRlp)
		} else {
			err = json.Unmarshal(data, &request.Input.Txs)
		}
		if err != nil {
			return fmt.Errorf("failed unmarshalling txs-file: %v", err)
		}
	}

	result, err := request.process(tracerGenerator(ctx))
	if err != nil {
		return err
	}
	return dispatchOutput(ctx, request.BaseDir, result.Result, result.Alloc, result.Body)
}

func TransitionServer(ctx *cli.Context) error {
	// Start the server
	server := &transitionServer{
		port:        ctx.Int(PortFlag.Name),
		unixSocket:  ctx.String(UnixSocketFlag.Name),
		tracerGenFn: tracerGenerator(ctx),
	}

	if err := server.start(); err != nil {
		return NewError(ErrorIO, fmt.Errorf("failed starting server: %v", err))
	}
	log.Info("Started server", "port", server.port)

	// Wait for a signal to shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-done

	log.Info("Shutting down server")
	if err := server.shutdown(); err != nil {
		return NewError(ErrorIO, fmt.Errorf("failed shutting down server: %v", err))
	}
	return nil
}

type transitionServer struct {
	port        int
	unixSocket  string
	tracerGenFn tracerFn
	httpSrv     *http.Server
}

func (server *transitionServer) start() error {
	// start the HTTP listener
	var (
		listener net.Listener
		err      error
	)
	if server.unixSocket != "" {
		listener, err = net.Listen("unix", server.unixSocket)
	} else {
		listener, err = net.Listen("tcp", fmt.Sprintf(":%d", server.port))
	}
	if err != nil {
		return err
	}

	// Bundle and start the HTTP server
	server.httpSrv = &http.Server{
		Handler: server,
	}
	go server.httpSrv.Serve(listener)
	if server.unixSocket == "" {
		server.port = listener.Addr().(*net.TCPAddr).Port
	}
	return err
}

func (server *transitionServer) shutdown() error {
	if server.httpSrv == nil {
		return nil
	}
	return server.httpSrv.Shutdown(context.Background())
}

func (server *transitionServer) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	reqStartTime := time.Now()
	decoder := json.NewDecoder(req.Body)
	var request transitionRequest
	// Parse this from the http request
	if err := decoder.Decode(&request); err != nil {
		http.Error(res, fmt.Sprintf("failed unmarshalling request: %v", err), http.StatusBadRequest)
		return
	}
	output, err := request.process(server.tracerGenFn)
	if err != nil {
		http.Error(res, fmt.Sprintf("failed processing request: %v", err), http.StatusInternalServerError)
		return
	}

	// Write the http response
	res.Header().Set("Content-Type", "application/json")
	json.NewEncoder(res).Encode(output)
	log.Info("Processed request", "duration", time.Since(reqStartTime))
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
		return NewError(ErrorConfig, errors.New("EIP-1559 config but missing 'currentBaseFee' in env section"))
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
