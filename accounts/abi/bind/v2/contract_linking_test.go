package v2

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/exp/rand"
	"testing"
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

type linkTestCaseInput struct {
	input          map[rune][]rune
	overrides      map[rune]struct{}
	expectDeployed map[rune]struct{}
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

	var (
		deployParams DeploymentParams
	)
	for pattern, bin := range tc.contractCodes {
		deployParams.Contracts = append(deployParams.Contracts, &bind.MetaData{Pattern: pattern, Bin: "0x" + bin})
	}
	for pattern, bin := range tc.libCodes {
		deployParams.Contracts = append(deployParams.Contracts, &bind.MetaData{
			Bin:     "0x" + bin,
			Pattern: pattern,
		})
	}

	overridePatterns := make(map[string]common.Address)
	for pattern, override := range tc.overrides {
		overridePatterns[pattern] = override
	}
	deployParams.Overrides = overridePatterns

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
	testLinkCase(t, linkTestCaseInput{
		map[rune][]rune{
			'a': {'b', 'c', 'd', 'e'}},
		map[rune]struct{}{},
		map[rune]struct{}{
			'a': {}, 'b': {}, 'c': {}, 'd': {}, 'e': {},
		},
	})

	testLinkCase(t, linkTestCaseInput{
		map[rune][]rune{
			'a': {'b', 'c', 'd', 'e'},
			'e': {'f', 'g', 'h', 'i'}},
		map[rune]struct{}{},
		map[rune]struct{}{
			'a': {}, 'b': {}, 'c': {}, 'd': {}, 'e': {}, 'f': {}, 'g': {}, 'h': {}, 'i': {},
		}})

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
			'a': {},
		}})

	// test one contract with overrides for some lib deps
	testLinkCase(t, linkTestCaseInput{
		map[rune][]rune{
			'a': {'b', 'c'}},
		map[rune]struct{}{'b': {}, 'c': {}},
		map[rune]struct{}{
			'a': {},
		}})

	// test deployment of a contract with overrides
	testLinkCase(t, linkTestCaseInput{
		map[rune][]rune{
			'a': {}},
		map[rune]struct{}{'a': {}},
		map[rune]struct{}{}})

	// two contracts share some dependencies.  one contract is marked as an override.  only the override contract
	// is not deployed.  its dependencies are all deployed (even the ones not used by f), and the ones shared with f
	// are not redeployed
	testLinkCase(t, linkTestCaseInput{map[rune][]rune{
		'a': {'b', 'c', 'd', 'e'},
		'f': {'g', 'c', 'd', 'h'}},
		map[rune]struct{}{'a': {}},
		map[rune]struct{}{
			'b': {}, 'c': {}, 'd': {}, 'e': {}, 'f': {}, 'g': {}, 'h': {},
		}})

	// test nested libraries that share deps at different levels of the tree... with override.
	testLinkCase(t, linkTestCaseInput{
		map[rune][]rune{
			'a': {'b', 'c', 'd', 'e'},
			'e': {'f', 'g', 'h', 'i', 'm'},
			'i': {'j', 'k', 'l', 'm'}},
		map[rune]struct{}{
			'i': {},
		},
		map[rune]struct{}{
			'a': {}, 'b': {}, 'c': {}, 'd': {}, 'e': {}, 'f': {}, 'g': {}, 'h': {}, 'j': {}, 'k': {}, 'l': {}, 'm': {},
		}})
	// TODO: same as the above case but nested one level of dependencies deep (?)
}
