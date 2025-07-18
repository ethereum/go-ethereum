package params

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params/forks"
)

// ChainID
var ChainID = Define(T[*big.Int]{
	Name:     "chainId",
	Optional: false,
	Validate: validateChainID,
})

func validateChainID(v *big.Int, cfg *Config2) error {
	if v.Sign() <= 0 {
		return fmt.Errorf("invalid chainID value %v", v)
	}
	return nil
}

// DAOForkSupport is the chain parameter that configures the DAO fork.
// true=supports or false=opposes the fork.
// The default value is true.
var DAOForkSupport = Define(T[bool]{
	Optional: true,
	Default:  true,
})

// TerminalTotalDifficulty (TTD) is the total difficulty value where
var TerminalTotalDifficulty = Define(T[*big.Int]{
	Name:     "terminalTotalDifficulty",
	Optional: true,
})

// DepositContractAddress configures the location of the deposit contract.
var DepositContractAddress = Define(T[common.Address]{
	Name:     "depositContractAddress",
	Optional: true,
})

// This configures the EIP-4844 parameters across forks.
// There must be an entry for each fork
var BlobSchedule = Define(T[map[forks.Fork]BlobConfig]{
	Name:     "blobSchedule",
	Optional: true,
	Validate: validateBlobSchedule,
})

func validateBlobSchedule(schedule map[forks.Fork]BlobConfig, cfg *Config2) error {
	// Check that all forks with blobs explicitly define the blob schedule configuration.
	for _, f := range forks.CanonOrder {
		if f.HasBlobs() {
			bcfg, defined := schedule[f]
			if cfg.Scheduled(f) && !defined {
				return fmt.Errorf("invalid chain configuration: missing entry for fork %q in blobSchedule", f)
			}
			if defined {
				if err := bcfg.validate(); err != nil {
					return fmt.Errorf("invalid chain configuration in blobSchedule for fork %q: %v", f, err)
				}
			}
		}
	}
	return nil
}
