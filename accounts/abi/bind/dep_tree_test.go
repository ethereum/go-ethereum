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
package bind

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/exp/rand"
)

type linkTestCase struct {
	// map of pattern to unlinked bytecode (for the purposes of tests just contains the patterns of its dependencies)
	libCodes      map[string]string
	contractCodes map[string]string

	overrides map[string]common.Address
}

func makeLinkTestCase(input map[rune][]rune, overrides map[rune]common.Address) *linkTestCase {
	codes := make(map[string]string)
	libCodes := make(map[string]string)
	contractCodes := make(map[string]string)

	inputMap := make(map[rune]map[rune]struct{})
	// set of solidity patterns for all contracts that are known to be libraries
	libs := make(map[string]struct{})

	// map of test contract id (rune) to the solidity library pattern (hash of that rune)
	patternMap := map[rune]string{}

	for contract, deps := range input {
		inputMap[contract] = make(map[rune]struct{})
		if _, ok := patternMap[contract]; !ok {
			patternMap[contract] = crypto.Keccak256Hash([]byte(string(contract))).String()[2:36]
		}

		for _, dep := range deps {
			if _, ok := patternMap[dep]; !ok {
				patternMap[dep] = crypto.Keccak256Hash([]byte(string(dep))).String()[2:36]
			}
			codes[patternMap[contract]] = codes[patternMap[contract]] + fmt.Sprintf("__$%s$__", patternMap[dep])
			inputMap[contract][dep] = struct{}{}
			libs[patternMap[dep]] = struct{}{}
		}
	}
	overridesPatterns := make(map[string]common.Address)
	for contractId, overrideAddr := range overrides {
		pattern := crypto.Keccak256Hash([]byte(string(contractId))).String()[2:36]
		overridesPatterns[pattern] = overrideAddr
	}

	for _, pattern := range patternMap {
		if _, ok := libs[pattern]; ok {
			// if the library didn't depend on others, give it some dummy code to not bork deployment logic down-the-line
			if len(codes[pattern]) == 0 {
				libCodes[pattern] = "ff"
			} else {
				libCodes[pattern] = codes[pattern]
			}
		} else {
			contractCodes[pattern] = codes[pattern]
		}
	}

	return &linkTestCase{
		libCodes,
		contractCodes,
		overridesPatterns,
	}
}

var testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

type linkTestCaseInput struct {
	input          map[rune][]rune
	overrides      map[rune]struct{}
	expectDeployed map[rune]struct{}
}

// linkDeps will return a set of root dependencies and their sub-dependencies connected via the Deps field
func linkDeps(deps map[string]*MetaData) []*MetaData {
	roots := make(map[string]struct{})
	for pattern, _ := range deps {
		roots[pattern] = struct{}{}
	}

	connectedDeps := make(map[string]MetaData)
	for pattern, dep := range deps {
		connectedDeps[pattern] = __linkDeps(*dep, deps, &roots)
	}
	rootMetadatas := []*MetaData{}
	for pattern, _ := range roots {
		dep := connectedDeps[pattern]
		rootMetadatas = append(rootMetadatas, &dep)
	}
	return rootMetadatas
}

func __linkDeps(metadata MetaData, depMap map[string]*MetaData, roots *map[string]struct{}) MetaData {
	linked := metadata
	depPatterns := parseLibraryDeps(metadata.Bin)
	for _, pattern := range depPatterns {
		delete(*roots, pattern)
		connectedDep := __linkDeps(*depMap[pattern], depMap, roots)
		linked.Deps = append(linked.Deps, &connectedDep)
	}
	return linked
}

func testLinkCase(t *testing.T, tcInput linkTestCaseInput) {
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)
	var testAddrNonce uint64
	overridesAddrs := make(map[common.Address]struct{})

	// generate deterministic addresses for the override set.
	rand.Seed(42)
	overrideAddrs := make(map[rune]common.Address)
	for contract, _ := range tcInput.overrides {
		var addr common.Address
		rand.Read(addr[:])
		overrideAddrs[contract] = addr
		overridesAddrs[addr] = struct{}{}
	}

	tc := makeLinkTestCase(tcInput.input, overrideAddrs)
	allContracts := make(map[rune]struct{})

	for contract, deps := range tcInput.input {
		allContracts[contract] = struct{}{}
		for _, dep := range deps {
			allContracts[dep] = struct{}{}
		}
	}

	mockDeploy := func(input []byte, deployer []byte) (common.Address, *types.Transaction, error) {
		contractAddr := crypto.CreateAddress(testAddr, testAddrNonce)
		testAddrNonce++

		if len(deployer) >= 20 {
			// assert that this contract only references libs that are known to be deployed or in the override set
			for i := 0; i < len(deployer); i += 20 {
				var dep common.Address
				dep.SetBytes(deployer[i : i+20])
				if _, ok := overridesAddrs[dep]; !ok {
					t.Fatalf("reference to dependent contract that has not yet been deployed: %x\n", dep)
				}
			}
		}
		overridesAddrs[contractAddr] = struct{}{}
		// we don't care about the txs themselves for the sake of the linking tests.  so we can return nil for them in the mock deployer
		return contractAddr, nil, nil
	}

	contracts := make(map[string]*MetaData)
	overrides := make(map[string]common.Address)

	for pattern, bin := range tc.contractCodes {
		contracts[pattern] = &MetaData{Pattern: pattern, Bin: "0x" + bin}
	}
	for pattern, bin := range tc.libCodes {
		contracts[pattern] = &MetaData{
			Bin:     "0x" + bin,
			Pattern: pattern,
		}
	}

	contractsList := linkDeps(contracts)

	for pattern, override := range tc.overrides {
		overrides[pattern] = override
	}

	deployParams := NewDeploymentParams(contractsList, nil, overrides)
	res, err := LinkAndDeploy(deployParams, mockDeploy)
	if err != nil {
		t.Fatalf("got error from LinkAndDeploy: %v\n", err)
	}

	if len(res.Addrs) != len(tcInput.expectDeployed) {
		t.Fatalf("got %d deployed contracts.  expected %d.\n", len(res.Addrs), len(tcInput.expectDeployed))
	}
	for contract, _ := range tcInput.expectDeployed {
		pattern := crypto.Keccak256Hash([]byte(string(contract))).String()[2:36]
		if _, ok := res.Addrs[pattern]; !ok {
			t.Fatalf("expected contract %s was not deployed\n", string(contract))
		}
	}
}

func TestContractLinking(t *testing.T) {
	// test simple contract without any dependencies or overrides
	testLinkCase(t, linkTestCaseInput{
		map[rune][]rune{
			'a': {}},
		map[rune]struct{}{},
		map[rune]struct{}{
			'a': {}}})
	// test deployment of a contract that depends on somes libraries.
	testLinkCase(t, linkTestCaseInput{
		map[rune][]rune{
			'a': {'b', 'c', 'd', 'e'}},
		map[rune]struct{}{},
		map[rune]struct{}{
			'a': {}, 'b': {}, 'c': {}, 'd': {}, 'e': {}}})
	// test deployment of a contract that depends on some libraries,
	// one of which has its own library dependencies.
	testLinkCase(t, linkTestCaseInput{
		map[rune][]rune{
			'a': {'b', 'c', 'd', 'e'},
			'e': {'f', 'g', 'h', 'i'}},
		map[rune]struct{}{},
		map[rune]struct{}{
			'a': {}, 'b': {}, 'c': {}, 'd': {}, 'e': {}, 'f': {}, 'g': {}, 'h': {}, 'i': {}}})
	// test single contract only without deps
	testLinkCase(t, linkTestCaseInput{
		map[rune][]rune{
			'a': {}},
		map[rune]struct{}{},
		map[rune]struct{}{
			'a': {},
		}})
	// test that libraries at different levels of the tree can share deps,
	// and that these shared deps will only be deployed once.
	testLinkCase(t, linkTestCaseInput{
		map[rune][]rune{
			'a': {'b', 'c', 'd', 'e'},
			'e': {'f', 'g', 'h', 'i', 'm'},
			'i': {'j', 'k', 'l', 'm'}},
		map[rune]struct{}{},
		map[rune]struct{}{
			'a': {}, 'b': {}, 'c': {}, 'd': {}, 'e': {}, 'f': {}, 'g': {}, 'h': {}, 'i': {}, 'j': {}, 'k': {}, 'l': {}, 'm': {},
		}})
	// test two contracts can be deployed which don't share deps
	testLinkCase(t, linkTestCaseInput{
		map[rune][]rune{
			'a': {'b', 'c', 'd', 'e'},
			'f': {'g', 'h', 'i', 'j'}},
		map[rune]struct{}{},
		map[rune]struct{}{
			'a': {}, 'b': {}, 'c': {}, 'd': {}, 'e': {}, 'f': {}, 'g': {}, 'h': {}, 'i': {}, 'j': {},
		}})
	// test two contracts can be deployed which share deps
	testLinkCase(t, linkTestCaseInput{
		map[rune][]rune{
			'a': {'b', 'c', 'd', 'e'},
			'f': {'g', 'c', 'd', 'h'}},
		map[rune]struct{}{},
		map[rune]struct{}{
			'a': {}, 'b': {}, 'c': {}, 'd': {}, 'e': {}, 'f': {}, 'g': {}, 'h': {},
		}})
	// test one contract with overrides for all lib deps
	testLinkCase(t, linkTestCaseInput{
		map[rune][]rune{
			'a': {'b', 'c', 'd', 'e'}},
		map[rune]struct{}{'b': {}, 'c': {}, 'd': {}, 'e': {}},
		map[rune]struct{}{
			'a': {}}})
	// test one contract with overrides for some lib deps
	testLinkCase(t, linkTestCaseInput{
		map[rune][]rune{
			'a': {'b', 'c'}},
		map[rune]struct{}{'b': {}, 'c': {}},
		map[rune]struct{}{
			'a': {}}})
	// test deployment of a contract with overrides
	testLinkCase(t, linkTestCaseInput{
		map[rune][]rune{
			'a': {}},
		map[rune]struct{}{'a': {}},
		map[rune]struct{}{}})
	// two contracts ('a' and 'f') share some dependencies.  contract 'a' is marked as an override.  expect that any of
	// its depdencies that aren't shared with 'f' are not deployed.
	testLinkCase(t, linkTestCaseInput{map[rune][]rune{
		'a': {'b', 'c', 'd', 'e'},
		'f': {'g', 'c', 'd', 'h'}},
		map[rune]struct{}{'a': {}},
		map[rune]struct{}{
			'f': {}, 'g': {}, 'c': {}, 'd': {}, 'h': {}}})
	// test nested libraries that share deps at different levels of the tree... with override.
	// same condition as above test:  no sub-dependencies of
	testLinkCase(t, linkTestCaseInput{
		map[rune][]rune{
			'a': {'b', 'c', 'd', 'e'},
			'e': {'f', 'g', 'h', 'i', 'm'},
			'i': {'j', 'k', 'l', 'm'},
			'l': {'n', 'o', 'p'}},
		map[rune]struct{}{
			'i': {},
		},
		map[rune]struct{}{
			'a': {}, 'b': {}, 'c': {}, 'd': {}, 'e': {}, 'f': {}, 'g': {}, 'h': {}, 'm': {}}})
}
