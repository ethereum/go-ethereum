package flagset

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFlagsetBool(t *testing.T) {
	f := NewFlagSet("")

	value := false
	f.BoolFlag(&BoolFlag{
		Name:  "flag",
		Value: &value,
	})

	assert.NoError(t, f.Parse([]string{"--flag", "true"}))
	assert.Equal(t, value, true)
}

func TestFlagsetSliceString(t *testing.T) {
	f := NewFlagSet("")

	value := []string{"a", "b", "c"}
	f.SliceStringFlag(&SliceStringFlag{
		Name:    "flag",
		Value:   &value,
		Default: value,
	})

	assert.NoError(t, f.Parse([]string{}))
	assert.Equal(t, value, []string{"a", "b", "c"})
	assert.NoError(t, f.Parse([]string{"--flag", "a,b"}))
	assert.Equal(t, value, []string{"a", "b"})
}

func TestFlagsetDuration(t *testing.T) {
	f := NewFlagSet("")

	value := time.Duration(0)
	f.DurationFlag(&DurationFlag{
		Name:  "flag",
		Value: &value,
	})

	assert.NoError(t, f.Parse([]string{"--flag", "1m"}))
	assert.Equal(t, value, 1*time.Minute)
}

func TestFlagsetMapString(t *testing.T) {
	f := NewFlagSet("")

	value := map[string]string{}
	f.MapStringFlag(&MapStringFlag{
		Name:  "flag",
		Value: &value,
	})

	assert.NoError(t, f.Parse([]string{"--flag", "a=b,c=d"}))
	assert.Equal(t, value, map[string]string{"a": "b", "c": "d"})
}
