package beacon

import (
	"errors"

	"github.com/protolambda/zrnt/eth2/beacon/altair"
	"github.com/protolambda/zrnt/eth2/beacon/capella"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/zrnt/eth2/configs"
	"github.com/protolambda/ztyp/codec"
)

var (
	Bellatrix common.ForkDigest = [4]byte{0x0, 0x0, 0x0, 0x0}
	Capella   common.ForkDigest = [4]byte{0xbb, 0xa4, 0xda, 0x96}
)

//go:generate sszgen --path types.go

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

func (flb *ForkedLightClientBootstrap) Deserialize(dr *codec.DecodingReader) error {
	_, err := dr.Read(flb.ForkDigest[:])
	if err != nil {
		return err
	}

	if flb.ForkDigest == Bellatrix {
		flb.Bootstrap = &altair.LightClientBootstrap{}
	} else if flb.ForkDigest == Capella {
		flb.Bootstrap = &capella.LightClientBootstrap{}
	} else {
		return errors.New("unknown fork digest")
	}

	err = flb.Bootstrap.Deserialize(configs.Mainnet, dr)
	if err != nil {
		return err
	}

	return nil
}

func (flb *ForkedLightClientBootstrap) FixedLength() uint64 {
	return 0
}
