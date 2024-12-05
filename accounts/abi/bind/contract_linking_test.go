package bind

import (
	v2 "github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"strings"
	"testing"
)

type linkTestCase struct {
	// map of a library to the order in which its dependencies appear in the EVM bytecode.
	codes map[string]string
}

func makeLinkTestCase(input map[rune][]rune) *linkTestCase {
	codes := make(map[string]string)
	inputMap := make(map[rune]map[rune]struct{})

	for contract, deps := range input {
		inputMap[contract] = make(map[rune]struct{})
		for _, dep := range deps {
			codes[string(contract)] = codes[string(contract)] + string(dep)
			inputMap[contract][dep] = struct{}{}
		}
	}
	return &linkTestCase{
		generateUnlinkedContracts(codes),
	}
}

func generateUnlinkedContracts(inputCodes map[string]string) map[string]string {
	// map of solidity library pattern to unlinked code
	codes := make(map[string]string)

	for name, code := range inputCodes {
		var prelinkCode []string
		for _, char := range code {
			prelinkCode = append(prelinkCode, crypto.Keccak256Hash([]byte(string(char))).String()[2:36])
		}
		pattern := crypto.Keccak256Hash([]byte(string(name))).String()[2:36]
		codes[pattern] = strings.Join(prelinkCode, "")
	}
	return codes
}

var testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

func testLinkCase(t *testing.T, input map[rune][]rune, overrides map[rune]common.Address) {
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)
	var testAddrNonce uint64

	tc := makeLinkTestCase(input)
	alreadyDeployed := make(map[common.Address]struct{})
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
				t.Fatalf("reference to dependent contract that has not yet been deployed.")
			}
		}
		alreadyDeployed[contractAddr] = struct{}{}
		// we don't care about the txs themselves for the sake of the linking tests.  so we can return nil for them in the mock deployer
		return contractAddr, nil, nil
	}
	doTest := func() {
		// convert the raw test case into working form

		deployParams := v2.DeploymentParams{
			Contracts: nil,
			Libraries: nil,
			Overrides: nil,
		}
		res, err := v2.LinkAndDeploy(deployParams, mockDeploy)
		if err != nil {
			t.Fatalf("got error from LinkAndDeploy: %v\n", err)
		}

		// assert that res contains everything we expected to be deployed
	}
}

func TestContractLinking(t *testing.T) {

	//test-case specific values (TODO: move these, mockDeploy, doTest into their own routines).

	// input:  a cycle of dependencies.
	// expected: [{set of deps deployed first}, {set of deps deployed second}, ..., {contract(s) deployed}]
}
