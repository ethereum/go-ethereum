package beacon

import (
	"errors"

	"github.com/protolambda/zrnt/eth2/beacon/altair"
	"github.com/protolambda/zrnt/eth2/beacon/capella"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/ztyp/codec"
	tree "github.com/protolambda/ztyp/tree"
)

var (
	Bellatrix common.ForkDigest = [4]byte{0x0, 0x0, 0x0, 0x0}
	Capella   common.ForkDigest = [4]byte{0xbb, 0xa4, 0xda, 0x96}
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
	return w.Container(flcb.ForkDigest, spec.Wrap(flcb.Bootstrap))
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
	return w.Container(flcu.ForkDigest, spec.Wrap(flcu.LightClientUpdate))
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
