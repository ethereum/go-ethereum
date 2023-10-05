package flagset

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFlagsetBool(t *testing.T) {
	t.Parallel()

	f := NewFlagSet("")

	value := true
	f.BoolFlag(&BoolFlag{
		Name:  "flag",
		Value: &value,
	})

	// Parse no value, should have default (of datatype)
	require.NoError(t, f.Parse([]string{}))
	require.Equal(t, false, value)

	// Parse --flag true
	require.NoError(t, f.Parse([]string{"--flag", "true"}))
	require.Equal(t, true, value)

	// Parse --flag=true
	require.NoError(t, f.Parse([]string{"--flag=true"}))
	require.Equal(t, true, value)

	// Parse --flag false: won't parse false
	require.NoError(t, f.Parse([]string{"--flag", "false"}))
	require.Equal(t, true, value)

	// Parse --flag=false
	require.NoError(t, f.Parse([]string{"--flag=false"}))
	require.Equal(t, false, value)

	// Parse --flag
	require.NoError(t, f.Parse([]string{"--flag"}))
	require.Equal(t, true, value)
}

func TestFlagsetString(t *testing.T) {
	t.Parallel()

	f := NewFlagSet("")

	value := "hello"
	f.StringFlag(&StringFlag{
		Name:  "flag",
		Value: &value,
	})

	// Parse no value, should have default
	require.NoError(t, f.Parse([]string{}))
	require.Equal(t, "", value)

	// Parse --flag value
	require.NoError(t, f.Parse([]string{"--flag", "value"}))
	require.Equal(t, "value", value)

	// Parse --flag ""
	require.NoError(t, f.Parse([]string{"--flag", ""}))
	require.Equal(t, "", value)

	// Parse --flag=newvalue
	require.NoError(t, f.Parse([]string{"--flag=newvalue"}))
	require.Equal(t, "newvalue", value)

	// Parse --flag: should fail due to no args
	require.Error(t, f.Parse([]string{"--flag"}))
}

func TestFlagsetInt(t *testing.T) {
	t.Parallel()

	f := NewFlagSet("")

	value := 10
	f.IntFlag(&IntFlag{
		Name:  "flag",
		Value: &value,
	})

	// Parse no value, should have default
	require.NoError(t, f.Parse([]string{}))
	require.Equal(t, 0, value)

	// Parse --flag 20
	require.NoError(t, f.Parse([]string{"--flag", "20"}))
	require.Equal(t, 20, value)

	// Parse --flag 0
	require.NoError(t, f.Parse([]string{"--flag", "0"}))
	require.Equal(t, 0, value)

	// Parse --flag=30
	require.NoError(t, f.Parse([]string{"--flag=30"}))
	require.Equal(t, 30, value)

	// Parse --flag: should fail due to no args
	require.Error(t, f.Parse([]string{"--flag"}))
}

func TestFlagsetFloat64(t *testing.T) {
	t.Parallel()

	f := NewFlagSet("")

	value := 10.0
	f.Float64Flag(&Float64Flag{
		Name:  "flag",
		Value: &value,
	})

	// Parse no value, should have default
	require.NoError(t, f.Parse([]string{}))
	require.Equal(t, 0.0, value)

	// Parse --flag 20
	require.NoError(t, f.Parse([]string{"--flag", "20.1"}))
	require.Equal(t, 20.1, value)

	// Parse --flag 0
	require.NoError(t, f.Parse([]string{"--flag", "0"}))
	require.Equal(t, 0.0, value)

	// Parse --flag 0.0
	require.NoError(t, f.Parse([]string{"--flag", "0.0"}))
	require.Equal(t, 0.0, value)

	// Parse --flag=30.1
	require.NoError(t, f.Parse([]string{"--flag=30.1"}))
	require.Equal(t, 30.1, value)

	// Parse --flag: should fail due to no args
	require.Error(t, f.Parse([]string{"--flag"}))
}

func TestFlagsetBigInt(t *testing.T) {
	t.Parallel()

	f := NewFlagSet("")

	value := big.NewInt(0)
	f.BigIntFlag(&BigIntFlag{
		Name:  "flag",
		Value: value,
	})

	// Parse no value, should have initial value (0 here)
	require.NoError(t, f.Parse([]string{}))
	require.Equal(t, big.NewInt(0), value)

	// Parse --flag 20
	require.NoError(t, f.Parse([]string{"--flag", "20"}))
	require.Equal(t, big.NewInt(20), value)

	// Parse --flag=30
	require.NoError(t, f.Parse([]string{"--flag=30"}))
	require.Equal(t, big.NewInt(30), value)

	// Parse --flag: should fail due to no args
	require.Error(t, f.Parse([]string{"--flag"}))
}

func TestFlagsetSliceString(t *testing.T) {
	t.Parallel()

	f := NewFlagSet("")

	value := []string{"a", "b", "c"}
	f.SliceStringFlag(&SliceStringFlag{
		Name:    "flag",
		Value:   &value,
		Default: value,
	})

	// Parse no value, should have initial value
	require.NoError(t, f.Parse([]string{}))
	require.Equal(t, []string{"a", "b", "c"}, value)

	// Parse --flag a,b
	require.NoError(t, f.Parse([]string{"--flag", "a,b"}))
	require.Equal(t, []string{"a", "b"}, value)

	// Parse --flag ""
	require.NoError(t, f.Parse([]string{"--flag", ""}))
	require.Equal(t, []string(nil), value)

	// Parse --flag: should fail due to no args
	require.Error(t, f.Parse([]string{"--flag"}))
}

func TestFlagsetDuration(t *testing.T) {
	t.Parallel()

	f := NewFlagSet("")

	value := time.Duration(0)
	f.DurationFlag(&DurationFlag{
		Name:  "flag",
		Value: &value,
	})

	// Parse no value, should have initial value
	require.NoError(t, f.Parse([]string{}))
	require.Equal(t, time.Duration(0), value)

	// Parse --flag 1m
	require.NoError(t, f.Parse([]string{"--flag", "1m"}))
	require.Equal(t, time.Minute, value)

	// Parse --flag=1h
	require.NoError(t, f.Parse([]string{"--flag=1h"}))
	require.Equal(t, time.Hour, value)

	// Parse --flag: should fail due to no args
	require.Error(t, f.Parse([]string{"--flag"}))
}

func TestFlagsetMapString(t *testing.T) {
	t.Parallel()

	f := NewFlagSet("")

	value := map[string]string{}
	f.MapStringFlag(&MapStringFlag{
		Name:  "flag",
		Value: &value,
	})

	// Parse no value, should have initial value
	require.NoError(t, f.Parse([]string{}))
	require.Equal(t, map[string]string{}, value)

	// Parse --flag a=b,c=d
	require.NoError(t, f.Parse([]string{"--flag", "a=b,c=d"}))
	require.Equal(t, map[string]string{"a": "b", "c": "d"}, value)

	// Parse --flag=x=y
	require.NoError(t, f.Parse([]string{"--flag=x=y"}))
	require.Equal(t, map[string]string{"x": "y"}, value)

	// Parse --flag: should fail due to no args
	require.Error(t, f.Parse([]string{"--flag"}))
}
