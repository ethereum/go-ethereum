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
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/testdata/v2/events"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/testdata/v2/nested_libraries"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"io"
	"math/big"
	"strings"
	"testing"
	"time"
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

	_, err := JSON(strings.NewReader(nested_libraries.C1MetaData.ABI))
	if err != nil {
		return nil, nil, err
	}

	signer := types.LatestSigner(params.AllDevChainProtocolChanges)
	opts := &bind.TransactOpts{
		From:  testAddr,
		Nonce: nil,
		Signer: func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			signature, err := crypto.Sign(signer.Hash(tx).Bytes(), testKey)
			if err != nil {
				panic(fmt.Sprintf("error signing tx: %v", err))
				return nil, err
			}
			signedTx, err := tx.WithSignature(signer, signature)
			if err != nil {
				panic(fmt.Sprintf("error creating tx with sig: %v", err))
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

// test deployment and interaction for a basic contract with no library deps
func TestDeployment(t *testing.T) {

}

// test that deploying a contract with library dependencies works,
// verifying by calling the deployed contract.
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
		Contracts: []ContractDeployParams{
			{
				Meta:  nested_libraries.C1MetaData,
				Input: constructorInput,
			},
		},
		Libraries: nested_libraries.C1LibraryDeps,
		Overrides: nil,
	}
	res, err := LinkAndDeploy(opts, bindBackend, deploymentParams)
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

	cABI, err := nested_libraries.C1MetaData.GetAbi()
	if err != nil {
		t.Fatalf("error getting abi object: %v", err)
	}
	contractAddr := res.Addrs[nested_libraries.C1MetaData.Pattern]
	boundC := bind.NewBoundContract(contractAddr, *cABI, bindBackend, bindBackend, bindBackend)
	callOpts := &bind.CallOpts{
		From:    common.Address{},
		Context: context.Background(),
	}
	callRes, err := boundC.CallRaw(callOpts, doInput)
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

func TestDeploymentWithOverrides(t *testing.T) {
	opts, bindBackend, err := testSetup()
	if err != nil {
		t.Fatalf("err setting up test: %v", err)
	}
	defer bindBackend.Backend.Close()

	// deploy some library deps
	deploymentParams := DeploymentParams{
		Libraries: nested_libraries.C1LibraryDeps,
	}

	res, err := LinkAndDeploy(opts, bindBackend, deploymentParams)
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
		Contracts: []ContractDeployParams{
			{
				Meta:  nested_libraries.C1MetaData,
				Input: constructorInput,
			},
		},
		Libraries: nil,
		Overrides: overrides,
	}
	res, err = LinkAndDeploy(opts, bindBackend, deploymentParams)
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
	boundC := bind.NewBoundContract(contractAddr, *cABI, bindBackend, bindBackend, bindBackend)
	callOpts := &bind.CallOpts{
		From:    common.Address{},
		Context: context.Background(),
	}
	callRes, err := boundC.CallRaw(callOpts, doInput)
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

/*
 *
 */
/*
	func TestDeploymentWithOverrides(t *testing.T) {
		// more deployment test case ideas:
		// 1)  deploy libraries, then deploy contract first with libraries as overrides
		// 2)  deploy contract without library dependencies.
	}
*/

// note to self: see how to filter logs in eth_call.

func TestEvents(t *testing.T) {
	// test watch/filter logs method on a contract that emits various kinds of events (struct-containing, etc.)
	txAuth, backend, err := testSetup()
	if err != nil {
		t.Fatalf("error setting up testing env: %v", err)
	}

	deploymentParams := DeploymentParams{
		Contracts: []ContractDeployParams{
			{
				Meta: events.CMetaData,
			},
		},
	}

	res, err := LinkAndDeploy(txAuth, backend, deploymentParams)
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

	abi, err := events.CMetaData.GetAbi()
	if err != nil {
		t.Fatalf("error getting contract abi: %v", err)
	}

	// TODO: why did I introduce separate type, and not just use bound contract?
	boundContract := ContractInstance{
		res.Addrs[events.CMetaData.Pattern],
		backend,
	}

	unpackBasic := func(raw *types.Log) (*events.CBasic1, error) {
		return &events.CBasic1{
			Id:   (new(big.Int)).SetBytes(raw.Topics[0].Bytes()),
			Data: (new(big.Int)).SetBytes(raw.Data),
		}, nil
	}
	unpackBasic2 := func(raw *types.Log) (*events.CBasic2, error) {
		return &events.CBasic2{
			Flag: false, // TODO: how to unpack different types to go types?  this should be exposed via abi package.
			Data: (new(big.Int)).SetBytes(raw.Data),
		}, nil
	}
	newCBasic1Ch := make(chan *events.CBasic1)
	newCBasic2Ch := make(chan *events.CBasic2)
	watchOpts := &bind.WatchOpts{
		Start:   nil,
		Context: context.Background(),
	}
	sub1, err := WatchLogs[events.CBasic1](&boundContract, *abi, watchOpts, events.CBasic1EventID(), unpackBasic, newCBasic1Ch)
	sub2, err := WatchLogs[events.CBasic2](&boundContract, *abi, watchOpts, events.CBasic2EventID(), unpackBasic2, newCBasic2Ch)
	defer sub1.Unsubscribe()
	defer sub2.Unsubscribe()

	crtctInstance := &ContractInstance{
		Address: res.Addrs[events.CMetaData.Pattern],
		Backend: backend,
	}
	_ = ctrct
	packedCall, err := ctrct.PackEmitMulti()
	if err != nil {
		t.Fatalf("failed to pack EmitMulti arguments")
	}
	tx, err := Transact(crtctInstance, txAuth, packedCall)
	if err != nil {
		t.Fatalf("failed to send transaction...")
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
		case _ = <-newCBasic1Ch:
			e1Count++
		case _ = <-newCBasic2Ch:
			e2Count++
		case _ = <-timeout.C:
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

	// now, test that we can filter those events that were just caught through the subscription

	filterOpts := &bind.FilterOpts{
		Start:   0,
		Context: context.Background(),
	}
	// TODO: test that returning error from unpack prevents event from being received by sink.
	it, err := FilterLogs[events.CBasic1](crtctInstance, filterOpts, events.CBasic1EventID(), unpackBasic)
	if err != nil {
		t.Fatalf("error filtering logs %v\n", err)
	}
	it2, err := FilterLogs[events.CBasic2](crtctInstance, filterOpts, events.CBasic1EventID(), unpackBasic2)
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
	if e2Count != 1 {
		t.Fatalf("bad")
	}
	if e1Count != 1 {
		t.Fatalf("bad")
	}
}

func TestEventsUnpackFailure(t *testing.T) {
	// test watch/filter logs method on a contract that emits various kinds of events (struct-containing, etc.)
	txAuth, backend, err := testSetup()
	if err != nil {
		t.Fatalf("error setting up testing env: %v", err)
	}

	deploymentParams := DeploymentParams{
		Contracts: []ContractDeployParams{
			{
				Meta: events.CMetaData,
			},
		},
	}

	res, err := LinkAndDeploy(txAuth, backend, deploymentParams)
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

	abi, err := events.CMetaData.GetAbi()
	if err != nil {
		t.Fatalf("error getting contract abi: %v", err)
	}

	// TODO: why did I introduce separate type, and not just use bound contract?
	boundContract := ContractInstance{
		res.Addrs[events.CMetaData.Pattern],
		backend,
	}

	unpackBasic := func(raw *types.Log) (*events.CBasic1, error) {
		return nil, fmt.Errorf("this error should stop the filter that uses this unpack.")
	}
	unpackBasic2 := func(raw *types.Log) (*events.CBasic2, error) {
		return &events.CBasic2{
			Flag: false, // TODO: how to unpack different types to go types?  this should be exposed via abi package.
			Data: (new(big.Int)).SetBytes(raw.Data),
		}, nil
	}
	newCBasic1Ch := make(chan *events.CBasic1)
	newCBasic2Ch := make(chan *events.CBasic2)
	watchOpts := &bind.WatchOpts{
		Start:   nil,
		Context: context.Background(),
	}
	sub1, err := WatchLogs[events.CBasic1](&boundContract, *abi, watchOpts, events.CBasic1EventID(), unpackBasic, newCBasic1Ch)
	sub2, err := WatchLogs[events.CBasic2](&boundContract, *abi, watchOpts, events.CBasic2EventID(), unpackBasic2, newCBasic2Ch)
	defer sub1.Unsubscribe()
	defer sub2.Unsubscribe()

	crtctInstance := &ContractInstance{
		Address: res.Addrs[events.CMetaData.Pattern],
		Backend: backend,
	}
	_ = ctrct
	packedCall, err := ctrct.PackEmitMulti()
	if err != nil {
		t.Fatalf("failed to pack EmitMulti arguments")
	}
	tx, err := Transact(crtctInstance, txAuth, packedCall)
	if err != nil {
		t.Fatalf("failed to send transaction...")
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
		case _ = <-newCBasic1Ch:
			e1Count++
		case _ = <-newCBasic2Ch:
			e2Count++
		case _ = <-timeout.C:
			goto done
		}
		if e1Count == 2 && e2Count == 1 {
			break
		}
	}
done:
	if e1Count != 0 {
		t.Fatalf("expected event type 1 count to be 0.  got %d", e1Count)
	}
	if e2Count != 1 {
		t.Fatalf("expected event type 2 count to be 1.  got %d", e2Count)
	}
}
