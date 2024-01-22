package utils

import (
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	"io"
	"net"
	"net/http"
)

func StatelessExecute(logOutput io.Writer, chainCfg *params.ChainConfig, witness *state.Witness) (root common.Hash, err error) {
	rawDb := rawdb.NewMemoryDatabase()
	if err := witness.PopulateDB(rawDb); err != nil {
		return common.Hash{}, err
	}
	_, prestateRoot := rawdb.ReadAccountTrieNode(rawDb, nil)

	db, err := state.New(prestateRoot, state.NewDatabaseWithConfig(rawDb, trie.PathDefaults), nil)
	if err != nil {
		return common.Hash{}, err
	}
	engine := beacon.New(ethash.NewFaker())
	validator := core.NewStatelessBlockValidator(chainCfg, engine)
	chainCtx := core.NewStatelessChainContext(rawDb, engine)
	processor := core.NewStatelessStateProcessor(chainCfg, chainCtx, engine)

	receipts, _, usedGas, err := processor.ProcessStateless(witness, witness.Block, db, vm.Config{})
	if err != nil {
		return common.Hash{}, err
	}
	// compute the state root.  skip validation of computed root against
	// the one provided in the block because this value is omitted from
	// the witness.
	if root, err = validator.ValidateState(witness.Block, db, receipts, usedGas, false); err != nil {
		return common.Hash{}, err
	}
	// TODO: how to differentiate between errors that are definitely not consensus-failure caused, and ones
	// that could be?
	return root, nil
}

// RunLocalServer runs an http server at the specified port (or 0 to use a random port).
// The server provides a POST endpoint /verify_block which takes input as an RLP-encoded
// block witness proof in the body, executes the block proof and returns the computed state root.
func RunLocalServer(chainConfig *params.ChainConfig, port int) (closeChan chan<- struct{}, actualPort int, err error) {
	mux := http.NewServeMux()
	mux.Handle("/verify_block", &verifyHandler{chainConfig})
	srv := http.Server{Handler: mux}
	listener, err := net.Listen("tcp", ":"+fmt.Sprintf("%d", port))
	if err != nil {
		return nil, 0, err
	}
	actualPort = listener.Addr().(*net.TCPAddr).Port

	go func() {
		if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	closeCh := make(chan struct{})
	go func() {
		select {
		case <-closeCh:
			if err := srv.Close(); err != nil {
				panic(err)
			}
		}
	}()
	return closeCh, actualPort, nil
}

type verifyHandler struct {
	chainConfig *params.ChainConfig
}

func (v *verifyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	respError := func(descr string, err error) {
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(fmt.Sprintf("%s: %s", descr, err))); err != nil {
			log.Error("write failed", "error", err)
		}

		log.Error("responded with error", "descr", descr, "error", err)
	}
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respError("error reading body", err)
		return
	}
	if len(body) == 0 {
		respError("error", fmt.Errorf("empty body"))
		return
	}
	witness, err := state.DecodeWitnessRLP(body)
	if err != nil {
		respError("error decoding body witness rlp", err)
		return
	}
	defer func() {
		if err := recover(); err != nil {
			errr, _ := err.(error)
			respError("execution error", errr)
			return
		}
	}()

	root, err := StatelessExecute(nil, v.chainConfig, witness)
	if err != nil {
		respError("error verifying stateless proof", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(root[:]); err != nil {
		log.Error("error writing response", "error", err)
	}
}
