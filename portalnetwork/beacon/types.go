package beacon

import (
	"errors"

	"github.com/protolambda/zrnt/eth2/beacon/altair"
	"github.com/protolambda/zrnt/eth2/beacon/capella"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/zrnt/eth2/beacon/deneb"
	"github.com/protolambda/ztyp/codec"
	"github.com/protolambda/ztyp/tree"
)

const MaxRequestLightClientUpdates = 128

var (
	Bellatrix common.ForkDigest = [4]byte{0x0, 0x0, 0x0, 0x0}
	Capella   common.ForkDigest = [4]byte{0xbb, 0xa4, 0xda, 0x96}
	Deneb     common.ForkDigest = [4]byte{0x6a, 0x95, 0xa1, 0xa9}
)

//go:generate sszgen --path types.go --exclude-objs ForkedLightClientBootstrap,ForkedLightClientUpdate,LightClientUpdateRange

type LightClientUpdateKey struct {
	StartPeriod uint64
	Count       uint64
}

type LightClientBootstrapKey struct {
	BlockHash []byte `ssz-size:"32"`
}

type LightClientFinalityUpdateKey struct {
	FinalizedSlot uint64
}

type LightClientOptimisticUpdateKey struct {
	OptimisticSlot uint64
}

type ForkedLightClientBootstrap struct {
	ForkDigest common.ForkDigest
	Bootstrap  common.SpecObj
}

func (flcb *ForkedLightClientBootstrap) Deserialize(spec *common.Spec, dr *codec.DecodingReader) error {
	_, err := dr.Read(flcb.ForkDigest[:])
	if err != nil {
		return err
	}

	if flcb.ForkDigest == Bellatrix {
		flcb.Bootstrap = &altair.LightClientBootstrap{}
	} else if flcb.ForkDigest == Capella {
		flcb.Bootstrap = &capella.LightClientBootstrap{}
	} else if flcb.ForkDigest == Deneb {
		flcb.Bootstrap = &deneb.LightClientBootstrap{}
	} else {
		return errors.New("unknown fork digest")
	}

	err = flcb.Bootstrap.Deserialize(spec, dr)
	if err != nil {
		return err
	}

	return nil
}

func (flcb *ForkedLightClientBootstrap) Serialize(spec *common.Spec, w *codec.EncodingWriter) error {
	if err := w.Write(flcb.ForkDigest[:]); err != nil {
		return err
	}
	return flcb.Bootstrap.Serialize(spec, w)
}

func (flcb *ForkedLightClientBootstrap) FixedLength(_ *common.Spec) uint64 {
	return 0
}

func (flcb *ForkedLightClientBootstrap) ByteLength(spec *common.Spec) uint64 {
	return 4 + flcb.Bootstrap.ByteLength(spec)
}

func (flcb *ForkedLightClientBootstrap) HashTreeRoot(spec *common.Spec, h tree.HashFn) common.Root {
	return h.HashTreeRoot(flcb.ForkDigest, spec.Wrap(flcb.Bootstrap))
}

type ForkedLightClientUpdate struct {
	ForkDigest        common.ForkDigest
	LightClientUpdate common.SpecObj
}

func (flcu *ForkedLightClientUpdate) Deserialize(spec *common.Spec, dr *codec.DecodingReader) error {
	_, err := dr.Read(flcu.ForkDigest[:])
	if err != nil {
		return err
	}

	if flcu.ForkDigest == Bellatrix {
		flcu.LightClientUpdate = &altair.LightClientUpdate{}
	} else if flcu.ForkDigest == Capella {
		flcu.LightClientUpdate = &capella.LightClientUpdate{}
	} else if flcu.ForkDigest == Deneb {
		flcu.LightClientUpdate = &deneb.LightClientUpdate{}
	} else {
		return errors.New("unknown fork digest")
	}

	err = flcu.LightClientUpdate.Deserialize(spec, dr)
	if err != nil {
		return err
	}

	return nil
}

func (flcu *ForkedLightClientUpdate) Serialize(spec *common.Spec, w *codec.EncodingWriter) error {
	if err := w.Write(flcu.ForkDigest[:]); err != nil {
		return err
	}
	return flcu.LightClientUpdate.Serialize(spec, w)
}

func (flcu *ForkedLightClientUpdate) FixedLength(_ *common.Spec) uint64 {
	return 0
}

func (flcu *ForkedLightClientUpdate) ByteLength(spec *common.Spec) uint64 {
	return 4 + flcu.LightClientUpdate.ByteLength(spec)
}

func (flcu *ForkedLightClientUpdate) HashTreeRoot(spec *common.Spec, h tree.HashFn) common.Root {
	return h.HashTreeRoot(flcu.ForkDigest, spec.Wrap(flcu.LightClientUpdate))
}

type LightClientUpdateRange []ForkedLightClientUpdate

func (r *LightClientUpdateRange) Deserialize(spec *common.Spec, dr *codec.DecodingReader) error {
	return dr.List(func() codec.Deserializable {
		i := len(*r)
		*r = append(*r, ForkedLightClientUpdate{})
		return spec.Wrap(&((*r)[i]))
	}, 0, 128)
}

func (r LightClientUpdateRange) Serialize(spec *common.Spec, w *codec.EncodingWriter) error {
	return w.List(func(i uint64) codec.Serializable {
		return spec.Wrap(&r[i])
	}, 0, uint64(len(r)))
}

func (r LightClientUpdateRange) ByteLength(spec *common.Spec) (out uint64) {
	for _, v := range r {
		out += v.ByteLength(spec) + codec.OFFSET_SIZE
	}
	return
}

func (r *LightClientUpdateRange) FixedLength(_ *common.Spec) uint64 {
	return 0
}

func (r LightClientUpdateRange) HashTreeRoot(spec *common.Spec, hFn tree.HashFn) common.Root {
	length := uint64(len(r))
	return hFn.ComplexListHTR(func(i uint64) tree.HTR {
		if i < length {
			return spec.Wrap(&r[i])
		}
		return nil
	}, length, 128)
}

type ForkedLightClientOptimisticUpdate struct {
	ForkDigest                  common.ForkDigest
	LightClientOptimisticUpdate common.SpecObj
}

func (flcou *ForkedLightClientOptimisticUpdate) Deserialize(spec *common.Spec, dr *codec.DecodingReader) error {
	_, err := dr.Read(flcou.ForkDigest[:])
	if err != nil {
		return err
	}

	if flcou.ForkDigest == Bellatrix {
		flcou.LightClientOptimisticUpdate = &altair.LightClientOptimisticUpdate{}
	} else if flcou.ForkDigest == Capella {
		flcou.LightClientOptimisticUpdate = &capella.LightClientOptimisticUpdate{}
	} else if flcou.ForkDigest == Deneb {
		flcou.LightClientOptimisticUpdate = &deneb.LightClientOptimisticUpdate{}
	} else {
		return errors.New("unknown fork digest")
	}

	err = flcou.LightClientOptimisticUpdate.Deserialize(spec, dr)
	if err != nil {
		return err
	}

	return nil
}

func (flcou *ForkedLightClientOptimisticUpdate) Serialize(spec *common.Spec, w *codec.EncodingWriter) error {
	if err := w.Write(flcou.ForkDigest[:]); err != nil {
		return err
	}
	return flcou.LightClientOptimisticUpdate.Serialize(spec, w)
}

func (flcou *ForkedLightClientOptimisticUpdate) FixedLength(_ *common.Spec) uint64 {
	return 0
}

func (flcou *ForkedLightClientOptimisticUpdate) ByteLength(spec *common.Spec) uint64 {
	return 4 + flcou.LightClientOptimisticUpdate.ByteLength(spec)
}

func (flcou *ForkedLightClientOptimisticUpdate) HashTreeRoot(spec *common.Spec, h tree.HashFn) common.Root {
	return h.HashTreeRoot(flcou.ForkDigest, spec.Wrap(flcou.LightClientOptimisticUpdate))
}

type ForkedLightClientFinalityUpdate struct {
	ForkDigest                common.ForkDigest
	LightClientFinalityUpdate common.SpecObj
}

func (flcfu *ForkedLightClientFinalityUpdate) Deserialize(spec *common.Spec, dr *codec.DecodingReader) error {
	_, err := dr.Read(flcfu.ForkDigest[:])
	if err != nil {
		return err
	}

	if flcfu.ForkDigest == Bellatrix {
		flcfu.LightClientFinalityUpdate = &altair.LightClientFinalityUpdate{}
	} else if flcfu.ForkDigest == Capella {
		flcfu.LightClientFinalityUpdate = &capella.LightClientFinalityUpdate{}
	} else if flcfu.ForkDigest == Deneb {
		flcfu.LightClientFinalityUpdate = &deneb.LightClientFinalityUpdate{}
	} else {
		return errors.New("unknown fork digest")
	}

	err = flcfu.LightClientFinalityUpdate.Deserialize(spec, dr)
	if err != nil {
		return err
	}

	return nil
}

func (flcfu *ForkedLightClientFinalityUpdate) Serialize(spec *common.Spec, w *codec.EncodingWriter) error {
	if err := w.Write(flcfu.ForkDigest[:]); err != nil {
		return err
	}
	return flcfu.LightClientFinalityUpdate.Serialize(spec, w)
}

func (flcfu *ForkedLightClientFinalityUpdate) FixedLength(_ *common.Spec) uint64 {
	return 0
}

func (flcfu *ForkedLightClientFinalityUpdate) ByteLength(spec *common.Spec) uint64 {
	return 4 + flcfu.LightClientFinalityUpdate.ByteLength(spec)
}

func (flcfu *ForkedLightClientFinalityUpdate) HashTreeRoot(spec *common.Spec, h tree.HashFn) common.Root {
	return h.HashTreeRoot(flcfu.ForkDigest, spec.Wrap(flcfu.LightClientFinalityUpdate))
}

type HistoricalSummariesProof struct {
	Proof [5]common.Bytes32
}

func (hsp *HistoricalSummariesProof) Deserialize(dr *codec.DecodingReader) error {
	roots := hsp.Proof[:]
	return tree.ReadRoots(dr, &roots, 5)
}

func (hsp *HistoricalSummariesProof) Serialize(w *codec.EncodingWriter) error {
	return tree.WriteRoots(w, hsp.Proof[:])
}

func (hsp *HistoricalSummariesProof) ByteLength() uint64 {
	return 32 * 5
}

func (hsp *HistoricalSummariesProof) FixedLength() uint64 {
	return 32 * 5
}

func (hsp *HistoricalSummariesProof) HashTreeRoot(hFn tree.HashFn) common.Root {
	return hFn.ComplexVectorHTR(func(i uint64) tree.HTR {
		if i < 5 {
			return &hsp.Proof[i]
		}
		return nil
	}, 5)
}

// TODO: Add tests for HistoricalSummariesWithProof

type HistoricalSummariesWithProof struct {
	EPOCH               common.Epoch
	HistoricalSummaries capella.HistoricalSummaries
	Proof               *HistoricalSummariesProof
}

func (hswp *HistoricalSummariesWithProof) Deserialize(spec *common.Spec, dr *codec.DecodingReader) error {
	return dr.Container(&hswp.EPOCH, spec.Wrap(&hswp.HistoricalSummaries), hswp.Proof)
}

func (hswp *HistoricalSummariesWithProof) Serialize(spec *common.Spec, w *codec.EncodingWriter) error {
	return w.Container(hswp.EPOCH, spec.Wrap(&hswp.HistoricalSummaries), hswp.Proof)
}

func (hswp *HistoricalSummariesWithProof) ByteLength(spec *common.Spec) uint64 {
	return codec.ContainerLength(hswp.EPOCH, spec.Wrap(&hswp.HistoricalSummaries), hswp.Proof)
}

func (hswp *HistoricalSummariesWithProof) FixedLength(_ *common.Spec) uint64 {
	return 0
}

func (hswp *HistoricalSummariesWithProof) HashTreeRoot(spec *common.Spec, hFn tree.HashFn) common.Root {
	return hFn.HashTreeRoot(hswp.EPOCH, spec.Wrap(&hswp.HistoricalSummaries), hswp.Proof)
}
