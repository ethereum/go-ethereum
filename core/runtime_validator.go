package core

import (
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
)

// Config is a basic type specifying certain configuration flags for running
// the EVM.
type Config struct {
	ChainConfig *params.ChainConfig
	Difficulty  *big.Int
	Origin      common.Address
	Coinbase    common.Address
	BlockNumber *big.Int
	Time        *big.Int
	GasLimit    uint64
	GasPrice    *big.Int
	Value       *big.Int
	Debug       bool
	EVMConfig   vm.Config

	State     *state.StateDB
	GetHashFn func(n uint64) common.Hash
}

var abiValidator = `[{"constant":false,"inputs":[{"name":"_candidate","type":"address"}],"name":"propose","outputs":[],"payable":true,"stateMutability":"payable","type":"function"},{"constant":false,"inputs":[{"name":"_candidate","type":"address"},{"name":"_cap","type":"uint256"}],"name":"unvote","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"getCandidates","outputs":[{"name":"","type":"address[]"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"_blockNumber","type":"uint256"}],"name":"getWithdrawCap","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"_candidate","type":"address"}],"name":"getVoters","outputs":[{"name":"","type":"address[]"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"getWithdrawBlockNumbers","outputs":[{"name":"","type":"uint256[]"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"_candidate","type":"address"},{"name":"_voter","type":"address"}],"name":"getVoterCap","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"","type":"uint256"}],"name":"candidates","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_blockNumber","type":"uint256"},{"name":"_index","type":"uint256"}],"name":"withdraw","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"_candidate","type":"address"}],"name":"getCandidateCap","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_candidate","type":"address"}],"name":"vote","outputs":[],"payable":true,"stateMutability":"payable","type":"function"},{"constant":true,"inputs":[],"name":"candidateCount","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"voterWithdrawDelay","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_candidate","type":"address"}],"name":"resign","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"_candidate","type":"address"}],"name":"getCandidateOwner","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"maxValidatorNumber","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"candidateWithdrawDelay","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"_candidate","type":"address"}],"name":"isCandidate","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"minCandidateCap","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"minVoterCap","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"inputs":[{"name":"_candidates","type":"address[]"},{"name":"_caps","type":"uint256[]"},{"name":"_firstOwner","type":"address"},{"name":"_minCandidateCap","type":"uint256"},{"name":"_minVoterCap","type":"uint256"},{"name":"_maxValidatorNumber","type":"uint256"},{"name":"_candidateWithdrawDelay","type":"uint256"},{"name":"_voterWithdrawDelay","type":"uint256"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"},{"anonymous":false,"inputs":[{"indexed":false,"name":"_voter","type":"address"},{"indexed":false,"name":"_candidate","type":"address"},{"indexed":false,"name":"_cap","type":"uint256"}],"name":"Vote","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"name":"_voter","type":"address"},{"indexed":false,"name":"_candidate","type":"address"},{"indexed":false,"name":"_cap","type":"uint256"}],"name":"Unvote","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"name":"_owner","type":"address"},{"indexed":false,"name":"_candidate","type":"address"},{"indexed":false,"name":"_cap","type":"uint256"}],"name":"Propose","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"name":"_owner","type":"address"},{"indexed":false,"name":"_candidate","type":"address"}],"name":"Resign","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"name":"_owner","type":"address"},{"indexed":false,"name":"_blockNumber","type":"uint256"},{"indexed":false,"name":"_cap","type":"uint256"}],"name":"Withdraw","type":"event"}]`

// sets defaults on the config
func setDefaults(cfg *Config) {
	if cfg.ChainConfig == nil {
		cfg.ChainConfig = &params.ChainConfig{
			ChainId:        big.NewInt(1),
			HomesteadBlock: new(big.Int),
			DAOForkBlock:   new(big.Int),
			DAOForkSupport: false,
			EIP150Block:    new(big.Int),
			EIP155Block:    new(big.Int),
			EIP158Block:    new(big.Int),
		}
	}

	if cfg.Difficulty == nil {
		cfg.Difficulty = new(big.Int)
	}
	if cfg.Time == nil {
		cfg.Time = big.NewInt(time.Now().Unix())
	}
	if cfg.GasLimit == 0 {
		cfg.GasLimit = math.MaxUint64
	}
	if cfg.GasPrice == nil {
		cfg.GasPrice = new(big.Int)
	}
	if cfg.Value == nil {
		cfg.Value = new(big.Int)
	}
	if cfg.BlockNumber == nil {
		cfg.BlockNumber = new(big.Int)
	}
	if cfg.GetHashFn == nil {
		cfg.GetHashFn = func(n uint64) common.Hash {
			return common.BytesToHash(crypto.Keccak256([]byte(new(big.Int).SetUint64(n).String())))
		}
	}
}

// Execute executes the code using the input as call data during the execution.
// It returns the EVM's return value, the new state and an error if it failed.
//
// Executes sets up a in memory, temporarily, environment for the execution of
// the given code. It makes sure that it's restored to it's original state afterwards.
/*
func Execute(code, input []byte, cfg *Config) ([]byte, *state.StateDB, error) {
	if cfg == nil {
		cfg = new(Config)
	}
	setDefaults(cfg)

	if cfg.State == nil {
		db, _ := ethdb.NewMemDatabase()
		cfg.State, _ = state.New(common.Hash{}, state.NewDatabase(db))
	}
	var (
		address = common.StringToAddress("contract")
		vmenv   = NewEnv(cfg)
		sender  = vm.AccountRef(cfg.Origin)
	)
	cfg.State.CreateAccount(address)
	// set the receiver's (the executing contract) code for execution.
	cfg.State.SetCode(address, code)
	// Call the code with the given configuration.
	ret, _, err := vmenv.Call(
		sender,
		common.StringToAddress("contract"),
		input,
		cfg.GasLimit,
		cfg.Value,
	)

	return ret, cfg.State, err
}
*/
func NewRuntimeEVM(chainState *state.StateDB) *vm.EVM {
	cfg := new(Config)
	setDefaults(cfg)
	db, _ := ethdb.NewMemDatabase()
	cfg.State, _ = state.New(common.Hash{}, state.NewDatabase(db))

	var (
		address = common.HexToAddress(common.MasternodeVotingSMC)
		vmenv   = NewEnv(cfg)
	)
	cfg.State.CreateAccount(address)
	code := chainState.GetCode(common.HexToAddress(common.MasternodeVotingSMC))
	cfg.State.SetCode(address, code)

	f := func(key, val common.Hash) bool {
		cfg.State.SetState(address, key, val)
		return true
	}
	chainState.ForEachStorage(common.HexToAddress(common.MasternodeVotingSMC), f)

	return vmenv
}

func GetVoters(candidate common.Address, vmenv *vm.EVM) ([]common.Address, error) {

	abi, err := abi.JSON(strings.NewReader(abiValidator))
	getVoters, err := abi.Pack("getVoters", candidate)

	// Call the code with the given configuration.
	voters, _, err := vmenv.Call(
		vm.AccountRef(common.HexToAddress(common.MasternodeVotingSMC)),
		common.HexToAddress(common.MasternodeVotingSMC),
		getVoters,
		math.MaxUint64,
		new(big.Int),
	)

	ret := common.ExtractAddressFromBytes(voters)

	return ret, err
}

func GetCandidateOwner(candidate common.Address, vmenv *vm.EVM) (common.Address, error) {

	abi, err := abi.JSON(strings.NewReader(abiValidator))
	getCandidateOwner, err := abi.Pack("getCandidateOwner", candidate)

	// Call the code with the given configuration.
	ret, _, err := vmenv.Call(
		vm.AccountRef(common.HexToAddress(common.MasternodeVotingSMC)),
		common.HexToAddress(common.MasternodeVotingSMC),
		getCandidateOwner,
		math.MaxUint64,
		new(big.Int),
	)

	return common.BytesToAddress(ret), err
}

func GetCandidateCap(candidate common.Address, vmenv *vm.EVM) (*big.Int, error) {

	abi, err := abi.JSON(strings.NewReader(abiValidator))
	getCandidateCap, err := abi.Pack("getCandidateCap", candidate)

	// Call the code with the given configuration.
	ret, _, err := vmenv.Call(
		vm.AccountRef(common.HexToAddress(common.MasternodeVotingSMC)),
		common.HexToAddress(common.MasternodeVotingSMC),
		getCandidateCap,
		math.MaxUint64,
		new(big.Int),
	)

	return new(big.Int).SetBytes(ret), err
}

func GetVoterCap(candidate common.Address, voter common.Address, vmenv *vm.EVM) (*big.Int, error) {

	abi, err := abi.JSON(strings.NewReader(abiValidator))
	getVoterCap, err := abi.Pack("getVoterCap", candidate, voter)

	// Call the code with the given configuration.
	ret, _, err := vmenv.Call(
		vm.AccountRef(common.HexToAddress(common.MasternodeVotingSMC)),
		common.HexToAddress(common.MasternodeVotingSMC),
		getVoterCap,
		math.MaxUint64,
		new(big.Int),
	)

	return new(big.Int).SetBytes(ret), err
}
