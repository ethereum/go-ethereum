package eth

import (
	"math/big"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/rawdb"
	"github.com/XinFinOrg/XDPoSChain/eth/util"
	"github.com/XinFinOrg/XDPoSChain/params"
)

func TestRewardInflation(t *testing.T) {
	for i := 0; i < 100; i++ {
		// the first 2 years
		chainReward := new(big.Int).Mul(new(big.Int).SetUint64(250), new(big.Int).SetUint64(params.Ether))
		chainReward = util.RewardInflation(nil, chainReward, uint64(i), 10)

		// 3rd year, 4th year, 5th year
		halfReward := new(big.Int).Mul(new(big.Int).SetUint64(125), new(big.Int).SetUint64(params.Ether))
		if 20 <= i && i < 50 && chainReward.Cmp(halfReward) != 0 {
			t.Error("Fail tor calculate reward inflation for 2 -> 5 years", "chainReward", chainReward)
		}

		// after 5 years
		quarterReward := new(big.Int).Mul(new(big.Int).SetUint64(62.5*1000), new(big.Int).SetUint64(params.Finney))
		if 50 <= i && chainReward.Cmp(quarterReward) != 0 {
			t.Error("Fail tor calculate reward inflation above 6 years", "chainReward", chainReward)
		}
	}
}

func TestSetupGenesisBlockRepairsMissingV2Config(t *testing.T) {
	db := rawdb.NewMemoryDatabase()

	legacyGenesis := legacyTestnetGenesisWithoutV2()
	legacyGenesis.MustCommit(db)

	loadedCfg, _, err := core.LoadChainConfig(db, core.DefaultTestnetGenesisBlock())
	if err != nil {
		t.Fatalf("LoadChainConfig failed: %v", err)
	}
	if loadedCfg.XDPoS == nil {
		t.Fatal("expected XDPoS config in loaded chain config")
	}
	if loadedCfg.XDPoS.V2 != nil {
		t.Fatal("expected stored legacy chain config to have nil XDPoS.V2 before setup")
	}

	finalCfg, _, err := core.SetupGenesisBlock(db, core.DefaultTestnetGenesisBlock())
	if err != nil {
		t.Fatalf("SetupGenesisBlock failed: %v", err)
	}
	if finalCfg.XDPoS == nil || finalCfg.XDPoS.V2 == nil {
		t.Fatal("expected SetupGenesisBlock to return a config with XDPoS.V2")
	}
	if finalCfg.XDPoS.V2.SwitchBlock.Cmp(params.TestnetChainConfig.XDPoS.V2.SwitchBlock) != 0 {
		t.Fatalf("unexpected switch block after setup: have %v want %v", finalCfg.XDPoS.V2.SwitchBlock, params.TestnetChainConfig.XDPoS.V2.SwitchBlock)
	}
}

func TestSetupGenesisBlockIsIdempotentForTestnet(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	genesis := core.DefaultTestnetGenesisBlock()

	cfg1, hash1, err := core.SetupGenesisBlock(db, genesis)
	if err != nil {
		t.Fatalf("first SetupGenesisBlock failed: %v", err)
	}
	cfg2, hash2, err := core.SetupGenesisBlock(db, genesis)
	if err != nil {
		t.Fatalf("second SetupGenesisBlock failed: %v", err)
	}
	if hash1 != hash2 {
		t.Fatalf("genesis hash changed across SetupGenesisBlock calls: first %v second %v", hash1, hash2)
	}
	if cfg1.XDPoS == nil || cfg2.XDPoS == nil || cfg1.XDPoS.V2 == nil || cfg2.XDPoS.V2 == nil {
		t.Fatal("expected both returned configs to include XDPoS.V2")
	}
	if cfg1.XDPoS.V2.SwitchBlock.Cmp(cfg2.XDPoS.V2.SwitchBlock) != 0 {
		t.Fatalf("switch block changed across SetupGenesisBlock calls: first %v second %v", cfg1.XDPoS.V2.SwitchBlock, cfg2.XDPoS.V2.SwitchBlock)
	}
}

func legacyTestnetGenesisWithoutV2() *core.Genesis {
	legacyGenesis := *core.DefaultTestnetGenesisBlock()
	legacyChainConfig := *params.TestnetChainConfig
	legacyXDPoS := *params.TestnetChainConfig.XDPoS
	legacyXDPoS.V2 = nil
	legacyChainConfig.XDPoS = &legacyXDPoS
	legacyGenesis.Config = &legacyChainConfig
	return &legacyGenesis
}
