package bor

import (
	"fmt"

	"github.com/maticnetwork/bor/common"
)

// Will include any new bor consensus errors here in an attempt to make error messages more descriptive

// ProposerNotFoundError is returned if the given proposer address is not present in the validator set
type ProposerNotFoundError struct {
	Address common.Address
}

func (e *ProposerNotFoundError) Error() string {
	return fmt.Sprintf("Proposer: %s not found", e.Address.Hex())
}

// SignerNotFoundError is returned when the signer address is not present in the validator set
type SignerNotFoundError struct {
	Address common.Address
}

func (e *SignerNotFoundError) Error() string {
	return fmt.Sprintf("Signer: %s not found", e.Address.Hex())
}

// TotalVotingPowerExceededError is returned when the maximum allowed total voting power is exceeded
type TotalVotingPowerExceededError struct {
	Sum        int64
	Validators []*Validator
}

func (e *TotalVotingPowerExceededError) Error() string {
	return fmt.Sprintf(
		"Total voting power should be guarded to not exceed %v; got: %v; for validator set: %v",
		MaxTotalVotingPower,
		e.Sum,
		e.Validators,
	)
}

type InvalidStartEndBlockError struct {
	Start         uint64
	End           uint64
	CurrentHeader uint64
}

func (e *InvalidStartEndBlockError) Error() string {
	return fmt.Sprintf(
		"Invalid parameters start: %d and end block: %d params",
		e.Start,
		e.End,
	)
}

type MaxCheckpointLengthExceededError struct {
	Start uint64
	End   uint64
}

func (e *MaxCheckpointLengthExceededError) Error() string {
	return fmt.Sprintf(
		"Start: %d and end block: %d exceed max allowed checkpoint length: %d",
		e.Start,
		e.End,
		MaxCheckpointLength,
	)
}
