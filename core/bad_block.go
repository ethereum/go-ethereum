package core

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rlp"
)

// DisabledBadBlockReporting can be set to prevent blocks being reported.
var DisableBadBlockReporting = true

// ReportBlock reports the block to the block reporting tool found at
// badblocks.ethdev.com
func ReportBlock(block *types.Block, err error) {
	if DisableBadBlockReporting {
		return
	}

	const url = "https://badblocks.ethdev.com"

	blockRlp, _ := rlp.EncodeToBytes(block)
	data := map[string]interface{}{
		"block":     common.Bytes2Hex(blockRlp),
		"errortype": err.Error(),
		"hints": map[string]interface{}{
			"receipts": "NYI",
			"vmtrace":  "NYI",
		},
	}
	jsonStr, _ := json.Marshal(map[string]interface{}{"method": "eth_badBlock", "params": []interface{}{data}, "id": "1", "jsonrpc": "2.0"})

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		glog.V(logger.Error).Infoln("POST err:", err)
		return
	}
	defer resp.Body.Close()

	if glog.V(logger.Debug) {
		glog.Infoln("response Status:", resp.Status)
		glog.Infoln("response Headers:", resp.Header)
		body, _ := ioutil.ReadAll(resp.Body)
		glog.Infoln("response Body:", string(body))
	}
}
