package params_test

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/libevm/ethtest"
	"github.com/ethereum/go-ethereum/libevm/hookstest"
	"github.com/ethereum/go-ethereum/params"
)

func TestChainConfigHooks_Description(t *testing.T) {
	const suffix = "Arran was here"
	c := new(params.ChainConfig)
	want := c.Description() + suffix

	hooks := &hookstest.Stub{
		DescriptionSuffix: "Arran was here",
	}
	hooks.Register(t).SetOnChainConfig(c, hooks)
	require.Equal(t, want, c.Description(), "ChainConfigHooks.Description() is appended to non-extras equivalent")
}

func TestChainConfigHooks_CheckConfigForkOrder(t *testing.T) {
	err := errors.New("uh oh")

	c := new(params.ChainConfig)
	require.NoError(t, c.CheckConfigForkOrder(), "CheckConfigForkOrder() with no hooks")

	hooks := &hookstest.Stub{
		CheckConfigForkOrderFn: func() error { return err },
	}
	hooks.Register(t).SetOnChainConfig(c, hooks)
	require.Equal(t, err, c.CheckConfigForkOrder(), "CheckConfigForkOrder() with error-producing hook")
}

func TestChainConfigHooks_CheckConfigCompatible(t *testing.T) {
	rng := ethtest.NewPseudoRand(1234567890)
	newcfg := &params.ChainConfig{
		ChainID: rng.BigUint64(),
	}
	headNumber := rng.Uint64()
	headTimestamp := rng.Uint64()

	c := new(params.ChainConfig)
	require.Nil(t, c.CheckCompatible(newcfg, headNumber, headTimestamp), "CheckCompatible() with no hooks")

	makeCompatErr := func(newcfg *params.ChainConfig, headNumber *big.Int, headTimestamp uint64) *params.ConfigCompatError {
		return &params.ConfigCompatError{
			What: fmt.Sprintf("ChainID: %v Head #: %v Head Time: %d", newcfg.ChainID, headNumber, headTimestamp),
		}
	}
	hooks := &hookstest.Stub{
		CheckConfigCompatibleFn: makeCompatErr,
	}
	hooks.Register(t).SetOnChainConfig(c, hooks)
	want := makeCompatErr(newcfg, new(big.Int).SetUint64(headNumber), headTimestamp)
	require.Equal(t, want, c.CheckCompatible(newcfg, headNumber, headTimestamp), "CheckCompatible() with error-producing hook")
}
