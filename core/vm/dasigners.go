package vm

import (
	"bytes"
	"math/big"
	"sort"
	"strings"

	"github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/bn254util"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm/precompiles"
	"github.com/ethereum/go-ethereum/core/vm/precompiles/dasigners"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/vmihailenco/msgpack/v5"
)

const (
	DASignersRequiredGasMax uint64 = 1000_000_000

	DASignersFunctionParams            = "params"
	DASignersFunctionEpochNumber       = "epochNumber"
	DASignersFunctionQuorumCount       = "quorumCount"
	DASignersFunctionGetSigner         = "getSigner"
	DASignersFunctionGetQuorum         = "getQuorum"
	DASignersFunctionGetQuorumRow      = "getQuorumRow"
	DASignersFunctionRegisterSigner    = "registerSigner"
	DASignersFunctionUpdateSocket      = "updateSocket"
	DASignersFunctionRegisterNextEpoch = "registerNextEpoch"
	DASignersFunctionGetAggPkG1        = "getAggPkG1"
	DASignersFunctionIsSigner          = "isSigner"
	DASignersFunctionRegisteredEpoch   = "registeredEpoch"
	DASignersFunctionMakeEpoch         = "makeEpoch"
)

var DASignersRequiredGasBasic = map[string]uint64{
	DASignersFunctionParams:            1_000,
	DASignersFunctionEpochNumber:       1_000,
	DASignersFunctionQuorumCount:       1_000,
	DASignersFunctionGetSigner:         100_000,
	DASignersFunctionGetQuorum:         100_000,
	DASignersFunctionGetQuorumRow:      10_000,
	DASignersFunctionRegisterSigner:    100_000,
	DASignersFunctionUpdateSocket:      50_000,
	DASignersFunctionRegisterNextEpoch: 100_000,
	DASignersFunctionGetAggPkG1:        1_000_000,
	DASignersFunctionIsSigner:          10_000,
	DASignersFunctionRegisteredEpoch:   10_000,
	DASignersFunctionMakeEpoch:         100_000,
}

const (
	DASignersNewSignerEvent     = "NewSigner"
	DASignersSocketUpdatedEvent = "SocketUpdated"
)

var _ StatefulPrecompiledContract = &DASignersPrecompile{}

type DASignersPrecompile struct {
	abi abi.ABI
}

func NewDASignersPrecompile() *DASignersPrecompile {
	abi, err := abi.JSON(strings.NewReader(dasigners.DASignersABI))
	if err != nil {
		panic(err)
	}
	return &DASignersPrecompile{
		abi: abi,
	}
}

// Address implements vm.PrecompiledContract.
func (d *DASignersPrecompile) Address() common.Address {
	return common.HexToAddress("0x0000000000000000000000000000000000001000")
}

// RequiredGas implements vm.PrecompiledContract.
func (d *DASignersPrecompile) RequiredGas(input []byte) uint64 {
	method, err := d.abi.MethodById(input[:4])
	if err != nil {
		return DASignersRequiredGasMax
	}
	if gas, ok := DASignersRequiredGasBasic[method.Name]; ok {
		return gas
	}
	return DASignersRequiredGasMax
}

func (d *DASignersPrecompile) IsTx(method string) bool {
	switch method {
	case DASignersFunctionUpdateSocket,
		DASignersFunctionRegisterSigner,
		DASignersFunctionRegisterNextEpoch:
		return true
	default:
		return false
	}
}

func (d *DASignersPrecompile) Abi() *abi.ABI {
	return &d.abi
}

// Run implements vm.PrecompiledContract.
func (d *DASignersPrecompile) Run(evm *EVM, contract *Contract, readonly bool) ([]byte, error) {
	method, args, err := InitializeStatefulPrecompileCall(d, evm, contract, readonly)
	if err != nil {
		return nil, err
	}

	var bz []byte
	switch method.Name {
	// queries
	case DASignersFunctionParams:
		bz, err = d.Params(evm, method, args)
	case DASignersFunctionEpochNumber:
		bz, err = d.EpochNumber(evm, method, args)
	case DASignersFunctionQuorumCount:
		bz, err = d.QuorumCount(evm, method, args)
	case DASignersFunctionGetSigner:
		bz, err = d.GetSigner(evm, method, args)
	case DASignersFunctionGetQuorum:
		bz, err = d.GetQuorum(evm, method, args)
	case DASignersFunctionGetQuorumRow:
		bz, err = d.GetQuorumRow(evm, method, args)
	case DASignersFunctionGetAggPkG1:
		bz, err = d.GetAggPkG1(evm, method, args)
	case DASignersFunctionIsSigner:
		bz, err = d.IsSigner(evm, method, args)
	case DASignersFunctionRegisteredEpoch:
		bz, err = d.RegisteredEpoch(evm, method, args)
	// txs
	case DASignersFunctionRegisterSigner:
		bz, err = d.RegisterSigner(evm, contract, method, args)
	case DASignersFunctionRegisterNextEpoch:
		bz, err = d.RegisterNextEpoch(evm, contract, method, args)
	case DASignersFunctionUpdateSocket:
		bz, err = d.UpdateSocket(evm, contract, method, args)
	case DASignersFunctionMakeEpoch:
		bz, err = d.MakeEpoch(evm, contract, method, args)
	}

	if err != nil {
		return nil, err
	}

	return bz, nil
}

func (d *DASignersPrecompile) EmitNewSignerEvent(evm *EVM, signer dasigners.IDASignersSignerDetail) error {
	event := d.abi.Events[DASignersNewSignerEvent]
	quries := make([]interface{}, 2)
	quries[0] = event.ID
	quries[1] = signer.Signer
	topics, err := abi.MakeTopics(quries)
	if err != nil {
		return err
	}
	arguments := abi.Arguments{event.Inputs[1], event.Inputs[2]}
	b, err := arguments.Pack(signer.PkG1, signer.PkG2)
	if err != nil {
		return err
	}
	evm.StateDB.AddLog(&types.Log{
		Address:     d.Address(),
		Topics:      topics[0],
		Data:        b,
		BlockNumber: evm.Context.BlockNumber.Uint64(),
	})
	return d.EmitSocketUpdatedEvent(evm, signer.Signer, signer.Socket)
}

func (d *DASignersPrecompile) EmitSocketUpdatedEvent(evm *EVM, signer common.Address, socket string) error {
	event := d.abi.Events[DASignersSocketUpdatedEvent]
	quries := make([]interface{}, 2)
	quries[0] = event.ID
	quries[1] = signer
	topics, err := abi.MakeTopics(quries)
	if err != nil {
		return err
	}
	arguments := abi.Arguments{event.Inputs[1]}
	b, err := arguments.Pack(socket)
	if err != nil {
		return err
	}
	evm.StateDB.AddLog(&types.Log{
		Address:     d.Address(),
		Topics:      topics[0],
		Data:        b,
		BlockNumber: evm.Context.BlockNumber.Uint64(),
	})
	return nil
}

type Ballot struct {
	account common.Address
	content []byte
}

func (d *DASignersPrecompile) MakeEpoch(
	evm *EVM,
	contract *Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != 0 {
		return nil, ErrExecutionReverted
	}
	params := d.params()
	epoch := d.epochNumber(evm)
	epochBlock := d.epochBlock(evm, epoch)
	blockHeight := evm.Context.BlockNumber.Uint64()
	if epochBlock > 0 && blockHeight < epochBlock+params.EpochBlocks.Uint64() {
		// not yet to the next epoch
		return method.Outputs.Pack()
	}
	// new epoch
	epoch += 1
	cnt := d.epochRegistration(evm, epoch)
	ballots := []Ballot{}
	for index := range cnt {
		account := d.epochRegisteredSigner(evm, epoch, index)
		sigHash, _ := d.getRegistration(evm, epoch, account)
		votes := d.getVotes(evm, epoch, account)
		// MaxVotesPerSigner is hard limit
		if params.MaxVotesPerSigner.Int64() < int64(votes) {
			votes = int(params.MaxVotesPerSigner.Int64())
		}
		content := sigHash
		for j := 0; j < votes; j += 1 {
			ballots = append(ballots, Ballot{
				account: account,
				content: content,
			})
			content = crypto.Keccak256(content)
		}
	}
	sort.Slice(ballots, func(i, j int) bool {
		return bytes.Compare(ballots[i].content, ballots[j].content) < 0
	})

	quorums := make([][]common.Address, 0)
	encodedSlices := params.EncodedSlices.Uint64()
	maxQuorums := params.MaxQuorums.Uint64()
	if len(ballots) >= int(encodedSlices) {
		for i := 0; i+int(encodedSlices) <= len(ballots); i += int(encodedSlices) {
			if int(maxQuorums) <= len(quorums) {
				break
			}
			quorum := make([]common.Address, encodedSlices)
			for j := 0; j < int(encodedSlices); j += 1 {
				quorum[j] = ballots[i+j].account
			}
			quorums = append(quorums, quorum)
		}
		if len(ballots)%int(encodedSlices) != 0 && int(maxQuorums) > len(quorums) {
			quorum := make([]common.Address, 0)
			for j := len(ballots) - int(encodedSlices); j < len(ballots); j += 1 {
				quorum = append(quorum, ballots[j].account)
			}
			quorums = append(quorums, quorum)
		}
	} else if len(ballots) > 0 {
		quorum := make([]common.Address, encodedSlices)
		n := len(ballots)
		for i := 0; i < int(encodedSlices); i += 1 {
			quorum[i] = ballots[i%n].account
		}
		quorums = append(quorums, quorum)
	}

	// save quorums
	for index, quorum := range quorums {
		b, err := msgpack.Marshal(quorum)
		if err != nil {
			return nil, err
		}
		StoreBytes(evm.StateDB, d.Address(), dasigners.QuorumKey(epoch, uint64(index)), b)
	}
	evm.StateDB.SetState(d.Address(), dasigners.QuorumCountKey(epoch), common.BigToHash(big.NewInt(int64(len(quorums)))))
	// save epoch number & block height
	evm.StateDB.SetState(d.Address(), dasigners.EpochNumberKey(), common.BigToHash(big.NewInt(int64(epoch))))
	evm.StateDB.SetState(d.Address(), dasigners.EpochBlockKey(epoch), common.BigToHash(big.NewInt(int64(blockHeight))))
	return method.Outputs.Pack()
}

func (d *DASignersPrecompile) setSigner(evm *EVM, signer dasigners.IDASignersSignerDetail) error {
	b, err := msgpack.Marshal(signer)
	if err != nil {
		return err
	}
	StoreBytes(evm.StateDB, d.Address(), dasigners.SignerKey(signer.Signer), b)
	return nil
}

func (d *DASignersPrecompile) getRegistry() common.Address {
	// This is a upgradeable contract deployed in Beacon-Proxy pattern in three raw transaction:
	// raw tx params:
	//	 from: 0xeb995d37799ad4a2db524e5ff0825ae2d4711757
	//   nonce: 0..2
	//	 gasPrice: 100 Gwei
	//   gasLimit: 1000000
	// The sender is an ephemeral account, nobody holds its private key and this is the only transaction it signed.
	// This transaction is a legacy transaction without chain ID so it can be deployed at any EVM chain which supports pre-EIP155 transactions.
	// raw tx #0(implementation): 0xf90b708085174876e800830f42408080b90b1d608060405234801561001057600080fd5b50610afd806100206000396000f3fe608060405234801561001057600080fd5b506004361061007d5760003560e01c8063807f063a1161005b578063807f063a146100b25780638129fc1c146100d75780638da5cb5b146100df578063f2fde38b1461010f57600080fd5b806356a3237214610082578063715018a6146100975780637ca4dd5e1461009f575b600080fd5b6100956100903660046106e6565b610122565b005b610095610242565b6100956100ad3660046107b5565b610256565b6100bb61100081565b6040516001600160a01b03909116815260200160405180910390f35b610095610395565b7f9016d09d72d40fdae2fd8ceac6b6234c7706214fd39c1cd1e609a0528c199300546001600160a01b03166100bb565b61009561011d3660046108bd565b6104b7565b33321461014257604051630f15d65160e01b815260040160405180910390fd5b6000806110006001600160a01b0316630f62bda560e01b3385600160405160240161016f939291906108d8565b60408051601f198184030181529181526020820180516001600160e01b03166001600160e01b03199094169390931790925290516101ad9190610934565b6000604051808303816000865af19150503d80600081146101ea576040519150601f19603f3d011682016040523d82523d6000602084013e6101ef565b606091505b509150915081816040516020016102069190610950565b6040516020818303038152906040529061023c5760405162461bcd60e51b815260040161023391906109c1565b60405180910390fd5b50505050565b61024a6104f5565b6102546000610550565b565b33321461027657604051630f15d65160e01b815260040160405180910390fd5b81516001600160a01b031633146102a057604051631024390d60e21b815260040160405180910390fd5b6000806110006001600160a01b0316637ca4dd5e60e01b85856040516024016102ca9291906109f7565b60408051601f198184030181529181526020820180516001600160e01b03166001600160e01b03199094169390931790925290516103089190610934565b6000604051808303816000865af19150503d8060008114610345576040519150601f19603f3d011682016040523d82523d6000602084013e61034a565b606091505b509150915081816040516020016103619190610a82565b6040516020818303038152906040529061038e5760405162461bcd60e51b815260040161023391906109c1565b5050505050565b7ff0c57e16840df040f15088dc2f81fe391c3923bec73e23a9662efc9c229c6a008054600160401b810460ff16159067ffffffffffffffff166000811580156103db5750825b905060008267ffffffffffffffff1660011480156103f85750303b155b905081158015610406575080155b156104245760405163f92ee8a960e01b815260040160405180910390fd5b845467ffffffffffffffff19166001178555831561044e57845460ff60401b1916600160401b1785555b61046b732d7f2d2286994477ba878f321b17a7e40e52cda46105c1565b831561038e57845460ff60401b19168555604051600181527fc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d29060200160405180910390a15050505050565b6104bf6104f5565b6001600160a01b0381166104e957604051631e4fbdf760e01b815260006004820152602401610233565b6104f281610550565b50565b336105277f9016d09d72d40fdae2fd8ceac6b6234c7706214fd39c1cd1e609a0528c199300546001600160a01b031690565b6001600160a01b0316146102545760405163118cdaa760e01b8152336004820152602401610233565b7f9016d09d72d40fdae2fd8ceac6b6234c7706214fd39c1cd1e609a0528c19930080546001600160a01b031981166001600160a01b03848116918217845560405192169182907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a3505050565b6105c96105d2565b6104f28161061b565b7ff0c57e16840df040f15088dc2f81fe391c3923bec73e23a9662efc9c229c6a0054600160401b900460ff1661025457604051631afcd79f60e31b815260040160405180910390fd5b6104bf6105d2565b634e487b7160e01b600052604160045260246000fd5b6040805190810167ffffffffffffffff8111828210171561065c5761065c610623565b60405290565b6040516080810167ffffffffffffffff8111828210171561065c5761065c610623565b604051601f8201601f1916810167ffffffffffffffff811182821017156106ae576106ae610623565b604052919050565b6000604082840312156106c857600080fd5b6106d0610639565b9050813581526020820135602082015292915050565b6000604082840312156106f857600080fd5b61070283836106b6565b9392505050565b80356001600160a01b038116811461072057600080fd5b919050565b600082601f83011261073657600080fd5b61073e610639565b80604084018581111561075057600080fd5b845b8181101561076a578035845260209384019301610752565b509095945050505050565b60006080828403121561078757600080fd5b61078f610639565b905061079b8383610725565b81526107aa8360408401610725565b602082015292915050565b600080606083850312156107c857600080fd5b823567ffffffffffffffff808211156107e057600080fd5b9084019061010082870312156107f557600080fd5b6107fd610662565b61080683610709565b81526020808401358381111561081b57600080fd5b8401601f8101891361082c57600080fd5b80358481111561083e5761083e610623565b610850601f8201601f19168401610685565b9450808552898382840101111561086657600080fd5b808383018487013760008382870101525050828183015261088a88604086016106b6565b604083015261089c8860808601610775565b60608301528195506108b0888289016106b6565b9450505050509250929050565b6000602082840312156108cf57600080fd5b61070282610709565b6001600160a01b0384168152608081016108ff602083018580518252602090810151910152565b60ff83166060830152949350505050565b60005b8381101561092b578181015183820152602001610913565b50506000910152565b60008251610946818460208701610910565b9190910192915050565b7f72656769737465724e65787445706f63682063616c6c206661696c65643a200081526000825161098881601f850160208701610910565b91909101601f0192915050565b600081518084526109ad816020860160208601610910565b601f01601f19169290920160200192915050565b6020815260006107026020830184610995565b8060005b600281101561023c5781518452602093840193909101906001016109d8565b606080825283516001600160a01b03169082015260208301516101006080830152600090610a29610160840182610995565b6040860151805160a0860152602081015160c0860152909150506060850151610a5660e0850182516109d4565b60200151610a686101208501826109d4565b509050610702602083018480518252602090810151910152565b7f72656769737465725369676e65722063616c6c206661696c65643a2000000000815260008251610aba81601c850160208701610910565b91909101601c019291505056fea2646970667358221220bb50c03fb5e1cc5e8bb317ff0bf7889ef0b368a37ea3568a9d02d8379ebc351164736f6c634300081400331ba04ce6e58373b6023b7ffc717cc0db01847ed8da661c7c3c98693957d3248af35fa0727edc90561444dcb4e91a329e8ef7916cb018dfef5f18e86e2bd90391e7a204
	// raw tx #1(beacon): 0xf904cb0185174876e800830f42408080b90478608060405234801561001057600080fd5b5060405161043838038061043883398101604081905261002f91610165565b806001600160a01b03811661005f57604051631e4fbdf760e01b8152600060048201526024015b60405180910390fd5b61006881610079565b50610072826100c9565b5050610198565b600080546001600160a01b038381166001600160a01b0319831681178455604051919092169283917f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e09190a35050565b806001600160a01b03163b6000036100ff5760405163211eb15960e21b81526001600160a01b0382166004820152602401610056565b600180546001600160a01b0319166001600160a01b0383169081179091556040517fbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b90600090a250565b80516001600160a01b038116811461016057600080fd5b919050565b6000806040838503121561017857600080fd5b61018183610149565b915061018f60208401610149565b90509250929050565b610291806101a76000396000f3fe608060405234801561001057600080fd5b50600436106100575760003560e01c80633659cfe61461005c5780635c60da1b14610071578063715018a61461009a5780638da5cb5b146100a2578063f2fde38b146100b3575b600080fd5b61006f61006a36600461022b565b6100c6565b005b6001546001600160a01b03165b6040516001600160a01b03909116815260200160405180910390f35b61006f6100da565b6000546001600160a01b031661007e565b61006f6100c136600461022b565b6100ee565b6100ce61012e565b6100d78161015b565b50565b6100e261012e565b6100ec60006101db565b565b6100f661012e565b6001600160a01b03811661012557604051631e4fbdf760e01b8152600060048201526024015b60405180910390fd5b6100d7816101db565b6000546001600160a01b031633146100ec5760405163118cdaa760e01b815233600482015260240161011c565b806001600160a01b03163b6000036101915760405163211eb15960e21b81526001600160a01b038216600482015260240161011c565b600180546001600160a01b0319166001600160a01b0383169081179091556040517fbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b90600090a250565b600080546001600160a01b038381166001600160a01b0319831681178455604051919092169283917f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e09190a35050565b60006020828403121561023d57600080fd5b81356001600160a01b038116811461025457600080fd5b939250505056fea26469706673582212205220e5b3095ab739313888ed7a605b359ca52e79f2a5a6297e03c439e8e8b30764736f6c634300081400330000000000000000000000007ad29425f6d68ed6bd8eb8a77d73bb2ad81b8afa0000000000000000000000002d7f2d2286994477ba878f321b17a7e40e52cda41ca0a1e38aac4e65cf5d87e9c2f857d3f0b3cc9f24d42639689f01c9876a45665ebda0330c349d845c3403c9be2b8e31ed0c17e5b3824d02c9e11e71564df71d6621ee
	// raw tx #2(proxy): 0xf906720285174876e800830f42408080b9061f60a06040526040516105bf3803806105bf83398101604081905261002291610387565b61002c828261003e565b506001600160a01b031660805261047e565b610047826100fe565b6040516001600160a01b038316907f1cf3b03a6cf19fa2baba4df148e9dcabedea7f8a5c07840e207e5c089be95d3e90600090a28051156100f2576100ed826001600160a01b0316635c60da1b6040518163ffffffff1660e01b8152600401602060405180830381865afa1580156100c3573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906100e79190610447565b82610211565b505050565b6100fa610288565b5050565b806001600160a01b03163b60000361013957604051631933b43b60e21b81526001600160a01b03821660048201526024015b60405180910390fd5b807fa3f0ad74e5423aebfd80d3ef4346578335a9a72aeaee59ff6cb3582b35133d5080546001600160a01b0319166001600160a01b0392831617905560408051635c60da1b60e01b81529051600092841691635c60da1b9160048083019260209291908290030181865afa1580156101b5573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906101d99190610447565b9050806001600160a01b03163b6000036100fa57604051634c9c8ce360e01b81526001600160a01b0382166004820152602401610130565b6060600080846001600160a01b03168460405161022e9190610462565b600060405180830381855af49150503d8060008114610269576040519150601f19603f3d011682016040523d82523d6000602084013e61026e565b606091505b50909250905061027f8583836102a9565b95945050505050565b34156102a75760405163b398979f60e01b815260040160405180910390fd5b565b6060826102be576102b982610308565b610301565b81511580156102d557506001600160a01b0384163b155b156102fe57604051639996b31560e01b81526001600160a01b0385166004820152602401610130565b50805b9392505050565b8051156103185780518082602001fd5b604051630a12f52160e11b815260040160405180910390fd5b80516001600160a01b038116811461034857600080fd5b919050565b634e487b7160e01b600052604160045260246000fd5b60005b8381101561037e578181015183820152602001610366565b50506000910152565b6000806040838503121561039a57600080fd5b6103a383610331565b60208401519092506001600160401b03808211156103c057600080fd5b818501915085601f8301126103d457600080fd5b8151818111156103e6576103e661034d565b604051601f8201601f19908116603f0116810190838211818310171561040e5761040e61034d565b8160405282815288602084870101111561042757600080fd5b610438836020830160208801610363565b80955050505050509250929050565b60006020828403121561045957600080fd5b61030182610331565b60008251610474818460208701610363565b9190910192915050565b6080516101276104986000396000601e01526101276000f3fe6080604052600a600c565b005b60186014601a565b60a0565b565b60007f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316635c60da1b6040518163ffffffff1660e01b8152600401602060405180830381865afa1580156079573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190609b919060c3565b905090565b3660008037600080366000845af43d6000803e80801560be573d6000f35b3d6000fd5b60006020828403121560d457600080fd5b81516001600160a01b038116811460ea57600080fd5b939250505056fea264697066735822122039e43d51fa1bcd8fe79599db2a7e6dd3e5358b756c53210827bbf02fda62be6c64736f6c63430008140033000000000000000000000000762662fb644cdd051f35e0dd8fb6ac15a4bf65ad000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000001ca04189ff3e2929af3203bb5041d582084f676c866a1faae88b0a1fad3f616b90ada0309a30d2c3fabc8e659b9a747c75f62556f1db3b269b1377fa936b7c236685cb
	// The owners of the contract and the beacon will be set to 0x2D7F2d2286994477Ba878f321b17A7e40E52cDa4,
	// and after the network has launched and reached a stable state, ownership will be transferred to a timelock contract controlled by a multisig
	return common.HexToAddress("0x20f30b2584f3096ea0d6c18c3b5cacc0585e12fc")
}

func (d *DASignersPrecompile) RegisterSigner(
	evm *EVM,
	contract *Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != 2 {
		return nil, ErrExecutionReverted
	}
	signer := args[0].(dasigners.IDASignersSignerDetail)
	signature := dasigners.SerializeG1(args[1].(dasigners.BN254G1Point))
	// validation
	if contract.caller != d.getRegistry() {
		return nil, precompiles.ErrSenderNotRegistry
	}
	// execute
	// validate sender
	// staked value is checked in registry contract
	_, found, err := d.getSigner(evm, signer.Signer)
	if err != nil {
		return nil, err
	}
	if found {
		return nil, dasigners.ErrSignerExists
	}
	// validate signature
	chainID := evm.chainConfig.ChainID
	hash := dasigners.PubkeyRegistrationHash(signer.Signer, chainID)
	if !dasigners.ValidateSignature(signer, hash, bn254util.DeserializeG1(signature)) {
		return nil, dasigners.ErrInvalidSignature
	}
	// save signer
	if err := d.setSigner(evm, signer); err != nil {
		return nil, err
	}
	// emit events
	err = d.EmitNewSignerEvent(evm, signer)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack()
}

func (d *DASignersPrecompile) epochRegistration(evm *EVM, epoch uint64) uint64 {
	return evm.StateDB.GetState(d.Address(), dasigners.EpochRegistrationKey(epoch)).Big().Uint64()
}

func (d *DASignersPrecompile) epochRegisteredSigner(evm *EVM, epoch uint64, index uint64) common.Address {
	h := evm.StateDB.GetState(d.Address(), dasigners.EpochRegisteredSignerKey(epoch, index))
	return common.Address(h[12:])
}

func (d *DASignersPrecompile) storeRegistration(evm *EVM, epoch uint64, signer common.Address, signature []byte, votes *big.Int) error {
	if _, found := d.getRegistration(evm, epoch, signer); found {
		return nil
	}
	// save signature hash
	evm.StateDB.SetState(d.Address(), dasigners.RegistrationKey(epoch, signer), crypto.Keccak256Hash(signature))
	// save votes
	evm.StateDB.SetState(d.Address(), dasigners.VotesKey(epoch, signer), common.BigToHash(votes))
	// increment epoch registration count
	registration := d.epochRegistration(evm, epoch)
	evm.StateDB.SetState(d.Address(), dasigners.EpochRegistrationKey(epoch), common.BigToHash(big.NewInt(int64(registration+1))))
	// save registered signer address
	evm.StateDB.SetState(d.Address(), dasigners.EpochRegisteredSignerKey(epoch, registration), common.BytesToHash(signer.Bytes()))
	return nil
}

func (d *DASignersPrecompile) RegisterNextEpoch(
	evm *EVM,
	contract *Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != 3 {
		return nil, ErrExecutionReverted
	}
	account := args[0].(common.Address)
	signature := dasigners.SerializeG1(args[1].(dasigners.BN254G1Point))
	votes := args[2].(*big.Int)
	// validation
	if contract.caller != d.getRegistry() {
		return nil, precompiles.ErrSenderNotRegistry
	}
	// execute
	// get signer
	// staked value is checked in registry contract
	signer, found, err := d.getSigner(evm, account)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, dasigners.ErrSignerNotFound
	}
	// validate signature
	epochNumber := d.epochNumber(evm)
	chainID := evm.chainConfig.ChainID
	hash := dasigners.EpochRegistrationHash(account, epochNumber+1, chainID)
	if !dasigners.ValidateSignature(signer, hash, bn254util.DeserializeG1(signature)) {
		return nil, dasigners.ErrInvalidSignature
	}
	// save registration
	if err := d.storeRegistration(evm, epochNumber+1, account, signature, votes); err != nil {
		return nil, err
	}
	return method.Outputs.Pack()
}

func (d *DASignersPrecompile) UpdateSocket(
	evm *EVM,
	contract *Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != 1 {
		return nil, ErrExecutionReverted
	}
	socket := args[0].(string)
	// validation
	if contract.caller != evm.Origin {
		return nil, precompiles.ErrSenderNotOrigin
	}
	// execute
	signer, found, err := d.getSigner(evm, contract.caller)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, dasigners.ErrSignerNotFound
	}
	signer.Socket = socket
	if err := d.setSigner(evm, signer); err != nil {
		return nil, err
	}
	// emit events
	err = d.EmitSocketUpdatedEvent(evm, contract.caller, socket)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack()
}

func (d *DASignersPrecompile) params() dasigners.IDASignersParams {
	return dasigners.IDASignersParams{
		TokensPerVote:     big.NewInt(10), // deprecated
		MaxVotesPerSigner: big.NewInt(1024),
		MaxQuorums:        big.NewInt(10),
		EpochBlocks:       big.NewInt(5760),
		EncodedSlices:     big.NewInt(3072),
	}
}

func (d *DASignersPrecompile) Params(evm *EVM, method *abi.Method, _ []interface{}) ([]byte, error) {
	return method.Outputs.Pack(d.params())
}

func (d *DASignersPrecompile) epochBlock(evm *EVM, epoch uint64) uint64 {
	return evm.StateDB.GetState(d.Address(), dasigners.EpochBlockKey(epoch)).Big().Uint64()
}

func (d *DASignersPrecompile) epochNumber(evm *EVM) uint64 {
	return evm.StateDB.GetState(d.Address(), dasigners.EpochNumberKey()).Big().Uint64()
}

func (d *DASignersPrecompile) EpochNumber(evm *EVM, method *abi.Method, args []interface{}) ([]byte, error) {
	if len(args) != 0 {
		return nil, ErrExecutionReverted
	}
	return method.Outputs.Pack(big.NewInt(int64(d.epochNumber(evm))))
}

func (d *DASignersPrecompile) quorumCount(evm *EVM, epochNumber uint64) uint64 {
	return evm.StateDB.GetState(d.Address(), dasigners.QuorumCountKey(epochNumber)).Big().Uint64()
}

func (d *DASignersPrecompile) QuorumCount(evm *EVM, method *abi.Method, args []interface{}) ([]byte, error) {
	if len(args) != 1 {
		return nil, ErrExecutionReverted
	}
	epochNumber := args[0].(*big.Int).Uint64()
	return method.Outputs.Pack(big.NewInt(int64(d.quorumCount(evm, epochNumber))))
}

func (d *DASignersPrecompile) getSigner(evm *EVM, account common.Address) (dasigners.IDASignersSignerDetail, bool, error) {
	b := LoadBytes(evm.StateDB, d.Address(), dasigners.SignerKey(account))
	if len(b) == 0 {
		return dasigners.IDASignersSignerDetail{}, false, nil
	}

	var signer dasigners.IDASignersSignerDetail
	err := msgpack.Unmarshal(b, &signer)
	if err != nil {
		return dasigners.IDASignersSignerDetail{}, false, err
	}
	return signer, true, nil
}

func (d *DASignersPrecompile) GetSigner(evm *EVM, method *abi.Method, args []interface{}) ([]byte, error) {
	if len(args) != 1 {
		return nil, ErrExecutionReverted
	}
	accounts := args[0].([]common.Address)
	signers := make([]dasigners.IDASignersSignerDetail, len(accounts))
	for i, account := range accounts {
		signer, found, err := d.getSigner(evm, account)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, dasigners.ErrSignerNotFound
		}
		signers[i] = signer
	}
	return method.Outputs.Pack(signers)
}

func (d *DASignersPrecompile) IsSigner(evm *EVM, method *abi.Method, args []interface{}) ([]byte, error) {
	if len(args) != 1 {
		return nil, ErrExecutionReverted
	}
	account := args[0].(common.Address)
	_, found, err := d.getSigner(evm, account)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(found)
}

func (d *DASignersPrecompile) getRegistration(evm *EVM, epoch uint64, account common.Address) ([]byte, bool) {
	h := evm.StateDB.GetState(d.Address(), dasigners.RegistrationKey(epoch, account))
	if h == (common.Hash{}) {
		return nil, false
	}
	return h.Bytes(), true
}

func (d *DASignersPrecompile) getVotes(evm *EVM, epoch uint64, account common.Address) int {
	return int(evm.StateDB.GetState(d.Address(), dasigners.VotesKey(epoch, account)).Big().Int64())
}

func (d *DASignersPrecompile) RegisteredEpoch(evm *EVM, method *abi.Method, args []interface{}) ([]byte, error) {
	if len(args) != 2 {
		return nil, ErrExecutionReverted
	}
	account := args[0].(common.Address)
	epoch := args[1].(*big.Int).Uint64()
	_, found := d.getRegistration(evm, epoch, account)
	return method.Outputs.Pack(found)
}

func (d *DASignersPrecompile) getQuorum(evm *EVM, epochNumber uint64, quorumId uint64) ([]common.Address, error) {
	if d.quorumCount(evm, epochNumber) <= quorumId {
		return nil, dasigners.ErrQuorumIdOutOfBound
	}
	if d.epochNumber(evm) < epochNumber {
		return nil, dasigners.ErrEpochOutOfBound
	}
	b := LoadBytes(evm.StateDB, d.Address(), dasigners.QuorumKey(epochNumber, quorumId))
	var quorum []common.Address
	err := msgpack.Unmarshal(b, &quorum)
	if err != nil {
		return nil, err
	}
	return quorum, nil
}

func (d *DASignersPrecompile) GetQuorum(evm *EVM, method *abi.Method, args []interface{}) ([]byte, error) {
	if len(args) != 2 {
		return nil, ErrExecutionReverted
	}
	epochNumber := args[0].(*big.Int).Uint64()
	quorumId := args[1].(*big.Int).Uint64()
	quorum, err := d.getQuorum(evm, epochNumber, quorumId)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(quorum)
}

func (d *DASignersPrecompile) GetQuorumRow(evm *EVM, method *abi.Method, args []interface{}) ([]byte, error) {
	if len(args) != 3 {
		return nil, ErrExecutionReverted
	}
	epochNumber := args[0].(*big.Int).Uint64()
	quorumId := args[1].(*big.Int).Uint64()
	rowIndex := args[2].(uint32)
	quorum, err := d.getQuorum(evm, epochNumber, quorumId)
	if err != nil {
		return nil, err
	}
	if int(rowIndex) >= len(quorum) {
		return nil, dasigners.ErrRowIdOfBound
	}
	return method.Outputs.Pack(quorum[rowIndex])
}

func (d *DASignersPrecompile) getAggPkG1(
	evm *EVM,
	epochNumber uint64,
	quorumId uint64,
	quorumBitmap []byte,
) (dasigners.BN254G1Point, *big.Int, *big.Int, error) {
	quorum, err := d.getQuorum(evm, epochNumber, quorumId)
	if err != nil {
		return dasigners.BN254G1Point{}, nil, nil, err
	}
	if (len(quorum)+7)/8 != len(quorumBitmap) {
		return dasigners.BN254G1Point{}, nil, nil, dasigners.ErrQuorumBitmapLengthMismatch
	}
	aggPubkeyG1 := new(bn254.G1Affine)
	hit := 0
	added := make(map[common.Address]struct{})
	for i, signer := range quorum {
		if _, ok := added[signer]; ok {
			hit += 1
			continue
		}
		b := quorumBitmap[i/8] & (1 << (i % 8))
		if b == 0 {
			continue
		}
		hit += 1
		added[signer] = struct{}{}
		signer, found, err := d.getSigner(evm, signer)
		if err != nil {
			return dasigners.BN254G1Point{}, nil, nil, err
		}
		if !found {
			return dasigners.BN254G1Point{}, nil, nil, dasigners.ErrSignerNotFound
		}
		aggPubkeyG1.Add(aggPubkeyG1, bn254util.DeserializeG1(dasigners.SerializeG1(signer.PkG1)))
	}
	return dasigners.NewBN254G1Point(bn254util.SerializeG1(aggPubkeyG1)), big.NewInt(int64(len(quorum))), big.NewInt(int64(hit)), nil
}

func (d *DASignersPrecompile) GetAggPkG1(evm *EVM, method *abi.Method, args []interface{}) ([]byte, error) {
	if len(args) != 3 {
		return nil, ErrExecutionReverted
	}
	epochNumber := args[0].(*big.Int).Uint64()
	quorumId := args[1].(*big.Int).Uint64()
	quorumBitmap := args[2].([]byte)
	aggPkG1, total, hit, err := d.getAggPkG1(evm, epochNumber, quorumId, quorumBitmap)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(aggPkG1, total, hit)
}
