package tracers

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rpc"
)

// erc20MetaData contains all meta data concerning the ERC20 contract.
var erc20MetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"pauser\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimal\",\"type\":\"uint8\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Paused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Unpaused\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"burn\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"new_operator\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"new_pauser\",\"type\":\"address\"}],\"name\":\"changeUser\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"subtractedValue\",\"type\":\"uint256\"}],\"name\":\"decreaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"addedValue\",\"type\":\"uint256\"}],\"name\":\"increaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"mint\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"unpause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Sigs: map[string]string{
		"dd62ed3e": "allowance(address,address)",
		"095ea7b3": "approve(address,uint256)",
		"70a08231": "balanceOf(address)",
		"9dc29fac": "burn(address,uint256)",
		"8e50817a": "changeUser(address,address)",
		"313ce567": "decimals()",
		"a457c2d7": "decreaseAllowance(address,uint256)",
		"39509351": "increaseAllowance(address,uint256)",
		"40c10f19": "mint(address,uint256)",
		"06fdde03": "name()",
		"8456cb59": "pause()",
		"5c975abb": "paused()",
		"95d89b41": "symbol()",
		"18160ddd": "totalSupply()",
		"a9059cbb": "transfer(address,uint256)",
		"23b872dd": "transferFrom(address,address,uint256)",
		"3f4ba83a": "unpause()",
	},
	Bin: "0x60806040523480156200001157600080fd5b50604051620014b2380380620014b2833981810160405260a08110156200003757600080fd5b815160208301516040808501805191519395929483019291846401000000008211156200006357600080fd5b9083019060208201858111156200007957600080fd5b82516401000000008111828201881017156200009457600080fd5b82525081516020918201929091019080838360005b83811015620000c3578181015183820152602001620000a9565b50505050905090810190601f168015620000f15780820380516001836020036101000a031916815260200191505b50604052602001805160405193929190846401000000008211156200011557600080fd5b9083019060208201858111156200012b57600080fd5b82516401000000008111828201881017156200014657600080fd5b82525081516020918201929091019080838360005b83811015620001755781810151838201526020016200015b565b50505050905090810190601f168015620001a35780820380516001836020036101000a031916815260200191505b5060405260209081015185519093508592508491620001c8916003918501906200026b565b508051620001de9060049060208401906200026b565b50506005805461ff001960ff1990911660121716905550600680546001600160a01b038088166001600160a01b0319928316179092556007805492871692909116919091179055620002308162000255565b50506005805462010000600160b01b0319163362010000021790555062000307915050565b6005805460ff191660ff92909216919091179055565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f10620002ae57805160ff1916838001178555620002de565b82800160010185558215620002de579182015b82811115620002de578251825591602001919060010190620002c1565b50620002ec929150620002f0565b5090565b5b80821115620002ec5760008155600101620002f1565b61119b80620003176000396000f3fe608060405234801561001057600080fd5b506004361061010b5760003560e01c80635c975abb116100a257806395d89b411161007157806395d89b41146103015780639dc29fac14610309578063a457c2d714610335578063a9059cbb14610361578063dd62ed3e1461038d5761010b565b80635c975abb1461029d57806370a08231146102a55780638456cb59146102cb5780638e50817a146102d35761010b565b8063313ce567116100de578063313ce5671461021d578063395093511461023b5780633f4ba83a1461026757806340c10f19146102715761010b565b806306fdde0314610110578063095ea7b31461018d57806318160ddd146101cd57806323b872dd146101e7575b600080fd5b6101186103bb565b6040805160208082528351818301528351919283929083019185019080838360005b8381101561015257818101518382015260200161013a565b50505050905090810190601f16801561017f5780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b6101b9600480360360408110156101a357600080fd5b506001600160a01b038135169060200135610451565b604080519115158252519081900360200190f35b6101d561046e565b60408051918252519081900360200190f35b6101b9600480360360608110156101fd57600080fd5b506001600160a01b03813581169160208101359091169060400135610474565b6102256104fb565b6040805160ff9092168252519081900360200190f35b6101b96004803603604081101561025157600080fd5b506001600160a01b038135169060200135610504565b61026f610552565b005b61026f6004803603604081101561028757600080fd5b506001600160a01b0381351690602001356105a9565b6101b9610654565b6101d5600480360360208110156102bb57600080fd5b50356001600160a01b0316610662565b61026f61067d565b61026f600480360360408110156102e957600080fd5b506001600160a01b03813581169160200135166106d2565b610118610757565b61026f6004803603604081101561031f57600080fd5b506001600160a01b0381351690602001356107b8565b6101b96004803603604081101561034b57600080fd5b506001600160a01b03813516906020013561085f565b6101b96004803603604081101561037757600080fd5b506001600160a01b0381351690602001356108c7565b6101d5600480360360408110156103a357600080fd5b506001600160a01b03813581169160200135166108db565b60038054604080516020601f60026000196101006001881615020190951694909404938401819004810282018101909252828152606093909290918301828280156104475780601f1061041c57610100808354040283529160200191610447565b820191906000526020600020905b81548152906001019060200180831161042a57829003601f168201915b5050505050905090565b600061046561045e610906565b848461090a565b50600192915050565b60025490565b60006104818484846109f6565b6104f18461048d610906565b6104ec85604051806060016040528060288152602001611085602891396001600160a01b038a166000908152600160205260408120906104cb610906565b6001600160a01b031681526020810191909152604001600020549190610b51565b61090a565b5060019392505050565b60055460ff1690565b6000610465610511610906565b846104ec8560016000610522610906565b6001600160a01b03908116825260208083019390935260409182016000908120918c168152925290205490610be8565b6007546001600160a01b0316331461059f576040805162461bcd60e51b815260206004820152600b60248201526a1b9bdd08185b1b1bddd95960aa1b604482015290519081900360640190fd5b6105a7610c49565b565b600554610100900460ff16156105f9576040805162461bcd60e51b815260206004820152601060248201526f14185d5cd8589b194e881c185d5cd95960821b604482015290519081900360640190fd5b6006546001600160a01b03163314610646576040805162461bcd60e51b815260206004820152600b60248201526a1b9bdd08185b1b1bddd95960aa1b604482015290519081900360640190fd5b6106508282610ced565b5050565b600554610100900460ff1690565b6001600160a01b031660009081526020819052604090205490565b6007546001600160a01b031633146106ca576040805162461bcd60e51b815260206004820152600b60248201526a1b9bdd08185b1b1bddd95960aa1b604482015290519081900360640190fd5b6105a7610ddd565b6005546201000090046001600160a01b03163314610726576040805162461bcd60e51b815260206004820152600c60248201526b6f6e6c7920466163746f727960a01b604482015290519081900360640190fd5b600780546001600160a01b039283166001600160a01b03199182161790915560068054939092169216919091179055565b60048054604080516020601f60026000196101006001881615020190951694909404938401819004810282018101909252828152606093909290918301828280156104475780601f1061041c57610100808354040283529160200191610447565b600554610100900460ff1615610808576040805162461bcd60e51b815260206004820152601060248201526f14185d5cd8589b194e881c185d5cd95960821b604482015290519081900360640190fd5b6006546001600160a01b03163314610855576040805162461bcd60e51b815260206004820152600b60248201526a1b9bdd08185b1b1bddd95960aa1b604482015290519081900360640190fd5b6106508282610e65565b600061046561086c610906565b846104ec856040518060600160405280602581526020016111176025913960016000610896610906565b6001600160a01b03908116825260208083019390935260409182016000908120918d16815292529020549190610b51565b60006104656108d4610906565b84846109f6565b6001600160a01b03918216600090815260016020908152604080832093909416825291909152205490565b3390565b6001600160a01b03831661094f5760405162461bcd60e51b81526004018080602001828103825260248152602001806110f36024913960400191505060405180910390fd5b6001600160a01b0382166109945760405162461bcd60e51b815260040180806020018281038252602281526020018061103d6022913960400191505060405180910390fd5b6001600160a01b03808416600081815260016020908152604080832094871680845294825291829020859055815185815291517f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b9259281900390910190a3505050565b6001600160a01b038316610a3b5760405162461bcd60e51b81526004018080602001828103825260258152602001806110ce6025913960400191505060405180910390fd5b6001600160a01b038216610a805760405162461bcd60e51b8152600401808060200182810382526023815260200180610ff86023913960400191505060405180910390fd5b610a8b838383610f61565b610ac88160405180606001604052806026815260200161105f602691396001600160a01b0386166000908152602081905260409020549190610b51565b6001600160a01b038085166000908152602081905260408082209390935590841681522054610af79082610be8565b6001600160a01b038084166000818152602081815260409182902094909455805185815290519193928716927fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef92918290030190a3505050565b60008184841115610be05760405162461bcd60e51b81526004018080602001828103825283818151815260200191508051906020019080838360005b83811015610ba5578181015183820152602001610b8d565b50505050905090810190601f168015610bd25780820380516001836020036101000a031916815260200191505b509250505060405180910390fd5b505050900390565b600082820183811015610c42576040805162461bcd60e51b815260206004820152601b60248201527f536166654d6174683a206164646974696f6e206f766572666c6f770000000000604482015290519081900360640190fd5b9392505050565b600554610100900460ff16610c9c576040805162461bcd60e51b815260206004820152601460248201527314185d5cd8589b194e881b9bdd081c185d5cd95960621b604482015290519081900360640190fd5b6005805461ff00191690557f5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa610cd0610906565b604080516001600160a01b039092168252519081900360200190a1565b6001600160a01b038216610d48576040805162461bcd60e51b815260206004820152601f60248201527f45524332303a206d696e7420746f20746865207a65726f206164647265737300604482015290519081900360640190fd5b610d5460008383610f61565b600254610d619082610be8565b6002556001600160a01b038216600090815260208190526040902054610d879082610be8565b6001600160a01b0383166000818152602081815260408083209490945583518581529351929391927fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef9281900390910190a35050565b600554610100900460ff1615610e2d576040805162461bcd60e51b815260206004820152601060248201526f14185d5cd8589b194e881c185d5cd95960821b604482015290519081900360640190fd5b6005805461ff0019166101001790557f62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258610cd0610906565b6001600160a01b038216610eaa5760405162461bcd60e51b81526004018080602001828103825260218152602001806110ad6021913960400191505060405180910390fd5b610eb682600083610f61565b610ef38160405180606001604052806022815260200161101b602291396001600160a01b0385166000908152602081905260409020549190610b51565b6001600160a01b038316600090815260208190526040902055600254610f199082610fb5565b6002556040805182815290516000916001600160a01b038516917fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef9181900360200190a35050565b610f6c838383610fb0565b610f74610654565b15610fb05760405162461bcd60e51b815260040180806020018281038252602a81526020018061113c602a913960400191505060405180910390fd5b505050565b6000610c4283836040518060400160405280601e81526020017f536166654d6174683a207375627472616374696f6e206f766572666c6f770000815250610b5156fe45524332303a207472616e7366657220746f20746865207a65726f206164647265737345524332303a206275726e20616d6f756e7420657863656564732062616c616e636545524332303a20617070726f766520746f20746865207a65726f206164647265737345524332303a207472616e7366657220616d6f756e7420657863656564732062616c616e636545524332303a207472616e7366657220616d6f756e74206578636565647320616c6c6f77616e636545524332303a206275726e2066726f6d20746865207a65726f206164647265737345524332303a207472616e736665722066726f6d20746865207a65726f206164647265737345524332303a20617070726f76652066726f6d20746865207a65726f206164647265737345524332303a2064656372656173656420616c6c6f77616e63652062656c6f77207a65726f45524332305061757361626c653a20746f6b656e207472616e73666572207768696c6520706175736564a2646970667358221220e96342bec8f6c2bf72815a39998973b64c3bed57770f402e9a7b7eeda0265d4c64736f6c634300060c0033",
}

func TestAPI_GetBlockTraceByNumberOrHash(t *testing.T) {
	t.Parallel()
	erc20Abi, err := erc20MetaData.GetAbi()
	assert.NoError(t, err)

	var (
		accounts = createAccounts(5)
		genesis  = &core.Genesis{Alloc: core.GenesisAlloc{
			accounts[0].From: {Balance: big.NewInt(params.Ether)},
			accounts[1].From: {Balance: big.NewInt(params.Ether)},
			accounts[2].From: {Balance: big.NewInt(params.Ether)},
			accounts[3].From: {Balance: big.NewInt(params.Ether)},
			accounts[4].From: {Balance: big.NewInt(params.Ether)},
		}}

		txs = make([]*types.Transaction, 0, 6)
	)

	// deploy erc20 contract
	backend := newTestBackend(t, 1, genesis, func(i int, b *core.BlockGen) {
		// deploy erc20 contract
		tx, err := createTx(b, accounts[0], nil, nil, common.FromHex(erc20MetaData.Bin))
		assert.NoError(t, err)
		txs = append(txs, tx)
	})
	parent, err := backend.BlockByNumber(context.Background(), 1)
	assert.NoError(t, err)

	// mint erc20 balance
	blocks, _ := core.GenerateChain(backend.chainConfig, parent, backend.engine, backend.chaindb, 1, func(i int, gen *core.BlockGen) {
		// mint erc20 tokens
		for _, acc := range accounts {
			to := common.BigToAddress(big.NewInt(int64(i)))
			data, err := erc20Abi.Pack("mint", acc.From, big.NewInt(1e18))
			assert.NoError(t, err)
			tx, err := createTx(gen, acc, &to, nil, data)
			assert.NoError(t, err)
			txs = append(txs, tx)
		}
	})
	_, err = backend.chain.InsertChain(blocks)
	assert.NoError(t, err)

	block, err := backend.BlockByNumber(context.Background(), 2)
	assert.NoError(t, err)

	api := NewAPI(backend)
	// get trace
	hash := block.Hash()
	blockTrace, err := api.GetBlockTraceByNumberOrHash(context.Background(), rpc.BlockNumberOrHash{
		BlockHash: &hash,
	}, nil)
	assert.NoError(t, err)

	// check chain status
	checkChainAndProof(t, backend, parent, block, blockTrace)

	// check txs.
	checkTxs(t, block.Transactions(), blockTrace.Transactions)

	txTraces, err := api.TraceBlockByNumber(context.Background(), 2, nil)
	assert.NoError(t, err)

	// check executionResults
	checkStructLogs(t, txTraces, blockTrace.ExecutionResults)

	// check coinbase
	checkCoinbase(t, backend, blockTrace.Coinbase)
}

func verifyProof(t *testing.T, expect [][]byte, actual []hexutil.Bytes) {
	assert.Equal(t, true, actual != nil)
	assert.Equal(t, len(expect), len(actual))
	for i, val := range expect {
		assert.Equal(t, hexutil.Encode(val), actual[i].String())
	}
}

func checkChainAndProof(t *testing.T, b *testBackend, parent *types.Block, block *types.Block, blockTrace *types.BlockTrace) {
	assert.Equal(t, parent.Hash(), block.ParentHash())
	storgeTrace := blockTrace.StorageTrace
	assert.Equal(t, parent.Root().String(), storgeTrace.RootBefore.String())
	assert.Equal(t, block.Root().String(), storgeTrace.RootAfter.String())

	statedb, err := b.chain.StateAt(parent.Root())
	assert.NoError(t, err)

	storageProof := blockTrace.StorageTrace.StorageProofs
	for _, tx := range blockTrace.Transactions {
		for _, addr := range []common.Address{tx.From, *tx.To} {
			// verify proofs
			if data2, ok := storgeTrace.Proofs[addr.String()]; ok {
				data1, err := statedb.GetProof(addr)
				assert.NoError(t, err)
				verifyProof(t, data1, data2)
			}
			// verify storage proofs
			for addr, wrappedProof := range storageProof {
				for keyStr, data2 := range wrappedProof {
					data1, err := statedb.GetStorageProof(common.HexToAddress(addr), common.HexToHash(keyStr))
					if len(data2) > 0 {
						assert.NoError(t, err)
					}
					verifyProof(t, data1, data2)
				}
			}
		}
	}
}

func checkTxs(t *testing.T, expect []*types.Transaction, actual []*types.TransactionData) {
	assert.Equal(t, len(expect), len(actual))
	for i := range expect {
		eTx, aTx := expect[i], actual[i]
		assert.Equal(t, eTx.Hash().String(), aTx.TxHash)
		assert.Equal(t, eTx.Gas(), aTx.Gas)
	}
}

func checkStructLogs(t *testing.T, expect []*txTraceResult, actual []*types.ExecutionResult) {
	// assert executionResults
	assert.Equal(t, len(expect), len(actual))
	for i, val := range expect {
		trace1, trace2 := val.Result.(*types.ExecutionResult), actual[i]
		assert.Equal(t, trace1.L1DataFee, trace2.L1DataFee)
		assert.Equal(t, trace1.Failed, trace2.Failed)
		assert.Equal(t, trace1.Gas, trace2.Gas)
		assert.Equal(t, trace1.ReturnValue, trace2.ReturnValue)
		assert.Equal(t, len(trace1.StructLogs), len(trace2.StructLogs))
		// TODO: compare other fields

		for i, opcode := range trace1.StructLogs {
			data1, err := json.Marshal(opcode)
			assert.NoError(t, err)
			data2, err := json.Marshal(trace2.StructLogs[i])
			assert.NoError(t, err)
			assert.Equal(t, string(data1), string(data2))
		}
	}
}

func checkCoinbase(t *testing.T, b *testBackend, wrapper *types.AccountWrapper) {
	var coinbase common.Address
	if b.chainConfig.Scroll.FeeVaultEnabled() {
		coinbase = *b.chainConfig.Scroll.FeeVaultAddress
	} else {
		header, err := b.HeaderByNumber(context.Background(), 1)
		assert.NoError(t, err)
		coinbase, err = b.engine.Author(header)
		assert.NoError(t, err)
	}
	assert.Equal(t, true, coinbase.String() == wrapper.Address.String())
}

func createAccounts(n int) (auths []*bind.TransactOpts) {
	accounts := newAccounts(5)
	for _, acc := range accounts {
		auth, _ := bind.NewKeyedTransactorWithChainID(acc.key, big.NewInt(1))
		auth.Nonce = big.NewInt(0)
		auths = append(auths, auth)
	}
	return
}

func createTx(b *core.BlockGen, auth *bind.TransactOpts, to *common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	nonce := auth.Nonce.Uint64()
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       to,
		Value:    value,
		Gas:      200000,
		GasPrice: b.BaseFee(),
		Data:     data,
	})
	signedTx, err := auth.Signer(auth.From, tx)
	if err == nil {
		auth.Nonce = big.NewInt(int64(nonce + 1))
		b.AddTx(signedTx)
	}
	return signedTx, err
}
