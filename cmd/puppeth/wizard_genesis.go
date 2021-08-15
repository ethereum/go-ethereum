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
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	blockSignerContract "github.com/ethereum/go-ethereum/contracts/blocksigner"
	multiSignWalletContract "github.com/ethereum/go-ethereum/contracts/multisigwallet"
	randomizeContract "github.com/ethereum/go-ethereum/contracts/randomize"
	validatorContract "github.com/ethereum/go-ethereum/contracts/validator"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// makeGenesis creates a new genesis struct based on some user input.
func (w *wizard) makeGenesis() {
	// Construct a default genesis block
	genesis := &core.Genesis{
		Timestamp:  uint64(time.Now().Unix()),
		GasLimit:   4700000,
		Difficulty: big.NewInt(524288),
		Alloc:      make(core.GenesisAlloc),
		Config: &params.ChainConfig{
			HomesteadBlock:      big.NewInt(1),
			EIP150Block:         big.NewInt(2),
			EIP155Block:         big.NewInt(3),
			EIP158Block:         big.NewInt(3),
			ByzantiumBlock:      big.NewInt(4),
			ConstantinopleBlock: big.NewInt(5),
		},
	}
	// Figure out which consensus engine to choose
	fmt.Println()
	fmt.Println("Which consensus engine to use? (default = XDPoS)")
	fmt.Println(" 1. Ethash - proof-of-work")
	fmt.Println(" 2. Clique - proof-of-authority")
	fmt.Println(" 3. XDPoS - delegated-proof-of-stake")

	choice := w.read()
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
			Epoch:  30000,
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
		genesis.ExtraData = make([]byte, 32+len(signers)*common.AddressLength+65)
		for i, signer := range signers {
			copy(genesis.ExtraData[32+i*common.AddressLength:], signer[:])
		}

	case choice == "" || choice == "3":
		genesis.Difficulty = big.NewInt(1)
		genesis.Config.XDPoS = &params.XDPoSConfig{
			Period: 15,
			Epoch:  30000,
			Reward: 0,
		}
		fmt.Println()
		fmt.Println("How many seconds should blocks take? (default = 2)")
		genesis.Config.XDPoS.Period = uint64(w.readDefaultInt(2))

		fmt.Println()
		fmt.Println("How many XDC should be rewarded to masternode? (default = 5000)")
		genesis.Config.XDPoS.Reward = uint64(w.readDefaultInt(5000))

		fmt.Println()
		fmt.Println("Who own the first masternodes? (mandatory)")
		owner := *w.readAddress()

		// We also need the initial list of signers
		fmt.Println()
		fmt.Println("Which accounts are allowed to seal (signers)? (mandatory at least one)")

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
		validatorCap := new(big.Int)
		validatorCap.SetString("10000000000000000000000000", 10)
		var validatorCaps []*big.Int
		genesis.ExtraData = make([]byte, 32+len(signers)*common.AddressLength+65)
		for i, signer := range signers {
			validatorCaps = append(validatorCaps, validatorCap)
			copy(genesis.ExtraData[32+i*common.AddressLength:], signer[:])
		}

		fmt.Println()
		fmt.Println("How many blocks per epoch? (default = 900)")
		epochNumber := uint64(w.readDefaultInt(900))
		genesis.Config.XDPoS.Epoch = epochNumber
		genesis.Config.XDPoS.RewardCheckpoint = epochNumber

		fmt.Println()
		fmt.Println("How many blocks before checkpoint need to prepare new set of masternodes? (default = 450)")
		genesis.Config.XDPoS.Gap = uint64(w.readDefaultInt(450))

		fmt.Println()
		fmt.Println("What is foundation wallet address? (default = xdc746249C61f5832C5eEd53172776b460491bDcd5C)")
		genesis.Config.XDPoS.FoudationWalletAddr = w.readDefaultAddress(common.HexToAddress(common.FoudationAddr))

		// Validator Smart Contract Code
		pKey, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr := crypto.PubkeyToAddress(pKey.PublicKey)
		contractBackend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: big.NewInt(1000000000)}}, 10000000)
		transactOpts := bind.NewKeyedTransactor(pKey)

		validatorAddress, _, err := validatorContract.DeployValidator(transactOpts, contractBackend, signers, validatorCaps, owner)
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
			decode := []byte{}
			trim := bytes.TrimLeft(val.Bytes(), "\x00")
			rlp.DecodeBytes(trim, &decode)
			storage[key] = common.BytesToHash(decode)
			log.Info("DecodeBytes", "value", val.String(), "decode", storage[key].String())
			return true
		}
		contractBackend.ForEachStorageAt(ctx, validatorAddress, nil, f)
		genesis.Alloc[common.HexToAddress(common.MasternodeVotingSMC)] = core.GenesisAccount{
			Balance: validatorCap.Mul(validatorCap, big.NewInt(int64(len(validatorCaps)))),
			Code:    code,
			Storage: storage,
		}

		fmt.Println()
		fmt.Println("Which accounts are allowed to confirm in Foudation MultiSignWallet?")
		var owners []common.Address
		for {
			if address := w.readAddress(); address != nil {
				owners = append(owners, *address)
				continue
			}
			if len(owners) > 0 {
				break
			}
		}
		fmt.Println()
		fmt.Println("How many require for confirm tx in Foudation MultiSignWallet? (default = 2)")
		required := int64(w.readDefaultInt(2))

		// MultiSigWallet.
		multiSignWalletAddr, _, err := multiSignWalletContract.DeployMultiSigWallet(transactOpts, contractBackend, owners, big.NewInt(required))
		if err != nil {
			fmt.Println("Can't deploy MultiSignWallet SMC")
		}
		contractBackend.Commit()
		code, _ = contractBackend.CodeAt(ctx, multiSignWalletAddr, nil)
		storage = make(map[common.Hash]common.Hash)
		contractBackend.ForEachStorageAt(ctx, multiSignWalletAddr, nil, f)
		fBalance := big.NewInt(0) // 3 billion
		fBalance.Add(fBalance, big.NewInt(0*1000*1000*1000))
		fBalance.Mul(fBalance, big.NewInt(1000000000000000000))
		genesis.Alloc[common.HexToAddress(common.FoudationAddr)] = core.GenesisAccount{
			Balance: fBalance,
			Code:    code,
			Storage: storage,
		}

		// Block Signers Smart Contract
		blockSignerAddress, _, err := blockSignerContract.DeployBlockSigner(transactOpts, contractBackend, big.NewInt(int64(epochNumber)))
		if err != nil {
			fmt.Println("Can't deploy root registry")
		}
		contractBackend.Commit()

		code, _ = contractBackend.CodeAt(ctx, blockSignerAddress, nil)
		storage = make(map[common.Hash]common.Hash)
		contractBackend.ForEachStorageAt(ctx, blockSignerAddress, nil, f)
		genesis.Alloc[common.HexToAddress(common.BlockSigners)] = core.GenesisAccount{
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
		genesis.Alloc[common.HexToAddress(common.RandomizeSMC)] = core.GenesisAccount{
			Balance: big.NewInt(0),
			Code:    code,
			Storage: storage,
		}

		fmt.Println()
		fmt.Println("Which accounts are allowed to confirm in Team MultiSignWallet?")
		var teams []common.Address
		for {
			if address := w.readAddress(); address != nil {
				teams = append(teams, *address)
				continue
			}
			if len(teams) > 0 {
				break
			}
		}
		fmt.Println()
		fmt.Println("How many require for confirm tx in Team MultiSignWallet? (default = 2)")
		required = int64(w.readDefaultInt(2))

		// MultiSigWallet.
		multiSignWalletTeamAddr, _, err := multiSignWalletContract.DeployMultiSigWallet(transactOpts, contractBackend, teams, big.NewInt(required))
		if err != nil {
			fmt.Println("Can't deploy MultiSignWallet SMC")
		}
		contractBackend.Commit()
		code, _ = contractBackend.CodeAt(ctx, multiSignWalletTeamAddr, nil)
		storage = make(map[common.Hash]common.Hash)
		contractBackend.ForEachStorageAt(ctx, multiSignWalletTeamAddr, nil, f)
		// Team balance.
		balance := big.NewInt(0) // 20 billion
		balance.Add(balance, big.NewInt(30*1000*1000))
		balance.Mul(balance, big.NewInt(1000000000000000000))
		subBalance := big.NewInt(0) // i * 50k
		subBalance.Add(subBalance, big.NewInt(int64(len(signers))*10*1000*1000))
		subBalance.Mul(subBalance, big.NewInt(1000000000000000000))
		balance.Sub(balance, subBalance) // 12m - i * 50k
		genesis.Alloc[common.HexToAddress(common.TeamAddr)] = core.GenesisAccount{
			Balance: balance,
			Code:    code,
			Storage: storage,
		}

		fmt.Println()
		fmt.Println("What is swap wallet address for fund 37.47Billion XDC?")
		swapAddr := *w.readAddress()
		baseBalance := big.NewInt(0) // 14.5Billion
		baseBalance.Add(baseBalance, big.NewInt(3747*1000*1000*10))
		baseBalance.Mul(baseBalance, big.NewInt(1000000000000000000))
		genesis.Alloc[swapAddr] = core.GenesisAccount{
			Balance: baseBalance,
		}

	default:
		log.Crit("Invalid consensus engine choice", "choice", choice)
	}
	// Consensus all set, just ask for initial funds and go
	fmt.Println()
	fmt.Println("Which accounts should be pre-funded? (advisable at least one)")
	for {
		// Read the address of the account to fund
		if address := w.readAddress(); address != nil {
			genesis.Alloc[*address] = core.GenesisAccount{
				Balance: new(big.Int).Lsh(big.NewInt(1), 256-7), // 2^256 / 128 (allow many pre-funds without balance overflows)
			}
			continue
		}
		break
	}
	// Add a batch of precompile balances to avoid them getting deleted
	for i := int64(0); i < 2; i++ {
		genesis.Alloc[common.BigToAddress(big.NewInt(i))] = core.GenesisAccount{Balance: big.NewInt(0)}
	}
	// Query the user for some custom extras
	fmt.Println()
	fmt.Println("Specify your chain/network ID if you want an explicit one (default = random)")
	genesis.Config.ChainID = new(big.Int).SetUint64(uint64(w.readDefaultInt(rand.Intn(65536))))

	// All done, store the genesis and flush to disk
	log.Info("Configured new genesis block")

	w.conf.Genesis = genesis
	w.conf.flush()
}

// importGenesis imports a Geth genesis spec into puppeth.
func (w *wizard) importGenesis() {
	// Request the genesis JSON spec URL from the user
	fmt.Println()
	fmt.Println("Where's the genesis file? (local file or http/https url)")
	url := w.readURL()

	// Convert the various allowed URLs to a reader stream
	var reader io.Reader

	switch url.Scheme {
	case "http", "https":
		// Remote web URL, retrieve it via an HTTP client
		res, err := http.Get(url.String())
		if err != nil {
			log.Error("Failed to retrieve remote genesis", "err", err)
			return
		}
		defer res.Body.Close()
		reader = res.Body

	case "":
		// Schemaless URL, interpret as a local file
		file, err := os.Open(url.String())
		if err != nil {
			log.Error("Failed to open local genesis", "err", err)
			return
		}
		defer file.Close()
		reader = file

	default:
		log.Error("Unsupported genesis URL scheme", "scheme", url.Scheme)
		return
	}
	// Parse the genesis file and inject it successful
	var genesis core.Genesis
	if err := json.NewDecoder(reader).Decode(&genesis); err != nil {
		log.Error("Invalid genesis spec: %v", err)
		return
	}
	log.Info("Imported genesis block")

	w.conf.Genesis = &genesis
	w.conf.flush()
}

// manageGenesis permits the modification of chain configuration parameters in
// a genesis config and the export of the entire genesis spec.
func (w *wizard) manageGenesis() {
	// Figure out whether to modify or export the genesis
	fmt.Println()
	fmt.Println(" 1. Modify existing fork rules")
	fmt.Println(" 2. Export genesis configurations")
	fmt.Println(" 3. Remove genesis configuration")

	choice := w.read()
	switch choice {
	case "1":
		// Fork rule updating requested, iterate over each fork
		fmt.Println()
		fmt.Printf("Which block should Homestead come into effect? (default = %v)\n", w.conf.Genesis.Config.HomesteadBlock)
		w.conf.Genesis.Config.HomesteadBlock = w.readDefaultBigInt(w.conf.Genesis.Config.HomesteadBlock)

		fmt.Println()
		fmt.Printf("Which block should EIP150 (Tangerine Whistle) come into effect? (default = %v)\n", w.conf.Genesis.Config.EIP150Block)
		w.conf.Genesis.Config.EIP150Block = w.readDefaultBigInt(w.conf.Genesis.Config.EIP150Block)

		fmt.Println()
		fmt.Printf("Which block should EIP155 (Spurious Dragon) come into effect? (default = %v)\n", w.conf.Genesis.Config.EIP155Block)
		w.conf.Genesis.Config.EIP155Block = w.readDefaultBigInt(w.conf.Genesis.Config.EIP155Block)

		fmt.Println()
		fmt.Printf("Which block should EIP158/161 (also Spurious Dragon) come into effect? (default = %v)\n", w.conf.Genesis.Config.EIP158Block)
		w.conf.Genesis.Config.EIP158Block = w.readDefaultBigInt(w.conf.Genesis.Config.EIP158Block)

		fmt.Println()
		fmt.Printf("Which block should Byzantium come into effect? (default = %v)\n", w.conf.Genesis.Config.ByzantiumBlock)
		w.conf.Genesis.Config.ByzantiumBlock = w.readDefaultBigInt(w.conf.Genesis.Config.ByzantiumBlock)

		fmt.Println()
		fmt.Printf("Which block should Constantinople come into effect? (default = %v)\n", w.conf.Genesis.Config.ConstantinopleBlock)
		w.conf.Genesis.Config.ConstantinopleBlock = w.readDefaultBigInt(w.conf.Genesis.Config.ConstantinopleBlock)
		if w.conf.Genesis.Config.PetersburgBlock == nil {
			w.conf.Genesis.Config.PetersburgBlock = w.conf.Genesis.Config.ConstantinopleBlock
		}
		fmt.Println()
		fmt.Printf("Which block should Constantinople-Fix (remove EIP-1283) come into effect? (default = %v)\n", w.conf.Genesis.Config.PetersburgBlock)
		w.conf.Genesis.Config.PetersburgBlock = w.readDefaultBigInt(w.conf.Genesis.Config.PetersburgBlock)

		out, _ := json.MarshalIndent(w.conf.Genesis.Config, "", "  ")
		fmt.Printf("Chain configuration updated:\n\n%s\n", out)

		w.conf.flush()

	case "2":
		// Save whatever genesis configuration we currently have
		fmt.Println()
		fmt.Printf("Which folder to save the genesis specs into? (default = current)\n")
		fmt.Printf("  Will create %s.json, %s-aleth.json, %s-harmony.json, %s-parity.json\n", w.network, w.network, w.network, w.network)

		folder := w.readDefaultString(".")
		if err := os.MkdirAll(folder, 0755); err != nil {
			log.Error("Failed to create spec folder", "folder", folder, "err", err)
			return
		}
		out, _ := json.MarshalIndent(w.conf.Genesis, "", "  ")

		// Export the native genesis spec used by puppeth and Geth
		gethJson := filepath.Join(folder, fmt.Sprintf("%s.json", w.network))
		if err := ioutil.WriteFile((gethJson), out, 0644); err != nil {
			log.Error("Failed to save genesis file", "err", err)
			return
		}
		log.Info("Saved native genesis chain spec", "path", gethJson)

		// Export the genesis spec used by Aleth (formerly C++ Ethereum)
		if spec, err := newAlethGenesisSpec(w.network, w.conf.Genesis); err != nil {
			log.Error("Failed to create Aleth chain spec", "err", err)
		} else {
			saveGenesis(folder, w.network, "aleth", spec)
		}
		// Export the genesis spec used by Parity
		if spec, err := newParityChainSpec(w.network, w.conf.Genesis, []string{}); err != nil {
			log.Error("Failed to create Parity chain spec", "err", err)
		} else {
			saveGenesis(folder, w.network, "parity", spec)
		}
		// Export the genesis spec used by Harmony (formerly EthereumJ
		saveGenesis(folder, w.network, "harmony", w.conf.Genesis)

	case "3":
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
		return
	}
}

// saveGenesis JSON encodes an arbitrary genesis spec into a pre-defined file.
func saveGenesis(folder, network, client string, spec interface{}) {
	path := filepath.Join(folder, fmt.Sprintf("%s-%s.json", network, client))

	out, _ := json.Marshal(spec)
	if err := ioutil.WriteFile(path, out, 0644); err != nil {
		log.Error("Failed to save genesis file", "client", client, "err", err)
		return
	}
	log.Info("Saved genesis chain spec", "client", client, "path", path)
}
