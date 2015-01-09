/*
	This file is part of go-ethereum

	go-ethereum is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	go-ethereum is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.
*/
/**
 * @authors
 * 	Jeffrey Wilcke <i@jev.io>
 */
package utils

import (
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/event/filter"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/ui/qt"
	"github.com/ethereum/go-ethereum/websocket"
	"github.com/ethereum/go-ethereum/xeth"
)

var wslogger = logger.NewLogger("WS")

func args(v ...interface{}) []interface{} {
	return v
}

type WebSocketServer struct {
	eth           *eth.Ethereum
	filterManager *filter.FilterManager
}

func NewWebSocketServer(eth *eth.Ethereum) *WebSocketServer {
	filterManager := filter.NewFilterManager(eth.EventMux())
	go filterManager.Start()

	return &WebSocketServer{eth, filterManager}
}

func (self *WebSocketServer) Serv() {
	pipe := xeth.NewJSXEth(self.eth)

	wsServ := websocket.NewServer("/eth", ":40404")
	wsServ.MessageFunc(func(c *websocket.Client, msg *websocket.Message) {
		switch msg.Call {
		case "compile":
			data := ethutil.NewValue(msg.Args)
			bcode, err := ethutil.Compile(data.Get(0).Str(), false)
			if err != nil {
				c.Write(args(nil, err.Error()), msg.Id)
			}

			code := ethutil.Bytes2Hex(bcode)
			c.Write(args(code, nil), msg.Id)
		case "eth_blockByNumber":
			args := msg.Arguments()

			block := pipe.BlockByNumber(int32(args.Get(0).Uint()))
			c.Write(block, msg.Id)

		case "eth_blockByHash":
			args := msg.Arguments()

			c.Write(pipe.BlockByHash(args.Get(0).Str()), msg.Id)

		case "eth_transact":
			if mp, ok := msg.Args[0].(map[string]interface{}); ok {
				object := mapToTxParams(mp)
				c.Write(
					args(pipe.Transact(pipe.Key().PrivateKey, object["to"], object["value"], object["gas"], object["gasPrice"], object["data"])),
					msg.Id,
				)

			}
		case "eth_gasPrice":
			c.Write("10000000000000", msg.Id)
		case "eth_coinbase":
			c.Write(pipe.CoinBase(), msg.Id)

		case "eth_listening":
			c.Write(pipe.IsListening(), msg.Id)

		case "eth_mining":
			c.Write(pipe.IsMining(), msg.Id)

		case "eth_peerCount":
			c.Write(pipe.PeerCount(), msg.Id)

		case "eth_countAt":
			args := msg.Arguments()

			c.Write(pipe.TxCountAt(args.Get(0).Str()), msg.Id)

		case "eth_codeAt":
			args := msg.Arguments()

			c.Write(len(pipe.CodeAt(args.Get(0).Str())), msg.Id)

		case "eth_storageAt":
			args := msg.Arguments()

			c.Write(pipe.StorageAt(args.Get(0).Str(), args.Get(1).Str()), msg.Id)

		case "eth_balanceAt":
			args := msg.Arguments()

			c.Write(pipe.BalanceAt(args.Get(0).Str()), msg.Id)

		case "eth_accounts":
			c.Write(pipe.Accounts(), msg.Id)

		case "eth_newFilter":
			if mp, ok := msg.Args[0].(map[string]interface{}); ok {
				var id int
				filter := qt.NewFilterFromMap(mp, self.eth)
				filter.MessageCallback = func(messages state.Messages) {
					c.Event(toMessages(messages), "eth_changed", id)
				}
				id = self.filterManager.InstallFilter(filter)
				c.Write(id, msg.Id)
			}
		case "eth_newFilterString":
			var id int
			filter := core.NewFilter(self.eth)
			filter.BlockCallback = func(block *types.Block) {
				c.Event(nil, "eth_changed", id)
			}
			id = self.filterManager.InstallFilter(filter)
			c.Write(id, msg.Id)
		case "eth_filterLogs":
			filter := self.filterManager.GetFilter(int(msg.Arguments().Get(0).Uint()))
			if filter != nil {
				c.Write(toMessages(filter.Find()), msg.Id)
			}
		}

	})

	wsServ.Listen()
}

func toMessages(messages state.Messages) (msgs []xeth.JSMessage) {
	msgs = make([]xeth.JSMessage, len(messages))
	for i, msg := range messages {
		msgs[i] = xeth.NewJSMessage(msg)
	}

	return
}

func StartWebSockets(eth *eth.Ethereum) {
	wslogger.Infoln("Starting WebSockets")

	sock := NewWebSocketServer(eth)
	go sock.Serv()
}

// TODO This is starting to become a generic method. Move to utils
func mapToTxParams(object map[string]interface{}) map[string]string {
	// Default values
	if object["from"] == nil {
		object["from"] = ""
	}
	if object["to"] == nil {
		object["to"] = ""
	}
	if object["value"] == nil {
		object["value"] = ""
	}
	if object["gas"] == nil {
		object["gas"] = ""
	}
	if object["gasPrice"] == nil {
		object["gasPrice"] = ""
	}

	var dataStr string
	var data []string
	if str, ok := object["data"].(string); ok {
		data = []string{str}
	}

	for _, str := range data {
		if ethutil.IsHex(str) {
			str = str[2:]

			if len(str) != 64 {
				str = ethutil.LeftPadString(str, 64)
			}
		} else {
			str = ethutil.Bytes2Hex(ethutil.LeftPadBytes(ethutil.Big(str).Bytes(), 32))
		}

		dataStr += str
	}
	object["data"] = dataStr

	conv := make(map[string]string)
	for key, value := range object {
		if v, ok := value.(string); ok {
			conv[key] = v
		}
	}

	return conv
}
