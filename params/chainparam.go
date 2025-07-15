package params

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params/forks"
)

// ChainID
type ChainID big.Int

func NewChainID(input string) *ChainID {
	b, ok := new(big.Int).SetString(input, 0)
	if !ok {
		panic("invalid chainID: " + input)
	}
	return (*ChainID)(b)
}

func (v *ChainID) MarshalText() ([]byte, error) {
	return (*big.Int)(v).MarshalText()
}

func (v *ChainID) UnmarshalText(input []byte) error {
	return (*big.Int)(v).UnmarshalText(input)
}

func (v *ChainID) BigInt() *big.Int {
	return (*big.Int)(v)
}

func (v *ChainID) Validate(cfg *Config2) error {
	b := (*big.Int)(v)
	if b.Sign() <= 0 {
		return fmt.Errorf("invalid chainID value %v", b)
	}
	return nil
}

// TerminalTotalDifficulty (TTD) is the total difficulty value where
type TerminalTotalDifficulty big.Int

func NewTerminalTotalDifficulty(input string) *TerminalTotalDifficulty {
	b, ok := new(big.Int).SetString(input, 0)
	if !ok {
		panic("invalid terminal total difficulty: " + input)
	}
	return (*TerminalTotalDifficulty)(b)
}

func (v *TerminalTotalDifficulty) MarshalText() ([]byte, error) {
	return (*big.Int)(v).MarshalText()
}

func (v *TerminalTotalDifficulty) UnmarshalText(input []byte) error {
	return (*big.Int)(v).UnmarshalText(input)
}

func (v *TerminalTotalDifficulty) BigInt() *big.Int {
	return (*big.Int)(v)
}

func (v *TerminalTotalDifficulty) Validate(cfg *Config2) error {
	return nil
}

// DepositContractAddress configures the location of the deposit contract.
type DepositContractAddress common.Address

func (v DepositContractAddress) Validate(cfg *Config2) error {
	return nil
}

// This configures the EIP-4844 parameters across forks.
// There must be an entry for each fork
type BlobSchedule map[forks.Fork]BlobConfig

// Validate checks that all forks with blobs explicitly define the blob schedule configuration.
func (v BlobSchedule) Validate(cfg *Config2) error {
	for _, f := range forks.CanonOrder {
		if f.HasBlobs() {
			schedule := Get[BlobSchedule](cfg)
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

// DAOForkSupport is the chain parameter that configures the DAO fork.
// true=supports or false=opposes the fork.
// The default value is true.
type DAOForkSupport bool

func (v DAOForkSupport) Validate(cfg *Config2) error {
	return nil
}

func init() {
	Define(Parameter[*ChainID]{
		Name:     "chainId",
		Optional: false,
	})
	Define(Parameter[*TerminalTotalDifficulty]{
		Name:     "terminalTotalDifficulty",
		Optional: false,
	})
	Define(Parameter[DAOForkSupport]{
		Name:     "daoForkSupport",
		Optional: true,
		Default:  true,
	})
	Define(Parameter[BlobSchedule]{
		Name:     "blobSchedule",
		Optional: true,
	})
	Define(Parameter[DepositContractAddress]{
		Name:     "depositContractAddress",
		Optional: true,
	})
}
