package dasigners

import "errors"

var (
	ErrQuorumIdOutOfBound         = errors.New("quorum id out of bound")
	ErrEpochOutOfBound            = errors.New("epoch out of bound")
	ErrRowIdOfBound               = errors.New("row id out of bound")
	ErrQuorumBitmapLengthMismatch = errors.New("quorum bitmap length mismatch")
	ErrSignerNotFound             = errors.New("signer not found")
	ErrInvalidSender              = errors.New("invalid sender")
	ErrSignerExists               = errors.New("signer exists")
	ErrInvalidSignature           = errors.New("invalid signature")
)
