package beacon

import (
	"bytes"
	"testing"

	"github.com/protolambda/ztyp/codec"
	"github.com/stretchr/testify/require"
)

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
	historySummariesWithProof, root, err := GetHistorySummariesWithProof()
	require.NoError(t, err)

	key := &HistoricalSummariesWithProofKey{
		Epoch: 450508969718611630,
	}
	var keyBuf bytes.Buffer
	err = key.Serialize(codec.NewEncodingWriter(&keyBuf))
	require.NoError(t, err)
	contentKey := make([]byte, 0)
	contentKey = append(contentKey, byte(HistoricalSummaries))
	contentKey = append(contentKey, keyBuf.Bytes()...)

	bn := NewBeaconNetwork(nil)
	var buf bytes.Buffer
	err = historySummariesWithProof.Serialize(bn.spec, codec.NewEncodingWriter(&buf))
	require.NoError(t, err)
	content := make([]byte, 0)
	content = append(content, Deneb[:]...)
	content = append(content, buf.Bytes()...)

	forkedHistorySummaries, err := bn.generalSummariesValidation(contentKey, content)
	require.NoError(t, err)
	valid := bn.stateSummariesValidation(*forkedHistorySummaries, root)
	require.True(t, valid)
}
