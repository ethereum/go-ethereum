package bor

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/maticnetwork/bor/common"
)

// Validator represets Volatile state for each Validator
// NOTE: The ProposerPriority is not included in Validator.Hash();
// make sure to update that method if changes are made here
type Validator struct {
	ID               uint64         `json:"ID"`
	Address          common.Address `json:"signer"`
	VotingPower      int64          `json:"power"`
	ProposerPriority int64          `json:"accum"`
}

// NewValidator creates new validator
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

// HeaderBytes return header bytes
func (v *Validator) HeaderBytes() []byte {
	result := make([]byte, 40)
	copy(result[:20], v.Address.Bytes())
	copy(result[20:], v.PowerBytes())
	return result
}

// PowerBytes return power bytes
func (v *Validator) PowerBytes() []byte {
	powerBytes := big.NewInt(0).SetInt64(v.VotingPower).Bytes()
	result := make([]byte, 20)
	copy(result[20-len(powerBytes):], powerBytes)
	return result
}

// MinimalVal returns block number of last validator update
func (v *Validator) MinimalVal() MinimalVal {
	return MinimalVal{
		ID:          v.ID,
		VotingPower: uint64(v.VotingPower),
		Signer:      v.Address,
	}
}

// ParseValidators returns validator set bytes
func ParseValidators(validatorsBytes []byte) ([]*Validator, error) {
	if len(validatorsBytes)%40 != 0 {
		return nil, errors.New("Invalid validators bytes")
	}

	result := make([]*Validator, len(validatorsBytes)/40)
	for i := 0; i < len(validatorsBytes); i += 40 {
		address := make([]byte, 20)
		power := make([]byte, 20)

		copy(address, validatorsBytes[i:i+20])
		copy(power, validatorsBytes[i+20:i+40])

		result[i/40] = NewValidator(common.BytesToAddress(address), big.NewInt(0).SetBytes(power).Int64())
	}

	return result, nil
}

// ---

// MinimalVal is the minimal validator representation
// Used to send validator information to bor validator contract
type MinimalVal struct {
	ID          uint64         `json:"ID"`
	VotingPower uint64         `json:"power"` // TODO add 10^-18 here so that we dont overflow easily
	Signer      common.Address `json:"signer"`
}

// SortMinimalValByAddress sorts validators
func SortMinimalValByAddress(a []MinimalVal) []MinimalVal {
	sort.Slice(a, func(i, j int) bool {
		return bytes.Compare(a[i].Signer.Bytes(), a[j].Signer.Bytes()) < 0
	})
	return a
}

// ValidatorsToMinimalValidators converts array of validators to minimal validators
func ValidatorsToMinimalValidators(vals []Validator) (minVals []MinimalVal) {
	for _, val := range vals {
		minVals = append(minVals, val.MinimalVal())
	}
	return
}
