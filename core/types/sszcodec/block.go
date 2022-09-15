package sszcodec

import (
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
)

type Block struct {
	Header       *Header    `ssz-max:"604"`
	Transactions [][]byte   `ssz-max:"1048576,1073741824" ssz-size:"?,?"`
	Uncles       []*Header  `ssz-max:"6040"`
	Receipts     []*Receipt `ssz-max:"4194452"`
}

type Header struct {
	ParentHash    []byte `ssz-size:"32"`
	UncleHash     []byte `ssz-size:"32"`
	FeeRecipient  []byte `ssz-size:"20"` // 84
	StateRoot     []byte `ssz-size:"32"`
	TxHash        []byte `ssz-size:"32"`
	ReceiptsRoot  []byte `ssz-size:"32"`  // 180
	LogsBloom     []byte `ssz-size:"256"` // 436
	Difficulty    []byte `ssz-size:"32"`
	BlockNumber   uint64
	GasLimit      uint64
	GasUsed       uint64
	Timestamp     uint64 // 500
	ExtraData     []byte `ssz-max:"32"`
	BaseFeePerGas []byte `ssz-size:"32"`
	MixDigest     []byte `ssz-size:"32"`
	Nonce         []byte `ssz-size:"8"` // 604
	//	BlockHash     []byte   `ssz-size:"32"`
}

type Receipt struct {
	PostState         []byte `ssz-max:"32"`
	Status            uint64
	CumulativeGasUsed uint64
	Logs              []*Log `ssz-max:"4194452"` // xxx
}

type Log struct {
	Address []byte   `ssz-size:"20"`
	Topics  [][]byte `ssz-max:"4" ssz-size:"?,32"` // 148
	Data    []byte   `ssz-max:"4194304"`           // 4194452
}

func FromBlock() *Block {
	return &Block{}
}

func FromHeader(h *types.Header) (*Header, error) {
	sh := &Header{}
	sh.ParentHash = h.ParentHash[:]
	sh.UncleHash = h.UncleHash[:]
	sh.FeeRecipient = h.Coinbase[:]
	sh.StateRoot = h.Root[:]
	sh.TxHash = h.TxHash[:]
	sh.ReceiptsRoot = h.ReceiptHash[:]
	sh.LogsBloom = h.Bloom[:]

	sh.Difficulty = make([]byte, 32)
	h.Difficulty.FillBytes(sh.Difficulty)

	sh.BlockNumber = h.Number.Uint64()
	sh.GasLimit = h.GasLimit
	sh.GasUsed = h.GasUsed
	sh.Timestamp = h.Time

	if len(h.Extra) > 32 {
		return nil, fmt.Errorf("invalid extradata length in block %d: %v", sh.BlockNumber, len(h.Extra))
	}
	sh.ExtraData = h.Extra

	sh.BaseFeePerGas = make([]byte, 32)
	if h.BaseFee != nil {
		h.BaseFee.FillBytes(sh.BaseFeePerGas)
	}
	sh.MixDigest = h.MixDigest[:]
	sh.Nonce = h.Nonce[:]
	//	e.BlockHash = make([]byte, 32)
	return sh, nil
}

func FillBlock(sb *Block, b types.Block) error {
	eh, err := FromHeader(b.Header())
	if err != nil {
		return err
	}
	sb.Header = eh

	txs := b.Transactions()
	for i := 0; i < len(txs); i++ {
		b, err := txs[i].MarshalBinary()
		if err != nil {
			return err
		}
		sb.Transactions = append(sb.Transactions, b)
	}
	uncles := b.Uncles()
	for i := 0; i < len(uncles); i++ {
		eh, err := FromHeader(uncles[i])
		if err != nil {
			return err
		}
		sb.Uncles = append(sb.Uncles, eh)
	}
	return nil
}

func FillReceipts(sb *Block, receipts []*types.Receipt) {
	for i := 0; i < len(receipts); i++ {
		p := &Receipt{}
		if len(receipts[i].PostState) > 0 {
			p.PostState = receipts[i].PostState
		} else {
			p.Status = receipts[i].Status
		}
		p.CumulativeGasUsed = receipts[i].CumulativeGasUsed
		for _, rlplog := range receipts[i].Logs {
			log := &Log{Address: rlplog.Address[:], Data: rlplog.Data}
			for j := 0; j < len(rlplog.Topics); j++ {
				topic := rlplog.Topics[j]
				// xxx ugly conversion from []common.Hash to [][]byte...
				// maybe just common.Hash directly? (here and elsewhere)
				log.Topics = append(log.Topics, []byte(topic[:]))
			}
			p.Logs = append(p.Logs, log)
		}
		sb.Receipts = append(sb.Receipts, p)
	}
}
