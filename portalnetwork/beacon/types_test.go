package beacon

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/protolambda/zrnt/eth2/beacon/capella"
	"github.com/protolambda/zrnt/eth2/configs"
	"github.com/protolambda/ztyp/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestForkedLightClientBootstrap(t *testing.T) {
	filePath := "testdata/types/light_client_bootstrap.json"

	f, err := os.Open(filePath)
	require.NoError(t, err)
	jsonStr, err := io.ReadAll(f)
	require.NoError(t, err)
	var result map[string]interface{}
	_ = json.Unmarshal(jsonStr, &result)

	for k, v := range result {
		b, _ := hexutil.Decode(v.(map[string]interface{})["content_value"].(string))
		dec := codec.NewDecodingReader(bytes.NewReader(b), uint64(len(b)))
		var f ForkedLightClientBootstrap
		err := f.Deserialize(configs.Mainnet, dec)
		assert.NoError(t, err)
		assert.Equal(t, k, f.Bootstrap.(*capella.LightClientBootstrap).Header.Beacon.Slot.String())

		var buf bytes.Buffer
		err = f.Serialize(configs.Mainnet, codec.NewEncodingWriter(&buf))
		assert.NoError(t, err)
		assert.Equal(t, b, buf.Bytes())
	}
}

func TestLightClientUpdateRange(t *testing.T) {
	filePath := "testdata/types/light_client_updates_by_range.json"

	f, err := os.Open(filePath)
	require.NoError(t, err)
	jsonStr, err := io.ReadAll(f)
	require.NoError(t, err)
	var result map[string]interface{}
	_ = json.Unmarshal(jsonStr, &result)

	for k, v := range result {
		b, _ := hexutil.Decode(v.(map[string]interface{})["content_value"].(string))
		dec := codec.NewDecodingReader(bytes.NewReader(b), uint64(len(b)))
		var f LightClientUpdateRange = make([]ForkedLightClientUpdate, 0)
		err := f.Deserialize(configs.Mainnet, dec)
		assert.NoError(t, err)
		assert.Equal(t, k, f[0].LightClientUpdate.(*capella.LightClientUpdate).AttestedHeader.Beacon.Slot.String())
		assert.Equal(t, 4, len(f))

		var buf bytes.Buffer
		err = f.Serialize(configs.Mainnet, codec.NewEncodingWriter(&buf))
		assert.NoError(t, err)
		assert.Equal(t, b, buf.Bytes())
	}
}

func TestForkedLightClientOptimisticUpdate(t *testing.T) {
	filePath := "testdata/types/light_client_optimistic_update.json"

	f, err := os.Open(filePath)
	require.NoError(t, err)
	jsonStr, err := io.ReadAll(f)
	require.NoError(t, err)

	var result map[string]interface{}
	_ = json.Unmarshal(jsonStr, &result)

	for k, v := range result {
		b, _ := hexutil.Decode(v.(map[string]interface{})["content_value"].(string))
		dec := codec.NewDecodingReader(bytes.NewReader(b), uint64(len(b)))
		var f ForkedLightClientOptimisticUpdate
		err := f.Deserialize(configs.Mainnet, dec)
		assert.NoError(t, err)
		assert.Equal(t, k, f.LightClientOptimisticUpdate.(*capella.LightClientOptimisticUpdate).AttestedHeader.Beacon.Slot.String())

		var buf bytes.Buffer
		err = f.Serialize(configs.Mainnet, codec.NewEncodingWriter(&buf))
		assert.NoError(t, err)
		assert.Equal(t, b, buf.Bytes())
	}
}

func TestForkedLightClientFinalityUpdate(t *testing.T) {
	filePath := "testdata/types/light_client_finality_update.json"

	f, err := os.Open(filePath)
	require.NoError(t, err)
	jsonStr, err := io.ReadAll(f)
	require.NoError(t, err)

	var result map[string]interface{}
	_ = json.Unmarshal(jsonStr, &result)

	for k, v := range result {
		b, _ := hexutil.Decode(v.(map[string]interface{})["content_value"].(string))
		dec := codec.NewDecodingReader(bytes.NewReader(b), uint64(len(b)))
		var f ForkedLightClientFinalityUpdate
		err := f.Deserialize(configs.Mainnet, dec)
		assert.NoError(t, err)
		assert.Equal(t, k, f.LightClientFinalityUpdate.(*capella.LightClientFinalityUpdate).AttestedHeader.Beacon.Slot.String())

		var buf bytes.Buffer
		err = f.Serialize(configs.Mainnet, codec.NewEncodingWriter(&buf))
		assert.NoError(t, err)
		assert.Equal(t, b, buf.Bytes())
	}
}

type TestProof struct {
	ContentKey                    string   `yaml:"content_key"`
	ContentValue                  string   `yaml:"content_value"`
	BeaconStateRoot               string   `yaml:"beacon_state_root"`
	HistoricalSummariesRoot       string   `yaml:"historical_summaries_root"`
	HistoricalSummariesStateProof []string `yaml:"historical_summaries_state_proof"`
	Epoch                         uint64   `yaml:"epoch"`
}

func TestForkedHistoricalSummariesWithProof(t *testing.T) {
	filePath := "testdata/types/historical_summaries_with_proof.yaml"

	f, err := os.Open(filePath)
	require.NoError(t, err)
	contentBytes, err := io.ReadAll(f)
	require.NoError(t, err)
	testData := TestProof{}
	err = yaml.Unmarshal(contentBytes, &testData)
	require.NoError(t, err)

	historyKey := &HistoricalSummariesWithProofKey{}
	contentKey := hexutil.MustDecode(testData.ContentKey)
	err = historyKey.Deserialize(codec.NewDecodingReader(bytes.NewReader(contentKey[1:]), uint64(len(contentKey)-1)))
	require.NoError(t, err)
	require.Equal(t, testData.Epoch, historyKey.Epoch)

	historyProof := &ForkedHistoricalSummariesWithProof{}
	content := hexutil.MustDecode(testData.ContentValue)
	err = historyProof.Deserialize(configs.Mainnet, codec.NewDecodingReader(bytes.NewReader(content), uint64(len(content))))
	require.NoError(t, err)
	require.Equal(t, uint64(historyProof.HistoricalSummariesWithProof.EPOCH), testData.Epoch)
}
