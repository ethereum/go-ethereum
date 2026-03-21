package engine_v1_tests

import (
	"context"
	"math/big"
	"math/rand"
	"strconv"
	"strings"
	"testing"

	ethereum "github.com/XinFinOrg/XDPoSChain"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind/backends"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/contracts"
	randomizeContract "github.com/XinFinOrg/XDPoSChain/contracts/randomize/contract"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/eth/hooks"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/require"
)

type randomizeBackendMock struct {
	*backends.SimulatedBackend
	apiABI  abi.ABI
	opening [32]byte
	latest  map[common.Address]int64
	byBlock map[uint64]map[common.Address]int64
}

func newRandomizeBackendMock(t *testing.T, backend *backends.SimulatedBackend) *randomizeBackendMock {
	t.Helper()
	parsed, err := abi.JSON(strings.NewReader(randomizeContract.XDCRandomizeABI))
	require.NoError(t, err)

	var opening [32]byte
	copy(opening[:], []byte("checkpoint-sync-randomize-key-000")) // 32 bytes prefix, deterministic

	return &randomizeBackendMock{
		SimulatedBackend: backend,
		apiABI:           parsed,
		opening:          opening,
		latest:           make(map[common.Address]int64),
		byBlock:          make(map[uint64]map[common.Address]int64),
	}
}

func (m *randomizeBackendMock) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	if contract == common.RandomizeSMCBinary {
		return []byte{1}, nil
	}
	return m.SimulatedBackend.CodeAt(ctx, contract, blockNumber)
}

func (m *randomizeBackendMock) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	if call.To == nil || *call.To != common.RandomizeSMCBinary || len(call.Data) < 4 {
		return m.SimulatedBackend.CallContract(ctx, call, blockNumber)
	}
	method, err := m.apiABI.MethodById(call.Data[:4])
	if err != nil {
		return nil, err
	}
	inputs, err := method.Inputs.Unpack(call.Data[4:])
	if err != nil {
		return nil, err
	}
	var addr common.Address
	if len(inputs) > 0 {
		addr = inputs[0].(common.Address)
	}

	switch method.Name {
	case "getSecret":
		random := m.lookupRandom(addr, blockNumber)
		encrypted := contracts.Encrypt(m.opening[:], strconv.FormatInt(random, 10))
		var secret [32]byte
		copy(secret[:], common.LeftPadBytes([]byte(encrypted), 32))
		return method.Outputs.Pack([][32]byte{secret})
	case "getOpening":
		return method.Outputs.Pack(m.opening)
	default:
		return m.SimulatedBackend.CallContract(ctx, call, blockNumber)
	}
}

func (m *randomizeBackendMock) lookupRandom(addr common.Address, blockNumber *big.Int) int64 {
	if blockNumber == nil {
		return m.latest[addr]
	}
	if vals, ok := m.byBlock[blockNumber.Uint64()]; ok {
		if random, ok := vals[addr]; ok {
			return random
		}
	}
	return m.latest[addr]
}

// Regression test for sync-time checkpoint verification.
//
// Scenario:
// 1) Build chain up to block 899.
// 2) Build checkpoint header #900 and precompute its validators from parent(#899) state.
// 3) Advance canonical chain with #900/#901 that mutate randomize contract state.
// 4) Re-verify old checkpoint header #900 against the updated chain head.
//
// Before the fix, HookVerifyMNs used latest-state randomize reads and could return
// ErrInvalidCheckpointValidators for step (4). After the fix, verification is pinned
// to parent block state and should pass.
func TestCheckpointSyncValidatorVerificationUsesParentState(t *testing.T) {
	const checkpointNumber = uint64(900)

	blockchain, backend, parentBlock, _, _ := PrepareXDCTestBlockChain(t, int(checkpointNumber-1), params.TestXDPoSMockChainConfig)
	require.Equal(t, checkpointNumber-1, parentBlock.NumberU64())

	engine := blockchain.Engine().(*XDPoS.XDPoS)
	hooks.AttachConsensusV1Hooks(engine, blockchain, blockchain.Config())
	masternodes, err := engine.EngineV1.GetAuthorisedSignersFromSnapshot(blockchain, parentBlock.Header())
	require.NoError(t, err)
	require.NotEmpty(t, masternodes)

	mockBackend := newRandomizeBackendMock(t, backend)
	parentRandoms := make(map[common.Address]int64)
	latestRandoms := make(map[common.Address]int64)
	for i, addr := range masternodes {
		parentRandoms[addr] = int64(i + 1)
		latestRandoms[addr] = int64(len(masternodes) - i + 100)
	}
	mockBackend.byBlock[checkpointNumber-1] = parentRandoms
	mockBackend.latest = latestRandoms
	blockchain.Client = mockBackend

	checkpointHeader := &types.Header{
		Root:       common.HexToHash("0xea465415b60d88429f181fec9fae67c0f19cbf5a4fa10971d96d4faa57d96ffa"),
		Number:     new(big.Int).SetUint64(checkpointNumber),
		ParentHash: parentBlock.Hash(),
		Coinbase:   common.HexToAddress("0xaaa0000000000000000000000000000000000900"),
	}

	// Build expected validators from parent-state randomize values directly,
	// independent from HookValidator implementation.
	validatorsAtParent, err := validatorsFromRandomizeAtNumber(blockchain.Client, masternodes, new(big.Int).SetUint64(checkpointNumber-1))
	require.NoError(t, err)
	validatorsAtLatest, err := validatorsFromRandomizeAtNumber(blockchain.Client, masternodes, nil)
	require.NoError(t, err)
	require.NotEqual(t, validatorsAtLatest, validatorsAtParent)
	checkpointHeader.Validators = validatorsAtParent

	// Re-verify checkpoint header while latest randomize differs from parent block.
	err = engine.EngineV1.HookVerifyMNs(checkpointHeader, masternodes)
	require.NoError(t, err)

	// Sanity: latest randomize view has diverged from parent-state view for at least
	// one masternode, proving this test exercises the historical-state requirement.
	var diverged bool
	for _, addr := range masternodes {
		latest, lerr := contracts.GetRandomizeFromContractAtNumber(blockchain.Client, addr, nil)
		require.NoError(t, lerr)
		atParent, perr := contracts.GetRandomizeFromContractAtNumber(blockchain.Client, addr, new(big.Int).SetUint64(checkpointNumber-1))
		require.NoError(t, perr)
		if latest != atParent {
			diverged = true
			break
		}
	}
	require.True(t, diverged)

	// Keep explicit guard for the historical failure signature.
	require.NotEqual(t, utils.ErrInvalidCheckpointValidators, err)
}

func validatorsFromRandomizeAtNumber(client bind.ContractBackend, masternodes []common.Address, blockNumber *big.Int) ([]byte, error) {
	randoms := make([]int64, 0, len(masternodes))
	for _, addr := range masternodes {
		random, err := contracts.GetRandomizeFromContractAtNumber(client, addr, blockNumber)
		if err != nil {
			return nil, err
		}
		randoms = append(randoms, random)
	}
	m2 := deterministicM2FromRandomize(randoms, int64(len(masternodes)))
	return contracts.BuildValidatorFromM2(m2), nil
}

// deterministicM2FromRandomize mirrors contracts.GenM2FromRandomize but uses a
// local RNG source so the test does not depend on global math/rand state.
func deterministicM2FromRandomize(randomizes []int64, lenSigners int64) []int64 {
	blockValidator := make([]int64, lenSigners)
	for i := int64(0); i < lenSigners; i++ {
		blockValidator[i] = i
	}
	randIndexs := make([]int64, lenSigners)
	total := int64(0)
	for _, v := range randomizes {
		total += v
	}
	rng := rand.New(rand.NewSource(total))

	for i := len(blockValidator) - 1; i >= 0; i-- {
		blockLength := len(blockValidator) - 1
		if blockLength <= 1 {
			blockLength = 1
		}
		randomIndex := rng.Intn(blockLength)
		temp := blockValidator[randomIndex]
		blockValidator[randomIndex] = blockValidator[i]
		blockValidator[i] = temp
		blockValidator = append(blockValidator[:i], blockValidator[i+1:]...)
		randIndexs[i] = temp
	}
	return randIndexs
}
