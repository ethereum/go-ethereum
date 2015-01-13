package rpc

import (
	"encoding/json"
	"io"
	"net/http"
)

type jsonWrapper struct{}

func (self jsonWrapper) Send(writer io.Writer, v interface{}) (n int, err error) {
	var payload []byte
	payload, err = json.Marshal(v)
	if err != nil {
		jsonlogger.Fatalln("Error marshalling JSON", err)
		return 0, err
	}
	jsonlogger.Infof("Sending payload: %s", payload)

	return writer.Write(payload)
}

func (self jsonWrapper) ParseRequestBody(req *http.Request) (RpcRequest, error) {
	var reqParsed RpcRequest

	// Convert JSON to native types
	d := json.NewDecoder(req.Body)
	// d.UseNumber()
	defer req.Body.Close()
	err := d.Decode(&reqParsed)

	if err != nil {
		jsonlogger.Errorln("Error decoding JSON: ", err)
		return reqParsed, err
	}
	jsonlogger.DebugDetailf("Parsed request: %s", reqParsed)

	return reqParsed, nil
}

func (self jsonWrapper) GetRequestReply(xeth *EthereumApi, req *RpcRequest, reply *interface{}) error {
	// call function for request method
	jsonlogger.DebugDetailf("%T %s", req.Params, req.Params)
	switch req.Method {
	case "eth_coinbase":
		return xeth.GetCoinbase(reply)
	case "eth_listening":
		return xeth.GetIsListening(reply)
	case "eth_mining":
		return xeth.GetIsMining(reply)
	case "eth_peerCount":
		return xeth.GetPeerCount(reply)
	case "eth_countAt":
		args, err := req.ToGetTxCountArgs()
		if err != nil {
			return err
		}
		return xeth.GetTxCountAt(args, reply)
	// case "eth_codeAt":
	// return nil
	case "eth_balanceAt":
		args, err := req.ToGetBalanceArgs()
		if err != nil {
			return err
		}
		return xeth.GetBalanceAt(args, reply)
	case "eth_stateAt":
		args, err := req.ToGetStorageArgs()
		if err != nil {
			return err
		}
		return xeth.GetStorageAt(args, reply)
	case "eth_blockByNumber", "eth_blockByHash":
		args, err := req.ToGetBlockArgs()
		if err != nil {
			return err
		}
		return xeth.GetBlock(args, reply)
	default:
		return NewErrorResponse(ErrorNotImplemented)
	}

	jsonlogger.DebugDetailf("Reply: %T %s", reply, reply)
	return nil
}

var JSON jsonWrapper
