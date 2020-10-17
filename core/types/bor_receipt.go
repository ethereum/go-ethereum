package types

import (
	"io"
	"math/big"

	"github.com/maticnetwork/bor/common"
	"github.com/maticnetwork/bor/rlp"
)

// BorReceipt represents the results of a block state syncs
type BorReceipt struct {
	// Consensus fields
	Bloom Bloom  `json:"logsBloom"         gencodec:"required"`
	Logs  []*Log `json:"logs"              gencodec:"required"`

	// Inclusion information: These fields provide information about the inclusion of the
	// transaction corresponding to this receipt.
	BlockHash   common.Hash `json:"blockHash,omitempty"`
	BlockNumber *big.Int    `json:"blockNumber,omitempty"`
}

// borReceiptRLP is the consensus encoding of a block receipt.
type borReceiptRLP struct {
	Bloom Bloom
	Logs  []*Log
}

// storedBorReceiptRLP is the storage encoding of a block receipt.
type storedBorReceiptRLP struct {
	Logs []*LogForStorage
}

// EncodeRLP implements rlp.Encoder, and flattens the consensus fields of a block receipt
// into an RLP stream. If no post state is present, byzantium fork is assumed.
func (r *BorReceipt) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &borReceiptRLP{r.Bloom, r.Logs})
}

// DecodeRLP implements rlp.Decoder, and loads the consensus fields of a block receipt
// from an RLP stream.
func (r *BorReceipt) DecodeRLP(s *rlp.Stream) error {
	var dec receiptRLP
	if err := s.Decode(&dec); err != nil {
		return err
	}
	r.Bloom, r.Logs = dec.Bloom, dec.Logs
	return nil
}

// BorReceiptForStorage is a wrapper around a Bor Receipt that flattens and parses the
// entire content of a receipt, as opposed to only the consensus fields originally.
type BorReceiptForStorage BorReceipt

// EncodeRLP implements rlp.Encoder, and flattens all content fields of a receipt
// into an RLP stream.
func (r *BorReceiptForStorage) EncodeRLP(w io.Writer) error {
	enc := &storedBorReceiptRLP{
		Logs: make([]*LogForStorage, len(r.Logs)),
	}

	for i, log := range r.Logs {
		enc.Logs[i] = (*LogForStorage)(log)
	}
	return rlp.Encode(w, enc)
}

// DecodeRLP implements rlp.Decoder, and loads both consensus and implementation
// fields of a receipt from an RLP stream.
func (r *BorReceiptForStorage) DecodeRLP(s *rlp.Stream) error {
	// Retrieve the entire receipt blob as we need to try multiple decoders
	blob, err := s.Raw()
	if err != nil {
		return err
	}

	return decodeStoredBorReceiptRLP(r, blob)
}

func decodeStoredBorReceiptRLP(r *BorReceiptForStorage, blob []byte) error {
	var stored storedBorReceiptRLP
	if err := rlp.DecodeBytes(blob, &stored); err != nil {
		return err
	}

	r.Logs = make([]*Log, len(stored.Logs))
	for i, log := range stored.Logs {
		r.Logs[i] = (*Log)(log)
	}
	r.Bloom = BytesToBloom(LogsBloom(r.Logs).Bytes())

	return nil
}

// DeriveFields fills the receipts with their computed fields based on consensus
// data and contextual infos like containing block and transactions.
func (r *BorReceipt) DeriveFields(hash common.Hash, number uint64) error {
	// txHash := common.BytesToHash(crypto.Keccak256(append([]byte("bor-receipt-"), hash.Bytes()...)))

	// The derived log fields can simply be set from the block and transaction
	for j := 0; j < len(r.Logs); j++ {
		r.Logs[j].BlockNumber = number
		r.Logs[j].BlockHash = hash
		// r.Logs[j].TxHash = txHash
		r.Logs[j].TxIndex = uint(0)
		r.Logs[j].Index = uint(j)
	}
	return nil
}
