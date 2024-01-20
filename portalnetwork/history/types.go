package history

import (
	"errors"

	ssz "github.com/ferranbt/fastssz"
)

//go:generate sszgen --path types.go --exclude-objs BlockHeaderProof,PortalReceipts

type BlockHeaderProofType uint8

const (
	none             BlockHeaderProofType = 0
	accumulatorProof BlockHeaderProofType = 1
)

type HeaderRecord struct {
	BlockHash       []byte `ssz-size:"32"`
	TotalDifficulty []byte `ssz-size:"32"`
}
type EpochAccumulator struct {
	HeaderRecords [][]byte `ssz-size:"8192,64"`
}
type BlockBodyLegacy struct {
	Transactions [][]byte `ssz-max:"16384,16777216"`
	Uncles       []byte   `ssz-max:"131072"`
}

type PortalBlockBodyShanghai struct {
	Transactions [][]byte `ssz-max:"16384,16777216"`
	Uncles       []byte   `ssz-max:"131072"`
	Withdrawals  [][]byte `ssz-max:"16,192"`
}

type BlockHeaderWithProof struct {
	Header []byte            `ssz-max:"8192"`
	Proof  *BlockHeaderProof `ssz-max:"512"`
}

type SSZProof struct {
	Leaf      []byte   `ssz-size:"32"`
	Witnesses [][]byte `ssz-max:"65536,32" ssz-size:"?,32"`
}

type MasterAccumulator struct {
	HistoricalEpochs [][]byte `ssz-max:"1897,32" ssz-size:"?,32"`
}

// BlockHeaderProof is a ssz union type
// Union[None, AccumulatorProof]
type BlockHeaderProof struct {
	Selector BlockHeaderProofType
	Proof    [][]byte `ssz-size:"15,32"`
}

func (p *BlockHeaderProof) MarshalSSZ() ([]byte, error) {
	return ssz.MarshalSSZ(p)
}

func (p *BlockHeaderProof) MarshalSSZTo(buf []byte) (dst []byte, err error) {
	return ssz.MarshalSSZ(p)
}

func (p *BlockHeaderProof) UnmarshalSSZ(buf []byte) (err error) {
	p.Selector = BlockHeaderProofType(buf[0])
	if p.Selector == none {
		return
	}

	if p.Selector != accumulatorProof {
		return errors.New("unknown accumulatorProofType, shoud be 0x00 or 0x01")
	}

	proofBytes := buf[1:]

	if len(proofBytes) != 32*15 {
		return ssz.ErrBytesLengthFn("AccumulatorProof", len(proofBytes), 32*15)
	}
	proof := make([][]byte, 15)

	for i := 0; i < 15; i++ {
		proof[i] = proofBytes[i*32 : (i+1)*32]
	}

	p.Proof = AccumulatorProof(proof)
	return
}

func (p *BlockHeaderProof) SizeSSZ() (size int) {
	size = 0

	// Field (0) 'Selector'
	size += 1

	if p.Selector == none {
		return size
	}

	// Field (1) 'Proof'
	size += 15 * 32

	return size
}

func (p *BlockHeaderProof) HashTreeRootWith(hh ssz.HashWalker) (err error) {
	panic("implement me")
}

type PortalReceipts struct {
	Receipts [][]byte `ssz-max:"134217728,16384"`
}

// MarshalSSZ ssz marshals the PortalReceipts object
func (p *PortalReceipts) MarshalSSZ() ([]byte, error) {
	return ssz.MarshalSSZ(p)
}

// MarshalSSZTo ssz marshals the PortalReceipts object to a target array
func (p *PortalReceipts) MarshalSSZTo(buf []byte) (dst []byte, err error) {
	dst = buf

	if size := len(p.Receipts); size > 134217728 {
		err = ssz.ErrListTooBigFn("PortalReceipts.Receipts", size, 134217728)
		return
	}
	{
		offset := 4 * len(p.Receipts)
		for ii := 0; ii < len(p.Receipts); ii++ {
			dst = ssz.WriteOffset(dst, offset)
			offset += len(p.Receipts[ii])
		}
	}
	for ii := 0; ii < len(p.Receipts); ii++ {
		if size := len(p.Receipts[ii]); size > 16384 {
			err = ssz.ErrBytesLengthFn("PortalReceipts.Receipts[ii]", size, 16384)
			return
		}
		dst = append(dst, p.Receipts[ii]...)
	}

	return
}

// UnmarshalSSZ ssz unmarshals the PortalReceipts object
func (p *PortalReceipts) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 4 {
		return ssz.ErrSize
	}
	// Field (0) 'Receipts'
	{
		num, err := ssz.DecodeDynamicLength(buf, 134217728)
		if err != nil {
			return err
		}
		p.Receipts = make([][]byte, num)
		err = ssz.UnmarshalDynamic(buf, num, func(indx int, buf []byte) (err error) {
			if len(buf) > 16384 {
				return ssz.ErrBytesLength
			}
			if cap(p.Receipts[indx]) == 0 {
				p.Receipts[indx] = make([]byte, 0, len(buf))
			}
			p.Receipts[indx] = append(p.Receipts[indx], buf...)
			return nil
		})
		if err != nil {
			return err
		}
	}
	return err
}

// SizeSSZ returns the ssz encoded size in bytes for the PortalReceipts object
func (p *PortalReceipts) SizeSSZ() (size int) {
	size = 0

	// Field (0) 'Receipts'
	for ii := 0; ii < len(p.Receipts); ii++ {
		size += 4
		size += len(p.Receipts[ii])
	}

	return
}

// HashTreeRootWith ssz hashes the PortalReceipts object with a hasher
func (p *PortalReceipts) HashTreeRootWith(hh ssz.HashWalker) (err error) {
	panic("implement me")
}
