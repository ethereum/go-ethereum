package beacon

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/protolambda/zrnt/eth2/beacon/capella"
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
		err := f.Deserialize(dec)
		assert.NoError(t, err)
		assert.Equal(t, k, f.Bootstrap.(*capella.LightClientBootstrap).Header.Beacon.Slot.String())
	}
}
