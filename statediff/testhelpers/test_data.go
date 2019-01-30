package testhelpers

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/statediff/builder"
	"math/big"
	"math/rand"
)

var (
	BlockNumber     = rand.Int63()
	BlockHash       = "0xfa40fbe2d98d98b3363a778d52f2bcd29d6790b9b3f3cab2b167fd12d3550f73"
	CodeHash        = "0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"
	NewNonceValue   = rand.Uint64()
	NewBalanceValue = rand.Int63()
	ContractRoot    = "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"
	StoragePath     = "0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"
	StorageKey      = "0000000000000000000000000000000000000000000000000000000000000001"
	StorageValue    = "0x03"
	storage         = map[string]builder.DiffStorage{StoragePath: {
		Key:   &StorageKey,
		Value: &StorageValue,
	}}
	emptyStorage           = map[string]builder.DiffStorage{}
	address                = common.HexToAddress("0xaE9BEa628c4Ce503DcFD7E305CaB4e29E7476592")
	anotherAddress         = common.HexToAddress("0xaE9BEa628c4Ce503DcFD7E305CaB4e29E7476593")
	ContractAddress        = address.String()
	AnotherContractAddress = anotherAddress.String()
	CreatedAccountDiffs    = map[common.Address]builder.AccountDiff{
		address: {
			Nonce:        builder.DiffUint64{Value: &NewNonceValue},
			Balance:      builder.DiffBigInt{Value: big.NewInt(NewBalanceValue)},
			ContractRoot: builder.DiffString{Value: &ContractRoot},
			CodeHash:     CodeHash,
			Storage:      storage,
		},
		anotherAddress: {
			Nonce:        builder.DiffUint64{Value: &NewNonceValue},
			Balance:      builder.DiffBigInt{Value: big.NewInt(NewBalanceValue)},
			CodeHash:     CodeHash,
			ContractRoot: builder.DiffString{Value: &ContractRoot},
			Storage:      emptyStorage,
		},
	}

	UpdatedAccountDiffs = map[common.Address]builder.AccountDiff{address: {
		Nonce:        builder.DiffUint64{Value: &NewNonceValue},
		Balance:      builder.DiffBigInt{Value: big.NewInt(NewBalanceValue)},
		CodeHash:     CodeHash,
		ContractRoot: builder.DiffString{Value: &ContractRoot},
		Storage:      storage,
	}}

	DeletedAccountDiffs = map[common.Address]builder.AccountDiff{address: {
		Nonce:        builder.DiffUint64{Value: &NewNonceValue},
		Balance:      builder.DiffBigInt{Value: big.NewInt(NewBalanceValue)},
		ContractRoot: builder.DiffString{Value: &ContractRoot},
		CodeHash:     CodeHash,
		Storage:      storage,
	}}

	TestStateDiff = builder.StateDiff{
		BlockNumber:     BlockNumber,
		BlockHash:       common.HexToHash(BlockHash),
		CreatedAccounts: CreatedAccountDiffs,
		DeletedAccounts: DeletedAccountDiffs,
		UpdatedAccounts: UpdatedAccountDiffs,
	}
)
