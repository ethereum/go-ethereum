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
	dst = buf
	dst = append(dst, byte(p.Selector))
	if p.Selector != none {
		if len(p.Proof) != 15 {
			err = ssz.ErrBytesLengthFn("proofs size should be", len(p.Proof), 15)
			return
		}
		for _, item := range p.Proof {
			if len(item) != 32 {
				err = ssz.ErrBytesLengthFn("single proof size should be", len(item), 32)
				return
			}
			dst = append(dst, item...)
		}
	}
	return
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

	p.Proof = proof
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

func (p *BlockHeaderProof) HashTreeRootWith(_ ssz.HashWalker) (err error) {
	panic("implement me")
}

type PortalReceipts struct {
	Receipts [][]byte `ssz-max:"16384,134217728"`
}

// MarshalSSZ ssz marshals the PortalReceipts object
func (p *PortalReceipts) MarshalSSZ() ([]byte, error) {
	return ssz.MarshalSSZ(p)
}

// MarshalSSZTo ssz marshals the PortalReceipts object to a target array
func (p *PortalReceipts) MarshalSSZTo(buf []byte) (dst []byte, err error) {
	dst = buf
	// Field (0) 'Receipts'
	if size := len(p.Receipts); size > 16384 {
		err = ssz.ErrListTooBigFn("PortalReceipts.Receipts", size, 16384)
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
		if size := len(p.Receipts[ii]); size > 134217728 {
			err = ssz.ErrBytesLengthFn("PortalReceipts.Receipts[ii]", size, 134217728)
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
		num, err := ssz.DecodeDynamicLength(buf, 16384)
		if err != nil {
			return err
		}
		p.Receipts = make([][]byte, num)
		err = ssz.UnmarshalDynamic(buf, num, func(indx int, buf []byte) (err error) {
			if len(buf) > 134217728 {
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

// HashTreeRoot ssz hashes the PortalReceipts object
func (p *PortalReceipts) HashTreeRoot() ([32]byte, error) {
	return ssz.HashWithDefaultHasher(p)
}

// HashTreeRootWith ssz hashes the PortalReceipts object with a hasher
func (p *PortalReceipts) HashTreeRootWith(hh ssz.HashWalker) (err error) {
	indx := hh.Index()

	// Field (0) 'Receipts'
	{
		subIndx := hh.Index()
		num := uint64(len(p.Receipts))
		if num > 16384 {
			err = ssz.ErrIncorrectListSize
			return
		}
		for _, elem := range p.Receipts {
			{
				elemIndx := hh.Index()
				byteLen := uint64(len(elem))
				if byteLen > 134217728 {
					err = ssz.ErrIncorrectListSize
					return
				}
				hh.AppendBytes32(elem)
				hh.MerkleizeWithMixin(elemIndx, byteLen, (134217728+31)/32)
			}
		}
		hh.MerkleizeWithMixin(subIndx, num, 16384)
	}

	hh.Merkleize(indx)
	return
}

// GetTree ssz hashes the PortalReceipts object
func (p *PortalReceipts) GetTree() (*ssz.Node, error) {
	return ssz.ProofTree(p)
}
