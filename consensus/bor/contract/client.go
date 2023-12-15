package contract

import (
	"context"
	"math"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/bor/api"
	"github.com/ethereum/go-ethereum/consensus/bor/clerk"
	"github.com/ethereum/go-ethereum/consensus/bor/statefull"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	vABI, _ = abi.JSON(strings.NewReader(validatorsetABI))
	sABI, _ = abi.JSON(strings.NewReader(stateReceiverABI))
)

func ValidatorSet() abi.ABI {
	return vABI
}

func StateReceiver() abi.ABI {
	return sABI
}

type GenesisContractsClient struct {
	validatorSetABI       abi.ABI
	stateReceiverABI      abi.ABI
	ValidatorContract     string
	StateReceiverContract string
	chainConfig           *params.ChainConfig
	ethAPI                api.Caller
}

const (
	validatorsetABI  = `[{"constant":true,"inputs":[],"name":"SPRINT","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"SYSTEM_ADDRESS","outputs":[{"internalType":"address","name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"CHAIN","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"FIRST_END_BLOCK","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"internalType":"uint256","name":"","type":"uint256"},{"internalType":"uint256","name":"","type":"uint256"}],"name":"producers","outputs":[{"internalType":"uint256","name":"id","type":"uint256"},{"internalType":"uint256","name":"power","type":"uint256"},{"internalType":"address","name":"signer","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"ROUND_TYPE","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"BOR_ID","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"internalType":"uint256","name":"","type":"uint256"}],"name":"spanNumbers","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"VOTE_TYPE","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"internalType":"uint256","name":"","type":"uint256"},{"internalType":"uint256","name":"","type":"uint256"}],"name":"validators","outputs":[{"internalType":"uint256","name":"id","type":"uint256"},{"internalType":"uint256","name":"power","type":"uint256"},{"internalType":"address","name":"signer","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"internalType":"uint256","name":"","type":"uint256"}],"name":"spans","outputs":[{"internalType":"uint256","name":"number","type":"uint256"},{"internalType":"uint256","name":"startBlock","type":"uint256"},{"internalType":"uint256","name":"endBlock","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"inputs":[],"payable":false,"stateMutability":"nonpayable","type":"constructor"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"uint256","name":"id","type":"uint256"},{"indexed":true,"internalType":"uint256","name":"startBlock","type":"uint256"},{"indexed":true,"internalType":"uint256","name":"endBlock","type":"uint256"}],"name":"NewSpan","type":"event"},{"constant":true,"inputs":[],"name":"currentSprint","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"internalType":"uint256","name":"span","type":"uint256"}],"name":"getSpan","outputs":[{"internalType":"uint256","name":"number","type":"uint256"},{"internalType":"uint256","name":"startBlock","type":"uint256"},{"internalType":"uint256","name":"endBlock","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"getCurrentSpan","outputs":[{"internalType":"uint256","name":"number","type":"uint256"},{"internalType":"uint256","name":"startBlock","type":"uint256"},{"internalType":"uint256","name":"endBlock","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"getNextSpan","outputs":[{"internalType":"uint256","name":"number","type":"uint256"},{"internalType":"uint256","name":"startBlock","type":"uint256"},{"internalType":"uint256","name":"endBlock","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"internalType":"uint256","name":"number","type":"uint256"}],"name":"getSpanByBlock","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"currentSpanNumber","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"internalType":"uint256","name":"span","type":"uint256"}],"name":"getValidatorsTotalStakeBySpan","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"internalType":"uint256","name":"span","type":"uint256"}],"name":"getProducersTotalStakeBySpan","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"internalType":"uint256","name":"span","type":"uint256"},{"internalType":"address","name":"signer","type":"address"}],"name":"getValidatorBySigner","outputs":[{"components":[{"internalType":"uint256","name":"id","type":"uint256"},{"internalType":"uint256","name":"power","type":"uint256"},{"internalType":"address","name":"signer","type":"address"}],"internalType":"struct BorValidatorSet.Validator","name":"result","type":"tuple"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"internalType":"uint256","name":"span","type":"uint256"},{"internalType":"address","name":"signer","type":"address"}],"name":"isValidator","outputs":[{"internalType":"bool","name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"internalType":"uint256","name":"span","type":"uint256"},{"internalType":"address","name":"signer","type":"address"}],"name":"isProducer","outputs":[{"internalType":"bool","name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"internalType":"address","name":"signer","type":"address"}],"name":"isCurrentValidator","outputs":[{"internalType":"bool","name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"internalType":"address","name":"signer","type":"address"}],"name":"isCurrentProducer","outputs":[{"internalType":"bool","name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"internalType":"uint256","name":"number","type":"uint256"}],"name":"getBorValidators","outputs":[{"internalType":"address[]","name":"","type":"address[]"},{"internalType":"uint256[]","name":"","type":"uint256[]"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"getInitialValidators","outputs":[{"internalType":"address[]","name":"","type":"address[]"},{"internalType":"uint256[]","name":"","type":"uint256[]"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"getValidators","outputs":[{"internalType":"address[]","name":"","type":"address[]"},{"internalType":"uint256[]","name":"","type":"uint256[]"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"internalType":"uint256","name":"newSpan","type":"uint256"},{"internalType":"uint256","name":"startBlock","type":"uint256"},{"internalType":"uint256","name":"endBlock","type":"uint256"},{"internalType":"bytes","name":"validatorBytes","type":"bytes"},{"internalType":"bytes","name":"producerBytes","type":"bytes"}],"name":"commitSpan","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"internalType":"uint256","name":"span","type":"uint256"},{"internalType":"bytes32","name":"dataHash","type":"bytes32"},{"internalType":"bytes","name":"sigs","type":"bytes"}],"name":"getStakePowerBySigs","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"internalType":"bytes32","name":"rootHash","type":"bytes32"},{"internalType":"bytes32","name":"leaf","type":"bytes32"},{"internalType":"bytes","name":"proof","type":"bytes"}],"name":"checkMembership","outputs":[{"internalType":"bool","name":"","type":"bool"}],"payable":false,"stateMutability":"pure","type":"function"},{"constant":true,"inputs":[{"internalType":"bytes32","name":"d","type":"bytes32"}],"name":"leafNode","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"payable":false,"stateMutability":"pure","type":"function"},{"constant":true,"inputs":[{"internalType":"bytes32","name":"left","type":"bytes32"},{"internalType":"bytes32","name":"right","type":"bytes32"}],"name":"innerNode","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"payable":false,"stateMutability":"pure","type":"function"}]`
	stateReceiverABI = `[{"constant":true,"inputs":[],"name":"SYSTEM_ADDRESS","outputs":[{"internalType":"address","name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"lastStateId","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"internalType":"uint256","name":"syncTime","type":"uint256"},{"internalType":"bytes","name":"recordBytes","type":"bytes"}],"name":"commitState","outputs":[{"internalType":"bool","name":"success","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"}]`
)

func NewGenesisContractsClient(
	chainConfig *params.ChainConfig,
	validatorContract,
	stateReceiverContract string,
	ethAPI api.Caller,
) *GenesisContractsClient {
	return &GenesisContractsClient{
		validatorSetABI:       ValidatorSet(),
		stateReceiverABI:      StateReceiver(),
		ValidatorContract:     validatorContract,
		StateReceiverContract: stateReceiverContract,
		chainConfig:           chainConfig,
		ethAPI:                ethAPI,
	}
}

func (gc *GenesisContractsClient) CommitState(
	event *clerk.EventRecordWithTime,
	state *state.StateDB,
	header *types.Header,
	chCtx statefull.ChainContext,
) (uint64, error) {
	eventRecord := event.BuildEventRecord()

	recordBytes, err := rlp.EncodeToBytes(eventRecord)
	if err != nil {
		return 0, err
	}

	const method = "commitState"

	t := event.Time.Unix()

	data, err := gc.stateReceiverABI.Pack(method, big.NewInt(0).SetInt64(t), recordBytes)
	if err != nil {
		log.Error("Unable to pack tx for commitState", "error", err)
		return 0, err
	}

	msg := statefull.GetSystemMessage(common.HexToAddress(gc.StateReceiverContract), data)

	log.Info("→ committing new state", "eventRecord", event.ID)

	gasUsed, err := statefull.ApplyMessage(context.Background(), msg, state, header, gc.chainConfig, chCtx)

	// Logging event log with time and individual gasUsed
	log.Info("→ committed new state", "eventRecord", event.String(gasUsed))

	if err != nil {
		return 0, err
	}

	return gasUsed, nil
}

func (gc *GenesisContractsClient) LastStateId(state *state.StateDB, number uint64, hash common.Hash) (*big.Int, error) {
	blockNr := rpc.BlockNumber(number)

	const method = "lastStateId"

	data, err := gc.stateReceiverABI.Pack(method)
	if err != nil {
		log.Error("Unable to pack tx for LastStateId", "error", err)

		return nil, err
	}

	msgData := (hexutil.Bytes)(data)
	toAddress := common.HexToAddress(gc.StateReceiverContract)
	gas := (hexutil.Uint64)(uint64(math.MaxUint64 / 2))

	// Do a call with state so that we can fetch the last state ID from a given (incoming)
	// state instead of local(canonical) chain.
	result, err := gc.ethAPI.Call(context.Background(), ethapi.TransactionArgs{
		Gas:  &gas,
		To:   &toAddress,
		Data: &msgData,
	}, &rpc.BlockNumberOrHash{BlockNumber: &blockNr, BlockHash: &hash}, nil, nil)
	if err != nil {
		return nil, err
	}

	ret := new(*big.Int)
	if err := gc.stateReceiverABI.UnpackIntoInterface(ret, method, result); err != nil {
		return nil, err
	}

	return *ret, nil
}
