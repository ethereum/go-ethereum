package v2

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
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

func testLinkCase(t *testing.T, input map[rune][]rune, overrides map[rune]common.Address) {
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)
	var testAddrNonce uint64

	tc := makeLinkTestCase(input, overrides)
	alreadyDeployed := make(map[common.Address]struct{})
	allContracts := make(map[rune]struct{})

	for contract, deps := range input {
		allContracts[contract] = struct{}{}
		for _, dep := range deps {
			allContracts[dep] = struct{}{}
		}
	}

	// TODO: include in link test case: set of contracts that we expect to be deployed at the end.
	// generate this in makeLinkTestCase
	// ^ overrides are not included in this case.
	mockDeploy := func(input []byte, deployer []byte) (common.Address, *types.Transaction, error) {
		contractAddr := crypto.CreateAddress(testAddr, testAddrNonce)
		testAddrNonce++

		// assert that this contract only references libs that are known to be deployed or in the override set
		for i := 0; i < len(deployer)/20; i += 20 {
			var dep common.Address
			dep.SetBytes(deployer[i : i+20])
			if _, ok := alreadyDeployed[dep]; !ok {
				t.Fatalf("reference to dependent contract that has not yet been deployed: %x\n", dep)
			}
		}
		alreadyDeployed[contractAddr] = struct{}{}
		// we don't care about the txs themselves for the sake of the linking tests.  so we can return nil for them in the mock deployer
		return contractAddr, nil, nil
	}

	var (
		contracts []ContractDeployParams
		libs      []*bind.MetaData
	)
	for pattern, bin := range tc.contractCodes {
		contracts = append(contracts, ContractDeployParams{
			Meta:  &bind.MetaData{Pattern: pattern, Bin: "0x" + bin},
			Input: nil,
		})
	}
	for pattern, bin := range tc.libCodes {
		libs = append(libs, &bind.MetaData{
			Bin:     "0x" + bin,
			Pattern: pattern,
		})
	}
	deployParams := DeploymentParams{
		Contracts: contracts,
		Libraries: libs,
		Overrides: nil,
	}

	res, err := LinkAndDeploy(deployParams, mockDeploy)
	if err != nil {
		t.Fatalf("got error from LinkAndDeploy: %v\n", err)
	}

	// TODO: assert that the result consists of the input contracts minus the overrides.

	if len(res.Addrs) != len(allContracts)-len(overrides) {
		for val, _ := range allContracts {
			fmt.Println(string(val))
		}
		t.Fatalf("expected %d contracts to be deployed.  got %d\n", len(allContracts)-len(overrides), len(res.Addrs))
	}

	// note that the link-and-deploy functionality assumes that the combined-abi is well-formed.

	// test-case ideas:
	// * libraries that are disjount from the rest of dep graph (they don't get deployed)
}

func TestContractLinking(t *testing.T) {
	testLinkCase(t, map[rune][]rune{
		'a': {'b', 'c', 'd', 'e'},
		'e': {'f', 'g', 'h', 'i'}},
		map[rune]common.Address{})

	testLinkCase(t, map[rune][]rune{
		'a': {'b', 'c', 'd', 'e'}},
		map[rune]common.Address{})

	// test single contract only without deps
	testLinkCase(t, map[rune][]rune{
		'a': {}},
		map[rune]common.Address{})

	// test that libraries at different levels of the tree can share deps,
	// and that these shared deps will only be deployed once.
	testLinkCase(t, map[rune][]rune{
		'a': {'b', 'c', 'd', 'e'},
		'e': {'f', 'g', 'h', 'i', 'm'},
		'i': {'j', 'k', 'l', 'm'}},
		map[rune]common.Address{})

	// test two contracts can be deployed which don't share deps
	testLinkCase(t, map[rune][]rune{
		'a': {'b', 'c', 'd', 'e'},
		'f': {'g', 'h', 'i', 'j'}},
		map[rune]common.Address{})

	// test two contracts can be deployed which share deps
	testLinkCase(t, map[rune][]rune{
		'a': {'b', 'c', 'd', 'e'},
		'f': {'g', 'c', 'd', 'j'}},
		map[rune]common.Address{})
}
