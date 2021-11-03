package engine_v2

import (
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/ethdb"
	"github.com/XinFinOrg/XDPoSChain/params"
)

type XDPoS_v2 struct {
	config *params.XDPoSConfig // Consensus engine configuration parameters
	db     ethdb.Database      // Database to store and retrieve snapshot checkpoints
}

func New(config *params.XDPoSConfig, db ethdb.Database) *XDPoS_v2 {
	return &XDPoS_v2{
		config: config,
		db:     db,
	}
}

func NewFaker(db ethdb.Database, config *params.XDPoSConfig) *XDPoS_v2 {
	var fakeEngine *XDPoS_v2
	// Set any missing consensus parameters to their defaults
	conf := config

	// Allocate the snapshot caches and create the engine
	fakeEngine = &XDPoS_v2{
		config: conf,
		db:     db,
	}
	return fakeEngine
}

func (consensus *XDPoS_v2) Author(header *types.Header) (common.Address, error) {
	return common.Address{}, nil
}

func (consensus *XDPoS_v2) VerifyHeader(chain consensus.ChainReader, header *types.Header, fullVerify bool) error {
	return nil
}
