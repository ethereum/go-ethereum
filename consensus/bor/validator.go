package bor

import (
	"bytes"
	// "encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/maticnetwork/bor/common"
)

// Validator represets Volatile state for each Validator
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

// Copy creates a new copy of the validator so we can mutate ProposerPriority.
// Panics if the validator is nil.
func (v *Validator) Copy() *Validator {
	vCopy := *v
	return &vCopy
}

// Cmp returns the one validator with a higher ProposerPriority.
// If ProposerPriority is same, it returns the validator with lexicographically smaller address
func (v *Validator) Cmp(other *Validator) *Validator {
	// if both of v and other are nil, nil will be returned and that could possibly lead to nil pointer dereference bubbling up the stack
	if v == nil {
		return other
	}
	if other == nil {
		return v
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
