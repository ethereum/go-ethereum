package chains

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// GetDeveloperChain returns the developer mode configs.
func GetDeveloperChain(period uint64, gasLimitt uint64, faucet common.Address) *Chain {
	// Override the default period to the user requested one
	config := *params.AllCliqueProtocolChanges
	config.Clique = &params.CliqueConfig{
		Period: period,
		Epoch:  config.Clique.Epoch,
	}

	// Assemble and return the chain having genesis with the
	// precompiles and faucet pre-funded
	return &Chain{
		Hash:      common.Hash{},
		NetworkId: 1337,
		Genesis: &core.Genesis{
			Config:     &config,
			ExtraData:  append(append(make([]byte, 32), faucet[:]...), make([]byte, crypto.SignatureLength)...),
			GasLimit:   gasLimitt,
			BaseFee:    big.NewInt(params.InitialBaseFee),
			Difficulty: big.NewInt(1),
			Alloc: map[common.Address]core.GenesisAccount{
				common.BytesToAddress([]byte{1}): {Balance: big.NewInt(1)}, // ECRecover
				common.BytesToAddress([]byte{2}): {Balance: big.NewInt(1)}, // SHA256
				common.BytesToAddress([]byte{3}): {Balance: big.NewInt(1)}, // RIPEMD
				common.BytesToAddress([]byte{4}): {Balance: big.NewInt(1)}, // Identity
				common.BytesToAddress([]byte{5}): {Balance: big.NewInt(1)}, // ModExp
				common.BytesToAddress([]byte{6}): {Balance: big.NewInt(1)}, // ECAdd
				common.BytesToAddress([]byte{7}): {Balance: big.NewInt(1)}, // ECScalarMul
				common.BytesToAddress([]byte{8}): {Balance: big.NewInt(1)}, // ECPairing
				common.BytesToAddress([]byte{9}): {Balance: big.NewInt(1)}, // BLAKE2b
				faucet:                           {Balance: new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(9))},
			},
		},
		Bootnodes: []string{},
	}
}
