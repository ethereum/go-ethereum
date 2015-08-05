// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

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
