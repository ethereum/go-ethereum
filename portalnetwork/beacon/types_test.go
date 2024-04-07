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
)

func TestForkedLightClientBootstrap(t *testing.T) {
	filePath := "testdata/light_client_bootstrap.json"

	f, _ := os.Open(filePath)
	jsonStr, _ := io.ReadAll(f)

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
	filePath := "testdata/light_client_updates_by_range.json"

	f, _ := os.Open(filePath)
	jsonStr, _ := io.ReadAll(f)

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
	filePath := "testdata/light_client_optimistic_update.json"

	f, _ := os.Open(filePath)
	jsonStr, _ := io.ReadAll(f)

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
	filePath := "testdata/light_client_finality_update.json"

	f, _ := os.Open(filePath)
	jsonStr, _ := io.ReadAll(f)

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
