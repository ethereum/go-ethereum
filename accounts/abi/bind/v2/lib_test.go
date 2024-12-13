// Copyright 2024 The go-ethereum Authors
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

package v2

import (
	"context"
	"encoding/json"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2/internal/events"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2/internal/nested_libraries"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2/internal/solc_errors"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/compiler"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
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

func testSetup() (*bind.TransactOpts, *backends.SimulatedBackend, error) {
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)
	backend := simulated.NewBackend(
		types.GenesisAlloc{
			testAddr: {Balance: big.NewInt(10000000000000000)},
		},
		func(nodeConf *node.Config, ethConf *ethconfig.Config) {
			ethConf.Genesis.Difficulty = big.NewInt(0)
		},
	)

	signer := types.LatestSigner(params.AllDevChainProtocolChanges)
	opts := &bind.TransactOpts{
		From:  testAddr,
		Nonce: nil,
		Signer: func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			signature, err := crypto.Sign(signer.Hash(tx).Bytes(), testKey)
			if err != nil {
				return nil, err
			}
			signedTx, err := tx.WithSignature(signer, signature)
			if err != nil {
				return nil, err
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
	return opts, &bindBackend, nil
}

func makeTestDeployer(auth *bind.TransactOpts, backend bind.ContractBackend) func(input, deployer []byte) (common.Address, *types.Transaction, error) {
	return func(input, deployer []byte) (common.Address, *types.Transaction, error) {
		addr, tx, _, err := bind.DeployContractRaw(auth, deployer, backend, input)
		return addr, tx, err
	}
}

// test that deploying a contract with library dependencies works,
// verifying by calling method on the deployed contract.
func TestDeploymentLibraries(t *testing.T) {
	opts, bindBackend, err := testSetup()
	if err != nil {
		t.Fatalf("err setting up test: %v", err)
	}
	defer bindBackend.Backend.Close()

	ctrct, err := nested_libraries.NewC1()
	if err != nil {
		panic(err)
	}

	constructorInput, err := ctrct.PackConstructor(big.NewInt(42), big.NewInt(1))
	if err != nil {
		t.Fatalf("failed to pack constructor: %v", err)
	}
	deploymentParams := DeploymentParams{
		Contracts: append(nested_libraries.C1LibraryDeps, nested_libraries.C1MetaData),
		Inputs:    map[string][]byte{nested_libraries.C1MetaData.Pattern: constructorInput},
		Overrides: nil,
	}

	res, err := LinkAndDeploy(deploymentParams, makeTestDeployer(opts, bindBackend))
	if err != nil {
		t.Fatalf("err: %+v\n", err)
	}
	bindBackend.Commit()

	if len(res.Addrs) != 5 {
		t.Fatalf("deployment should have generated 5 addresses.  got %d", len(res.Addrs))
	}
	for _, tx := range res.Txs {
		_, err = bind.WaitDeployed(context.Background(), bindBackend, tx)
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

	contractAddr := res.Addrs[nested_libraries.C1MetaData.Pattern]
	callOpts := &bind.CallOpts{
		From:    common.Address{},
		Context: context.Background(),
	}

	ctrctInstance := &ContractInstance{
		Address: contractAddr,
		Backend: bindBackend,
	}
	internalCallCount, err := Call[big.Int](ctrctInstance, callOpts, doInput, ctrct.UnpackDo)
	if err != nil {
		t.Fatalf("err unpacking result: %v", err)
	}
	if internalCallCount.Uint64() != 6 {
		t.Fatalf("expected internal call count of 6.  got %d.", internalCallCount.Uint64())
	}
}

// Same as TestDeployment.  However, stagger the deployments with overrides:
// first deploy the library deps and then the contract.
func TestDeploymentWithOverrides(t *testing.T) {
	opts, bindBackend, err := testSetup()
	if err != nil {
		t.Fatalf("err setting up test: %v", err)
	}
	defer bindBackend.Backend.Close()

	// deploy some library deps
	deploymentParams := DeploymentParams{
		Contracts: nested_libraries.C1LibraryDeps,
	}

	res, err := LinkAndDeploy(deploymentParams, makeTestDeployer(opts, bindBackend))
	if err != nil {
		t.Fatalf("err: %+v\n", err)
	}
	bindBackend.Commit()

	if len(res.Addrs) != 4 {
		t.Fatalf("deployment should have generated 4 addresses.  got %d", len(res.Addrs))
	}
	for _, tx := range res.Txs {
		_, err = bind.WaitDeployed(context.Background(), bindBackend, tx)
		if err != nil {
			t.Fatalf("error deploying library: %+v", err)
		}
	}

	ctrct, err := nested_libraries.NewC1()
	if err != nil {
		panic(err)
	}
	constructorInput, err := ctrct.PackConstructor(big.NewInt(42), big.NewInt(1))
	if err != nil {
		t.Fatalf("failed to pack constructor: %v", err)
	}
	overrides := res.Addrs
	// deploy the contract
	deploymentParams = DeploymentParams{
		Contracts: []*bind.MetaData{nested_libraries.C1MetaData},
		Inputs:    map[string][]byte{nested_libraries.C1MetaData.Pattern: constructorInput},
		Overrides: overrides,
	}
	res, err = LinkAndDeploy(deploymentParams, makeTestDeployer(opts, bindBackend))
	if err != nil {
		t.Fatalf("err: %+v\n", err)
	}
	bindBackend.Commit()

	if len(res.Addrs) != 1 {
		t.Fatalf("deployment should have generated 1 address.  got %d", len(res.Addrs))
	}
	for _, tx := range res.Txs {
		_, err = bind.WaitDeployed(context.Background(), bindBackend, tx)
		if err != nil {
			t.Fatalf("error deploying library: %+v", err)
		}
	}

	// call the deployed contract and make sure it returns the correct result
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
	boundContract := bind.NewBoundContract(contractAddr, *cABI, bindBackend, bindBackend, bindBackend)
	callOpts := &bind.CallOpts{
		From:    common.Address{},
		Context: context.Background(),
	}
	callRes, err := boundContract.CallRaw(callOpts, doInput)
	if err != nil {
		t.Fatalf("err calling contract: %v", err)
	}
	internalCallCount, err := c.UnpackDo(callRes)
	if err != nil {
		t.Fatalf("err unpacking result: %v", err)
	}
	if internalCallCount.Uint64() != 6 {
		t.Fatalf("expected internal call count of 6.  got %d.", internalCallCount.Uint64())
	}
}

func TestEvents(t *testing.T) {
	// test watch/filter logs method on a contract that emits various kinds of events (struct-containing, etc.)
	txAuth, backend, err := testSetup()
	if err != nil {
		t.Fatalf("error setting up testing env: %v", err)
	}

	deploymentParams := DeploymentParams{
		Contracts: []*bind.MetaData{events.CMetaData},
	}

	res, err := LinkAndDeploy(deploymentParams, makeTestDeployer(txAuth, backend))
	if err != nil {
		t.Fatalf("error deploying contract for testing: %v", err)
	}

	backend.Commit()
	if _, err := bind.WaitDeployed(context.Background(), backend, res.Txs[events.CMetaData.Pattern]); err != nil {
		t.Fatalf("WaitDeployed failed %v", err)
	}

	ctrct, err := events.NewC()
	if err != nil {
		t.Fatalf("error instantiating contract instance: %v", err)
	}

	boundContract := ContractInstance{
		res.Addrs[events.CMetaData.Pattern],
		backend,
	}

	newCBasic1Ch := make(chan *events.CBasic1)
	newCBasic2Ch := make(chan *events.CBasic2)
	watchOpts := &bind.WatchOpts{
		Start:   nil,
		Context: context.Background(),
	}
	sub1, err := WatchEvents(&boundContract, watchOpts, events.CBasic1EventID(), ctrct.UnpackBasic1Event, newCBasic1Ch)
	if err != nil {
		t.Fatalf("WatchEvents returned error: %v", err)
	}
	sub2, err := WatchEvents(&boundContract, watchOpts, events.CBasic2EventID(), ctrct.UnpackBasic2Event, newCBasic2Ch)
	if err != nil {
		t.Fatalf("WatchEvents returned error: %v", err)
	}
	defer sub1.Unsubscribe()
	defer sub2.Unsubscribe()

	crtctInstance := &ContractInstance{
		Address: res.Addrs[events.CMetaData.Pattern],
		Backend: backend,
	}
	packedInput, _ := ctrct.PackEmitMulti()
	tx, err := Transact(crtctInstance, txAuth, packedInput)
	if err != nil {
		t.Fatalf("failed to send transaction: %v", err)
	}
	backend.Commit()
	if _, err := bind.WaitMined(context.Background(), backend, tx); err != nil {
		t.Fatalf("error waiting for tx to be mined: %v", err)
	}

	timeout := time.NewTimer(2 * time.Second)
	e1Count := 0
	e2Count := 0
	for {
		select {
		case <-newCBasic1Ch:
			e1Count++
		case <-newCBasic2Ch:
			e2Count++
		case <-timeout.C:
			goto done
		}
		if e1Count == 2 && e2Count == 1 {
			break
		}
	}
done:
	if e1Count != 2 {
		t.Fatalf("expected event type 1 count to be 2.  got %d", e1Count)
	}
	if e2Count != 1 {
		t.Fatalf("expected event type 2 count to be 1.  got %d", e2Count)
	}

	// now, test that we can filter those same logs after they were included in the chain

	filterOpts := &bind.FilterOpts{
		Start:   0,
		Context: context.Background(),
	}
	it, err := FilterEvents[events.CBasic1](crtctInstance, filterOpts, events.CBasic1EventID(), ctrct.UnpackBasic1Event)
	if err != nil {
		t.Fatalf("error filtering logs %v\n", err)
	}
	it2, err := FilterEvents[events.CBasic2](crtctInstance, filterOpts, events.CBasic2EventID(), ctrct.UnpackBasic2Event)
	if err != nil {
		t.Fatalf("error filtering logs %v\n", err)
	}
	e1Count = 0
	e2Count = 0
	for it.Next() {
		e1Count++
	}
	for it2.Next() {
		e2Count++
	}
	if e1Count != 2 {
		t.Fatalf("expected e1Count of 2 from filter call.  got %d", e1Count)
	}
	if e2Count != 1 {
		t.Fatalf("expected e2Count of 1 from filter call.  got %d", e1Count)
	}
}

func TestErrors(t *testing.T) {
	// test watch/filter logs method on a contract that emits various kinds of events (struct-containing, etc.)
	txAuth, backend, err := testSetup()
	if err != nil {
		t.Fatalf("error setting up testing env: %v", err)
	}

	deploymentParams := DeploymentParams{
		Contracts: []*bind.MetaData{solc_errors.CMetaData},
	}

	res, err := LinkAndDeploy(deploymentParams, makeTestDeployer(txAuth, backend))
	if err != nil {
		t.Fatalf("error deploying contract for testing: %v", err)
	}

	backend.Commit()
	if _, err := bind.WaitDeployed(context.Background(), backend, res.Txs[solc_errors.CMetaData.Pattern]); err != nil {
		t.Fatalf("WaitDeployed failed %v", err)
	}

	ctrct, _ := solc_errors.NewC()

	var packedInput []byte
	opts := &bind.CallOpts{
		From: res.Addrs[solc_errors.CMetaData.Pattern],
	}
	packedInput, _ = ctrct.PackFoo()

	instance := ContractInstance{res.Addrs[solc_errors.CMetaData.Pattern], backend}
	_, err = Call[struct{}](&instance, opts, packedInput, func(packed []byte) (*struct{}, error) { return nil, nil })
	if err == nil {
		t.Fatalf("expected call to fail")
	}
	raw, hasRevertErrorData := ethclient.RevertErrorData(err)
	if !hasRevertErrorData {
		t.Fatalf("expected call error to contain revert error data.")
	}
	unpackedErr := ctrct.UnpackError(raw)
	if unpackedErr == nil {
		t.Fatalf("expected to unpack error")
	}
	// TODO: check anything about the error?
}

// TODO: this test will pass if the code is changed but not compiled to a combined-abi.json.
// Not really possible to test this without including solc in the path when running CI.
func TestBindingGeneration(t *testing.T) {
	matches, _ := filepath.Glob("internal/*")
	var dirs []string
	for _, match := range matches {
		f, _ := os.Stat(match)
		if f.IsDir() {
			dirs = append(dirs, f.Name())
		}
	}

	for _, dir := range dirs {
		var (
			abis  []string
			bins  []string
			types []string
			sigs  []map[string]string
			libs  = make(map[string]string)
		)
		basePath := filepath.Join("internal", dir)
		combinedJsonPath := filepath.Join(basePath, "combined-abi.json")
		abiBytes, err := os.ReadFile(combinedJsonPath)
		if err != nil {
			t.Fatalf("error trying to read file %s: %v", combinedJsonPath, err)
		}
		contracts, err := compiler.ParseCombinedJSON(abiBytes, "", "", "", "")
		if err != nil {
			t.Fatalf("Failed to read contract information from json output: %v", err)
		}

		for name, contract := range contracts {
			// fully qualified name is of the form <solFilePath>:<type>
			nameParts := strings.Split(name, ":")
			typeName := nameParts[len(nameParts)-1]
			abi, err := json.Marshal(contract.Info.AbiDefinition) // Flatten the compiler parse
			if err != nil {
				utils.Fatalf("Failed to parse ABIs from compiler output: %v", err)
			}
			abis = append(abis, string(abi))
			bins = append(bins, contract.Code)
			sigs = append(sigs, contract.Hashes)
			types = append(types, typeName)

			// Derive the library placeholder which is a 34 character prefix of the
			// hex encoding of the keccak256 hash of the fully qualified library name.
			// Note that the fully qualified library name is the path of its source
			// file and the library name separated by ":".
			libPattern := crypto.Keccak256Hash([]byte(name)).String()[2:36] // the first 2 chars are 0x
			libs[libPattern] = typeName
		}
		code, err := bind.BindV2(types, abis, bins, sigs, dir, libs, make(map[string]string))
		if err != nil {
			t.Fatalf("error creating bindings for package %s: %v", dir, err)
		}

		existingBindings, err := os.ReadFile(filepath.Join(basePath, "bindings.go"))
		if err != nil {
			t.Fatalf("ReadFile returned error: %v", err)
		}
		if code != string(existingBindings) {
			t.Fatalf("code mismatch for %s", dir)
		}
	}
}
