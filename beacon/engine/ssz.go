// Copyright 2025 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package engine

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// SSZ status codes for PayloadStatusSSZ (EIP-8161).
const (
	SSZStatusValid            uint8 = 0
	SSZStatusInvalid          uint8 = 1
	SSZStatusSyncing          uint8 = 2
	SSZStatusAccepted         uint8 = 3
	SSZStatusInvalidBlockHash uint8 = 4
)

// EngineStatusToSSZ converts a string engine status to the SSZ uint8 representation.
func EngineStatusToSSZ(status string) uint8 {
	switch status {
	case VALID:
		return SSZStatusValid
	case INVALID:
		return SSZStatusInvalid
	case SYNCING:
		return SSZStatusSyncing
	case ACCEPTED:
		return SSZStatusAccepted
	case "INVALID_BLOCK_HASH":
		return SSZStatusInvalidBlockHash
	default:
		return SSZStatusInvalid
	}
}

// SSZToEngineStatus converts an SSZ uint8 status to the string engine status.
func SSZToEngineStatus(status uint8) string {
	switch status {
	case SSZStatusValid:
		return VALID
	case SSZStatusInvalid:
		return INVALID
	case SSZStatusSyncing:
		return SYNCING
	case SSZStatusAccepted:
		return ACCEPTED
	case SSZStatusInvalidBlockHash:
		return "INVALID_BLOCK_HASH"
	default:
		return INVALID
	}
}

// --- PayloadStatus SSZ ---

const payloadStatusFixedSize = 9 // status(1) + hash_offset(4) + err_offset(4)

// EncodePayloadStatusSSZ encodes a PayloadStatusV1 to SSZ bytes per EIP-8161.
func EncodePayloadStatusSSZ(ps *PayloadStatusV1) []byte {
	// Build Union[None, Hash32] for latest_valid_hash
	var hashUnion []byte
	if ps.LatestValidHash != nil {
		hashUnion = make([]byte, 33) // selector(1) + hash(32)
		hashUnion[0] = 1
		copy(hashUnion[1:33], ps.LatestValidHash[:])
	} else {
		hashUnion = []byte{0}
	}

	var errorBytes []byte
	if ps.ValidationError != nil {
		errorBytes = []byte(*ps.ValidationError)
	}

	buf := make([]byte, payloadStatusFixedSize+len(hashUnion)+len(errorBytes))
	buf[0] = EngineStatusToSSZ(ps.Status)
	binary.LittleEndian.PutUint32(buf[1:5], uint32(payloadStatusFixedSize))
	binary.LittleEndian.PutUint32(buf[5:9], uint32(payloadStatusFixedSize+len(hashUnion)))

	copy(buf[payloadStatusFixedSize:], hashUnion)
	copy(buf[payloadStatusFixedSize+len(hashUnion):], errorBytes)
	return buf
}

// DecodePayloadStatusSSZ decodes SSZ bytes into a PayloadStatusV1.
func DecodePayloadStatusSSZ(buf []byte) (*PayloadStatusV1, error) {
	if len(buf) < payloadStatusFixedSize {
		return nil, fmt.Errorf("PayloadStatusSSZ: buffer too short (%d < %d)", len(buf), payloadStatusFixedSize)
	}

	ps := &PayloadStatusV1{
		Status: SSZToEngineStatus(buf[0]),
	}

	hashOffset := binary.LittleEndian.Uint32(buf[1:5])
	errOffset := binary.LittleEndian.Uint32(buf[5:9])

	if hashOffset > uint32(len(buf)) || errOffset > uint32(len(buf)) || hashOffset > errOffset {
		return nil, fmt.Errorf("PayloadStatusSSZ: offsets out of bounds")
	}

	// Decode Union[None, Hash32]
	unionData := buf[hashOffset:errOffset]
	if len(unionData) > 0 {
		if unionData[0] == 1 {
			if len(unionData) < 33 {
				return nil, fmt.Errorf("PayloadStatusSSZ: Union hash data too short")
			}
			hash := common.BytesToHash(unionData[1:33])
			ps.LatestValidHash = &hash
		}
	}

	// Decode validation_error
	if errOffset < uint32(len(buf)) {
		errLen := uint32(len(buf)) - errOffset
		if errLen > 1024 {
			return nil, fmt.Errorf("PayloadStatusSSZ: validation error too long (%d > 1024)", errLen)
		}
		s := string(buf[errOffset:])
		ps.ValidationError = &s
	}

	return ps, nil
}

// --- ForkchoiceState SSZ ---

// EncodeForkchoiceStateSSZ encodes a ForkchoiceStateV1 to SSZ bytes (96 bytes fixed).
func EncodeForkchoiceStateSSZ(fcs *ForkchoiceStateV1) []byte {
	buf := make([]byte, 96)
	copy(buf[0:32], fcs.HeadBlockHash[:])
	copy(buf[32:64], fcs.SafeBlockHash[:])
	copy(buf[64:96], fcs.FinalizedBlockHash[:])
	return buf
}

// DecodeForkchoiceStateSSZ decodes SSZ bytes into a ForkchoiceStateV1.
func DecodeForkchoiceStateSSZ(buf []byte) (*ForkchoiceStateV1, error) {
	if len(buf) < 96 {
		return nil, fmt.Errorf("ForkchoiceState: buffer too short (%d < 96)", len(buf))
	}
	fcs := &ForkchoiceStateV1{}
	copy(fcs.HeadBlockHash[:], buf[0:32])
	copy(fcs.SafeBlockHash[:], buf[32:64])
	copy(fcs.FinalizedBlockHash[:], buf[64:96])
	return fcs, nil
}

// --- ForkchoiceUpdated Response SSZ ---

const forkchoiceUpdatedResponseFixedSize = 8

// EncodeForkChoiceResponseSSZ encodes a ForkChoiceResponse to SSZ bytes.
func EncodeForkChoiceResponseSSZ(resp *ForkChoiceResponse) []byte {
	psBytes := EncodePayloadStatusSSZ(&resp.PayloadStatus)

	// Build Union[None, uint64] for payload ID
	var pidUnion []byte
	if resp.PayloadID != nil {
		pidUnion = make([]byte, 9) // selector(1) + 8 bytes
		pidUnion[0] = 1
		copy(pidUnion[1:9], resp.PayloadID[:])
	} else {
		pidUnion = []byte{0}
	}

	buf := make([]byte, forkchoiceUpdatedResponseFixedSize+len(psBytes)+len(pidUnion))
	binary.LittleEndian.PutUint32(buf[0:4], uint32(forkchoiceUpdatedResponseFixedSize))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(forkchoiceUpdatedResponseFixedSize+len(psBytes)))

	copy(buf[forkchoiceUpdatedResponseFixedSize:], psBytes)
	copy(buf[forkchoiceUpdatedResponseFixedSize+len(psBytes):], pidUnion)
	return buf
}

// DecodeForkChoiceResponseSSZ decodes SSZ bytes into a ForkChoiceResponse.
func DecodeForkChoiceResponseSSZ(buf []byte) (*ForkChoiceResponse, error) {
	if len(buf) < forkchoiceUpdatedResponseFixedSize {
		return nil, fmt.Errorf("ForkChoiceResponseSSZ: buffer too short (%d < %d)", len(buf), forkchoiceUpdatedResponseFixedSize)
	}

	psOffset := binary.LittleEndian.Uint32(buf[0:4])
	pidOffset := binary.LittleEndian.Uint32(buf[4:8])

	if psOffset > uint32(len(buf)) || pidOffset > uint32(len(buf)) || psOffset > pidOffset {
		return nil, fmt.Errorf("ForkChoiceResponseSSZ: offsets out of bounds")
	}

	resp := &ForkChoiceResponse{}

	ps, err := DecodePayloadStatusSSZ(buf[psOffset:pidOffset])
	if err != nil {
		return nil, err
	}
	resp.PayloadStatus = *ps

	// Decode Union[None, PayloadID]
	pidData := buf[pidOffset:]
	if len(pidData) > 0 && pidData[0] == 1 {
		if len(pidData) < 9 {
			return nil, fmt.Errorf("ForkChoiceResponseSSZ: Union payload_id data too short")
		}
		var pid PayloadID
		copy(pid[:], pidData[1:9])
		resp.PayloadID = &pid
	}

	return resp, nil
}

// --- CommunicationChannel SSZ ---

// EncodeCommunicationChannelsSSZ encodes communication channels to SSZ.
func EncodeCommunicationChannelsSSZ(channels []CommunicationChannel) []byte {
	if len(channels) == 0 {
		return []byte{}
	}

	var totalSize int
	for _, ch := range channels {
		totalSize += 4 + len(ch.Protocol) + 4 + len(ch.URL)
	}

	buf := make([]byte, 4+totalSize)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(len(channels)))

	offset := 4
	for _, ch := range channels {
		protBytes := []byte(ch.Protocol)
		urlBytes := []byte(ch.URL)

		binary.LittleEndian.PutUint32(buf[offset:offset+4], uint32(len(protBytes)))
		offset += 4
		copy(buf[offset:], protBytes)
		offset += len(protBytes)

		binary.LittleEndian.PutUint32(buf[offset:offset+4], uint32(len(urlBytes)))
		offset += 4
		copy(buf[offset:], urlBytes)
		offset += len(urlBytes)
	}

	return buf
}

// DecodeCommunicationChannelsSSZ decodes communication channels from SSZ bytes.
func DecodeCommunicationChannelsSSZ(buf []byte) ([]CommunicationChannel, error) {
	if len(buf) < 4 {
		return nil, fmt.Errorf("CommunicationChannels: buffer too short")
	}

	count := binary.LittleEndian.Uint32(buf[0:4])
	if count > 16 {
		return nil, fmt.Errorf("CommunicationChannels: too many channels (%d > 16)", count)
	}

	channels := make([]CommunicationChannel, 0, count)
	offset := uint32(4)

	for i := uint32(0); i < count; i++ {
		if offset+4 > uint32(len(buf)) {
			return nil, fmt.Errorf("CommunicationChannels: unexpected end of buffer")
		}
		protLen := binary.LittleEndian.Uint32(buf[offset : offset+4])
		offset += 4
		if protLen > 32 || offset+protLen > uint32(len(buf)) {
			return nil, fmt.Errorf("CommunicationChannels: protocol too long or truncated")
		}
		protocol := string(buf[offset : offset+protLen])
		offset += protLen

		if offset+4 > uint32(len(buf)) {
			return nil, fmt.Errorf("CommunicationChannels: unexpected end of buffer")
		}
		urlLen := binary.LittleEndian.Uint32(buf[offset : offset+4])
		offset += 4
		if urlLen > 256 || offset+urlLen > uint32(len(buf)) {
			return nil, fmt.Errorf("CommunicationChannels: URL too long or truncated")
		}
		url := string(buf[offset : offset+urlLen])
		offset += urlLen

		channels = append(channels, CommunicationChannel{Protocol: protocol, URL: url})
	}

	return channels, nil
}

// --- Capabilities SSZ ---

// EncodeCapabilitiesSSZ encodes a list of capability strings to SSZ.
func EncodeCapabilitiesSSZ(capabilities []string) []byte {
	var totalSize int
	for _, cap := range capabilities {
		totalSize += 4 + len(cap)
	}

	buf := make([]byte, 4+totalSize)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(len(capabilities)))

	offset := 4
	for _, cap := range capabilities {
		capBytes := []byte(cap)
		binary.LittleEndian.PutUint32(buf[offset:offset+4], uint32(len(capBytes)))
		offset += 4
		copy(buf[offset:], capBytes)
		offset += len(capBytes)
	}

	return buf
}

// DecodeCapabilitiesSSZ decodes a list of capability strings from SSZ bytes.
func DecodeCapabilitiesSSZ(buf []byte) ([]string, error) {
	if len(buf) < 4 {
		return nil, fmt.Errorf("Capabilities: buffer too short")
	}

	count := binary.LittleEndian.Uint32(buf[0:4])
	if count > 128 {
		return nil, fmt.Errorf("Capabilities: too many capabilities (%d > 128)", count)
	}

	capabilities := make([]string, 0, count)
	offset := uint32(4)

	for i := uint32(0); i < count; i++ {
		if offset+4 > uint32(len(buf)) {
			return nil, fmt.Errorf("Capabilities: unexpected end of buffer")
		}
		capLen := binary.LittleEndian.Uint32(buf[offset : offset+4])
		offset += 4
		if capLen > 64 || offset+capLen > uint32(len(buf)) {
			return nil, fmt.Errorf("Capabilities: capability too long or truncated")
		}
		capabilities = append(capabilities, string(buf[offset:offset+capLen]))
		offset += capLen
	}

	return capabilities, nil
}

// --- ClientVersion SSZ ---

// EncodeClientVersionSSZ encodes a ClientVersionV1 to SSZ.
func EncodeClientVersionSSZ(cv *ClientVersionV1) []byte {
	codeBytes := []byte(cv.Code)
	nameBytes := []byte(cv.Name)
	versionBytes := []byte(cv.Version)
	commitBytes := []byte(cv.Commit)

	totalLen := 4 + len(codeBytes) + 4 + len(nameBytes) + 4 + len(versionBytes) + 4 + len(commitBytes)
	buf := make([]byte, totalLen)

	offset := 0
	for _, field := range [][]byte{codeBytes, nameBytes, versionBytes, commitBytes} {
		binary.LittleEndian.PutUint32(buf[offset:offset+4], uint32(len(field)))
		offset += 4
		copy(buf[offset:], field)
		offset += len(field)
	}

	return buf
}

// DecodeClientVersionSSZ decodes a ClientVersionV1 from SSZ bytes.
func DecodeClientVersionSSZ(buf []byte) (*ClientVersionV1, error) {
	if len(buf) < 16 {
		return nil, fmt.Errorf("ClientVersion: buffer too short")
	}

	cv := &ClientVersionV1{}
	offset := uint32(0)

	readString := func(maxLen uint32) (string, error) {
		if offset+4 > uint32(len(buf)) {
			return "", fmt.Errorf("ClientVersion: unexpected end of buffer")
		}
		strLen := binary.LittleEndian.Uint32(buf[offset : offset+4])
		offset += 4
		if strLen > maxLen || offset+strLen > uint32(len(buf)) {
			return "", fmt.Errorf("ClientVersion: string too long or truncated")
		}
		s := string(buf[offset : offset+strLen])
		offset += strLen
		return s, nil
	}

	var err error
	if cv.Code, err = readString(8); err != nil {
		return nil, err
	}
	if cv.Name, err = readString(64); err != nil {
		return nil, err
	}
	if cv.Version, err = readString(64); err != nil {
		return nil, err
	}
	if cv.Commit, err = readString(64); err != nil {
		return nil, err
	}

	return cv, nil
}

// EncodeClientVersionsSSZ encodes a list of ClientVersionV1 to SSZ.
func EncodeClientVersionsSSZ(versions []ClientVersionV1) []byte {
	var parts [][]byte
	for i := range versions {
		parts = append(parts, EncodeClientVersionSSZ(&versions[i]))
	}

	totalLen := 4
	for _, p := range parts {
		totalLen += 4 + len(p)
	}

	buf := make([]byte, totalLen)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(len(versions)))

	offset := 4
	for _, p := range parts {
		binary.LittleEndian.PutUint32(buf[offset:offset+4], uint32(len(p)))
		offset += 4
		copy(buf[offset:], p)
		offset += len(p)
	}

	return buf
}

// DecodeClientVersionsSSZ decodes a list of ClientVersionV1 from SSZ bytes.
func DecodeClientVersionsSSZ(buf []byte) ([]ClientVersionV1, error) {
	if len(buf) < 4 {
		return nil, fmt.Errorf("ClientVersions: buffer too short")
	}

	count := binary.LittleEndian.Uint32(buf[0:4])
	if count > 16 {
		return nil, fmt.Errorf("ClientVersions: too many versions (%d > 16)", count)
	}

	versions := make([]ClientVersionV1, 0, count)
	offset := uint32(4)

	for i := uint32(0); i < count; i++ {
		if offset+4 > uint32(len(buf)) {
			return nil, fmt.Errorf("ClientVersions: unexpected end of buffer")
		}
		cvLen := binary.LittleEndian.Uint32(buf[offset : offset+4])
		offset += 4
		if offset+cvLen > uint32(len(buf)) {
			return nil, fmt.Errorf("ClientVersions: truncated")
		}
		cv, err := DecodeClientVersionSSZ(buf[offset : offset+cvLen])
		if err != nil {
			return nil, err
		}
		versions = append(versions, *cv)
		offset += cvLen
	}

	return versions, nil
}

// --- ExecutionPayload SSZ ---

// engineVersionToPayloadVersion maps Engine API versions to ExecutionPayload SSZ versions.
func engineVersionToPayloadVersion(engineVersion int) int {
	if engineVersion == 4 {
		return 3 // Electra uses Deneb payload layout
	}
	if engineVersion >= 5 {
		return 4 // Amsterdam and beyond use extended layout
	}
	return engineVersion
}

// executionPayloadFixedSize returns the fixed part size for a given version.
func executionPayloadFixedSize(version int) int {
	size := 508 // V1 base
	if version >= 2 {
		size += 4 // withdrawals_offset
	}
	if version >= 3 {
		size += 8 + 8 // blob_gas_used + excess_blob_gas
	}
	if version >= 4 {
		size += 8 + 4 // slot_number + block_access_list_offset
	}
	return size
}

// uint256ToSSZBytes converts a big.Int to 32-byte little-endian SSZ representation.
func uint256ToSSZBytes(val *big.Int) []byte {
	buf := make([]byte, 32)
	if val == nil {
		return buf
	}
	b := val.Bytes() // big-endian, minimal
	for i, v := range b {
		buf[len(b)-1-i] = v
	}
	return buf
}

// sszBytesToUint256 converts 32-byte little-endian SSZ bytes to a big.Int.
func sszBytesToUint256(buf []byte) *big.Int {
	be := make([]byte, 32)
	for i := 0; i < 32; i++ {
		be[31-i] = buf[i]
	}
	return new(big.Int).SetBytes(be)
}

// encodeTransactionsSSZ encodes a list of transactions as SSZ list of variable-length items.
func encodeTransactionsSSZ(txs [][]byte) []byte {
	if len(txs) == 0 {
		return nil
	}
	offsetsSize := len(txs) * 4
	dataSize := 0
	for _, tx := range txs {
		dataSize += len(tx)
	}
	buf := make([]byte, offsetsSize+dataSize)

	dataStart := offsetsSize
	for i, tx := range txs {
		binary.LittleEndian.PutUint32(buf[i*4:(i+1)*4], uint32(dataStart))
		dataStart += len(tx)
	}
	pos := offsetsSize
	for _, tx := range txs {
		copy(buf[pos:], tx)
		pos += len(tx)
	}
	return buf
}

// decodeTransactionsSSZ decodes SSZ-encoded list of variable-length transactions.
func decodeTransactionsSSZ(buf []byte) ([][]byte, error) {
	if len(buf) == 0 {
		return nil, nil
	}
	if len(buf) < 4 {
		return nil, fmt.Errorf("transactions SSZ: buffer too short")
	}
	firstOffset := binary.LittleEndian.Uint32(buf[0:4])
	if firstOffset%4 != 0 {
		return nil, fmt.Errorf("transactions SSZ: first offset not aligned (%d)", firstOffset)
	}
	count := firstOffset / 4
	if count == 0 {
		return nil, nil
	}
	if firstOffset > uint32(len(buf)) {
		return nil, fmt.Errorf("transactions SSZ: first offset out of bounds")
	}

	offsets := make([]uint32, count)
	for i := uint32(0); i < count; i++ {
		offsets[i] = binary.LittleEndian.Uint32(buf[i*4 : (i+1)*4])
	}

	txs := make([][]byte, count)
	for i := uint32(0); i < count; i++ {
		start := offsets[i]
		var end uint32
		if i+1 < count {
			end = offsets[i+1]
		} else {
			end = uint32(len(buf))
		}
		if start > uint32(len(buf)) || end > uint32(len(buf)) || start > end {
			return nil, fmt.Errorf("transactions SSZ: invalid offset at index %d", i)
		}
		tx := make([]byte, end-start)
		copy(tx, buf[start:end])
		txs[i] = tx
	}
	return txs, nil
}

// Withdrawal SSZ: index(8) + validator_index(8) + address(20) + amount(8) = 44 bytes
const withdrawalSSZSize = 44

func encodeWithdrawalsSSZ(withdrawals []*types.Withdrawal) []byte {
	if withdrawals == nil {
		return nil
	}
	buf := make([]byte, len(withdrawals)*withdrawalSSZSize)
	for i, w := range withdrawals {
		off := i * withdrawalSSZSize
		binary.LittleEndian.PutUint64(buf[off:off+8], w.Index)
		binary.LittleEndian.PutUint64(buf[off+8:off+16], w.Validator)
		copy(buf[off+16:off+36], w.Address[:])
		binary.LittleEndian.PutUint64(buf[off+36:off+44], w.Amount)
	}
	return buf
}

func decodeWithdrawalsSSZ(buf []byte) ([]*types.Withdrawal, error) {
	if len(buf) == 0 {
		return []*types.Withdrawal{}, nil
	}
	if len(buf)%withdrawalSSZSize != 0 {
		return nil, fmt.Errorf("withdrawals SSZ: buffer length %d not divisible by %d", len(buf), withdrawalSSZSize)
	}
	count := len(buf) / withdrawalSSZSize
	withdrawals := make([]*types.Withdrawal, count)
	for i := 0; i < count; i++ {
		off := i * withdrawalSSZSize
		withdrawals[i] = &types.Withdrawal{
			Index:     binary.LittleEndian.Uint64(buf[off : off+8]),
			Validator: binary.LittleEndian.Uint64(buf[off+8 : off+16]),
			Amount:    binary.LittleEndian.Uint64(buf[off+36 : off+44]),
		}
		copy(withdrawals[i].Address[:], buf[off+16:off+36])
	}
	return withdrawals, nil
}

// EncodeExecutableDataSSZ encodes an ExecutableData to SSZ bytes.
// version: 1=Bellatrix, 2=Capella, 3=Deneb, 4=Amsterdam
func EncodeExecutableDataSSZ(ep *ExecutableData, version int) []byte {
	fixedSize := executionPayloadFixedSize(version)

	extraData := ep.ExtraData
	txData := encodeTransactionsSSZ(ep.Transactions)
	var withdrawalData []byte
	if version >= 2 {
		withdrawalData = encodeWithdrawalsSSZ(ep.Withdrawals)
	}

	totalVarSize := len(extraData) + len(txData)
	if version >= 2 {
		totalVarSize += len(withdrawalData)
	}

	buf := make([]byte, fixedSize+totalVarSize)
	pos := 0

	// Fixed fields
	copy(buf[pos:pos+32], ep.ParentHash[:])
	pos += 32
	copy(buf[pos:pos+20], ep.FeeRecipient[:])
	pos += 20
	copy(buf[pos:pos+32], ep.StateRoot[:])
	pos += 32
	copy(buf[pos:pos+32], ep.ReceiptsRoot[:])
	pos += 32
	if len(ep.LogsBloom) >= 256 {
		copy(buf[pos:pos+256], ep.LogsBloom[:256])
	}
	pos += 256
	copy(buf[pos:pos+32], ep.Random[:])
	pos += 32
	binary.LittleEndian.PutUint64(buf[pos:pos+8], ep.Number)
	pos += 8
	binary.LittleEndian.PutUint64(buf[pos:pos+8], ep.GasLimit)
	pos += 8
	binary.LittleEndian.PutUint64(buf[pos:pos+8], ep.GasUsed)
	pos += 8
	binary.LittleEndian.PutUint64(buf[pos:pos+8], ep.Timestamp)
	pos += 8

	// extra_data offset
	extraDataOffset := fixedSize
	binary.LittleEndian.PutUint32(buf[pos:pos+4], uint32(extraDataOffset))
	pos += 4

	// base_fee_per_gas (uint256, 32 bytes LE)
	copy(buf[pos:pos+32], uint256ToSSZBytes(ep.BaseFeePerGas))
	pos += 32

	copy(buf[pos:pos+32], ep.BlockHash[:])
	pos += 32

	// transactions offset
	txOffset := extraDataOffset + len(extraData)
	binary.LittleEndian.PutUint32(buf[pos:pos+4], uint32(txOffset))
	pos += 4

	if version >= 2 {
		wdOffset := txOffset + len(txData)
		binary.LittleEndian.PutUint32(buf[pos:pos+4], uint32(wdOffset))
		pos += 4
	}

	if version >= 3 {
		var blobGasUsed, excessBlobGas uint64
		if ep.BlobGasUsed != nil {
			blobGasUsed = *ep.BlobGasUsed
		}
		if ep.ExcessBlobGas != nil {
			excessBlobGas = *ep.ExcessBlobGas
		}
		binary.LittleEndian.PutUint64(buf[pos:pos+8], blobGasUsed)
		pos += 8
		binary.LittleEndian.PutUint64(buf[pos:pos+8], excessBlobGas)
		pos += 8
	}

	if version >= 4 {
		var slotNumber uint64
		if ep.SlotNumber != nil {
			slotNumber = *ep.SlotNumber
		}
		binary.LittleEndian.PutUint64(buf[pos:pos+8], slotNumber)
		pos += 8
		// Note: For V4 we'd have block_access_list offset here, but Geth doesn't have that field.
		// We write the end of data as the offset (no block_access_list data).
		balOffset := extraDataOffset + len(extraData) + len(txData)
		if version >= 2 {
			balOffset += len(withdrawalData)
		}
		binary.LittleEndian.PutUint32(buf[pos:pos+4], uint32(balOffset))
		pos += 4
	}

	// Variable part
	copy(buf[extraDataOffset:], extraData)
	copy(buf[txOffset:], txData)
	if version >= 2 {
		wdOffset := txOffset + len(txData)
		copy(buf[wdOffset:], withdrawalData)
	}

	return buf
}

// DecodeExecutableDataSSZ decodes SSZ bytes into an ExecutableData.
func DecodeExecutableDataSSZ(buf []byte, version int) (*ExecutableData, error) {
	fixedSize := executionPayloadFixedSize(version)
	if len(buf) < fixedSize {
		return nil, fmt.Errorf("ExecutableData SSZ: buffer too short (%d < %d)", len(buf), fixedSize)
	}

	ep := &ExecutableData{}
	pos := 0

	copy(ep.ParentHash[:], buf[pos:pos+32])
	pos += 32
	copy(ep.FeeRecipient[:], buf[pos:pos+20])
	pos += 20
	copy(ep.StateRoot[:], buf[pos:pos+32])
	pos += 32
	copy(ep.ReceiptsRoot[:], buf[pos:pos+32])
	pos += 32
	ep.LogsBloom = make([]byte, 256)
	copy(ep.LogsBloom, buf[pos:pos+256])
	pos += 256
	copy(ep.Random[:], buf[pos:pos+32])
	pos += 32
	ep.Number = binary.LittleEndian.Uint64(buf[pos : pos+8])
	pos += 8
	ep.GasLimit = binary.LittleEndian.Uint64(buf[pos : pos+8])
	pos += 8
	ep.GasUsed = binary.LittleEndian.Uint64(buf[pos : pos+8])
	pos += 8
	ep.Timestamp = binary.LittleEndian.Uint64(buf[pos : pos+8])
	pos += 8

	extraDataOffset := binary.LittleEndian.Uint32(buf[pos : pos+4])
	pos += 4

	ep.BaseFeePerGas = sszBytesToUint256(buf[pos : pos+32])
	pos += 32

	copy(ep.BlockHash[:], buf[pos:pos+32])
	pos += 32

	txOffset := binary.LittleEndian.Uint32(buf[pos : pos+4])
	pos += 4

	var wdOffset uint32
	if version >= 2 {
		wdOffset = binary.LittleEndian.Uint32(buf[pos : pos+4])
		pos += 4
	}

	if version >= 3 {
		blobGasUsed := binary.LittleEndian.Uint64(buf[pos : pos+8])
		ep.BlobGasUsed = &blobGasUsed
		pos += 8
		excessBlobGas := binary.LittleEndian.Uint64(buf[pos : pos+8])
		ep.ExcessBlobGas = &excessBlobGas
		pos += 8
	}

	var balOffset uint32
	if version >= 4 {
		slotNumber := binary.LittleEndian.Uint64(buf[pos : pos+8])
		ep.SlotNumber = &slotNumber
		pos += 8
		balOffset = binary.LittleEndian.Uint32(buf[pos : pos+4])
		pos += 4
	}

	// Decode variable-length fields
	if extraDataOffset > uint32(len(buf)) || txOffset > uint32(len(buf)) || extraDataOffset > txOffset {
		return nil, fmt.Errorf("ExecutableData SSZ: invalid extra_data/transactions offsets")
	}
	ep.ExtraData = make([]byte, txOffset-extraDataOffset)
	copy(ep.ExtraData, buf[extraDataOffset:txOffset])

	var txEnd uint32
	if version >= 2 {
		txEnd = wdOffset
	} else {
		txEnd = uint32(len(buf))
	}
	if txOffset > txEnd {
		return nil, fmt.Errorf("ExecutableData SSZ: transactions offset > end")
	}

	txBuf := buf[txOffset:txEnd]
	txs, err := decodeTransactionsSSZ(txBuf)
	if err != nil {
		return nil, fmt.Errorf("ExecutableData SSZ: %w", err)
	}
	ep.Transactions = txs
	if ep.Transactions == nil {
		ep.Transactions = [][]byte{}
	}

	if version >= 2 {
		var wdEnd uint32
		if version >= 4 {
			wdEnd = balOffset
		} else {
			wdEnd = uint32(len(buf))
		}
		if wdOffset > wdEnd || wdEnd > uint32(len(buf)) {
			return nil, fmt.Errorf("ExecutableData SSZ: invalid withdrawals offset")
		}
		wds, err := decodeWithdrawalsSSZ(buf[wdOffset:wdEnd])
		if err != nil {
			return nil, fmt.Errorf("ExecutableData SSZ: %w", err)
		}
		ep.Withdrawals = wds
	}

	return ep, nil
}

// --- NewPayload request SSZ ---

// EncodeNewPayloadRequestSSZ encodes a newPayload request to SSZ.
func EncodeNewPayloadRequestSSZ(
	ep *ExecutableData,
	versionedHashes []common.Hash,
	parentBeaconBlockRoot *common.Hash,
	executionRequests [][]byte,
	version int,
) []byte {
	payloadVersion := engineVersionToPayloadVersion(version)
	if version <= 2 {
		return EncodeExecutableDataSSZ(ep, payloadVersion)
	}

	epBytes := EncodeExecutableDataSSZ(ep, payloadVersion)
	blobHashBytes := make([]byte, len(versionedHashes)*32)
	for i, h := range versionedHashes {
		copy(blobHashBytes[i*32:(i+1)*32], h[:])
	}

	if version == 3 {
		fixedSize := 40
		buf := make([]byte, fixedSize+len(epBytes)+len(blobHashBytes))
		binary.LittleEndian.PutUint32(buf[0:4], uint32(fixedSize))
		binary.LittleEndian.PutUint32(buf[4:8], uint32(fixedSize+len(epBytes)))
		if parentBeaconBlockRoot != nil {
			copy(buf[8:40], parentBeaconBlockRoot[:])
		}
		copy(buf[fixedSize:], epBytes)
		copy(buf[fixedSize+len(epBytes):], blobHashBytes)
		return buf
	}

	// V4+
	reqBytes := encodeStructuredExecutionRequestsSSZ(executionRequests)

	fixedSize := 44
	buf := make([]byte, fixedSize+len(epBytes)+len(blobHashBytes)+len(reqBytes))
	binary.LittleEndian.PutUint32(buf[0:4], uint32(fixedSize))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(fixedSize+len(epBytes)))
	if parentBeaconBlockRoot != nil {
		copy(buf[8:40], parentBeaconBlockRoot[:])
	}
	binary.LittleEndian.PutUint32(buf[40:44], uint32(fixedSize+len(epBytes)+len(blobHashBytes)))

	copy(buf[fixedSize:], epBytes)
	copy(buf[fixedSize+len(epBytes):], blobHashBytes)
	copy(buf[fixedSize+len(epBytes)+len(blobHashBytes):], reqBytes)
	return buf
}

// DecodeNewPayloadRequestSSZ decodes a newPayload request from SSZ.
func DecodeNewPayloadRequestSSZ(buf []byte, version int) (
	ep *ExecutableData,
	versionedHashes []common.Hash,
	parentBeaconBlockRoot *common.Hash,
	executionRequests [][]byte,
	err error,
) {
	payloadVersion := engineVersionToPayloadVersion(version)
	if version <= 2 {
		ep, err = DecodeExecutableDataSSZ(buf, payloadVersion)
		return
	}

	if version == 3 {
		if len(buf) < 40 {
			err = fmt.Errorf("NewPayloadV3 SSZ: buffer too short (%d < 40)", len(buf))
			return
		}
		epOffset := binary.LittleEndian.Uint32(buf[0:4])
		blobHashOffset := binary.LittleEndian.Uint32(buf[4:8])
		root := common.BytesToHash(buf[8:40])
		parentBeaconBlockRoot = &root

		if epOffset > uint32(len(buf)) || blobHashOffset > uint32(len(buf)) || epOffset > blobHashOffset {
			err = fmt.Errorf("NewPayloadV3 SSZ: invalid offsets")
			return
		}
		ep, err = DecodeExecutableDataSSZ(buf[epOffset:blobHashOffset], payloadVersion)
		if err != nil {
			return
		}
		blobHashBuf := buf[blobHashOffset:]
		if len(blobHashBuf)%32 != 0 {
			err = fmt.Errorf("NewPayloadV3 SSZ: blob hashes not aligned")
			return
		}
		versionedHashes = make([]common.Hash, len(blobHashBuf)/32)
		for i := range versionedHashes {
			copy(versionedHashes[i][:], blobHashBuf[i*32:(i+1)*32])
		}
		return
	}

	// V4+
	if len(buf) < 44 {
		err = fmt.Errorf("NewPayloadV4 SSZ: buffer too short (%d < 44)", len(buf))
		return
	}
	epOffset := binary.LittleEndian.Uint32(buf[0:4])
	blobHashOffset := binary.LittleEndian.Uint32(buf[4:8])
	root := common.BytesToHash(buf[8:40])
	parentBeaconBlockRoot = &root
	reqOffset := binary.LittleEndian.Uint32(buf[40:44])

	if epOffset > uint32(len(buf)) || blobHashOffset > uint32(len(buf)) || reqOffset > uint32(len(buf)) {
		err = fmt.Errorf("NewPayloadV4 SSZ: offsets out of bounds")
		return
	}
	ep, err = DecodeExecutableDataSSZ(buf[epOffset:blobHashOffset], payloadVersion)
	if err != nil {
		return
	}
	blobHashBuf := buf[blobHashOffset:reqOffset]
	if len(blobHashBuf)%32 != 0 {
		err = fmt.Errorf("NewPayloadV4 SSZ: blob hashes not aligned")
		return
	}
	versionedHashes = make([]common.Hash, len(blobHashBuf)/32)
	for i := range versionedHashes {
		copy(versionedHashes[i][:], blobHashBuf[i*32:(i+1)*32])
	}

	executionRequests, err = decodeStructuredExecutionRequestsSSZ(buf[reqOffset:])
	return
}

// --- Execution Requests SSZ (structured container for Prysm compatibility) ---

func encodeStructuredExecutionRequestsSSZ(reqs [][]byte) []byte {
	var depositsData, withdrawalsData, consolidationsData []byte
	for _, r := range reqs {
		if len(r) < 1 {
			continue
		}
		switch r[0] {
		case 0x00:
			depositsData = append(depositsData, r[1:]...)
		case 0x01:
			withdrawalsData = append(withdrawalsData, r[1:]...)
		case 0x02:
			consolidationsData = append(consolidationsData, r[1:]...)
		}
	}

	fixedSize := 12
	totalVar := len(depositsData) + len(withdrawalsData) + len(consolidationsData)
	buf := make([]byte, fixedSize+totalVar)

	depositsOffset := fixedSize
	withdrawalsOffset := depositsOffset + len(depositsData)
	consolidationsOffset := withdrawalsOffset + len(withdrawalsData)

	binary.LittleEndian.PutUint32(buf[0:4], uint32(depositsOffset))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(withdrawalsOffset))
	binary.LittleEndian.PutUint32(buf[8:12], uint32(consolidationsOffset))

	copy(buf[depositsOffset:], depositsData)
	copy(buf[withdrawalsOffset:], withdrawalsData)
	copy(buf[consolidationsOffset:], consolidationsData)

	return buf
}

func decodeStructuredExecutionRequestsSSZ(buf []byte) ([][]byte, error) {
	if len(buf) == 0 {
		return [][]byte{}, nil
	}
	if len(buf) < 12 {
		return nil, fmt.Errorf("structured execution requests SSZ: buffer too short (%d < 12)", len(buf))
	}

	depositsOffset := binary.LittleEndian.Uint32(buf[0:4])
	withdrawalsOffset := binary.LittleEndian.Uint32(buf[4:8])
	consolidationsOffset := binary.LittleEndian.Uint32(buf[8:12])

	if depositsOffset > uint32(len(buf)) || withdrawalsOffset > uint32(len(buf)) || consolidationsOffset > uint32(len(buf)) {
		return nil, fmt.Errorf("structured execution requests SSZ: offsets out of bounds")
	}
	if depositsOffset > withdrawalsOffset || withdrawalsOffset > consolidationsOffset {
		return nil, fmt.Errorf("structured execution requests SSZ: offsets not in order")
	}

	reqs := make([][]byte, 0, 3)

	depositsData := buf[depositsOffset:withdrawalsOffset]
	if len(depositsData) > 0 {
		r := make([]byte, 1+len(depositsData))
		r[0] = 0x00
		copy(r[1:], depositsData)
		reqs = append(reqs, r)
	}

	withdrawalsData := buf[withdrawalsOffset:consolidationsOffset]
	if len(withdrawalsData) > 0 {
		r := make([]byte, 1+len(withdrawalsData))
		r[0] = 0x01
		copy(r[1:], withdrawalsData)
		reqs = append(reqs, r)
	}

	consolidationsData := buf[consolidationsOffset:]
	if len(consolidationsData) > 0 {
		r := make([]byte, 1+len(consolidationsData))
		r[0] = 0x02
		copy(r[1:], consolidationsData)
		reqs = append(reqs, r)
	}

	return reqs, nil
}

// --- GetPayload response SSZ ---

const getPayloadResponseFixedSize = 45

// EncodeExecutionPayloadEnvelopeSSZ encodes a GetPayload response to SSZ.
func EncodeExecutionPayloadEnvelopeSSZ(resp *ExecutionPayloadEnvelope, version int) []byte {
	if version == 1 {
		return EncodeExecutableDataSSZ(resp.ExecutionPayload, 1)
	}

	payloadVersion := engineVersionToPayloadVersion(version)
	epBytes := EncodeExecutableDataSSZ(resp.ExecutionPayload, payloadVersion)
	blobsBytes := encodeBlobsBundleSSZ(resp.BlobsBundle)
	reqBytes := encodeStructuredExecutionRequestsSSZ(resp.Requests)

	buf := make([]byte, getPayloadResponseFixedSize+len(epBytes)+len(blobsBytes)+len(reqBytes))

	// ep offset
	binary.LittleEndian.PutUint32(buf[0:4], uint32(getPayloadResponseFixedSize))

	// block_value (uint256 LE)
	if resp.BlockValue != nil {
		copy(buf[4:36], uint256ToSSZBytes(resp.BlockValue))
	}

	// blobs_bundle offset
	blobsOffset := getPayloadResponseFixedSize + len(epBytes)
	binary.LittleEndian.PutUint32(buf[36:40], uint32(blobsOffset))

	// should_override_builder
	if resp.Override {
		buf[40] = 1
	}

	// execution_requests offset
	reqOffset := blobsOffset + len(blobsBytes)
	binary.LittleEndian.PutUint32(buf[41:45], uint32(reqOffset))

	// Variable data
	copy(buf[getPayloadResponseFixedSize:], epBytes)
	copy(buf[blobsOffset:], blobsBytes)
	copy(buf[reqOffset:], reqBytes)

	return buf
}

// DecodeExecutionPayloadEnvelopeSSZ decodes SSZ bytes into an ExecutionPayloadEnvelope.
func DecodeExecutionPayloadEnvelopeSSZ(buf []byte, version int) (*ExecutionPayloadEnvelope, error) {
	if version == 1 {
		ep, err := DecodeExecutableDataSSZ(buf, 1)
		if err != nil {
			return nil, err
		}
		return &ExecutionPayloadEnvelope{ExecutionPayload: ep}, nil
	}

	if len(buf) < getPayloadResponseFixedSize {
		return nil, fmt.Errorf("ExecutionPayloadEnvelope SSZ: buffer too short (%d < %d)", len(buf), getPayloadResponseFixedSize)
	}

	resp := &ExecutionPayloadEnvelope{}

	epOffset := binary.LittleEndian.Uint32(buf[0:4])
	resp.BlockValue = sszBytesToUint256(buf[4:36])
	blobsOffset := binary.LittleEndian.Uint32(buf[36:40])
	resp.Override = buf[40] == 1
	reqOffset := binary.LittleEndian.Uint32(buf[41:45])

	if epOffset > uint32(len(buf)) || blobsOffset > uint32(len(buf)) {
		return nil, fmt.Errorf("ExecutionPayloadEnvelope SSZ: offsets out of bounds")
	}
	payloadVersion := engineVersionToPayloadVersion(version)
	ep, err := DecodeExecutableDataSSZ(buf[epOffset:blobsOffset], payloadVersion)
	if err != nil {
		return nil, err
	}
	resp.ExecutionPayload = ep

	if blobsOffset > reqOffset || reqOffset > uint32(len(buf)) {
		return nil, fmt.Errorf("ExecutionPayloadEnvelope SSZ: invalid blobs/requests offsets")
	}
	bundle, err := decodeBlobsBundleSSZ(buf[blobsOffset:reqOffset])
	if err != nil {
		return nil, err
	}
	resp.BlobsBundle = bundle

	if reqOffset < uint32(len(buf)) {
		reqs, err := decodeStructuredExecutionRequestsSSZ(buf[reqOffset:])
		if err != nil {
			return nil, err
		}
		resp.Requests = reqs
	}

	return resp, nil
}

// --- BlobsBundle SSZ ---

const blobsBundleFixedSize = 12

func encodeBlobsBundleSSZ(bundle *BlobsBundle) []byte {
	if bundle == nil {
		return nil
	}

	commitmentsData := encodeFixedSizeList(bundle.Commitments)
	proofsData := encodeFixedSizeList(bundle.Proofs)
	blobsData := encodeFixedSizeList(bundle.Blobs)

	totalVar := len(commitmentsData) + len(proofsData) + len(blobsData)
	buf := make([]byte, blobsBundleFixedSize+totalVar)

	commitmentsOffset := blobsBundleFixedSize
	proofsOffset := commitmentsOffset + len(commitmentsData)
	blobsOffset := proofsOffset + len(proofsData)

	binary.LittleEndian.PutUint32(buf[0:4], uint32(commitmentsOffset))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(proofsOffset))
	binary.LittleEndian.PutUint32(buf[8:12], uint32(blobsOffset))

	copy(buf[commitmentsOffset:], commitmentsData)
	copy(buf[proofsOffset:], proofsData)
	copy(buf[blobsOffset:], blobsData)

	return buf
}

func decodeBlobsBundleSSZ(buf []byte) (*BlobsBundle, error) {
	if len(buf) == 0 {
		return nil, nil
	}
	if len(buf) < blobsBundleFixedSize {
		return nil, fmt.Errorf("BlobsBundle SSZ: buffer too short")
	}

	commitmentsOffset := binary.LittleEndian.Uint32(buf[0:4])
	proofsOffset := binary.LittleEndian.Uint32(buf[4:8])
	blobsOffset := binary.LittleEndian.Uint32(buf[8:12])

	if commitmentsOffset > uint32(len(buf)) || proofsOffset > uint32(len(buf)) || blobsOffset > uint32(len(buf)) {
		return nil, fmt.Errorf("BlobsBundle SSZ: offsets out of bounds")
	}

	bundle := &BlobsBundle{}

	commBuf := buf[commitmentsOffset:proofsOffset]
	if len(commBuf) > 0 {
		if len(commBuf)%48 != 0 {
			return nil, fmt.Errorf("BlobsBundle SSZ: commitments not aligned to 48 bytes")
		}
		bundle.Commitments = make([]hexutil.Bytes, len(commBuf)/48)
		for i := range bundle.Commitments {
			c := make(hexutil.Bytes, 48)
			copy(c, commBuf[i*48:(i+1)*48])
			bundle.Commitments[i] = c
		}
	}

	proofBuf := buf[proofsOffset:blobsOffset]
	if len(proofBuf) > 0 {
		if len(proofBuf)%48 != 0 {
			return nil, fmt.Errorf("BlobsBundle SSZ: proofs not aligned to 48 bytes")
		}
		bundle.Proofs = make([]hexutil.Bytes, len(proofBuf)/48)
		for i := range bundle.Proofs {
			p := make(hexutil.Bytes, 48)
			copy(p, proofBuf[i*48:(i+1)*48])
			bundle.Proofs[i] = p
		}
	}

	blobBuf := buf[blobsOffset:]
	if len(blobBuf) > 0 {
		if len(blobBuf)%131072 != 0 {
			return nil, fmt.Errorf("BlobsBundle SSZ: blobs not aligned to 131072 bytes")
		}
		bundle.Blobs = make([]hexutil.Bytes, len(blobBuf)/131072)
		for i := range bundle.Blobs {
			b := make(hexutil.Bytes, 131072)
			copy(b, blobBuf[i*131072:(i+1)*131072])
			bundle.Blobs[i] = b
		}
	}

	return bundle, nil
}

func encodeFixedSizeList(items []hexutil.Bytes) []byte {
	totalLen := 0
	for _, item := range items {
		totalLen += len(item)
	}
	buf := make([]byte, totalLen)
	pos := 0
	for _, item := range items {
		copy(buf[pos:], item)
		pos += len(item)
	}
	return buf
}

// --- GetBlobs request SSZ ---

// EncodeGetBlobsRequestSSZ encodes a list of versioned hashes for the get_blobs SSZ request.
func EncodeGetBlobsRequestSSZ(hashes []common.Hash) []byte {
	buf := make([]byte, 4+len(hashes)*32)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(len(hashes)))
	for i, h := range hashes {
		copy(buf[4+i*32:4+(i+1)*32], h[:])
	}
	return buf
}

// DecodeGetBlobsRequestSSZ decodes a list of versioned hashes from SSZ bytes.
func DecodeGetBlobsRequestSSZ(buf []byte) ([]common.Hash, error) {
	if len(buf) < 4 {
		return nil, fmt.Errorf("GetBlobsRequest: buffer too short")
	}
	count := binary.LittleEndian.Uint32(buf[0:4])
	if 4+count*32 > uint32(len(buf)) {
		return nil, fmt.Errorf("GetBlobsRequest: buffer too short for %d hashes", count)
	}
	hashes := make([]common.Hash, count)
	for i := uint32(0); i < count; i++ {
		copy(hashes[i][:], buf[4+i*32:4+(i+1)*32])
	}
	return hashes, nil
}

// --- PayloadAttributes SSZ ---

// DecodePayloadAttributesSSZ decodes PayloadAttributes from SSZ bytes.
func DecodePayloadAttributesSSZ(buf []byte, version int) (*PayloadAttributes, error) {
	if len(buf) < 60 {
		return nil, fmt.Errorf("PayloadAttributes: buffer too short (%d < 60)", len(buf))
	}

	timestamp := binary.LittleEndian.Uint64(buf[0:8])
	pa := &PayloadAttributes{
		Timestamp:             timestamp,
		SuggestedFeeRecipient: common.BytesToAddress(buf[40:60]),
	}
	copy(pa.Random[:], buf[8:40])

	if version == 1 {
		return pa, nil
	}

	if len(buf) < 64 {
		return nil, fmt.Errorf("PayloadAttributes V2+: buffer too short (%d < 64)", len(buf))
	}
	withdrawalsOffset := binary.LittleEndian.Uint32(buf[60:64])

	if version >= 3 {
		if len(buf) < 96 {
			return nil, fmt.Errorf("PayloadAttributes V3: buffer too short (%d < 96)", len(buf))
		}
		root := common.BytesToHash(buf[64:96])
		pa.BeaconRoot = &root
	}

	if withdrawalsOffset <= uint32(len(buf)) {
		wdBuf := buf[withdrawalsOffset:]
		if len(wdBuf) > 0 {
			if len(wdBuf)%44 != 0 {
				return nil, fmt.Errorf("PayloadAttributes: withdrawals buffer length %d not divisible by 44", len(wdBuf))
			}
			count := len(wdBuf) / 44
			pa.Withdrawals = make([]*types.Withdrawal, count)
			for i := 0; i < count; i++ {
				off := i * 44
				w := &types.Withdrawal{
					Index:     binary.LittleEndian.Uint64(wdBuf[off : off+8]),
					Validator: binary.LittleEndian.Uint64(wdBuf[off+8 : off+16]),
					Amount:    binary.LittleEndian.Uint64(wdBuf[off+36 : off+44]),
				}
				copy(w.Address[:], wdBuf[off+16:off+36])
				pa.Withdrawals[i] = w
			}
		} else {
			pa.Withdrawals = []*types.Withdrawal{}
		}
	}

	return pa, nil
}
