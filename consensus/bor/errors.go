package bor

import (
	"fmt"
	"time"
)

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

// MismatchingValidatorsError is returned if a last block in sprint contains a
// list of validators different from the one that local node calculated
type MismatchingValidatorsError struct {
	Number             uint64
	ValidatorSetSnap   []byte
	ValidatorSetHeader []byte
}

func (e *MismatchingValidatorsError) Error() string {
	return fmt.Sprintf(
		"Mismatching validators at block %d\nValidatorBytes from snapshot: 0x%x\nValidatorBytes in Header: 0x%x\n",
		e.Number,
		e.ValidatorSetSnap,
		e.ValidatorSetHeader,
	)
}

type BlockTooSoonError struct {
	Number     uint64
	Succession int
}

func (e *BlockTooSoonError) Error() string {
	return fmt.Sprintf(
		"Block %d was created too soon. Signer turn-ness number is %d\n",
		e.Number,
		e.Succession,
	)
}

// UnauthorizedProposerError is returned if a header is [being] signed by an unauthorized entity.
type UnauthorizedProposerError struct {
	Number   uint64
	Proposer []byte
}

func (e *UnauthorizedProposerError) Error() string {
	return fmt.Sprintf(
		"Proposer 0x%x is not a part of the producer set at block %d",
		e.Proposer,
		e.Number,
	)
}

// UnauthorizedSignerError is returned if a header is [being] signed by an unauthorized entity.
type UnauthorizedSignerError struct {
	Number uint64
	Signer []byte
}

func (e *UnauthorizedSignerError) Error() string {
	return fmt.Sprintf(
		"Signer 0x%x is not a part of the producer set at block %d",
		e.Signer,
		e.Number,
	)
}

// WrongDifficultyError is returned if the difficulty of a block doesn't match the
// turn of the signer.
type WrongDifficultyError struct {
	Number   uint64
	Expected uint64
	Actual   uint64
	Signer   []byte
}

func (e *WrongDifficultyError) Error() string {
	return fmt.Sprintf(
		"Wrong difficulty at block %d, expected: %d, actual %d. Signer was %x\n",
		e.Number,
		e.Expected,
		e.Actual,
		e.Signer,
	)
}

type InvalidStateReceivedError struct {
	Number      uint64
	LastStateID uint64
	To          *time.Time
	Event       *EventRecordWithTime
}

func (e *InvalidStateReceivedError) Error() string {
	return fmt.Sprintf(
		"Received invalid event %s at block %d. Requested events until %s. Last state id was %d",
		e.Event,
		e.Number,
		e.To.Format(time.RFC3339),
		e.LastStateID,
	)
}
