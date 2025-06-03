package span

import (
	"context"
	"encoding/hex"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/bor/abi"
	"github.com/ethereum/go-ethereum/consensus/bor/api"
	"github.com/ethereum/go-ethereum/consensus/bor/statefull"
	"github.com/ethereum/go-ethereum/consensus/bor/valset"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
)

type ChainSpanner struct {
	ethAPI                   api.Caller
	validatorSet             abi.ABI
	chainConfig              *params.ChainConfig
	validatorContractAddress common.Address
}

// validator response on ValidatorSet contract
type contractValidator struct {
	Id     *big.Int
	Power  *big.Int
	Signer common.Address
}

func NewChainSpanner(ethAPI api.Caller, validatorSet abi.ABI, chainConfig *params.ChainConfig, validatorContractAddress common.Address) *ChainSpanner {
	return &ChainSpanner{
		ethAPI:                   ethAPI,
		validatorSet:             validatorSet,
		chainConfig:              chainConfig,
		validatorContractAddress: validatorContractAddress,
	}
}

// GetCurrentSpan get current span from contract
func (c *ChainSpanner) GetCurrentSpan(ctx context.Context, headerHash common.Hash) (*Span, error) {
	// block
	blockNr := rpc.BlockNumberOrHashWithHash(headerHash, false)

	// method
	const method = "getCurrentSpan"

	data, err := c.validatorSet.Pack(method)
	if err != nil {
		log.Error("Unable to pack tx for getCurrentSpan", "error", err)

		return nil, err
	}

	msgData := (hexutil.Bytes)(data)
	toAddress := c.validatorContractAddress
	gas := (hexutil.Uint64)(uint64(math.MaxUint64 / 2))

	// todo: would we like to have a timeout here?
	result, err := c.ethAPI.Call(ctx, ethapi.TransactionArgs{
		Gas:  &gas,
		To:   &toAddress,
		Data: &msgData,
	}, &blockNr, nil, nil)
	if err != nil {
		return nil, err
	}

	// span result
	ret := new(struct {
		Number     *big.Int
		StartBlock *big.Int
		EndBlock   *big.Int
	})

	if err := c.validatorSet.UnpackIntoInterface(ret, method, result); err != nil {
		return nil, err
	}

	// create new span
	span := Span{
		ID:         ret.Number.Uint64(),
		StartBlock: ret.StartBlock.Uint64(),
		EndBlock:   ret.EndBlock.Uint64(),
	}

	return &span, nil
}

// GetCurrentValidators get current validators
func (c *ChainSpanner) GetCurrentValidatorsByBlockNrOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash, blockNumber uint64) ([]*valset.Validator, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	toAddress := c.validatorContractAddress
	gas := (hexutil.Uint64)(uint64(math.MaxUint64 / 2))

	valz, err := c.tryGetBorValidatorsWithId(ctx, blockNrOrHash, blockNumber, toAddress, gas)
	if err != nil {
		return nil, err
	}

	return valz, nil
}

// tryGetBorValidatorsWithId Try to get bor validators with Id from ValidatorSet contract by querying each element on mapping(uint256 => Validator[]) public producers
// If fails then returns GetBorValidators without id
func (c *ChainSpanner) tryGetBorValidatorsWithId(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash, blockNumber uint64, toAddress common.Address, gas hexutil.Uint64) ([]*valset.Validator, error) {
	firstEndBlock, err := c.getFirstEndBlock(ctx, blockNrOrHash, toAddress, gas)
	if err != nil {
		return nil, err
	}
	var spanNumber *big.Int
	if big.NewInt(int64(blockNumber)).Cmp(firstEndBlock) <= 0 {
		spanNumber = big.NewInt(0)
	} else {
		spanNumber, err = c.getSpanByBlock(ctx, blockNrOrHash, blockNumber, toAddress, gas)
		if err != nil {
			return nil, err
		}
	}

	borValidatorsWithoutId, err := c.getBorValidatorsWithoutId(ctx, blockNrOrHash, blockNumber, toAddress, gas)
	if err != nil {
		return nil, err
	}

	producersCount := len(borValidatorsWithoutId)

	valz := make([]*valset.Validator, producersCount)

	for i := 0; i < producersCount; i++ {
		p, err := c.getProducersBySpanAndIndexMethod(ctx, blockNrOrHash, toAddress, gas, spanNumber, i)
		// if fails, return validators without id
		if err != nil {
			return borValidatorsWithoutId, nil
		}

		valz[i] = &valset.Validator{
			ID:          p.Id.Uint64(),
			Address:     p.Signer,
			VotingPower: p.Power.Int64(),
		}
	}

	return valz, nil
}

func (c *ChainSpanner) getSpanByBlock(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash, blockNumber uint64, toAddress common.Address, gas hexutil.Uint64) (*big.Int, error) {
	const getSpanByBlockMethod = "getSpanByBlock"
	spanData, err := c.validatorSet.Pack(getSpanByBlockMethod, big.NewInt(0).SetUint64(blockNumber))
	if err != nil {
		log.Error("Unable to pack tx for getSpanByBlock", "error", err)
		return nil, err
	}

	spanMsgData := (hexutil.Bytes)(spanData)

	spanResult, err := c.ethAPI.Call(ctx, ethapi.TransactionArgs{
		Gas:  &gas,
		To:   &toAddress,
		Data: &spanMsgData,
	}, &blockNrOrHash, nil, nil)
	if err != nil {
		return nil, err
	}

	var spanNumber *big.Int
	if err := c.validatorSet.UnpackIntoInterface(&spanNumber, getSpanByBlockMethod, spanResult); err != nil {
		return nil, err
	}
	return spanNumber, nil
}

func (c *ChainSpanner) getProducersBySpanAndIndexMethod(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash, toAddress common.Address, gas hexutil.Uint64, spanNumber *big.Int, index int) (*contractValidator, error) {
	const getProducersBySpanAndIndexMethod = "producers"
	producerData, err := c.validatorSet.Pack(getProducersBySpanAndIndexMethod, spanNumber, big.NewInt(int64(index)))
	if err != nil {
		log.Error("Unable to pack tx for producers", "error", err)
		return nil, err
	}

	producerMsgData := (hexutil.Bytes)(producerData)

	result, err := c.ethAPI.Call(ctx, ethapi.TransactionArgs{
		Gas:  &gas,
		To:   &toAddress,
		Data: &producerMsgData,
	}, &blockNrOrHash, nil, nil)
	if err != nil {
		return nil, err
	}

	var producer contractValidator
	if err := c.validatorSet.UnpackIntoInterface(&producer, getProducersBySpanAndIndexMethod, result); err != nil {
		return nil, err
	}
	return &producer, nil
}

func (c *ChainSpanner) getFirstEndBlock(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash, toAddress common.Address, gas hexutil.Uint64) (*big.Int, error) {
	const getFirstEndBlockMethod = "FIRST_END_BLOCK"
	firstEndBlockData, err := c.validatorSet.Pack(getFirstEndBlockMethod)
	if err != nil {
		log.Error("Unable to pack tx for getFirstEndBlock", "error", err)
		return nil, err
	}

	firstEndBlockMsgData := (hexutil.Bytes)(firstEndBlockData)

	firstEndBlockResult, err := c.ethAPI.Call(ctx, ethapi.TransactionArgs{
		Gas:  &gas,
		To:   &toAddress,
		Data: &firstEndBlockMsgData,
	}, &blockNrOrHash, nil, nil)
	if err != nil {
		return nil, err
	}

	var firstEndBlockNumber *big.Int
	if err := c.validatorSet.UnpackIntoInterface(&firstEndBlockNumber, getFirstEndBlockMethod, firstEndBlockResult); err != nil {
		return nil, err
	}
	return firstEndBlockNumber, nil
}

func (c *ChainSpanner) getBorValidatorsWithoutId(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash, blockNumber uint64, toAddress common.Address, gas hexutil.Uint64) ([]*valset.Validator, error) {
	// method
	const method = "getBorValidators"

	data, err := c.validatorSet.Pack(method, big.NewInt(0).SetUint64(blockNumber))
	if err != nil {
		log.Error("Unable to pack tx for getValidator", "error", err)
		return nil, err
	}

	// call
	msgData := (hexutil.Bytes)(data)

	result, err := c.ethAPI.Call(ctx, ethapi.TransactionArgs{
		Gas:  &gas,
		To:   &toAddress,
		Data: &msgData,
	}, &blockNrOrHash, nil, nil)
	if err != nil {
		return nil, err
	}

	var (
		ret0 = new([]common.Address)
		ret1 = new([]*big.Int)
	)

	out := &[]interface{}{
		ret0,
		ret1,
	}

	if err := c.validatorSet.UnpackIntoInterface(out, method, result); err != nil {
		return nil, err
	}

	valz := make([]*valset.Validator, len(*ret0))
	for i, a := range *ret0 {
		valz[i] = &valset.Validator{
			Address:     a,
			VotingPower: (*ret1)[i].Int64(),
		}
	}

	return valz, nil
}

func (c *ChainSpanner) GetCurrentValidatorsByHash(ctx context.Context, headerHash common.Hash, blockNumber uint64) ([]*valset.Validator, error) {
	blockNr := rpc.BlockNumberOrHashWithHash(headerHash, false)

	return c.GetCurrentValidatorsByBlockNrOrHash(ctx, blockNr, blockNumber)
}

const method = "commitSpan"

func (c *ChainSpanner) CommitSpan(ctx context.Context, heimdallSpan HeimdallSpan, state *state.StateDB, header *types.Header, chainContext core.ChainContext, tracer *tracing.Hooks) error {
	// get validators bytes
	validators := make([]valset.MinimalVal, 0, len(heimdallSpan.ValidatorSet.Validators))
	for _, val := range heimdallSpan.ValidatorSet.Validators {
		validators = append(validators, val.MinimalVal())
	}

	validatorBytes, err := rlp.EncodeToBytes(validators)
	if err != nil {
		return err
	}

	// get producers bytes
	producers := make([]valset.MinimalVal, 0, len(heimdallSpan.SelectedProducers))
	for _, val := range heimdallSpan.SelectedProducers {
		producers = append(producers, val.MinimalVal())
	}

	producerBytes, err := rlp.EncodeToBytes(producers)
	if err != nil {
		return err
	}

	log.Info("✅ Committing new span",
		"id", heimdallSpan.ID,
		"startBlock", heimdallSpan.StartBlock,
		"endBlock", heimdallSpan.EndBlock,
		"validatorBytes", hex.EncodeToString(validatorBytes),
		"producerBytes", hex.EncodeToString(producerBytes),
	)

	data, err := c.validatorSet.Pack(method,
		big.NewInt(0).SetUint64(heimdallSpan.ID),
		big.NewInt(0).SetUint64(heimdallSpan.StartBlock),
		big.NewInt(0).SetUint64(heimdallSpan.EndBlock),
		validatorBytes,
		producerBytes,
	)
	if err != nil {
		log.Error("Unable to pack tx for commitSpan", "error", err)

		return err
	}

	// get system message
	msg := statefull.GetSystemMessage(c.validatorContractAddress, data)

	// apply message
	_, err = statefull.ApplyMessage(ctx, msg, state, header, c.chainConfig, chainContext, 0, tracer)

	return err
}
