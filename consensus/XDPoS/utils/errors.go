package utils

import (
	"errors"
	"fmt"

	"github.com/XinFinOrg/XDPoSChain/core/types"
)

// Various error messages to mark blocks invalid. These should be private to
// prevent engine specific errors from being referenced in the remainder of the
// codebase, inherently breaking if the engine is swapped out. Please put common
// error types into the consensus package.
var (
	// errUnknownBlock is returned when the list of signers is requested for a block
	// that is not part of the local blockchain.
	ErrUnknownBlock = errors.New("unknown block")

	// errInvalidCheckpointBeneficiary is returned if a checkpoint/epoch transition
	// block has a beneficiary set to non-zeroes.
	ErrInvalidCheckpointBeneficiary = errors.New("beneficiary in checkpoint block non-zero")

	// errInvalidVote is returned if a nonce value is something else that the two
	// allowed constants of 0x00..0 or 0xff..f.
	ErrInvalidVote = errors.New("vote nonce not 0x00..0 or 0xff..f")

	// errInvalidCheckpointVote is returned if a checkpoint/epoch transition block
	// has a vote nonce set to non-zeroes.
	ErrInvalidCheckpointVote = errors.New("vote nonce in checkpoint block non-zero")

	// errMissingVanity is returned if a block's extra-data section is shorter than
	// 32 bytes, which is required to store the signer vanity.
	ErrMissingVanity = errors.New("extra-data 32 byte vanity prefix missing")

	// errMissingSignature is returned if a block's extra-data section doesn't seem
	// to contain a 65 byte secp256k1 signature.
	ErrMissingSignature = errors.New("extra-data 65 byte suffix signature missing")

	// errExtraSigners is returned if non-checkpoint block contain signer data in
	// their extra-data fields.
	ErrExtraSigners = errors.New("non-checkpoint block contains extra signer list")

	// errInvalidCheckpointSigners is returned if a checkpoint block contains an
	// invalid list of signers (i.e. non divisible by 20 bytes, or not the correct
	// ones).
	ErrInvalidCheckpointSigners = errors.New("invalid signer list on checkpoint block")

	ErrInvalidCheckpointPenalties = errors.New("invalid penalty list on checkpoint block")

	ErrValidatorsNotLegit = errors.New("validators does not match what's stored in snapshot minutes its penalty")
	ErrPenaltiesNotLegit  = errors.New("penalties does not match")

	// errInvalidMixDigest is returned if a block's mix digest is non-zero.
	ErrInvalidMixDigest = errors.New("non-zero mix digest")

	// errInvalidUncleHash is returned if a block contains an non-empty uncle list.
	ErrInvalidUncleHash = errors.New("non empty uncle hash")

	// errInvalidDifficulty is returned if the difficulty of a block is not either
	// of 1 or 2, or if the value does not match the turn of the signer.
	ErrInvalidDifficulty = errors.New("invalid difficulty")

	// ErrInvalidTimestamp is returned if the timestamp of a block is lower than
	// the previous block's timestamp + the minimum block period.
	ErrInvalidTimestamp = errors.New("invalid timestamp")

	// errInvalidVotingChain is returned if an authorization list is attempted to
	// be modified via out-of-range or non-contiguous headers.
	ErrInvalidVotingChain = errors.New("invalid voting chain")

	ErrInvalidHeaderOrder = errors.New("invalid header order")
	ErrInvalidChild       = errors.New("invalid header child")

	// errUnauthorized is returned if a header is signed by a non-authorized entity.
	ErrUnauthorized = errors.New("unauthorized")

	ErrFailedDoubleValidation = errors.New("wrong pair of creator-validator in double validation")

	// errWaitTransactions is returned if an empty block is attempted to be sealed
	// on an instant chain (0 second period). It's important to refuse these as the
	// block reward is zero, so an empty block just bloats the chain... fast.
	ErrWaitTransactions = errors.New("waiting for transactions")

	ErrInvalidCheckpointValidators = errors.New("invalid validators list on checkpoint block")

	ErrEmptyEpochSwitchValidators = errors.New("empty validators list on epoch switch block")

	ErrInvalidV2Extra                = errors.New("Invalid v2 extra in the block")
	ErrInvalidQC                     = errors.New("Invalid QC content")
	ErrInvalidQCSignatures           = errors.New("Invalid QC Signatures")
	ErrInvalidTC                     = errors.New("Invalid TC content")
	ErrInvalidTCSignatures           = errors.New("Invalid TC Signatures")
	ErrEmptyBlockInfoHash            = errors.New("BlockInfo hash is empty")
	ErrInvalidFieldInNonEpochSwitch  = errors.New("Invalid field exist in a non-epoch swtich block")
	ErrValidatorNotWithinMasternodes = errors.New("Validaotor address is not in the master node list")
	ErrCoinbaseAndValidatorMismatch  = errors.New("Validaotor and coinbase address in header does not match")
	ErrNotItsTurn                    = errors.New("Not validator's turn to mine this block")

	ErrRoundInvalid = errors.New("Invalid Round, it shall be bigger than QC round")

	ErrAlreadyMined = errors.New("Already mined")
)

type ErrIncomingMessageRoundNotEqualCurrentRound struct {
	Type          string
	IncomingRound types.Round
	CurrentRound  types.Round
}

func (e *ErrIncomingMessageRoundNotEqualCurrentRound) Error() string {
	return fmt.Sprintf("%s message round number: %v does not match currentRound: %v", e.Type, e.IncomingRound, e.CurrentRound)
}

type ErrIncomingMessageRoundTooFarFromCurrentRound struct {
	Type          string
	IncomingRound types.Round
	CurrentRound  types.Round
}

func (e *ErrIncomingMessageRoundTooFarFromCurrentRound) Error() string {
	return fmt.Sprintf("%s message round number: %v is too far away from currentRound: %v", e.Type, e.IncomingRound, e.CurrentRound)
}
