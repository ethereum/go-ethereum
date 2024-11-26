package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/testdata/v2/nested_libraries"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/testdata/v2/v2_generated_testcase"
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

	// TODO: allow for the flexibility of deploying only libraries.
	// also, i kind of hate this conversion.  But the API of LinkAndDeployContractWithOverrides feels cleaner this way... idk.
	libMetas := make(map[string]*bind.MetaData)
	for pattern, metadata := range nested_libraries.C1LibraryDeps {
		libMetas[pattern] = metadata
	}

	ctrct, err := nested_libraries.NewC1()
	if err != nil {
		panic(err)
	}

	constructorInput, err := ctrct.PackConstructor(big.NewInt(42), big.NewInt(1))
	if err != nil {
		t.Fatalf("fack %v", err)
	}
	// TODO: test case with arguments-containing constructor
	deploymentParams := DeploymentParams{
		Contracts: []ContractDeployParams{
			{
				Meta:        nested_libraries.C1MetaData,
				Constructor: constructorInput,
			},
		},
		Libraries: nested_libraries.C1LibraryDeps,
		Overrides: nil,
	}
	res, err := LinkAndDeployContractWithOverrides(&opts, bindBackend, deploymentParams)
	if err != nil {
		t.Fatalf("err: %+v\n", err)
	}
	bindBackend.Commit()

	// assert that only 4 txs were produced.
	/*
		if len(deployedLibs)+1 != 4 {
			panic(fmt.Sprintf("whoops %d\n", len(deployedLibs)))
		}
	*/
	for _, tx := range res.Txs {
		_, err = bind.WaitDeployed(context.Background(), &bindBackend, tx)
		if err != nil {
			t.Fatalf("error deploying library: %+v", err)
		}
	}
	c, err := nested_libraries.NewC1()
	if err != nil {
		t.Fatalf("err is %v", err)
	}
	doInput, err := c.PackDo(big.NewInt(1))
	if err != nil {
		t.Fatalf("pack function input err: %v\n", doInput)
	}

	cABI, err := nested_libraries.C1MetaData.GetAbi()
	if err != nil {
		t.Fatalf("error getting abi object: %v", err)
	}
	contractAddr := res.Addrs[nested_libraries.C1MetaData.Pattern]
	boundC := bind.NewBoundContract(contractAddr, *cABI, &bindBackend, &bindBackend, &bindBackend)
	callOpts := &bind.CallOpts{
		From:    common.Address{},
		Context: context.Background(),
	}
	c1Code, err := bindBackend.PendingCodeAt(context.Background(), contractAddr)
	if err != nil {
		t.Fatalf("error getting pending code at %x: %v", contractAddr, err)
	}
	fmt.Printf("contract code:\n%x\n", c1Code)
	fmt.Printf("contract input:\n%x\n", doInput)
	callRes, err := boundC.CallRaw(callOpts, doInput)
	if err != nil {
		t.Fatalf("err calling contract: %v", err)
	}
	unpacked, err := c.UnpackDo(callRes)
	if err != nil {
		t.Fatalf("err unpacking result: %v", err)
	}

	// TODO: test transact
	fmt.Println(unpacked.String())
}

func TestDeploymentWithOverrides(t *testing.T) {
	// test that libs sharing deps, if overrides not specified we will deploy multiple versions of the dependent deps
	// test that libs sharing deps, if overrides specified... overrides work.
}

func TestEvents(t *testing.T) {
	// test watch/filter logs method on a contract that emits various kinds of events (struct-containing, etc.)
}

/* test-cases that should be extracted from v1 tests

* EventChecker

 */
