package lightclient

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/crypto/merkle"
	lerr "github.com/tendermint/tendermint/lite/errors"
	tmtypes "github.com/tendermint/tendermint/types"
)

const (
	chainIDLength              uint64 = 32
	heightLength               uint64 = 8
	validatorSetHashLength     uint64 = 32
	validatorPubkeyLength      uint64 = 32
	validatorVotingPowerLength uint64 = 8
	appHashLength              uint64 = 32
	storeNameLengthBytesLength uint64 = 32
	keyLengthBytesLength       uint64 = 32
	valueLengthBytesLength     uint64 = 32
	maxConsensusStateLength    uint64 = 32 * (128 - 1) // maximum validator quantity 99
)

type ConsensusState struct {
	ChainID             string
	Height              uint64
	AppHash             []byte
	CurValidatorSetHash []byte
	NextValidatorSet    *tmtypes.ValidatorSet
}

// input:
// | chainID   | height   | appHash  | curValidatorSetHash | [{validator pubkey, voting power}] |
// | 32 bytes  | 8 bytes  | 32 bytes | 32 bytes            | [{32 bytes, 8 bytes}]              |
func DecodeConsensusState(input []byte) (ConsensusState, error) {

	minimumLength := chainIDLength + heightLength + appHashLength + validatorSetHashLength
	singleValidatorBytesLength := validatorPubkeyLength + validatorVotingPowerLength

	inputLen := uint64(len(input))
	if inputLen <= minimumLength || (inputLen-minimumLength)%singleValidatorBytesLength != 0 {
		return ConsensusState{}, fmt.Errorf("expected input size %d+%d*N, actual input size: %d", minimumLength, singleValidatorBytesLength, inputLen)
	}
	pos := uint64(0)

	chainID := string(bytes.Trim(input[pos:pos+chainIDLength], "\x00"))
	pos += chainIDLength

	height := binary.BigEndian.Uint64(input[pos : pos+heightLength])
	pos += heightLength

	appHash := input[pos : pos+appHashLength]
	pos += appHashLength

	curValidatorSetHash := input[pos : pos+validatorSetHashLength]
	pos += validatorSetHashLength

	nextValidatorSetLength := (inputLen - minimumLength) / singleValidatorBytesLength
	validatorSetBytes := input[pos:]
	var validatorSet []*tmtypes.Validator
	for index := uint64(0); index < nextValidatorSetLength; index++ {
		validatorAndPowerBytes := validatorSetBytes[singleValidatorBytesLength*index : singleValidatorBytesLength*(index+1)]
		var pubkey ed25519.PubKeyEd25519
		copy(pubkey[:], validatorAndPowerBytes[:validatorPubkeyLength])
		votingPower := int64(binary.BigEndian.Uint64(validatorAndPowerBytes[validatorPubkeyLength:]))

		validator := tmtypes.NewValidator(pubkey, votingPower)
		validatorSet = append(validatorSet, validator)
	}

	consensusState := ConsensusState{
		ChainID:             chainID,
		Height:              height,
		AppHash:             appHash,
		CurValidatorSetHash: curValidatorSetHash,
		NextValidatorSet: &tmtypes.ValidatorSet{
			Validators: validatorSet,
		},
	}

	return consensusState, nil
}

// output:
// | chainID   | height   | appHash  | curValidatorSetHash | [{validator pubkey, voting power}] |
// | 32 bytes  | 8 bytes  | 32 bytes | 32 bytes            | [{32 bytes, 8 bytes}]              |
func (cs ConsensusState) EncodeConsensusState() ([]byte, error) {
	validatorSetLength := uint64(len(cs.NextValidatorSet.Validators))
	serializeLength := chainIDLength + heightLength + appHashLength + validatorSetHashLength + validatorSetLength*(validatorPubkeyLength+validatorVotingPowerLength)
	if serializeLength > maxConsensusStateLength {
		return nil, fmt.Errorf("too many validators %d, consensus state bytes should not exceed %d", len(cs.NextValidatorSet.Validators), maxConsensusStateLength)
	}

	encodingBytes := make([]byte, serializeLength)

	pos := uint64(0)
	if uint64(len(cs.ChainID)) > chainIDLength {
		return nil, fmt.Errorf("chainID length should be no more than 32")
	}
	copy(encodingBytes[pos:pos+chainIDLength], cs.ChainID)
	pos += chainIDLength

	binary.BigEndian.PutUint64(encodingBytes[pos:pos+heightLength], uint64(cs.Height))
	pos += heightLength

	copy(encodingBytes[pos:pos+appHashLength], cs.AppHash)
	pos += appHashLength

	copy(encodingBytes[pos:pos+validatorSetHashLength], cs.CurValidatorSetHash)
	pos += validatorSetHashLength

	for index := uint64(0); index < validatorSetLength; index++ {
		validator := cs.NextValidatorSet.Validators[index]
		pubkey, ok := validator.PubKey.(ed25519.PubKeyEd25519)
		if !ok {
			return nil, fmt.Errorf("invalid pubkey type")
		}

		copy(encodingBytes[pos:pos+validatorPubkeyLength], pubkey[:])
		pos += validatorPubkeyLength

		binary.BigEndian.PutUint64(encodingBytes[pos:pos+validatorVotingPowerLength], uint64(validator.VotingPower))
		pos += validatorVotingPowerLength
	}

	return encodingBytes, nil
}

func (cs *ConsensusState) ApplyHeader(header *Header) (bool, error) {
	if uint64(header.Height) < cs.Height {
		return false, fmt.Errorf("header height < consensus height (%d < %d)", header.Height, cs.Height)
	}

	if err := header.Validate(cs.ChainID); err != nil {
		return false, err
	}

	trustedNextHash := cs.NextValidatorSet.Hash()
	if cs.Height == uint64(header.Height-1) {
		if !bytes.Equal(trustedNextHash, header.ValidatorsHash) {
			return false, lerr.ErrUnexpectedValidators(header.ValidatorsHash, trustedNextHash)
		}
		err := cs.NextValidatorSet.VerifyCommit(cs.ChainID, header.Commit.BlockID, header.Height, header.Commit)
		if err != nil {
			return false, err
		}
	} else {
		err := cs.NextValidatorSet.VerifyFutureCommit(header.ValidatorSet, cs.ChainID,
			header.Commit.BlockID, header.Height, header.Commit)
		if err != nil {
			return false, err
		}
	}
	validatorSetChanged := false
	if !bytes.Equal(cs.CurValidatorSetHash, header.ValidatorsHash) || !bytes.Equal(cs.NextValidatorSet.Hash(), header.NextValidatorsHash) {
		validatorSetChanged = true
	}
	// update consensus state
	cs.Height = uint64(header.Height)
	cs.AppHash = header.AppHash
	cs.CurValidatorSetHash = header.ValidatorsHash
	cs.NextValidatorSet = header.NextValidatorSet

	return validatorSetChanged, nil
}

type Header struct {
	tmtypes.SignedHeader
	ValidatorSet     *tmtypes.ValidatorSet `json:"validator_set"`
	NextValidatorSet *tmtypes.ValidatorSet `json:"next_validator_set"`
}

func (h *Header) Validate(chainID string) error {
	if err := h.SignedHeader.ValidateBasic(chainID); err != nil {
		return err
	}
	if h.ValidatorSet == nil {
		return fmt.Errorf("invalid header: validator set is nil")
	}
	if h.NextValidatorSet == nil {
		return fmt.Errorf("invalid header: next validator set is nil")
	}
	if !bytes.Equal(h.ValidatorsHash, h.ValidatorSet.Hash()) {
		return fmt.Errorf("invalid header: validator set does not match hash")
	}
	if !bytes.Equal(h.NextValidatorsHash, h.NextValidatorSet.Hash()) {
		return fmt.Errorf("invalid header: next validator set does not match hash")
	}
	return nil
}

func (h *Header) EncodeHeader() ([]byte, error) {
	bz, err := Cdc.MarshalBinaryLengthPrefixed(h)
	if err != nil {
		return nil, err
	}
	return bz, nil
}

func DecodeHeader(input []byte) (*Header, error) {
	var header Header
	err := Cdc.UnmarshalBinaryLengthPrefixed(input, &header)
	if err != nil {
		return nil, err
	}
	return &header, nil
}

type KeyValueMerkleProof struct {
	Key       []byte
	Value     []byte
	StoreName string
	AppHash   []byte
	Proof     *merkle.Proof
}

func (kvmp *KeyValueMerkleProof) Validate() bool {
	prt := DefaultProofRuntime()

	kp := merkle.KeyPath{}
	kp = kp.AppendKey([]byte(kvmp.StoreName), merkle.KeyEncodingURL)
	kp = kp.AppendKey(kvmp.Key, merkle.KeyEncodingURL)

	if len(kvmp.Value) == 0 {
		err := prt.VerifyAbsence(kvmp.Proof, kvmp.AppHash, kp.String())
		return err == nil
	}

	err := prt.VerifyValue(kvmp.Proof, kvmp.AppHash, kp.String(), kvmp.Value)
	return err == nil
}

// input:
// | storeName | key length | key | value length | value | appHash  | proof |
// | 32 bytes  | 32 bytes   |     | 32 bytes     |       | 32 bytes |       |
func DecodeKeyValueMerkleProof(input []byte) (*KeyValueMerkleProof, error) {
	inputLength := uint64(len(input))
	pos := uint64(0)

	if inputLength <= storeNameLengthBytesLength+keyLengthBytesLength+valueLengthBytesLength+appHashLength {
		return nil, fmt.Errorf("input length should be no less than %d", storeNameLengthBytesLength+keyLengthBytesLength+valueLengthBytesLength+appHashLength)
	}
	storeName := string(bytes.Trim(input[pos:pos+storeNameLengthBytesLength], "\x00"))
	pos += storeNameLengthBytesLength

	keyLength := binary.BigEndian.Uint64(input[pos+keyLengthBytesLength-8 : pos+keyLengthBytesLength])
	pos += keyLengthBytesLength

	if inputLength <= storeNameLengthBytesLength+keyLengthBytesLength+keyLength+valueLengthBytesLength {
		return nil, fmt.Errorf("invalid input, keyLength %d is too long", keyLength)
	}
	key := input[pos : pos+keyLength]
	pos += keyLength

	valueLength := binary.BigEndian.Uint64(input[pos+valueLengthBytesLength-8 : pos+valueLengthBytesLength])
	pos += valueLengthBytesLength

	if inputLength <= storeNameLengthBytesLength+keyLengthBytesLength+keyLength+valueLengthBytesLength+valueLength+appHashLength {
		return nil, fmt.Errorf("invalid input, valueLength %d is too long", valueLength)
	}
	value := input[pos : pos+valueLength]
	pos += valueLength

	appHash := input[pos : pos+appHashLength]
	pos += appHashLength

	proofBytes := input[pos:]
	var merkleProof merkle.Proof
	err := merkleProof.Unmarshal(proofBytes)
	if err != nil {
		return nil, err
	}

	keyValueMerkleProof := &KeyValueMerkleProof{
		Key:       key,
		Value:     value,
		StoreName: storeName,
		AppHash:   appHash,
		Proof:     &merkleProof,
	}

	return keyValueMerkleProof, nil
}
