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
	Name:     "daoForkSupport",
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

// validateBlobSchedule verifies that all forks after cancun explicitly define a blob
// schedule configuration.
func validateBlobSchedule(schedule map[forks.Fork]BlobConfig, cfg *Config2) error {
	for f := range forks.All() {
		if _, defined := schedule[f]; f.BlockBased() && defined {
			return fmt.Errorf("contains fork %q with block-number based scheduling", f.ConfigName())
		}
		if cfg.Scheduled(f) && f.After(forks.Cancun) {
			bcfg, defined := schedule[f]
			if !defined {
				return fmt.Errorf("missing entry for fork %q", f.ConfigName())
			} else {
				if err := bcfg.validate(); err != nil {
					return fmt.Errorf("invalid blob config for fork %q: %v", f.ConfigName(), err)
				}
			}
		}
	}
	return nil
}
