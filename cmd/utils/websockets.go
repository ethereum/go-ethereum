package utils

import (
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/websocket"
	"github.com/ethereum/go-ethereum/xeth"
)

var wslogger = logger.NewLogger("WS")

func args(v ...interface{}) []interface{} {
	return v
}

type WebSocketServer struct {
	ethereum        *eth.Ethereum
	filterCallbacks map[int][]int
}

func NewWebSocketServer(eth *eth.Ethereum) *WebSocketServer {
	return &WebSocketServer{eth, make(map[int][]int)}
}

func (self *WebSocketServer) Serv() {
	pipe := xeth.NewJSXEth(self.ethereum)

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

		case "eth_secretToAddress":
			args := msg.Arguments()

			c.Write(pipe.SecretToAddress(args.Get(0).Str()), msg.Id)

		case "eth_newFilter":
		case "eth_newFilterString":
		case "eth_messages":
			// TODO
		}

	})

	wsServ.Listen()
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
