package params

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params/forks"
)

// ChainID
type ChainID big.Int

// TerminalTotalDifficulty (TTD) is the total difficulty value where
type TerminalTotalDifficulty big.Int

func (v *TerminalTotalDifficulty) MarshalText() ([]byte, error) {
	return (*big.Int)(v).MarshalText()
}

func (v *TerminalTotalDifficulty) UnmarshalText(input []byte) error {
	return (*big.Int)(v).UnmarshalText(input)
}

// DepositContractAddress configures the location of the deposit contract.
type DepositContractAddress common.Address

// This configures the EIP-4844 parameters across forks.
// There must be an entry for each fork
type BlobSchedule map[forks.Fork]BlobConfig

// DAOForkSupport is the chain parameter that configures the DAO fork.
// true=supports or false=opposes the fork.
// The default value is true.
type DAOForkSupport bool

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
