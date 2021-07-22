package DDosAttack

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/monitor"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
	"strconv"
	"strings"
	"time"
)

type ExecuteResult struct {
	UseTime time.Duration
	Gas     uint64
}

func ExecuteByteCode(code []byte) (*ExecuteResult, error) {
	return ExecuteStringCode(hexutil.Encode(code))
}

func ExecuteStringCode(code string) (*ExecuteResult, error) {

	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	bgCtx := context.Background()
	signer := types.HomesteadSigner{}

	opts, _ := bind.NewKeyedTransactorWithChainID(key, big.NewInt(1337))
	opts.GasPrice = big.NewInt(1000000000)

	sim := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: big.NewInt(params.Ether)}}, 10000000)
	defer func(sim *backends.SimulatedBackend) {
		err := sim.Close()
		if err != nil {
			log.Warn("something wrong when closing sim")
		}
	}(sim)

	parsed, _ := abi.JSON(strings.NewReader("[{\n\t\t\"inputs\": [],\n\t\t\"payable\": false,\n\t\t\"stateMutability\": \"nonpayable\",\n\t\t\"type\": \"constructor\"\n\t}]"))
	contractAddr, _, _, err := bind.DeployContract(opts, parsed, common.FromHex(code), sim)
	if err != nil {
		log.Error("failed to executed the given code")
	}

	input, err := parsed.Pack("", []byte("test"))
	if err != nil {
		log.Warn("could not pack receive function on contract: %v", err)
	}

	// make sure you can call the contract in pending state
	fmt.Println("start PendingCallContract ===============================")
	res, err := sim.PendingCallContract(bgCtx, ethereum.CallMsg{
		From: addr,
		To:   &contractAddr,
		Data: input,
	})
	if err != nil {
		log.Warn("could not call receive method on contract: %v", err)
	}
	if len(res) == 0 {
		log.Warn("result of contract call was empty: %v", res)
	}

	log.Info("result of contract call: %v", res)

	nonce, err := sim.PendingNonceAt(bgCtx, addr)
	if err != nil {
		log.Info("could not get nonce for test addr: %v", err)
	}

	signedTx, err := types.SignTx(types.NewTransaction(nonce, contractAddr, new(big.Int), 52045, big.NewInt(1000000000), input), signer, key)
	if err != nil {
		log.Info("could not sign tx: %v", err)
	}

	return AddCallContractTx(sim, signedTx)
}

func ExecuteStringCode2(code string) (*ExecuteResult, error) {

	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	bgCtx := context.Background()
	signer := types.HomesteadSigner{}

	opts, _ := bind.NewKeyedTransactorWithChainID(key, big.NewInt(1337))
	opts.GasPrice = big.NewInt(1000000000)

	sim := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: big.NewInt(params.Ether)}}, 10000000)
	defer func(sim *backends.SimulatedBackend) {
		err := sim.Close()
		if err != nil {
			log.Warn("something wrong when closing sim")
		}
	}(sim)

	nonce, err := sim.PendingNonceAt(bgCtx, addr)
	if err != nil {
		log.Info("could not get nonce for test addr: %v", err)
	}

	signedTx, err := opts.Signer(opts.From, types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		GasPrice: big.NewInt(1000000000),
		Gas:      55656,
		Value:    new(big.Int),
		Data:     common.FromHex(code),
		To:       nil,
	}))
	if err != nil {
		log.Info("could not sign tx: %v", err)
	}
	_ = sim.SendTransaction(opts.Context, signedTx)
	contractAddr := crypto.CreateAddress(opts.From, signedTx.Nonce())
	state, _ := sim.GetPendingState(bgCtx)
	state.SetCode(contractAddr, common.FromHex(code))

	storageCode, _ := sim.PendingCodeAt(bgCtx, contractAddr)

	fmt.Println(hexutil.Encode(storageCode))

	parsed, _ := abi.JSON(strings.NewReader("[]"))
	input, err := parsed.Pack("", []byte("test"))
	if err != nil {
		log.Warn("could not pack receive function on contract: %v", err)
	}

	// make sure you can call the contract in pending state
	fmt.Println("start PendingCallContract ===============================")
	res, _ := sim.PendingCallContract(bgCtx, ethereum.CallMsg{
		From: addr,
		To:   &contractAddr,
		Data: input,
	})

	fmt.Printf("result of contract call: %v\n", res)

	nonce, _ = sim.PendingNonceAt(bgCtx, addr)

	sim.Commit()

	signedTx, err = types.SignTx(types.NewTransaction(nonce, contractAddr, new(big.Int), 52045, big.NewInt(1000000000), input), signer, key)
	if err != nil {
		log.Info("could not sign tx: %v", err)
	}

	return AddCallContractTx(sim, signedTx)
}

func AddCallContractTx(sim *backends.SimulatedBackend, tx *types.Transaction) (*ExecuteResult, error) {
	bgCtx := context.Background()

	err := sim.SendTransaction(bgCtx, tx)
	if err != nil {
		log.Info("could not add tx to pending block: %v", err)
	}

	sum := monitor.GetSystemUsageMonitor()
	usedTime, err := sum.GetUsedTimeByTxHash(tx.Hash().String())
	if err != nil {
		log.Error("can not get the used time")
	}
	usedGas, err := sum.GetUsedGasByTxHash(tx.Hash().String())
	if err != nil {
		log.Error("can not get the used gas")
	}

	sim.Commit()
	return &ExecuteResult{
		UseTime: *usedTime,
		Gas:     usedGas,
	}, nil
}

func EstimateGas(code string) (*ExecuteResult, error) {
	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	opts, _ := bind.NewKeyedTransactorWithChainID(key, big.NewInt(1337))
	opts.GasPrice = big.NewInt(1000000000)

	sim := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: big.NewInt(params.Ether)}}, 10000000)
	defer func(sim *backends.SimulatedBackend) {
		err := sim.Close()
		if err != nil {
			log.Warn("something wrong when closing sim")
		}
	}(sim)

	parsed, _ := abi.JSON(strings.NewReader("[]"))
	contractAddr, tx, _, err := bind.DeployContract(opts, parsed, common.FromHex(code), sim)
	if err != nil {
		log.Error("failed to executed the given code")
	}
	sim.Commit()
	gas := tx.Gas()
	sum := monitor.GetSystemUsageMonitor()
	useTime, err := sum.GetUsedTimeByTxHash(tx.Hash().String())
	if err != nil {
		return nil, err
	}
	log.Info("contractAddr : " + contractAddr.String() + "\n")
	log.Info("tx's gas : " + strconv.FormatUint(gas, 10))
	log.Info("use time: " + useTime.String())
	return &ExecuteResult{
		UseTime: *useTime,
		Gas:     gas,
	}, nil
}
