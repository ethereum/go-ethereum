// Copyright 2017 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/params"
	"gopkg.in/yaml.v3"

	"context"
	"math/big"

	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind/backends"
	blockSignerContract "github.com/XinFinOrg/XDPoSChain/contracts/blocksigner"
	multiSignWalletContract "github.com/XinFinOrg/XDPoSChain/contracts/multisigwallet"
	randomizeContract "github.com/XinFinOrg/XDPoSChain/contracts/randomize"
	validatorContract "github.com/XinFinOrg/XDPoSChain/contracts/validator"
	"github.com/XinFinOrg/XDPoSChain/crypto"
)

type GenesisInput struct {
	Name                    string
	ChainId                 uint64
	Denom                   string
	Period                  uint64
	Epoch                   uint64
	Gap                     uint64
	TimeoutPeriod           int
	TimeoutSyncThreshold    int
	V2SwitchBlock           uint64
	CertThreshold           float64
	MasternodesOwner        common.Address
	Masternodes             []common.Address
	StakingThreshold        uint64
	RewardYield             uint64
	FoundationWalletAddress common.Address
}

func NewGenesisInput() *GenesisInput {
	return &GenesisInput{
		Name:                    "xdc-custom-network",
		ChainId:                 5551,
		Denom:                   "xdc",
		Period:                  2,
		Epoch:                   900,
		Gap:                     450,
		TimeoutPeriod:           10,
		TimeoutSyncThreshold:    3,
		V2SwitchBlock:           0,
		CertThreshold:           0.667,
		StakingThreshold:        10_000_000, // 10M
		RewardYield:             10,
		FoundationWalletAddress: common.FoundationAddrBinary,
	}
}

func (w *wizard) loadGenesisInput() *GenesisInput {
	input := NewGenesisInput()
	file, err := os.Open(w.conf.inpath)
	if err != nil {
		log.Warn("Failed to open genesis input file", "err", err)
		os.Exit(1)
		return nil
	}
	defer file.Close()

	log.Warn("Decoding genesis input file", "path", w.conf.inpath)
	log.Warn("File content", "file", file)
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&input); err != nil {
		log.Warn("Failed to decode genesis input file (expect yaml format)", "err", err)
		os.Exit(1)
		return nil
	}
	fmt.Println("Generating genesis file with the below input")
	fmt.Printf("%+v\n", input)

	return input
}

// makeGenesis creates a new genesis struct based on some user input.
func (w *wizard) makeGenesis() {
	// Construct a default genesis block
	genesis := &core.Genesis{
		Timestamp:  uint64(time.Now().Unix()),
		GasLimit:   4700000,
		Difficulty: big.NewInt(524288),
		Alloc:      make(types.GenesisAlloc),
		Config: &params.ChainConfig{
			HomesteadBlock: big.NewInt(0),
			EIP150Block:    big.NewInt(0),
			EIP155Block:    big.NewInt(0),
			EIP158Block:    big.NewInt(0),
			ByzantiumBlock: big.NewInt(0),
		},
	}
	// Figure out which consensus engine to choose
	fmt.Println()
	fmt.Println("Which consensus engine to use? (default = XDPoS)")
	fmt.Println(" 1. Ethash - proof-of-work")
	fmt.Println(" 2. Clique - proof-of-authority")
	fmt.Println(" 3. XDPoS - delegated-proof-of-stake")

	input := w.loadGenesisInput()
	var choice string
	if input != nil {
		choice = "3"
	} else {
		choice = w.read()
	}
	switch {
	case choice == "1":
		// In case of ethash, we're pretty much done
		genesis.Config.Ethash = new(params.EthashConfig)
		genesis.ExtraData = make([]byte, 32)

	case choice == "2":
		// In the case of clique, configure the consensus parameters
		genesis.Difficulty = big.NewInt(1)
		genesis.Config.Clique = &params.CliqueConfig{
			Period: 15,
			Epoch:  900,
		}
		fmt.Println()
		fmt.Println("How many seconds should blocks take? (default = 15)")
		genesis.Config.Clique.Period = uint64(w.readDefaultInt(15))

		// We also need the initial list of signers
		fmt.Println()
		fmt.Println("Which accounts are allowed to seal? (mandatory at least one)")

		var signers []common.Address
		for {
			if address := w.readAddress(); address != nil {
				signers = append(signers, *address)
				continue
			}
			if len(signers) > 0 {
				break
			}
		}
		// Sort the signers and embed into the extra-data section
		for i := 0; i < len(signers); i++ {
			for j := i + 1; j < len(signers); j++ {
				if bytes.Compare(signers[i][:], signers[j][:]) > 0 {
					signers[i], signers[j] = signers[j], signers[i]
				}
			}
		}
		genesis.ExtraData = make([]byte, 32+len(signers)*common.AddressLength+crypto.SignatureLength)
		for i, signer := range signers {
			copy(genesis.ExtraData[32+i*common.AddressLength:], signer[:])
		}

	case choice == "" || choice == "3":
		genesis.Difficulty = big.NewInt(1)
		genesis.Config.XDPoS = &params.XDPoSConfig{
			Period: 15,
			Epoch:  900,
			Reward: 0,
			V2: &params.V2{
				SwitchBlock:   big.NewInt(0),
				CurrentConfig: &params.V2Config{},
				AllConfigs:    make(map[uint64]*params.V2Config),
			},
		}
		fmt.Println()
		fmt.Println("How many seconds should blocks take? (default = 2)")
		if input != nil {
			genesis.Config.XDPoS.Period = input.Period
		} else {
			genesis.Config.XDPoS.Period = uint64(w.readDefaultInt(2))
		}
		genesis.Config.XDPoS.V2.CurrentConfig.MinePeriod = int(genesis.Config.XDPoS.Period)

		fmt.Println()
		fmt.Println("Which block number start v2 consesus? (default = 0)")
		if input != nil {
			genesis.Config.XDPoS.V2.SwitchBlock = big.NewInt(int64(input.V2SwitchBlock))
		} else {
			genesis.Config.XDPoS.V2.SwitchBlock = w.readDefaultBigInt(genesis.Config.XDPoS.V2.SwitchBlock)
		}
		genesis.Config.XDPoS.V2.CurrentConfig.SwitchRound = 0

		fmt.Println()
		fmt.Println("How long is the v2 timeout period? (default = 10)")
		if input != nil {
			genesis.Config.XDPoS.V2.CurrentConfig.TimeoutPeriod = input.TimeoutPeriod
		} else {
			genesis.Config.XDPoS.V2.CurrentConfig.TimeoutPeriod = w.readDefaultInt(10)
		}

		fmt.Println()
		fmt.Println("How many v2 timeout reach to send Synchronize message? (default = 3)")
		if input != nil {
			genesis.Config.XDPoS.V2.CurrentConfig.TimeoutSyncThreshold = input.TimeoutSyncThreshold
		} else {
			genesis.Config.XDPoS.V2.CurrentConfig.TimeoutSyncThreshold = w.readDefaultInt(3)
		}

		fmt.Println()
		fmt.Printf("Proportion of total masternodes v2 vote collection to generate a QC (float value), should be two thirds of masternodes? (default = %f)\n", 0.667)
		if input != nil {
			genesis.Config.XDPoS.V2.CurrentConfig.CertThreshold = input.CertThreshold
		} else {
			genesis.Config.XDPoS.V2.CurrentConfig.CertThreshold = w.readDefaultFloat(0.667)
		}
		genesis.Config.XDPoS.V2.CurrentConfig.MaxMasternodes = 108
		// TODO: config to add after rewards upgrade enabled
		// genesis.Config.XDPoS.V2.CurrentConfig.MaxProtectornodes
		// genesis.Config.XDPoS.V2.CurrentConfig.MaxObservernodes
		// genesis.Config.XDPoS.V2.CurrentConfig.MinProtectornodes
		// genesis.Config.XDPoS.V2.CurrentConfig.MasternodeReward
		// genesis.Config.XDPoS.V2.CurrentConfig.ProtectornodeReward
		// genesis.Config.XDPoS.V2.CurrentConfig.ObservernodeReward

		genesis.Config.XDPoS.V2.AllConfigs[0] = genesis.Config.XDPoS.V2.CurrentConfig

		fmt.Println()
		fmt.Println("Who own the first masternodes? (mandatory)")
		var owner common.Address
		if input != nil {
			owner = input.MasternodesOwner
		} else {
			owner = *w.readAddress()
		}

		// We also need the initial list of signers
		fmt.Println()
		fmt.Println("Which accounts are Masternodes? (mandatory at least one)")

		var signers []common.Address
		if input != nil {
			signers = append(signers, input.Masternodes...)
		} else {
			for {
				if address := w.readAddress(); address != nil {
					signers = append(signers, *address)
					continue
				}
				if len(signers) > 0 {
					break
				}
			}
		}
		// Sort the signers and embed into the extra-data section
		for i := 0; i < len(signers); i++ {
			for j := i + 1; j < len(signers); j++ {
				if bytes.Compare(signers[i][:], signers[j][:]) > 0 {
					signers[i], signers[j] = signers[j], signers[i]
				}
			}
		}

		fmt.Println()
		fmt.Println("How many blocks per epoch? (default = 900)")
		if input != nil {
			genesis.Config.XDPoS.Epoch = input.Epoch
		} else {
			genesis.Config.XDPoS.Epoch = uint64(w.readDefaultInt(900))
		}
		genesis.Config.XDPoS.RewardCheckpoint = genesis.Config.XDPoS.Epoch

		fmt.Println()
		fmt.Println("How many blocks before checkpoint need to prepare new set of masternodes? (default = 450)")
		if input != nil {
			genesis.Config.XDPoS.Gap = input.Gap
		} else {
			genesis.Config.XDPoS.Gap = uint64(w.readDefaultInt(450))
		}

		fmt.Println()
		fmt.Println("What is minimum staking threshold to become a Validator? (default = 10M)")
		var threshold uint64
		if input != nil {
			threshold = input.StakingThreshold
		} else {
			threshold = uint64(w.readDefaultInt(10000000))
		}

		fmt.Println()
		// fmt.Println("How many Ethers should be rewarded to masternode each Epoch? (default = 10)")
		fmt.Println("What should be the reward yield of Masternodes in APY% (default = 10)")
		yield := uint64(0)
		if input != nil {
			yield = input.RewardYield
		} else {
			yield = uint64(w.readDefaultInt(10))
		}
		blocksPerYear := 31536000 / genesis.Config.XDPoS.Period
		epochsPerYear := blocksPerYear / genesis.Config.XDPoS.Epoch
		rewardsPerYear := float64(threshold) * (float64(yield) / float64(100))
		rewardPerEpochPerMN := uint64(rewardsPerYear / float64(epochsPerYear))
		totalRewardPerEpoch := rewardPerEpochPerMN * uint64(len(signers))
		fmt.Println()
		fmt.Println("Calculated Total Masternode rewards per epoch based on yield: ", totalRewardPerEpoch)
		genesis.Config.XDPoS.Reward = totalRewardPerEpoch

		fmt.Println()
		fmt.Println("What is foundation wallet address (collect 10% of all rewards)? (default = xdc0000000000000000000000000000000000000068)")
		if input != nil {
			genesis.Config.XDPoS.FoundationWalletAddr = input.FoundationWalletAddress
		} else {
			genesis.Config.XDPoS.FoundationWalletAddr = w.readDefaultAddress(common.FoundationAddrBinary)
		}

		// Validator Smart Contract Code
		pKey, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr := crypto.PubkeyToAddress(pKey.PublicKey)
		// Gas limit increased to 10,000,000,000 to support validator contract deployment with large masternode counts (>38).
		contractBackend := backends.NewXDCSimulatedBackend(types.GenesisAlloc{addr: {Balance: big.NewInt(1000000000)}}, 10_000_000_000, params.TestXDPoSMockChainConfig)
		//lint:ignore SA1019 chainID is not determined at this time
		transactOpts := bind.NewKeyedTransactor(pKey)

		minDeposit := new(big.Int).SetUint64(threshold)
		minDeposit.Mul(minDeposit, big.NewInt(1e18)) //convert to wei
		validatorCap := new(big.Int).Set(minDeposit)
		var validatorCaps []*big.Int
		genesis.ExtraData = make([]byte, 32+len(signers)*common.AddressLength+crypto.SignatureLength)
		for i, signer := range signers {
			validatorCaps = append(validatorCaps, validatorCap)
			copy(genesis.ExtraData[32+i*common.AddressLength:], signer[:])
		}
		validatorAddress, _, err := validatorContract.DeployValidator(transactOpts, contractBackend, signers, validatorCaps, owner, minDeposit, nil)
		if err != nil {
			fmt.Println("Can't deploy root registry")
		}
		contractBackend.Commit()

		d := time.Now().Add(1000 * time.Millisecond)
		ctx, cancel := context.WithDeadline(context.Background(), d)
		defer cancel()
		code, _ := contractBackend.CodeAt(ctx, validatorAddress, nil)
		storage := make(map[common.Hash]common.Hash)
		f := func(key, val common.Hash) bool {
			storage[key] = common.BytesToHash(val.Bytes())
			log.Info("DecodeBytes", "value", val, "decode", storage[key])
			return true
		}
		contractBackend.ForEachStorageAt(ctx, validatorAddress, nil, f)
		genesis.Alloc[common.MasternodeVotingSMCBinary] = types.Account{
			Balance: validatorCap.Mul(validatorCap, big.NewInt(int64(len(validatorCaps)))),
			Code:    code,
			Storage: storage,
		}

		fmt.Println()
		fmt.Println("Which accounts are allowed to confirm in Foundation MultiSignWallet?")
		var owners []common.Address
		if input != nil {
			owners = append(owners, input.MasternodesOwner)
		} else {
			for {
				if address := w.readAddress(); address != nil {
					owners = append(owners, *address)
					continue
				}
				if len(owners) > 0 {
					break
				}
			}
		}

		fmt.Println()
		fmt.Println("How many require for confirm tx in Foundation MultiSignWallet? (default = 1)")
		var required uint64
		if input != nil {
			required = 1
		} else {
			required = uint64(w.readDefaultInt(1))
		}

		// MultiSigWallet.
		multiSignWalletAddr, _, err := multiSignWalletContract.DeployMultiSigWallet(transactOpts, contractBackend, owners, big.NewInt(int64(required)))
		if err != nil {
			fmt.Println("Can't deploy MultiSignWallet SMC")
		}
		contractBackend.Commit()
		code, _ = contractBackend.CodeAt(ctx, multiSignWalletAddr, nil)
		storage = make(map[common.Hash]common.Hash)
		contractBackend.ForEachStorageAt(ctx, multiSignWalletAddr, nil, f)
		fBalance := big.NewInt(0) // 16m
		fBalance.Add(fBalance, big.NewInt(16*1000*1000))
		fBalance.Mul(fBalance, big.NewInt(1000000000000000000))
		genesis.Alloc[common.FoundationAddrBinary] = types.Account{
			Balance: fBalance,
			Code:    code,
			Storage: storage,
		}

		// Block Signers Smart Contract
		blockSignerAddress, _, err := blockSignerContract.DeployBlockSigner(transactOpts, contractBackend, big.NewInt(int64(genesis.Config.XDPoS.Epoch)))
		if err != nil {
			fmt.Println("Can't deploy root registry")
		}
		contractBackend.Commit()

		code, _ = contractBackend.CodeAt(ctx, blockSignerAddress, nil)
		storage = make(map[common.Hash]common.Hash)
		contractBackend.ForEachStorageAt(ctx, blockSignerAddress, nil, f)
		genesis.Alloc[common.BlockSignersBinary] = types.Account{
			Balance: big.NewInt(0),
			Code:    code,
			Storage: storage,
		}

		// Randomize Smart Contract Code
		randomizeAddress, _, err := randomizeContract.DeployRandomize(transactOpts, contractBackend)
		if err != nil {
			fmt.Println("Can't deploy root registry")
		}
		contractBackend.Commit()

		code, _ = contractBackend.CodeAt(ctx, randomizeAddress, nil)
		storage = make(map[common.Hash]common.Hash)
		contractBackend.ForEachStorageAt(ctx, randomizeAddress, nil, f)
		genesis.Alloc[common.RandomizeSMCBinary] = types.Account{
			Balance: big.NewInt(0),
			Code:    code,
			Storage: storage,
		}

		fmt.Println()
		fmt.Println("Which accounts are allowed to confirm in Team MultiSignWallet?")
		var teams []common.Address
		if input != nil {
			teams = append(teams, input.MasternodesOwner)
		} else {
			for {
				if address := w.readAddress(); address != nil {
					teams = append(teams, *address)
					continue
				}
				if len(teams) > 0 {
					break
				}
			}
		}

		fmt.Println()
		fmt.Println("How many require for confirm tx in Team MultiSignWallet? (default = 2)")
		var requiredTeam int64
		if input != nil {
			requiredTeam = 1
		} else {
			requiredTeam = int64(w.readDefaultInt(1))
		}

		// MultiSigWallet.
		multiSignWalletTeamAddr, _, err := multiSignWalletContract.DeployMultiSigWallet(transactOpts, contractBackend, teams, big.NewInt(requiredTeam))
		if err != nil {
			fmt.Println("Can't deploy MultiSignWallet SMC")
		}
		contractBackend.Commit()
		code, _ = contractBackend.CodeAt(ctx, multiSignWalletTeamAddr, nil)
		storage = make(map[common.Hash]common.Hash)
		contractBackend.ForEachStorageAt(ctx, multiSignWalletTeamAddr, nil, f)
		// Team balance.
		balance := big.NewInt(0) // 12m
		balance.Add(balance, big.NewInt(12*1000*1000))
		balance.Mul(balance, big.NewInt(1000000000000000000))
		subBalance := big.NewInt(0) // i * 50k
		subBalance.Add(subBalance, big.NewInt(int64(len(signers))*50*1000))
		subBalance.Mul(subBalance, big.NewInt(1000000000000000000))
		balance.Sub(balance, subBalance) // 12m - i * 50k
		genesis.Alloc[common.TeamAddrBinary] = types.Account{
			Balance: balance,
			Code:    code,
			Storage: storage,
		}

	default:
		log.Crit("Invalid consensus engine choice", "choice", choice)
	}
	// Consensus all set, just ask for initial funds and go
	fmt.Println()
	fmt.Println("Which accounts should be pre-funded? (advisable at least one)")
	var addresses []common.Address
	if input != nil {
		addresses = append(addresses, input.MasternodesOwner)
	} else {
		for {
			if address := w.readAddress(); address != nil {
				addresses = append(addresses, *address)
				continue
			}
			break
		}
	}
	for _, address := range addresses {
		baseBalance := big.NewInt(0) // 21m
		baseBalance.Add(baseBalance, big.NewInt(21_000_000))
		baseBalance.Mul(baseBalance, big.NewInt(1e18))
		genesis.Alloc[address] = types.Account{
			Balance: baseBalance,
		}
	}

	// Add a batch of precompile balances to avoid them getting deleted
	for i := int64(0); i < 2; i++ {
		genesis.Alloc[common.BigToAddress(big.NewInt(i))] = types.Account{Balance: big.NewInt(0)}
	}
	// Query the user for some custom extras
	fmt.Println()
	fmt.Println("Specify your chain/network ID if you want an explicit one (default = random)")
	if input != nil {
		genesis.Config.ChainID = new(big.Int).SetUint64(input.ChainId)
	} else {
		genesis.Config.ChainID = new(big.Int).SetUint64(uint64(w.readDefaultInt(rand.Intn(65536))))
	}

	// All done, store the genesis and flush to disk
	log.Info("Configured new genesis block")

	w.conf.Genesis = genesis
	w.conf.flush()
}

// manageGenesis permits the modification of chain configuration parameters in
// a genesis config and the export of the entire genesis spec.
func (w *wizard) manageGenesis() {
	// Figure out whether to modify or export the genesis
	fmt.Println()
	fmt.Println(" 1. Modify existing fork rules")
	fmt.Println(" 2. Export genesis configuration")
	fmt.Println(" 3. Remove genesis configuration")

	choice := w.read()
	switch {
	case choice == "1":
		// Fork rule updating requested, iterate over each fork
		fmt.Println()
		fmt.Printf("Which block should Homestead come into effect? (default = %v)\n", w.conf.Genesis.Config.HomesteadBlock)
		w.conf.Genesis.Config.HomesteadBlock = w.readDefaultBigInt(w.conf.Genesis.Config.HomesteadBlock)

		fmt.Println()
		fmt.Printf("Which block should EIP150 come into effect? (default = %v)\n", w.conf.Genesis.Config.EIP150Block)
		w.conf.Genesis.Config.EIP150Block = w.readDefaultBigInt(w.conf.Genesis.Config.EIP150Block)

		fmt.Println()
		fmt.Printf("Which block should EIP155 come into effect? (default = %v)\n", w.conf.Genesis.Config.EIP155Block)
		w.conf.Genesis.Config.EIP155Block = w.readDefaultBigInt(w.conf.Genesis.Config.EIP155Block)

		fmt.Println()
		fmt.Printf("Which block should EIP158 come into effect? (default = %v)\n", w.conf.Genesis.Config.EIP158Block)
		w.conf.Genesis.Config.EIP158Block = w.readDefaultBigInt(w.conf.Genesis.Config.EIP158Block)

		fmt.Println()
		fmt.Printf("Which block should Byzantium come into effect? (default = %v)\n", w.conf.Genesis.Config.ByzantiumBlock)
		w.conf.Genesis.Config.ByzantiumBlock = w.readDefaultBigInt(w.conf.Genesis.Config.ByzantiumBlock)

		out, _ := json.MarshalIndent(w.conf.Genesis.Config, "", "  ")
		fmt.Printf("Chain configuration updated:\n\n%s\n", out)

	case choice == "2":
		// Save whatever genesis configuration we currently have
		fmt.Println()
		fmt.Printf("Which file to save the genesis into? (default = %s.json)\n", w.network)
		out, _ := json.MarshalIndent(w.conf.Genesis, "", "  ")
		if err := os.WriteFile(w.readDefaultString(fmt.Sprintf("%s.json", w.network)), out, 0644); err != nil {
			log.Error("Failed to save genesis file", "err", err)
		}
		log.Info("Exported existing genesis block")

	case choice == "3":
		// Make sure we don't have any services running
		if len(w.conf.servers()) > 0 {
			log.Error("Genesis reset requires all services and servers torn down")
			return
		}
		log.Info("Genesis block destroyed")

		w.conf.Genesis = nil
		w.conf.flush()

	default:
		log.Error("That's not something I can do")
	}
}
