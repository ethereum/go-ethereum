package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/testdata/v2_generated_testcase"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/testdata/v2_testcase_library"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
	"strings"
	"testing"
)

var testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

// JSON returns a parsed ABI interface and error if it failed.
func JSON(reader io.Reader) (abi.ABI, error) {
	dec := json.NewDecoder(reader)

	var instance abi.ABI
	if err := dec.Decode(&instance); err != nil {
		return abi.ABI{}, err
	}
	return instance, nil
}

func TestV2(t *testing.T) {
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)
	backend := simulated.NewBackend(
		types.GenesisAlloc{
			testAddr: {Balance: big.NewInt(10000000000000000)},
		},
		func(nodeConf *node.Config, ethConf *ethconfig.Config) {
			ethConf.Genesis.Difficulty = big.NewInt(0)
		},
	)
	defer backend.Close()

	contractABI, err := JSON(strings.NewReader(v2_generated_testcase.V2GeneratedTestcaseMetaData.ABI))
	if err != nil {
		panic(err)
	}

	signer := types.LatestSigner(params.AllDevChainProtocolChanges)
	opts := bind.TransactOpts{
		From:  testAddr,
		Nonce: nil,
		Signer: func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			signature, err := crypto.Sign(signer.Hash(tx).Bytes(), testKey)
			if err != nil {
				t.Fatal(err)
			}
			signedTx, err := tx.WithSignature(signer, signature)
			if err != nil {
				t.Fatal(err)
			}
			return signedTx, nil
		},
		Context: context.Background(),
	}
	// we should just be able to use the backend directly, instead of using
	// this deprecated interface.  However, the simulated backend no longer
	// implements backends.SimulatedBackend...
	bindBackend := backends.SimulatedBackend{
		Backend: backend,
		Client:  backend.Client(),
	}
	address, tx, _, err := bind.DeployContract(&opts, contractABI, common.Hex2Bytes(v2_generated_testcase.V2GeneratedTestcaseMetaData.Bin), &bindBackend)
	if err != nil {
		t.Fatal(err)
	}

	_, err = bind.WaitDeployed(context.Background(), &bindBackend, tx)
	if err != nil {
		t.Fatalf("error deploying bound contract: %+v", err)
	}

	contract, err := v2_generated_testcase.NewV2GeneratedTestcase()
	if err != nil {
		t.Fatal(err) // can't happen here with the example used.  consider removing this block
	}
	//contractInstance := v2_generated_testcase.NewV2GeneratedTestcaseInstance(contract, address, bindBackend)
	contractInstance := ContractInstance{
		Address: address,
		Backend: bindBackend,
	}
	sinkCh := make(chan *v2_generated_testcase.V2GeneratedTestcaseStruct)
	// q:  what extra functionality is given by specifying this as a custom method, instead of catching emited methods
	// from the sync channel?
	unpackStruct := func(log *types.Log) (*v2_generated_testcase.V2GeneratedTestcaseStruct, error) {
		res, err := contract.UnpackStructEvent(log)
		return res, err
	}
	watchOpts := bind.WatchOpts{
		Start:   nil,
		Context: context.Background(),
	}
	// TODO: test using various topics
	// q: does nil topics mean to accept any?
	sub, err := WatchLogs[v2_generated_testcase.V2GeneratedTestcaseStruct](&contractInstance, &watchOpts, v2_generated_testcase.V2GeneratedTestcaseStructEventID(), unpackStruct, sinkCh, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Unsubscribe()
	// send a balance to our contract (contract must accept ether by default)
}

func TestDeployment(t *testing.T) {
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)
	backend := simulated.NewBackend(
		types.GenesisAlloc{
			testAddr: {Balance: big.NewInt(10000000000000000)},
		},
		func(nodeConf *node.Config, ethConf *ethconfig.Config) {
			ethConf.Genesis.Difficulty = big.NewInt(0)
		},
	)
	defer backend.Close()

	_, err := JSON(strings.NewReader(v2_generated_testcase.V2GeneratedTestcaseMetaData.ABI))
	if err != nil {
		panic(err)
	}

	signer := types.LatestSigner(params.AllDevChainProtocolChanges)
	opts := bind.TransactOpts{
		From:  testAddr,
		Nonce: nil,
		Signer: func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			signature, err := crypto.Sign(signer.Hash(tx).Bytes(), testKey)
			if err != nil {
				t.Fatal(err)
			}
			signedTx, err := tx.WithSignature(signer, signature)
			if err != nil {
				t.Fatal(err)
			}
			return signedTx, nil
		},
		Context: context.Background(),
	}
	// we should just be able to use the backend directly, instead of using
	// this deprecated interface.  However, the simulated backend no longer
	// implements backends.SimulatedBackend...
	bindBackend := backends.SimulatedBackend{
		Backend: backend,
		Client:  backend.Client(),
	}

	log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stdout, log.LevelDebug, true)))

	///LinkAndDeployContractsWithOverride(&opts, bindBackend, v2_test)
	deployTxs, err := DeployContracts(&opts, bindBackend, []byte{}, v2_testcase_library.TestArrayLibraryDeps)
	if err != nil {
		t.Fatalf("err: %+v\n", err)
	}
	for _, tx := range deployTxs {
		fmt.Println("waiting for deployment")
		_, err = bind.WaitDeployed(context.Background(), &bindBackend, tx)
		if err != nil {
			t.Fatalf("error deploying bound contract: %+v", err)
		}
	}
}

/* test-cases that should be extracted from v1 tests

* EventChecker

 */
