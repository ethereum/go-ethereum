// Copyright 2016 The go-ethereum Authors
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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ubiq/go-ubiq/common"
	"golang.org/x/tools/imports"
)

var bindTests = []struct {
	name     string
	contract string
	bytecode string
	abi      string
	tester   string
}{
	// Test that the binding is available in combined and separate forms too
	{
		`Empty`,
		`contract NilContract {}`,
		`606060405260068060106000396000f3606060405200`,
		`[]`,
		`
			if b, err := NewEmpty(common.Address{}, nil); b == nil || err != nil {
				t.Fatalf("combined binding (%v) nil or error (%v) not nil", b, nil)
			}
			if b, err := NewEmptyCaller(common.Address{}, nil); b == nil || err != nil {
				t.Fatalf("caller binding (%v) nil or error (%v) not nil", b, nil)
			}
			if b, err := NewEmptyTransactor(common.Address{}, nil); b == nil || err != nil {
				t.Fatalf("transactor binding (%v) nil or error (%v) not nil", b, nil)
			}
		`,
	},
	// Test that all the official sample contracts bind correctly
	{
		`Token`,
		`https://ethereum.org/token`,
		`60606040526040516107fd3803806107fd83398101604052805160805160a05160c051929391820192909101600160a060020a0333166000908152600360209081526040822086905581548551838052601f6002600019610100600186161502019093169290920482018390047f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e56390810193919290918801908390106100e857805160ff19168380011785555b506101189291505b8082111561017157600081556001016100b4565b50506002805460ff19168317905550505050610658806101a56000396000f35b828001600101855582156100ac579182015b828111156100ac5782518260005055916020019190600101906100fa565b50508060016000509080519060200190828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f1061017557805160ff19168380011785555b506100c89291506100b4565b5090565b82800160010185558215610165579182015b8281111561016557825182600050559160200191906001019061018756606060405236156100775760e060020a600035046306fdde03811461007f57806323b872dd146100dc578063313ce5671461010e57806370a082311461011a57806395d89b4114610132578063a9059cbb1461018e578063cae9ca51146101bd578063dc3080f21461031c578063dd62ed3e14610341575b610365610002565b61036760008054602060026001831615610100026000190190921691909104601f810182900490910260809081016040526060828152929190828280156104eb5780601f106104c0576101008083540402835291602001916104eb565b6103d5600435602435604435600160a060020a038316600090815260036020526040812054829010156104f357610002565b6103e760025460ff1681565b6103d560043560036020526000908152604090205481565b610367600180546020600282841615610100026000190190921691909104601f810182900490910260809081016040526060828152929190828280156104eb5780601f106104c0576101008083540402835291602001916104eb565b610365600435602435600160a060020a033316600090815260036020526040902054819010156103f157610002565b60806020604435600481810135601f8101849004909302840160405260608381526103d5948235946024803595606494939101919081908382808284375094965050505050505060006000836004600050600033600160a060020a03168152602001908152602001600020600050600087600160a060020a031681526020019081526020016000206000508190555084905080600160a060020a0316638f4ffcb1338630876040518560e060020a0281526004018085600160a060020a0316815260200184815260200183600160a060020a03168152602001806020018281038252838181518152602001915080519060200190808383829060006004602084601f0104600f02600301f150905090810190601f1680156102f25780820380516001836020036101000a031916815260200191505b50955050505050506000604051808303816000876161da5a03f11561000257505050509392505050565b6005602090815260043560009081526040808220909252602435815220546103d59081565b60046020818152903560009081526040808220909252602435815220546103d59081565b005b60405180806020018281038252838181518152602001915080519060200190808383829060006004602084601f0104600f02600301f150905090810190601f1680156103c75780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b60408051918252519081900360200190f35b6060908152602090f35b600160a060020a03821660009081526040902054808201101561041357610002565b806003600050600033600160a060020a03168152602001908152602001600020600082828250540392505081905550806003600050600084600160a060020a0316815260200190815260200160002060008282825054019250508190555081600160a060020a031633600160a060020a03167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef836040518082815260200191505060405180910390a35050565b820191906000526020600020905b8154815290600101906020018083116104ce57829003601f168201915b505050505081565b600160a060020a03831681526040812054808301101561051257610002565b600160a060020a0380851680835260046020908152604080852033949094168086529382528085205492855260058252808520938552929052908220548301111561055c57610002565b816003600050600086600160a060020a03168152602001908152602001600020600082828250540392505081905550816003600050600085600160a060020a03168152602001908152602001600020600082828250540192505081905550816005600050600086600160a060020a03168152602001908152602001600020600050600033600160a060020a0316815260200190815260200160002060008282825054019250508190555082600160a060020a031633600160a060020a03167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef846040518082815260200191505060405180910390a3939250505056`,
		`[{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"type":"function"},{"constant":false,"inputs":[{"name":"_from","type":"address"},{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"success","type":"bool"}],"type":"function"},{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"type":"function"},{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"type":"function"},{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[],"type":"function"},{"constant":false,"inputs":[{"name":"_spender","type":"address"},{"name":"_value","type":"uint256"},{"name":"_extraData","type":"bytes"}],"name":"approveAndCall","outputs":[{"name":"success","type":"bool"}],"type":"function"},{"constant":true,"inputs":[{"name":"","type":"address"},{"name":"","type":"address"}],"name":"spentAllowance","outputs":[{"name":"","type":"uint256"}],"type":"function"},{"constant":true,"inputs":[{"name":"","type":"address"},{"name":"","type":"address"}],"name":"allowance","outputs":[{"name":"","type":"uint256"}],"type":"function"},{"inputs":[{"name":"initialSupply","type":"uint256"},{"name":"tokenName","type":"string"},{"name":"decimalUnits","type":"uint8"},{"name":"tokenSymbol","type":"string"}],"type":"constructor"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]`,
		`
			if b, err := NewToken(common.Address{}, nil); b == nil || err != nil {
				t.Fatalf("binding (%v) nil or error (%v) not nil", b, nil)
			}
		`,
	},
	{
		`Crowdsale`,
		`https://ethereum.org/crowdsale`,
		`606060408190526007805460ff1916905560a0806105a883396101006040529051608051915160c05160e05160008054600160a060020a03199081169095178155670de0b6b3a7640000958602600155603c9093024201600355930260045560058054909216909217905561052f90819061007990396000f36060604052361561006c5760e060020a600035046301cb3b20811461008257806329dcb0cf1461014457806338af3eed1461014d5780636e66f6e91461015f5780637a3a0e84146101715780637b3e5e7b1461017a578063a035b1fe14610183578063dc0d3dff1461018c575b61020060075460009060ff161561032357610002565b61020060035460009042106103205760025460015490106103cb576002548154600160a060020a0316908290606082818181858883f150915460025460408051600160a060020a039390931683526020830191909152818101869052517fe842aea7a5f1b01049d752008c53c52890b1a6daf660cf39e8eec506112bbdf6945090819003909201919050a15b60405160008054600160a060020a039081169230909116319082818181858883f150506007805460ff1916600117905550505050565b6103a160035481565b6103ab600054600160a060020a031681565b6103ab600554600160a060020a031681565b6103a160015481565b6103a160025481565b6103a160045481565b6103be60043560068054829081101561000257506000526002027ff652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d3f8101547ff652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d409190910154600160a060020a03919091169082565b005b505050815481101561000257906000526020600020906002020160005060008201518160000160006101000a815481600160a060020a030219169083021790555060208201518160010160005055905050806002600082828250540192505081905550600560009054906101000a9004600160a060020a0316600160a060020a031663a9059cbb3360046000505484046040518360e060020a0281526004018083600160a060020a03168152602001828152602001925050506000604051808303816000876161da5a03f11561000257505060408051600160a060020a03331681526020810184905260018183015290517fe842aea7a5f1b01049d752008c53c52890b1a6daf660cf39e8eec506112bbdf692509081900360600190a15b50565b5060a0604052336060908152346080819052600680546001810180835592939282908280158290116102025760020281600202836000526020600020918201910161020291905b8082111561039d57805473ffffffffffffffffffffffffffffffffffffffff19168155600060019190910190815561036a565b5090565b6060908152602090f35b600160a060020a03166060908152602090f35b6060918252608052604090f35b5b60065481101561010e576006805482908110156100025760009182526002027ff652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d3f0190600680549254600160a060020a0316928490811015610002576002027ff652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d40015460405190915082818181858883f19350505050507fe842aea7a5f1b01049d752008c53c52890b1a6daf660cf39e8eec506112bbdf660066000508281548110156100025760008290526002027ff652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d3f01548154600160a060020a039190911691908490811015610002576002027ff652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d40015460408051600160a060020a0394909416845260208401919091526000838201525191829003606001919050a16001016103cc56`,
		`[{"constant":false,"inputs":[],"name":"checkGoalReached","outputs":[],"type":"function"},{"constant":true,"inputs":[],"name":"deadline","outputs":[{"name":"","type":"uint256"}],"type":"function"},{"constant":true,"inputs":[],"name":"beneficiary","outputs":[{"name":"","type":"address"}],"type":"function"},{"constant":true,"inputs":[],"name":"tokenReward","outputs":[{"name":"","type":"address"}],"type":"function"},{"constant":true,"inputs":[],"name":"fundingGoal","outputs":[{"name":"","type":"uint256"}],"type":"function"},{"constant":true,"inputs":[],"name":"amountRaised","outputs":[{"name":"","type":"uint256"}],"type":"function"},{"constant":true,"inputs":[],"name":"price","outputs":[{"name":"","type":"uint256"}],"type":"function"},{"constant":true,"inputs":[{"name":"","type":"uint256"}],"name":"funders","outputs":[{"name":"addr","type":"address"},{"name":"amount","type":"uint256"}],"type":"function"},{"inputs":[{"name":"ifSuccessfulSendTo","type":"address"},{"name":"fundingGoalInEthers","type":"uint256"},{"name":"durationInMinutes","type":"uint256"},{"name":"etherCostOfEachToken","type":"uint256"},{"name":"addressOfTokenUsedAsReward","type":"address"}],"type":"constructor"},{"anonymous":false,"inputs":[{"indexed":false,"name":"backer","type":"address"},{"indexed":false,"name":"amount","type":"uint256"},{"indexed":false,"name":"isContribution","type":"bool"}],"name":"FundTransfer","type":"event"}]`,
		`
			if b, err := NewCrowdsale(common.Address{}, nil); b == nil || err != nil {
				t.Fatalf("binding (%v) nil or error (%v) not nil", b, nil)
			}
		`,
	},
	// Test that named and anonymous inputs are handled correctly
	{
		`InputChecker`, ``, ``,
		`
			[
				{"type":"function","name":"noInput","constant":true,"inputs":[],"outputs":[]},
				{"type":"function","name":"namedInput","constant":true,"inputs":[{"name":"str","type":"string"}],"outputs":[]},
				{"type":"function","name":"anonInput","constant":true,"inputs":[{"name":"","type":"string"}],"outputs":[]},
				{"type":"function","name":"namedInputs","constant":true,"inputs":[{"name":"str1","type":"string"},{"name":"str2","type":"string"}],"outputs":[]},
				{"type":"function","name":"anonInputs","constant":true,"inputs":[{"name":"","type":"string"},{"name":"","type":"string"}],"outputs":[]},
				{"type":"function","name":"mixedInputs","constant":true,"inputs":[{"name":"","type":"string"},{"name":"str","type":"string"}],"outputs":[]}
			]
		`,
		`if b, err := NewInputChecker(common.Address{}, nil); b == nil || err != nil {
			 t.Fatalf("binding (%v) nil or error (%v) not nil", b, nil)
		 } else if false { // Don't run, just compile and test types
			 var err error

			 err = b.NoInput(nil)
			 err = b.NamedInput(nil, "")
			 err = b.AnonInput(nil, "")
			 err = b.NamedInputs(nil, "", "")
			 err = b.AnonInputs(nil, "", "")
			 err = b.MixedInputs(nil, "", "")

			 fmt.Println(err)
		 }`,
	},
	// Test that named and anonymous outputs are handled correctly
	{
		`OutputChecker`, ``, ``,
		`
			[
				{"type":"function","name":"noOutput","constant":true,"inputs":[],"outputs":[]},
				{"type":"function","name":"namedOutput","constant":true,"inputs":[],"outputs":[{"name":"str","type":"string"}]},
				{"type":"function","name":"anonOutput","constant":true,"inputs":[],"outputs":[{"name":"","type":"string"}]},
				{"type":"function","name":"namedOutputs","constant":true,"inputs":[],"outputs":[{"name":"str1","type":"string"},{"name":"str2","type":"string"}]},
				{"type":"function","name":"anonOutputs","constant":true,"inputs":[],"outputs":[{"name":"","type":"string"},{"name":"","type":"string"}]},
				{"type":"function","name":"mixedOutputs","constant":true,"inputs":[],"outputs":[{"name":"","type":"string"},{"name":"str","type":"string"}]}
			]
		`,
		`if b, err := NewOutputChecker(common.Address{}, nil); b == nil || err != nil {
			 t.Fatalf("binding (%v) nil or error (%v) not nil", b, nil)
		 } else if false { // Don't run, just compile and test types
			 var str1, str2 string
			 var err error

			 err              = b.NoOutput(nil)
			 str1, err        = b.NamedOutput(nil)
			 str1, err        = b.AnonOutput(nil)
			 res, _          := b.NamedOutputs(nil)
			 str1, str2, err  = b.AnonOutputs(nil)
			 str1, str2, err  = b.MixedOutputs(nil)

			 fmt.Println(str1, str2, res.Str1, res.Str2, err)
		 }`,
	},
	// Test that contract interactions (deploy, transact and call) generate working code
	{
		`Interactor`,
		`
			contract Interactor {
				string public deployString;
				string public transactString;

				function Interactor(string str) {
				  deployString = str;
				}

				function transact(string str) {
				  transactString = str;
				}
			}
		`,
		`6060604052604051610328380380610328833981016040528051018060006000509080519060200190828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f10608d57805160ff19168380011785555b50607c9291505b8082111560ba57838155600101606b565b50505061026a806100be6000396000f35b828001600101855582156064579182015b828111156064578251826000505591602001919060010190609e565b509056606060405260e060020a60003504630d86a0e181146100315780636874e8091461008d578063d736c513146100ea575b005b610190600180546020600282841615610100026000190190921691909104601f810182900490910260809081016040526060828152929190828280156102295780601f106101fe57610100808354040283529160200191610229565b61019060008054602060026001831615610100026000190190921691909104601f810182900490910260809081016040526060828152929190828280156102295780601f106101fe57610100808354040283529160200191610229565b60206004803580820135601f81018490049093026080908101604052606084815261002f946024939192918401918190838280828437509496505050505050508060016000509080519060200190828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f1061023157805160ff19168380011785555b506102619291505b808211156102665760008155830161017d565b60405180806020018281038252838181518152602001915080519060200190808383829060006004602084601f0104600f02600301f150905090810190601f1680156101f05780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b820191906000526020600020905b81548152906001019060200180831161020c57829003601f168201915b505050505081565b82800160010185558215610175579182015b82811115610175578251826000505591602001919060010190610243565b505050565b509056`,
		`[{"constant":true,"inputs":[],"name":"transactString","outputs":[{"name":"","type":"string"}],"type":"function"},{"constant":true,"inputs":[],"name":"deployString","outputs":[{"name":"","type":"string"}],"type":"function"},{"constant":false,"inputs":[{"name":"str","type":"string"}],"name":"transact","outputs":[],"type":"function"},{"inputs":[{"name":"str","type":"string"}],"type":"constructor"}]`,
		`
			// Generate a new random account and a funded simulator
			key, _ := crypto.GenerateKey()
			auth := bind.NewKeyedTransactor(key)
			sim := backends.NewSimulatedBackend(core.GenesisAccount{Address: auth.From, Balance: big.NewInt(10000000000)})

			// Deploy an interaction tester contract and call a transaction on it
			_, _, interactor, err := DeployInteractor(auth, sim, "Deploy string")
			if err != nil {
				t.Fatalf("Failed to deploy interactor contract: %v", err)
			}
			if _, err := interactor.Transact(auth, "Transact string"); err != nil {
				t.Fatalf("Failed to transact with interactor contract: %v", err)
			}
			// Commit all pending transactions in the simulator and check the contract state
			sim.Commit()

			if str, err := interactor.DeployString(nil); err != nil {
				t.Fatalf("Failed to retrieve deploy string: %v", err)
			} else if str != "Deploy string" {
				t.Fatalf("Deploy string mismatch: have '%s', want 'Deploy string'", str)
			}
			if str, err := interactor.TransactString(nil); err != nil {
				t.Fatalf("Failed to retrieve transact string: %v", err)
			} else if str != "Transact string" {
				t.Fatalf("Transact string mismatch: have '%s', want 'Transact string'", str)
			}
		`,
	},
	// Tests that plain values can be properly returned and deserialized
	{
		`Getter`,
		`
			contract Getter {
				function getter() constant returns (string, int, bytes32) {
					return ("Hi", 1, sha3(""));
				}
			}
		`,
		`606060405260dc8060106000396000f3606060405260e060020a6000350463993a04b78114601a575b005b600060605260c0604052600260809081527f486900000000000000000000000000000000000000000000000000000000000060a05260017fc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a47060e0829052610100819052606060c0908152600261012081905281906101409060a09080838184600060046012f1505081517fffff000000000000000000000000000000000000000000000000000000000000169091525050604051610160819003945092505050f3`,
		`[{"constant":true,"inputs":[],"name":"getter","outputs":[{"name":"","type":"string"},{"name":"","type":"int256"},{"name":"","type":"bytes32"}],"type":"function"}]`,
		`
			// Generate a new random account and a funded simulator
			key, _ := crypto.GenerateKey()
			auth := bind.NewKeyedTransactor(key)
			sim := backends.NewSimulatedBackend(core.GenesisAccount{Address: auth.From, Balance: big.NewInt(10000000000)})

			// Deploy a tuple tester contract and execute a structured call on it
			_, _, getter, err := DeployGetter(auth, sim)
			if err != nil {
				t.Fatalf("Failed to deploy getter contract: %v", err)
			}
			sim.Commit()

			if str, num, _, err := getter.Getter(nil); err != nil {
				t.Fatalf("Failed to call anonymous field retriever: %v", err)
			} else if str != "Hi" || num.Cmp(big.NewInt(1)) != 0 {
				t.Fatalf("Retrieved value mismatch: have %v/%v, want %v/%v", str, num, "Hi", 1)
			}
		`,
	},
	// Tests that tuples can be properly returned and deserialized
	{
		`Tupler`,
		`
			contract Tupler {
				function tuple() constant returns (string a, int b, bytes32 c) {
					return ("Hi", 1, sha3(""));
				}
			}
		`,
		`606060405260dc8060106000396000f3606060405260e060020a60003504633175aae28114601a575b005b600060605260c0604052600260809081527f486900000000000000000000000000000000000000000000000000000000000060a05260017fc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a47060e0829052610100819052606060c0908152600261012081905281906101409060a09080838184600060046012f1505081517fffff000000000000000000000000000000000000000000000000000000000000169091525050604051610160819003945092505050f3`,
		`[{"constant":true,"inputs":[],"name":"tuple","outputs":[{"name":"a","type":"string"},{"name":"b","type":"int256"},{"name":"c","type":"bytes32"}],"type":"function"}]`,
		`
			// Generate a new random account and a funded simulator
			key, _ := crypto.GenerateKey()
			auth := bind.NewKeyedTransactor(key)
			sim := backends.NewSimulatedBackend(core.GenesisAccount{Address: auth.From, Balance: big.NewInt(10000000000)})

			// Deploy a tuple tester contract and execute a structured call on it
			_, _, tupler, err := DeployTupler(auth, sim)
			if err != nil {
				t.Fatalf("Failed to deploy tupler contract: %v", err)
			}
			sim.Commit()

			if res, err := tupler.Tuple(nil); err != nil {
				t.Fatalf("Failed to call structure retriever: %v", err)
			} else if res.A != "Hi" || res.B.Cmp(big.NewInt(1)) != 0 {
				t.Fatalf("Retrieved value mismatch: have %v/%v, want %v/%v", res.A, res.B, "Hi", 1)
			}
		`,
	},
	// Tests that arrays/slices can be properly returned and deserialized.
	// Only addresses are tested, remainder just compiled to keep the test small.
	{
		`Slicer`,
		`
			contract Slicer {
				function echoAddresses(address[] input) constant returns (address[] output) {
					return input;
				}
				function echoInts(int[] input) constant returns (int[] output) {
					return input;
				}
				function echoFancyInts(uint24[23] input) constant returns (uint24[23] output) {
					return input;
				}
				function echoBools(bool[] input) constant returns (bool[] output) {
					return input;
				}
			}
		`,
		`606060405261015c806100126000396000f3606060405260e060020a6000350463be1127a3811461003c578063d88becc014610092578063e15a3db71461003c578063f637e5891461003c575b005b604080516020600480358082013583810285810185019096528085526100ee959294602494909392850192829185019084908082843750949650505050505050604080516020810190915260009052805b919050565b604080516102e0818101909252610138916004916102e491839060179083908390808284375090955050505050506102e0604051908101604052806017905b60008152602001906001900390816100d15790505081905061008d565b60405180806020018281038252838181518152602001915080519060200190602002808383829060006004602084601f0104600f02600301f1509050019250505060405180910390f35b60405180826102e0808381846000600461015cf15090500191505060405180910390f3`,
		`[{"constant":true,"inputs":[{"name":"input","type":"address[]"}],"name":"echoAddresses","outputs":[{"name":"output","type":"address[]"}],"type":"function"},{"constant":true,"inputs":[{"name":"input","type":"uint24[23]"}],"name":"echoFancyInts","outputs":[{"name":"output","type":"uint24[23]"}],"type":"function"},{"constant":true,"inputs":[{"name":"input","type":"int256[]"}],"name":"echoInts","outputs":[{"name":"output","type":"int256[]"}],"type":"function"},{"constant":true,"inputs":[{"name":"input","type":"bool[]"}],"name":"echoBools","outputs":[{"name":"output","type":"bool[]"}],"type":"function"}]`,
		`
			// Generate a new random account and a funded simulator
			key, _ := crypto.GenerateKey()
			auth := bind.NewKeyedTransactor(key)
			sim := backends.NewSimulatedBackend(core.GenesisAccount{Address: auth.From, Balance: big.NewInt(10000000000)})

			// Deploy a slice tester contract and execute a n array call on it
			_, _, slicer, err := DeploySlicer(auth, sim)
			if err != nil {
					t.Fatalf("Failed to deploy slicer contract: %v", err)
			}
			sim.Commit()

			if out, err := slicer.EchoAddresses(nil, []common.Address{auth.From, common.Address{}}); err != nil {
					t.Fatalf("Failed to call slice echoer: %v", err)
			} else if !reflect.DeepEqual(out, []common.Address{auth.From, common.Address{}}) {
					t.Fatalf("Slice return mismatch: have %v, want %v", out, []common.Address{auth.From, common.Address{}})
			}
		`,
	},
	// Tests that anonymous default methods can be correctly invoked
	{
		`Defaulter`,
		`
			contract Defaulter {
				address public caller;

				function() {
					caller = msg.sender;
				}
			}
		`,
		`6060604052606a8060106000396000f360606040523615601d5760e060020a6000350463fc9c8d3981146040575b605e6000805473ffffffffffffffffffffffffffffffffffffffff191633179055565b606060005473ffffffffffffffffffffffffffffffffffffffff1681565b005b6060908152602090f3`,
		`[{"constant":true,"inputs":[],"name":"caller","outputs":[{"name":"","type":"address"}],"type":"function"}]`,
		`
			// Generate a new random account and a funded simulator
			key, _ := crypto.GenerateKey()
			auth := bind.NewKeyedTransactor(key)
			sim := backends.NewSimulatedBackend(core.GenesisAccount{Address: auth.From, Balance: big.NewInt(10000000000)})

			// Deploy a default method invoker contract and execute its default method
			_, _, defaulter, err := DeployDefaulter(auth, sim)
			if err != nil {
				t.Fatalf("Failed to deploy defaulter contract: %v", err)
			}
			if _, err := (&DefaulterRaw{defaulter}).Transfer(auth); err != nil {
				t.Fatalf("Failed to invoke default method: %v", err)
			}
			sim.Commit()

			if caller, err := defaulter.Caller(nil); err != nil {
				t.Fatalf("Failed to call address retriever: %v", err)
			} else if (caller != auth.From) {
				t.Fatalf("Address mismatch: have %v, want %v", caller, auth.From)
			}
		`,
	},
	// Tests that non-existent contracts are reported as such (though only simulator test)
	{
		`NonExistent`,
		`
		contract NonExistent {
			function String() constant returns(string) {
				return "I don't exist";
			}
		}
		`,
		`6060604052609f8060106000396000f3606060405260e060020a6000350463f97a60058114601a575b005b600060605260c0604052600d60809081527f4920646f6e27742065786973740000000000000000000000000000000000000060a052602060c0908152600d60e081905281906101009060a09080838184600060046012f15050815172ffffffffffffffffffffffffffffffffffffff1916909152505060405161012081900392509050f3`,
		`[{"constant":true,"inputs":[],"name":"String","outputs":[{"name":"","type":"string"}],"type":"function"}]`,
		`
			// Create a simulator and wrap a non-deployed contract
			sim := backends.NewSimulatedBackend()

			nonexistent, err := NewNonExistent(common.Address{}, sim)
			if err != nil {
				t.Fatalf("Failed to access non-existent contract: %v", err)
			}
			// Ensure that contract calls fail with the appropriate error
			if res, err := nonexistent.String(nil); err == nil {
				t.Fatalf("Call succeeded on non-existent contract: %v", res)
			} else if (err != bind.ErrNoCode) {
				t.Fatalf("Error mismatch: have %v, want %v", err, bind.ErrNoCode)
			}
		`,
	},
}

// Tests that packages generated by the binder can be successfully compiled and
// the requested tester run against it.
func TestBindings(t *testing.T) {
	// Skip the test if no Go command can be found
	gocmd := runtime.GOROOT() + "/bin/go"
	if !common.FileExist(gocmd) {
		t.Skip("go sdk not found for testing")
	}
	// Skip the test if the go-ubiq sources are symlinked (https://github.com/golang/go/issues/14845)
	linkTestCode := fmt.Sprintf("package linktest\nfunc CheckSymlinks(){\nfmt.Println(backends.NewSimulatedBackend())\n}")
	linkTestDeps, err := imports.Process("", []byte(linkTestCode), nil)
	if err != nil {
		t.Fatalf("failed check for goimports symlink bug: %v", err)
	}
	if !strings.Contains(string(linkTestDeps), "go-ubiq") {
		t.Skip("symlinked environment doesn't support bind (https://github.com/golang/go/issues/14845)")
	}
	// Create a temporary workspace for the test suite
	ws, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("failed to create temporary workspace: %v", err)
	}
	defer os.RemoveAll(ws)

	pkg := filepath.Join(ws, "bindtest")
	if err = os.MkdirAll(pkg, 0700); err != nil {
		t.Fatalf("failed to create package: %v", err)
	}
	// Generate the test suite for all the contracts
	for i, tt := range bindTests {
		// Generate the binding and create a Go source file in the workspace
		bind, err := Bind([]string{tt.name}, []string{tt.abi}, []string{tt.bytecode}, "bindtest", LangGo)
		if err != nil {
			t.Fatalf("test %d: failed to generate binding: %v", i, err)
		}
		if err = ioutil.WriteFile(filepath.Join(pkg, strings.ToLower(tt.name)+".go"), []byte(bind), 0600); err != nil {
			t.Fatalf("test %d: failed to write binding: %v", i, err)
		}
		// Generate the test file with the injected test code
		code := fmt.Sprintf("package bindtest\nimport \"testing\"\nfunc Test%s(t *testing.T){\n%s\n}", tt.name, tt.tester)
		blob, err := imports.Process("", []byte(code), nil)
		if err != nil {
			t.Fatalf("test %d: failed to generate tests: %v", i, err)
		}
		if err := ioutil.WriteFile(filepath.Join(pkg, strings.ToLower(tt.name)+"_test.go"), blob, 0600); err != nil {
			t.Fatalf("test %d: failed to write tests: %v", i, err)
		}
	}
	// Test the entire package and report any failures
	cmd := exec.Command(gocmd, "test", "-v")
	cmd.Dir = pkg
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to run binding test: %v\n%s", err, out)
	}
}
