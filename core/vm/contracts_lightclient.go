package vm

import (
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/core/vm/lightclient"
	"github.com/ethereum/go-ethereum/params"
)

const (
	uint64TypeLength                      uint64 = 8
	precompileContractInputMetaDataLength uint64 = 32
	consensusStateLengthBytesLength       uint64 = 32

	tmHeaderValidateResultMetaDataLength uint64 = 32
	merkleProofValidateResultLength      uint64 = 32
)

// input:
// consensus state length | consensus state | tendermint header |
// 32 bytes               |                 |                   |
func decodeTendermintHeaderValidationInput(input []byte) (*lightclient.ConsensusState, *lightclient.Header, error) {
	csLen := binary.BigEndian.Uint64(input[consensusStateLengthBytesLength-uint64TypeLength : consensusStateLengthBytesLength])
	if uint64(len(input)) <= consensusStateLengthBytesLength+csLen {
		return nil, nil, fmt.Errorf("expected payload size %d, actual size: %d", consensusStateLengthBytesLength+csLen, len(input))
	}

	cs, err := lightclient.DecodeConsensusState(input[consensusStateLengthBytesLength : consensusStateLengthBytesLength+csLen])
	if err != nil {
		return nil, nil, err
	}
	header, err := lightclient.DecodeHeader(input[consensusStateLengthBytesLength+csLen:])
	if err != nil {
		return nil, nil, err
	}

	return &cs, header, nil
}

// tmHeaderValidate implemented as a native contract.
type tmHeaderValidate struct{}

func (c *tmHeaderValidate) RequiredGas(input []byte) uint64 {
	return params.TendermintHeaderValidateGas
}

func (c *tmHeaderValidate) Run(input []byte) (result []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("internal error: %v\n", r)
		}
	}()

	if uint64(len(input)) <= precompileContractInputMetaDataLength {
		return nil, fmt.Errorf("invalid input")
	}

	payloadLength := binary.BigEndian.Uint64(input[precompileContractInputMetaDataLength-uint64TypeLength : precompileContractInputMetaDataLength])
	if uint64(len(input)) != payloadLength+precompileContractInputMetaDataLength {
		return nil, fmt.Errorf("invalid input: input size should be %d, actual the size is %d", payloadLength+precompileContractInputMetaDataLength, len(input))
	}

	cs, header, err := decodeTendermintHeaderValidationInput(input[precompileContractInputMetaDataLength:])
	if err != nil {
		return nil, err
	}

	validatorSetChanged, err := cs.ApplyHeader(header)
	if err != nil {
		return nil, err
	}

	consensusStateBytes, err := cs.EncodeConsensusState()
	if err != nil {
		return nil, err
	}

	// result
	// | validatorSetChanged | empty      | consensusStateBytesLength |  new consensusState |
	// | 1 byte              | 23 bytes   | 8 bytes                   |                     |
	lengthBytes := make([]byte, tmHeaderValidateResultMetaDataLength)
	if validatorSetChanged {
		copy(lengthBytes[:1], []byte{0x01})
	}
	consensusStateBytesLength := uint64(len(consensusStateBytes))
	binary.BigEndian.PutUint64(lengthBytes[tmHeaderValidateResultMetaDataLength-uint64TypeLength:], consensusStateBytesLength)

	result = append(lengthBytes, consensusStateBytes...)

	return result, nil
}

//------------------------------------------------------------------------------------------------------------------------------------------------

// tmHeaderValidate implemented as a native contract.
type iavlMerkleProofValidate struct{}

func (c *iavlMerkleProofValidate) RequiredGas(input []byte) uint64 {
	return params.IAVLMerkleProofValidateGas
}

// input:
// | payload length | payload    |
// | 32 bytes       |            |
func (c *iavlMerkleProofValidate) Run(input []byte) (result []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("internal error: %v\n", r)
		}
	}()

	if uint64(len(input)) <= precompileContractInputMetaDataLength {
		return nil, fmt.Errorf("invalid input: input should include %d bytes payload length and payload", precompileContractInputMetaDataLength)
	}

	payloadLength := binary.BigEndian.Uint64(input[precompileContractInputMetaDataLength-uint64TypeLength : precompileContractInputMetaDataLength])
	if uint64(len(input)) != payloadLength+precompileContractInputMetaDataLength {
		return nil, fmt.Errorf("invalid input: input size should be %d, actual the size is %d", payloadLength+precompileContractInputMetaDataLength, len(input))
	}

	kvmp, err := lightclient.DecodeKeyValueMerkleProof(input[precompileContractInputMetaDataLength:])
	if err != nil {
		return nil, err
	}

	valid := kvmp.Validate()
	if !valid {
		return nil, fmt.Errorf("invalid merkle proof")
	}

	result = make([]byte, merkleProofValidateResultLength)
	binary.BigEndian.PutUint64(result[merkleProofValidateResultLength-uint64TypeLength:], 0x01)
	return result, nil
}
