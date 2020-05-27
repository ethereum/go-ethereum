package bor

// Tendermint leader selection algorithm

import (
	"bytes"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strings"

	"github.com/maticnetwork/bor/common"
	"github.com/maticnetwork/bor/log"
)

// MaxTotalVotingPower - the maximum allowed total voting power.
// It needs to be sufficiently small to, in all cases:
// 1. prevent clipping in incrementProposerPriority()
// 2. let (diff+diffMax-1) not overflow in IncrementProposerPriority()
// (Proof of 1 is tricky, left to the reader).
// It could be higher, but this is sufficiently large for our purposes,
// and leaves room for defensive purposes.
// PriorityWindowSizeFactor - is a constant that when multiplied with the total voting power gives
// the maximum allowed distance between validator priorities.

const (
	MaxTotalVotingPower      = int64(math.MaxInt64) / 8
	PriorityWindowSizeFactor = 2
)

// ValidatorSet represent a set of *Validator at a given height.
// The validators can be fetched by address or index.
// The index is in order of .Address, so the indices are fixed
// for all rounds of a given blockchain height - ie. the validators
// are sorted by their address.
// On the other hand, the .ProposerPriority of each validator and
// the designated .GetProposer() of a set changes every round,
// upon calling .IncrementProposerPriority().
// NOTE: Not goroutine-safe.
// NOTE: All get/set to validators should copy the value for safety.
type ValidatorSet struct {
	// NOTE: persisted via reflect, must be exported.
	Validators []*Validator `json:"validators"`
	Proposer   *Validator   `json:"proposer"`

	// cached (unexported)
	totalVotingPower int64
}

// NewValidatorSet initializes a ValidatorSet by copying over the
// values from `valz`, a list of Validators. If valz is nil or empty,
// the new ValidatorSet will have an empty list of Validators.
// The addresses of validators in `valz` must be unique otherwise the
// function panics.
func NewValidatorSet(valz []*Validator) *ValidatorSet {
	vals := &ValidatorSet{}
	err := vals.updateWithChangeSet(valz, false)
	if err != nil {
		panic(fmt.Sprintf("cannot create validator set: %s", err))
	}
	if len(valz) > 0 {
		vals.IncrementProposerPriority(1)
	}
	return vals
}

// Nil or empty validator sets are invalid.
func (vals *ValidatorSet) IsNilOrEmpty() bool {
	return vals == nil || len(vals.Validators) == 0
}

// Increment ProposerPriority and update the proposer on a copy, and return it.
func (vals *ValidatorSet) CopyIncrementProposerPriority(times int) *ValidatorSet {
	copy := vals.Copy()
	copy.IncrementProposerPriority(times)
	return copy
}

// IncrementProposerPriority increments ProposerPriority of each validator and updates the
// proposer. Panics if validator set is empty.
// `times` must be positive.
func (vals *ValidatorSet) IncrementProposerPriority(times int) {
	if vals.IsNilOrEmpty() {
		panic("empty validator set")
	}
	if times <= 0 {
		panic("Cannot call IncrementProposerPriority with non-positive times")
	}

	// Cap the difference between priorities to be proportional to 2*totalPower by
	// re-normalizing priorities, i.e., rescale all priorities by multiplying with:
	//  2*totalVotingPower/(maxPriority - minPriority)
	diffMax := PriorityWindowSizeFactor * vals.TotalVotingPower()
	vals.RescalePriorities(diffMax)
	vals.shiftByAvgProposerPriority()

	var proposer *Validator
	// Call IncrementProposerPriority(1) times times.
	for i := 0; i < times; i++ {
		proposer = vals.incrementProposerPriority()
	}

	vals.Proposer = proposer
}

func (vals *ValidatorSet) RescalePriorities(diffMax int64) {
	if vals.IsNilOrEmpty() {
		panic("empty validator set")
	}
	// NOTE: This check is merely a sanity check which could be
	// removed if all tests would init. voting power appropriately;
	// i.e. diffMax should always be > 0
	if diffMax <= 0 {
		return
	}

	// Calculating ceil(diff/diffMax):
	// Re-normalization is performed by dividing by an integer for simplicity.
	// NOTE: This may make debugging priority issues easier as well.
	diff := computeMaxMinPriorityDiff(vals)
	ratio := (diff + diffMax - 1) / diffMax
	if diff > diffMax {
		for _, val := range vals.Validators {
			val.ProposerPriority = val.ProposerPriority / ratio
		}
	}
}

func (vals *ValidatorSet) incrementProposerPriority() *Validator {
	for _, val := range vals.Validators {
		// Check for overflow for sum.
		newPrio := safeAddClip(val.ProposerPriority, val.VotingPower)
		val.ProposerPriority = newPrio
	}
	// Decrement the validator with most ProposerPriority.
	mostest := vals.getValWithMostPriority()
	// Mind the underflow.
	mostest.ProposerPriority = safeSubClip(mostest.ProposerPriority, vals.TotalVotingPower())

	return mostest
}

// Should not be called on an empty validator set.
func (vals *ValidatorSet) computeAvgProposerPriority() int64 {
	n := int64(len(vals.Validators))
	sum := big.NewInt(0)
	for _, val := range vals.Validators {
		sum.Add(sum, big.NewInt(val.ProposerPriority))
	}
	avg := sum.Div(sum, big.NewInt(n))
	if avg.IsInt64() {
		return avg.Int64()
	}

	// This should never happen: each val.ProposerPriority is in bounds of int64.
	panic(fmt.Sprintf("Cannot represent avg ProposerPriority as an int64 %v", avg))
}

// Compute the difference between the max and min ProposerPriority of that set.
func computeMaxMinPriorityDiff(vals *ValidatorSet) int64 {
	if vals.IsNilOrEmpty() {
		panic("empty validator set")
	}
	max := int64(math.MinInt64)
	min := int64(math.MaxInt64)
	for _, v := range vals.Validators {
		if v.ProposerPriority < min {
			min = v.ProposerPriority
		}
		if v.ProposerPriority > max {
			max = v.ProposerPriority
		}
	}
	diff := max - min
	if diff < 0 {
		return -1 * diff
	} else {
		return diff
	}
}

func (vals *ValidatorSet) getValWithMostPriority() *Validator {
	var res *Validator
	for _, val := range vals.Validators {
		res = res.Cmp(val)
	}
	return res
}

func (vals *ValidatorSet) shiftByAvgProposerPriority() {
	if vals.IsNilOrEmpty() {
		panic("empty validator set")
	}
	avgProposerPriority := vals.computeAvgProposerPriority()
	for _, val := range vals.Validators {
		val.ProposerPriority = safeSubClip(val.ProposerPriority, avgProposerPriority)
	}
}

// Makes a copy of the validator list.
func validatorListCopy(valsList []*Validator) []*Validator {
	if valsList == nil {
		return nil
	}
	valsCopy := make([]*Validator, len(valsList))
	for i, val := range valsList {
		valsCopy[i] = val.Copy()
	}
	return valsCopy
}

// Copy each validator into a new ValidatorSet.
func (vals *ValidatorSet) Copy() *ValidatorSet {
	return &ValidatorSet{
		Validators:       validatorListCopy(vals.Validators),
		Proposer:         vals.Proposer,
		totalVotingPower: vals.totalVotingPower,
	}
}

// HasAddress returns true if address given is in the validator set, false -
// otherwise.
func (vals *ValidatorSet) HasAddress(address []byte) bool {
	idx := sort.Search(len(vals.Validators), func(i int) bool {
		return bytes.Compare(address, vals.Validators[i].Address.Bytes()) <= 0
	})
	return idx < len(vals.Validators) && bytes.Equal(vals.Validators[idx].Address.Bytes(), address)
}

// GetByAddress returns an index of the validator with address and validator
// itself if found. Otherwise, -1 and nil are returned.
func (vals *ValidatorSet) GetByAddress(address common.Address) (index int, val *Validator) {
	idx := sort.Search(len(vals.Validators), func(i int) bool {
		return bytes.Compare(address.Bytes(), vals.Validators[i].Address.Bytes()) <= 0
	})
	if idx < len(vals.Validators) && bytes.Equal(vals.Validators[idx].Address.Bytes(), address.Bytes()) {
		return idx, vals.Validators[idx].Copy()
	}
	return -1, nil
}

// GetByIndex returns the validator's address and validator itself by index.
// It returns nil values if index is less than 0 or greater or equal to
// len(ValidatorSet.Validators).
func (vals *ValidatorSet) GetByIndex(index int) (address []byte, val *Validator) {
	if index < 0 || index >= len(vals.Validators) {
		return nil, nil
	}
	val = vals.Validators[index]
	return val.Address.Bytes(), val.Copy()
}

// Size returns the length of the validator set.
func (vals *ValidatorSet) Size() int {
	return len(vals.Validators)
}

// Force recalculation of the set's total voting power.
func (vals *ValidatorSet) updateTotalVotingPower() error {

	sum := int64(0)
	for _, val := range vals.Validators {
		// mind overflow
		sum = safeAddClip(sum, val.VotingPower)
		if sum > MaxTotalVotingPower {
			return &TotalVotingPowerExceededError{sum, vals.Validators}
		}
	}
	vals.totalVotingPower = sum
	return nil
}

// TotalVotingPower returns the sum of the voting powers of all validators.
// It recomputes the total voting power if required.
func (vals *ValidatorSet) TotalVotingPower() int64 {
	if vals.totalVotingPower == 0 {
		log.Info("invoking updateTotalVotingPower before returning it")
		if err := vals.updateTotalVotingPower(); err != nil {
			// Can/should we do better?
			panic(err)
		}
	}
	return vals.totalVotingPower
}

// GetProposer returns the current proposer. If the validator set is empty, nil
// is returned.
func (vals *ValidatorSet) GetProposer() (proposer *Validator) {
	if len(vals.Validators) == 0 {
		return nil
	}
	if vals.Proposer == nil {
		vals.Proposer = vals.findProposer()
	}
	return vals.Proposer.Copy()
}

func (vals *ValidatorSet) findProposer() *Validator {
	var proposer *Validator
	for _, val := range vals.Validators {
		if proposer == nil || !bytes.Equal(val.Address.Bytes(), proposer.Address.Bytes()) {
			proposer = proposer.Cmp(val)
		}
	}
	return proposer
}

// Hash returns the Merkle root hash build using validators (as leaves) in the
// set.
// func (vals *ValidatorSet) Hash() []byte {
// 	if len(vals.Validators) == 0 {
// 		return nil
// 	}
// 	bzs := make([][]byte, len(vals.Validators))
// 	for i, val := range vals.Validators {
// 		bzs[i] = val.Bytes()
// 	}
// 	return merkle.SimpleHashFromByteSlices(bzs)
// }

// Iterate will run the given function over the set.
func (vals *ValidatorSet) Iterate(fn func(index int, val *Validator) bool) {
	for i, val := range vals.Validators {
		stop := fn(i, val.Copy())
		if stop {
			break
		}
	}
}

// Checks changes against duplicates, splits the changes in updates and removals, sorts them by address.
//
// Returns:
// updates, removals - the sorted lists of updates and removals
// err - non-nil if duplicate entries or entries with negative voting power are seen
//
// No changes are made to 'origChanges'.
func processChanges(origChanges []*Validator) (updates, removals []*Validator, err error) {
	// Make a deep copy of the changes and sort by address.
	changes := validatorListCopy(origChanges)
	sort.Sort(ValidatorsByAddress(changes))

	removals = make([]*Validator, 0, len(changes))
	updates = make([]*Validator, 0, len(changes))
	var prevAddr common.Address

	// Scan changes by address and append valid validators to updates or removals lists.
	for _, valUpdate := range changes {
		if bytes.Equal(valUpdate.Address.Bytes(), prevAddr.Bytes()) {
			err = fmt.Errorf("duplicate entry %v in %v", valUpdate, changes)
			return nil, nil, err
		}
		if valUpdate.VotingPower < 0 {
			err = fmt.Errorf("voting power can't be negative: %v", valUpdate)
			return nil, nil, err
		}
		if valUpdate.VotingPower > MaxTotalVotingPower {
			err = fmt.Errorf("to prevent clipping/ overflow, voting power can't be higher than %v: %v ",
				MaxTotalVotingPower, valUpdate)
			return nil, nil, err
		}
		if valUpdate.VotingPower == 0 {
			removals = append(removals, valUpdate)
		} else {
			updates = append(updates, valUpdate)
		}
		prevAddr = valUpdate.Address
	}
	return updates, removals, err
}

// Verifies a list of updates against a validator set, making sure the allowed
// total voting power would not be exceeded if these updates would be applied to the set.
//
// Returns:
// updatedTotalVotingPower - the new total voting power if these updates would be applied
// numNewValidators - number of new validators
// err - non-nil if the maximum allowed total voting power would be exceeded
//
// 'updates' should be a list of proper validator changes, i.e. they have been verified
// by processChanges for duplicates and invalid values.
// No changes are made to the validator set 'vals'.
func verifyUpdates(updates []*Validator, vals *ValidatorSet) (updatedTotalVotingPower int64, numNewValidators int, err error) {

	updatedTotalVotingPower = vals.TotalVotingPower()

	for _, valUpdate := range updates {
		address := valUpdate.Address
		_, val := vals.GetByAddress(address)
		if val == nil {
			// New validator, add its voting power the the total.
			updatedTotalVotingPower += valUpdate.VotingPower
			numNewValidators++
		} else {
			// Updated validator, add the difference in power to the total.
			updatedTotalVotingPower += valUpdate.VotingPower - val.VotingPower
		}
		overflow := updatedTotalVotingPower > MaxTotalVotingPower
		if overflow {
			err = fmt.Errorf(
				"failed to add/update validator %v, total voting power would exceed the max allowed %v",
				valUpdate, MaxTotalVotingPower)
			return 0, 0, err
		}
	}

	return updatedTotalVotingPower, numNewValidators, nil
}

// Computes the proposer priority for the validators not present in the set based on 'updatedTotalVotingPower'.
// Leaves unchanged the priorities of validators that are changed.
//
// 'updates' parameter must be a list of unique validators to be added or updated.
// No changes are made to the validator set 'vals'.
func computeNewPriorities(updates []*Validator, vals *ValidatorSet, updatedTotalVotingPower int64) {

	for _, valUpdate := range updates {
		address := valUpdate.Address
		_, val := vals.GetByAddress(address)
		if val == nil {
			// add val
			// Set ProposerPriority to -C*totalVotingPower (with C ~= 1.125) to make sure validators can't
			// un-bond and then re-bond to reset their (potentially previously negative) ProposerPriority to zero.
			//
			// Contract: updatedVotingPower < MaxTotalVotingPower to ensure ProposerPriority does
			// not exceed the bounds of int64.
			//
			// Compute ProposerPriority = -1.125*totalVotingPower == -(updatedVotingPower + (updatedVotingPower >> 3)).
			valUpdate.ProposerPriority = -(updatedTotalVotingPower + (updatedTotalVotingPower >> 3))
		} else {
			valUpdate.ProposerPriority = val.ProposerPriority
		}
	}

}

// Merges the vals' validator list with the updates list.
// When two elements with same address are seen, the one from updates is selected.
// Expects updates to be a list of updates sorted by address with no duplicates or errors,
// must have been validated with verifyUpdates() and priorities computed with computeNewPriorities().
func (vals *ValidatorSet) applyUpdates(updates []*Validator) {

	existing := vals.Validators
	merged := make([]*Validator, len(existing)+len(updates))
	i := 0

	for len(existing) > 0 && len(updates) > 0 {
		if bytes.Compare(existing[0].Address.Bytes(), updates[0].Address.Bytes()) < 0 { // unchanged validator
			merged[i] = existing[0]
			existing = existing[1:]
		} else {
			// Apply add or update.
			merged[i] = updates[0]
			if bytes.Equal(existing[0].Address.Bytes(), updates[0].Address.Bytes()) {
				// Validator is present in both, advance existing.
				existing = existing[1:]
			}
			updates = updates[1:]
		}
		i++
	}

	// Add the elements which are left.
	for j := 0; j < len(existing); j++ {
		merged[i] = existing[j]
		i++
	}
	// OR add updates which are left.
	for j := 0; j < len(updates); j++ {
		merged[i] = updates[j]
		i++
	}

	vals.Validators = merged[:i]
}

// Checks that the validators to be removed are part of the validator set.
// No changes are made to the validator set 'vals'.
func verifyRemovals(deletes []*Validator, vals *ValidatorSet) error {

	for _, valUpdate := range deletes {
		address := valUpdate.Address
		_, val := vals.GetByAddress(address)
		if val == nil {
			return fmt.Errorf("failed to find validator %X to remove", address)
		}
	}
	if len(deletes) > len(vals.Validators) {
		panic("more deletes than validators")
	}
	return nil
}

// Removes the validators specified in 'deletes' from validator set 'vals'.
// Should not fail as verification has been done before.
func (vals *ValidatorSet) applyRemovals(deletes []*Validator) {

	existing := vals.Validators

	merged := make([]*Validator, len(existing)-len(deletes))
	i := 0

	// Loop over deletes until we removed all of them.
	for len(deletes) > 0 {
		if bytes.Equal(existing[0].Address.Bytes(), deletes[0].Address.Bytes()) {
			deletes = deletes[1:]
		} else { // Leave it in the resulting slice.
			merged[i] = existing[0]
			i++
		}
		existing = existing[1:]
	}

	// Add the elements which are left.
	for j := 0; j < len(existing); j++ {
		merged[i] = existing[j]
		i++
	}

	vals.Validators = merged[:i]
}

// Main function used by UpdateWithChangeSet() and NewValidatorSet().
// If 'allowDeletes' is false then delete operations (identified by validators with voting power 0)
// are not allowed and will trigger an error if present in 'changes'.
// The 'allowDeletes' flag is set to false by NewValidatorSet() and to true by UpdateWithChangeSet().
func (vals *ValidatorSet) updateWithChangeSet(changes []*Validator, allowDeletes bool) error {

	if len(changes) <= 0 {
		return nil
	}

	// Check for duplicates within changes, split in 'updates' and 'deletes' lists (sorted).
	updates, deletes, err := processChanges(changes)
	if err != nil {
		return err
	}

	if !allowDeletes && len(deletes) != 0 {
		return fmt.Errorf("cannot process validators with voting power 0: %v", deletes)
	}

	// Verify that applying the 'deletes' against 'vals' will not result in error.
	if err := verifyRemovals(deletes, vals); err != nil {
		return err
	}

	// Verify that applying the 'updates' against 'vals' will not result in error.
	updatedTotalVotingPower, numNewValidators, err := verifyUpdates(updates, vals)
	if err != nil {
		return err
	}

	// Check that the resulting set will not be empty.
	if numNewValidators == 0 && len(vals.Validators) == len(deletes) {
		return fmt.Errorf("applying the validator changes would result in empty set")
	}

	// Compute the priorities for updates.
	computeNewPriorities(updates, vals, updatedTotalVotingPower)

	// Apply updates and removals.
	vals.applyUpdates(updates)
	vals.applyRemovals(deletes)

	if err := vals.updateTotalVotingPower(); err != nil {
		return err
	}

	// Scale and center.
	vals.RescalePriorities(PriorityWindowSizeFactor * vals.TotalVotingPower())
	vals.shiftByAvgProposerPriority()

	return nil
}

// UpdateWithChangeSet attempts to update the validator set with 'changes'.
// It performs the following steps:
// - validates the changes making sure there are no duplicates and splits them in updates and deletes
// - verifies that applying the changes will not result in errors
// - computes the total voting power BEFORE removals to ensure that in the next steps the priorities
//   across old and newly added validators are fair
// - computes the priorities of new validators against the final set
// - applies the updates against the validator set
// - applies the removals against the validator set
// - performs scaling and centering of priority values
// If an error is detected during verification steps, it is returned and the validator set
// is not changed.
func (vals *ValidatorSet) UpdateWithChangeSet(changes []*Validator) error {
	return vals.updateWithChangeSet(changes, true)
}

//-----------------
// ErrTooMuchChange

func IsErrTooMuchChange(err error) bool {
	switch err.(type) {
	case errTooMuchChange:
		return true
	default:
		return false
	}
}

type errTooMuchChange struct {
	got    int64
	needed int64
}

func (e errTooMuchChange) Error() string {
	return fmt.Sprintf("Invalid commit -- insufficient old voting power: got %v, needed %v", e.got, e.needed)
}

//----------------

func (vals *ValidatorSet) String() string {
	return vals.StringIndented("")
}

func (vals *ValidatorSet) StringIndented(indent string) string {
	if vals == nil {
		return "nil-ValidatorSet"
	}
	var valStrings []string
	vals.Iterate(func(index int, val *Validator) bool {
		valStrings = append(valStrings, val.String())
		return false
	})
	return fmt.Sprintf(`ValidatorSet{
%s  Proposer: %v
%s  Validators:
%s    %v
%s}`,
		indent, vals.GetProposer().String(),
		indent,
		indent, strings.Join(valStrings, "\n"+indent+"    "),
		indent)

}

//-------------------------------------
// Implements sort for sorting validators by address.

// Sort validators by address.
type ValidatorsByAddress []*Validator

func (valz ValidatorsByAddress) Len() int {
	return len(valz)
}

func (valz ValidatorsByAddress) Less(i, j int) bool {
	return bytes.Compare(valz[i].Address.Bytes(), valz[j].Address.Bytes()) == -1
}

func (valz ValidatorsByAddress) Swap(i, j int) {
	it := valz[i]
	valz[i] = valz[j]
	valz[j] = it
}

///////////////////////////////////////////////////////////////////////////////
// safe addition/subtraction

func safeAdd(a, b int64) (int64, bool) {
	if b > 0 && a > math.MaxInt64-b {
		return -1, true
	} else if b < 0 && a < math.MinInt64-b {
		return -1, true
	}
	return a + b, false
}

func safeSub(a, b int64) (int64, bool) {
	if b > 0 && a < math.MinInt64+b {
		return -1, true
	} else if b < 0 && a > math.MaxInt64+b {
		return -1, true
	}
	return a - b, false
}

func safeAddClip(a, b int64) int64 {
	c, overflow := safeAdd(a, b)
	if overflow {
		if b < 0 {
			return math.MinInt64
		}
		return math.MaxInt64
	}
	return c
}

func safeSubClip(a, b int64) int64 {
	c, overflow := safeSub(a, b)
	if overflow {
		if b > 0 {
			return math.MinInt64
		}
		return math.MaxInt64
	}
	return c
}
