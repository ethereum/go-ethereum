package history

import (
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/ztyp/codec"
	"github.com/protolambda/ztyp/tree"
	"github.com/protolambda/ztyp/view"
)

const beaconBlockBodyProofLen = 8

type BeaconBlockBodyProof [beaconBlockBodyProofLen]common.Root

func (b *BeaconBlockBodyProof) Deserialize(dr *codec.DecodingReader) error {
	roots := b[:]
	return tree.ReadRoots(dr, &roots, beaconBlockBodyProofLen)
}

func (b *BeaconBlockBodyProof) Serialize(w *codec.EncodingWriter) error {
	return tree.WriteRoots(w, b[:])
}

func (b BeaconBlockBodyProof) ByteLength() (out uint64) {
	return beaconBlockBodyProofLen * 32
}

func (b BeaconBlockBodyProof) FixedLength() uint64 {
	return beaconBlockBodyProofLen * 32
}

func (b *BeaconBlockBodyProof) HashTreeRoot(hFn tree.HashFn) common.Root {
	return hFn.ComplexVectorHTR(func(i uint64) tree.HTR {
		if i < beaconBlockBodyProofLen {
			return &b[i]
		}
		return nil
	}, beaconBlockBodyProofLen)
}

const beaconBlockHeaderProofLen = 3

type BeaconBlockHeaderProof [beaconBlockHeaderProofLen]common.Root

func (b *BeaconBlockHeaderProof) Deserialize(dr *codec.DecodingReader) error {
	roots := b[:]
	return tree.ReadRoots(dr, &roots, beaconBlockHeaderProofLen)
}

func (b *BeaconBlockHeaderProof) Serialize(w *codec.EncodingWriter) error {
	return tree.WriteRoots(w, b[:])
}

func (b BeaconBlockHeaderProof) ByteLength() (out uint64) {
	return beaconBlockHeaderProofLen * 32
}

func (b BeaconBlockHeaderProof) FixedLength() uint64 {
	return beaconBlockHeaderProofLen * 32
}

func (b *BeaconBlockHeaderProof) HashTreeRoot(hFn tree.HashFn) common.Root {
	return hFn.ComplexVectorHTR(func(i uint64) tree.HTR {
		if i < beaconBlockHeaderProofLen {
			return &b[i]
		}
		return nil
	}, beaconBlockHeaderProofLen)
}

const historicalRootsProofLen = 14

type HistoricalRootsProof [historicalRootsProofLen]common.Root

func (b *HistoricalRootsProof) Deserialize(dr *codec.DecodingReader) error {
	roots := b[:]
	return tree.ReadRoots(dr, &roots, historicalRootsProofLen)
}

func (b *HistoricalRootsProof) Serialize(w *codec.EncodingWriter) error {
	return tree.WriteRoots(w, b[:])
}

func (b HistoricalRootsProof) ByteLength() (out uint64) {
	return historicalRootsProofLen * 32
}

func (b HistoricalRootsProof) FixedLength() uint64 {
	return historicalRootsProofLen * 32
}

func (b *HistoricalRootsProof) HashTreeRoot(hFn tree.HashFn) common.Root {
	return hFn.ComplexVectorHTR(func(i uint64) tree.HTR {
		if i < historicalRootsProofLen {
			return &b[i]
		}
		return nil
	}, historicalRootsProofLen)
}

type HistoricalRootsBlockProof struct {
	BeaconBlockBodyProof   BeaconBlockBodyProof   `yaml:"beacon_block_body_proof" json:"beacon_block_body_proof"`
	BeaconBlockBodyRoot    common.Root            `yaml:"beacon_block_body_root" json:"beacon_block_body_root"`
	BeaconBlockHeaderProof BeaconBlockHeaderProof `yaml:"beacon_block_header_proof" json:"beacon_block_header_proof"`
	BeaconBlockHeaderRoot  common.Root            `yaml:"beacon_block_header_root" json:"beacon_block_header_root"`
	HistoricalRootsProof   HistoricalRootsProof   `yaml:"historical_roots_proof" json:"historical_roots_proof"`
	Slot                   common.Slot            `yaml:"slot" json:"slot"`
}

func (h *HistoricalRootsBlockProof) Deserialize(dr *codec.DecodingReader) error {
	return dr.FixedLenContainer(
		&h.BeaconBlockBodyProof,
		&h.BeaconBlockBodyRoot,
		&h.BeaconBlockHeaderProof,
		&h.BeaconBlockHeaderProof,
		&h.HistoricalRootsProof,
		&h.Slot,
	)
}

func (h *HistoricalRootsBlockProof) Serialize(w *codec.EncodingWriter) error {
	return w.FixedLenContainer(
		&h.BeaconBlockBodyProof,
		&h.BeaconBlockBodyRoot,
		&h.BeaconBlockHeaderProof,
		&h.BeaconBlockHeaderProof,
		&h.HistoricalRootsProof,
		&h.Slot,
	)
}

func (h *HistoricalRootsBlockProof) ByteLength(spec *common.Spec) uint64 {
	return codec.ContainerLength(
		&h.BeaconBlockBodyProof,
		&h.BeaconBlockBodyRoot,
		&h.BeaconBlockHeaderProof,
		&h.BeaconBlockHeaderProof,
		&h.HistoricalRootsProof,
		&h.Slot,
	)
}

func (h *HistoricalRootsBlockProof) FixedLength(spec *common.Spec) uint64 {
	return codec.ContainerLength(
		&h.BeaconBlockBodyProof,
		&h.BeaconBlockBodyRoot,
		&h.BeaconBlockHeaderProof,
		&h.BeaconBlockHeaderProof,
		&h.HistoricalRootsProof,
		&h.Slot,
	)
}

func (h *HistoricalRootsBlockProof) HashTreeRoot(spec *common.Spec, hFn tree.HashFn) common.Root {
	return hFn.HashTreeRoot(
		&h.BeaconBlockBodyProof,
		&h.BeaconBlockBodyRoot,
		&h.BeaconBlockHeaderProof,
		&h.BeaconBlockHeaderProof,
		&h.HistoricalRootsProof,
		&h.Slot,
	)
}

type HistoricalRoots []common.Root

func (h *HistoricalRoots) Deserialize(spec *common.Spec, dr *codec.DecodingReader) error {
	return dr.List(func() codec.Deserializable {
		i := len(*h)
		*h = append(*h, common.Root{})
		return &(*h)[i]
	}, common.Root{}.ByteLength(), uint64(spec.HISTORICAL_ROOTS_LIMIT))
}

func (h HistoricalRoots) Serialize(spec *common.Spec, w *codec.EncodingWriter) error {
	return w.List(func(i uint64) codec.Serializable {
		return &h[i]
	}, common.Root{}.ByteLength(), uint64(spec.HISTORICAL_ROOTS_LIMIT))
}

func (h HistoricalRoots) ByteLength(spec *common.Spec) uint64 {
	return uint64(len(h)) * (common.Root{}.ByteLength())
}

func (h *HistoricalRoots) FixedLength(_ *common.Spec) uint64 {
	return 0
}

func (h HistoricalRoots) HashTreeRoot(spec *common.Spec, hFn tree.HashFn) common.Root {
	length := uint64(len(h))
	return hFn.ComplexListHTR(func(i uint64) tree.HTR {
		if i < length {
			return &h[i]
		}
		return nil
	}, length, uint64(spec.HISTORICAL_ROOTS_LIMIT))
}

type BlockNumberKey view.Uint64View
