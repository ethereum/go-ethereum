package beacon

import (
	"errors"

	"github.com/protolambda/zrnt/eth2/beacon/altair"
	"github.com/protolambda/zrnt/eth2/beacon/capella"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/zrnt/eth2/configs"
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

func (flcb *ForkedLightClientBootstrap) Deserialize(dr *codec.DecodingReader) error {
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

	err = flcb.Bootstrap.Deserialize(configs.Mainnet, dr)
	if err != nil {
		return err
	}

	return nil
}

func (flcb *ForkedLightClientBootstrap) Serialize(w *codec.EncodingWriter) error {
	if err := w.Write(flcb.ForkDigest[:]); err != nil {
		return err
	}
	return flcb.Bootstrap.Serialize(configs.Mainnet, w)
}

func (flcb *ForkedLightClientBootstrap) FixedLength() uint64 {
	return 0
}

func (flcb *ForkedLightClientBootstrap) ByteLength() uint64 {
	return 4 + flcb.Bootstrap.ByteLength(configs.Mainnet)
}

func (flcb *ForkedLightClientBootstrap) HashTreeRoot(h tree.HashFn) common.Root {
	return h.HashTreeRoot(flcb.ForkDigest, configs.Mainnet.Wrap(flcb.Bootstrap))
}

type ForkedLightClientUpdate struct {
	ForkDigest        common.ForkDigest
	LightClientUpdate common.SpecObj
}

func (flcu *ForkedLightClientUpdate) Deserialize(dr *codec.DecodingReader) error {
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

	err = flcu.LightClientUpdate.Deserialize(configs.Mainnet, dr)
	if err != nil {
		return err
	}

	return nil
}

func (flcu *ForkedLightClientUpdate) Serialize(w *codec.EncodingWriter) error {
	if err := w.Write(flcu.ForkDigest[:]); err != nil {
		return err
	}
	return flcu.LightClientUpdate.Serialize(configs.Mainnet, w)
}

func (flcu *ForkedLightClientUpdate) FixedLength() uint64 {
	return 0
}

func (flcu *ForkedLightClientUpdate) ByteLength() uint64 {
	return 4 + flcu.LightClientUpdate.ByteLength(configs.Mainnet)
}

func (flcu *ForkedLightClientUpdate) HashTreeRoot(h tree.HashFn) common.Root {
	return h.HashTreeRoot(flcu.ForkDigest, configs.Mainnet.Wrap(flcu.LightClientUpdate))
}

type LightClientUpdateRange []ForkedLightClientUpdate

func (r *LightClientUpdateRange) Deserialize(dr *codec.DecodingReader) error {
	return dr.List(func() codec.Deserializable {
		i := len(*r)
		*r = append(*r, ForkedLightClientUpdate{})
		return &((*r)[i])
	}, 0, 128)
}

func (r LightClientUpdateRange) Serialize(w *codec.EncodingWriter) error {
	return w.List(func(i uint64) codec.Serializable {
		return &r[i]
	}, 0, uint64(len(r)))
}

func (r LightClientUpdateRange) ByteLength() (out uint64) {
	for _, v := range r {
		out += v.ByteLength() + codec.OFFSET_SIZE
	}
	return
}

func (r *LightClientUpdateRange) FixedLength() uint64 {
	return 0
}

func (r LightClientUpdateRange) HashTreeRoot(hFn tree.HashFn) common.Root {
	length := uint64(len(r))
	return hFn.ComplexListHTR(func(i uint64) tree.HTR {
		if i < length {
			return &r[i]
		}
		return nil
	}, length, 128)
}
