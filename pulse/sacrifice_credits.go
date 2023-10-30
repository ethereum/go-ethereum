package pulse

import (
	_ "embed"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// see https://gitlab.com/pulsechaincom/compressed-allocations/-/tags/Mainnet
//
//go:embed sacrifice_credits_mainnet.bin
var mainnetRawCredits []byte

// see https://gitlab.com/pulsechaincom/compressed-allocations/-/tags/Testnet-V4
//
//go:embed sacrifice_credits_testnet_v4.bin
var testnetV4RawCredits []byte

// Applies the sacrifice credits for the PrimordialPulse fork.
func applySacrificeCredits(state *state.StateDB, treasury *params.Treasury, chainID *big.Int) {
	rawCredits := mainnetRawCredits
	if chainID.Cmp(params.PulseChainTestnetV4Config.ChainID) == 0 {
		rawCredits = testnetV4RawCredits
	}

	if treasury != nil {
		log.Info("Applying PrimordialPulse treasury allocation 💸")
		state.AddBalance(common.HexToAddress(treasury.Addr), (*big.Int)(treasury.Balance))
	}

	log.Info("Applying PrimordialPulse sacrifice credits 💸")
	for ptr := 0; ptr < len(rawCredits); {
		byteCount := int(rawCredits[ptr])
		ptr++

		record := rawCredits[ptr : ptr+byteCount]
		ptr += byteCount

		addr := common.BytesToAddress(record[:20])
		credit := new(big.Int).SetBytes(record[20:])
		state.AddBalance(addr, credit)
	}

	log.Info("Finished applying PrimordialPulse sacrifice credits 🤑")
}
