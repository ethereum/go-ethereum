package bor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// Volatile state for each Validator
// NOTE: The ProposerPriority is not included in Validator.Hash();
// make sure to update that method if changes are made here
type Validator struct {
	Address          common.Address `json:"address"`
	VotingPower      int64          `json:"voting_power"`
	ProposerPriority int64          `json:"proposer_priority"`
}

func NewValidator(address common.Address, votingPower int64) *Validator {
	return &Validator{
		Address:          address,
		VotingPower:      votingPower,
		ProposerPriority: 0,
	}
}

// Creates a new copy of the validator so we can mutate ProposerPriority.
// Panics if the validator is nil.
func (v *Validator) Copy() *Validator {
	vCopy := *v
	return &vCopy
}

// Returns the one with higher ProposerPriority.
func (v *Validator) CompareProposerPriority(other *Validator) *Validator {
	if v == nil {
		return other
	}
	if v.ProposerPriority > other.ProposerPriority {
		return v
	} else if v.ProposerPriority < other.ProposerPriority {
		return other
	} else {
		result := bytes.Compare(v.Address.Bytes(), other.Address.Bytes())
		if result < 0 {
			return v
		} else if result > 0 {
			return other
		} else {
			panic("Cannot compare identical validators")
			return nil
		}
	}
}

func (v *Validator) String() string {
	if v == nil {
		return "nil-Validator"
	}
	return fmt.Sprintf("Validator{%v Power:%v Priority:%v}",
		v.Address.Hex(),
		v.VotingPower,
		v.ProposerPriority)
}

// ValidatorListString returns a prettified validator list for logging purposes.
func ValidatorListString(vals []*Validator) string {
	chunks := make([]string, len(vals))
	for i, val := range vals {
		chunks[i] = fmt.Sprintf("%s:%d", val.Address, val.VotingPower)
	}

	return strings.Join(chunks, ",")
}

// Bytes computes the unique encoding of a validator with a given voting power.
// These are the bytes that gets hashed in consensus. It excludes address
// as its redundant with the pubkey. This also excludes ProposerPriority
// which changes every round.
func (v *Validator) Bytes() []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return b
	}
	return nil
}
