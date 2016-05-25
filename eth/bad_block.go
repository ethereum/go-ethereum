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

package eth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	// The Ethereum main network genesis block.
	defaultGenesisHash = "0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3"
	badBlocksURL       = "https://badblocks.ethdev.com"
)

var EnableBadBlockReporting = false

func sendBadBlockReport(block *types.Block, err error) {
	if !EnableBadBlockReporting {
		return
	}

	var (
		blockRLP, _ = rlp.EncodeToBytes(block)
		params      = map[string]interface{}{
			"block":     common.Bytes2Hex(blockRLP),
			"blockHash": block.Hash().Hex(),
			"errortype": err.Error(),
			"client":    "go",
		}
	)
	if !block.ReceivedAt.IsZero() {
		params["receivedAt"] = block.ReceivedAt.UTC().String()
	}
	if p, ok := block.ReceivedFrom.(*peer); ok {
		params["receivedFrom"] = map[string]interface{}{
			"enode":           fmt.Sprintf("enode://%x@%v", p.ID(), p.RemoteAddr()),
			"name":            p.Name(),
			"protocolVersion": p.version,
		}
	}
	jsonStr, _ := json.Marshal(map[string]interface{}{"method": "eth_badBlock", "id": "1", "jsonrpc": "2.0", "params": []interface{}{params}})
	client := http.Client{Timeout: 8 * time.Second}
	resp, err := client.Post(badBlocksURL, "application/json", bytes.NewReader(jsonStr))
	if err != nil {
		glog.V(logger.Debug).Infoln(err)
		return
	}
	glog.V(logger.Debug).Infof("Bad Block Report posted (%d)", resp.StatusCode)
	resp.Body.Close()
}
