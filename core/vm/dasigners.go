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
	ballots := make([]Ballot, cnt)
	for index := range cnt {
		account := d.epochRegisteredSigner(evm, epoch, index)
		sigHash, _ := d.getRegistration(evm, epoch, account)
		ballots[index] = Ballot{
			account: account,
			content: sigHash,
		}
	}
	// TODO: calculate ballots based on staked amount
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
		StoreBytes(evm.StateDB, d.Address(), dasigners.QuorumKey(epoch+1, uint64(index)), b)
	}
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
	if evm.Origin != signer.Signer {
		return nil, dasigners.ErrInvalidSender
	}
	if contract.caller != evm.Origin {
		return nil, precompiles.ErrSenderNotOrigin
	}
	// execute
	// validate sender
	// TODO: check staked
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

func (d *DASignersPrecompile) storeRegistration(evm *EVM, epoch uint64, signer common.Address, signature []byte) error {
	if _, found := d.getRegistration(evm, epoch, signer); found {
		return nil
	}
	// save signature hash
	evm.StateDB.SetState(d.Address(), dasigners.RegistrationKey(epoch, signer), crypto.Keccak256Hash(signature))
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
	if len(args) != 1 {
		return nil, ErrExecutionReverted
	}
	signature := dasigners.SerializeG1(args[0].(dasigners.BN254G1Point))
	// validation
	if contract.caller != evm.Origin {
		return nil, precompiles.ErrSenderNotOrigin
	}
	// execute
	// get signer
	// TODO: check staked
	signer, found, err := d.getSigner(evm, contract.caller)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, dasigners.ErrSignerNotFound
	}
	// validate signature
	epochNumber := d.epochNumber(evm)
	chainID := evm.chainConfig.ChainID
	hash := dasigners.EpochRegistrationHash(contract.caller, epochNumber+1, chainID)
	if !dasigners.ValidateSignature(signer, hash, bn254util.DeserializeG1(signature)) {
		return nil, dasigners.ErrInvalidSignature
	}
	// save registration
	if err := d.storeRegistration(evm, epochNumber+1, contract.caller, signature); err != nil {
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
		TokensPerVote:     big.NewInt(10),
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
	return signer, false, nil
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
