package beacon

import (
	"bytes"
	"testing"

	"github.com/protolambda/ztyp/codec"
	"github.com/stretchr/testify/require"
)

type Entry struct {
	ContentKey   string `yaml:"content_key"`
	ContentValue string `yaml:"content_value"`
}

func TestLightClientBootstrapValidation(t *testing.T) {
	bootstrap, err := GetLightClientBootstrap(0)
	require.NoError(t, err)
	contentKey := make([]byte, 33)
	contentKey[0] = byte(LightClientBootstrap)
	bn := NewBeaconNetwork(nil)
	var buf bytes.Buffer
	bootstrap.Serialize(bn.spec, codec.NewEncodingWriter(&buf))
	err = bn.validateContent(contentKey, buf.Bytes())
	require.NoError(t, err)
}

func TestLightClienUpdateValidation(t *testing.T) {
	update, err := GetClientUpdate(0)
	require.NoError(t, err)
	key := &LightClientUpdateKey{
		StartPeriod: 0,
		Count:       1,
	}
	updateRange := LightClientUpdateRange([]ForkedLightClientUpdate{update})
	keyData, err := key.MarshalSSZ()
	require.NoError(t, err)
	contentKey := make([]byte, 0)
	contentKey = append(contentKey, byte(LightClientUpdate))
	contentKey = append(contentKey, keyData...)
	bn := NewBeaconNetwork(nil)
	var buf bytes.Buffer
	updateRange.Serialize(bn.spec, codec.NewEncodingWriter(&buf))
	err = bn.validateContent(contentKey, buf.Bytes())
	require.NoError(t, err)
}

func TestLightClientFinalityUpdateValidation(t *testing.T) {
	update, err := GetLightClientFinalityUpdate(0)
	require.NoError(t, err)
	key := &LightClientFinalityUpdateKey{
		FinalizedSlot: 10934316269310501102,
	}
	keyData, err := key.MarshalSSZ()
	require.NoError(t, err)
	contentKey := make([]byte, 0)
	contentKey = append(contentKey, byte(LightClientFinalityUpdate))
	contentKey = append(contentKey, keyData...)
	bn := NewBeaconNetwork(nil)
	var buf bytes.Buffer
	update.Serialize(bn.spec, codec.NewEncodingWriter(&buf))
	err = bn.validateContent(contentKey, buf.Bytes())
	require.NoError(t, err)
}

func TestLightClientOptimisticUpdateValidation(t *testing.T) {
	update, err := GetLightClientOptimisticUpdate(0)
	require.NoError(t, err)
	key := &LightClientOptimisticUpdateKey{
		OptimisticSlot: 15067541596220156845,
	}
	keyData, err := key.MarshalSSZ()
	require.NoError(t, err)
	contentKey := make([]byte, 0)
	contentKey = append(contentKey, byte(LightClientOptimisticUpdate))
	contentKey = append(contentKey, keyData...)
	bn := NewBeaconNetwork(nil)
	var buf bytes.Buffer
	update.Serialize(bn.spec, codec.NewEncodingWriter(&buf))
	err = bn.validateContent(contentKey, buf.Bytes())
	require.NoError(t, err)
}

func TestHistorySummariesWithProofValidation(t *testing.T) {
	_, err := GetHistorySummariesWithProof()
	require.NoError(t, err)
}
