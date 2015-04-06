package rpc

import (
	"encoding/json"
	// "sync"
	"testing"
	// "time"

	// "github.com/ethereum/go-ethereum/xeth"
)

func TestWeb3Sha3(t *testing.T) {
	jsonstr := `{"jsonrpc":"2.0","method":"web3_sha3","params":["0x68656c6c6f20776f726c64"],"id":64}`
	expected := "0x47173285a8d7341e5e972fc677286384f802f8ef42a5ec5f03bbfa254cb01fad"

	api := &EthereumApi{}

	var req RpcRequest
	json.Unmarshal([]byte(jsonstr), &req)

	var response interface{}
	_ = api.GetRequestReply(&req, &response)

	if response.(string) != expected {
		t.Errorf("Expected %s got %s", expected, response)
	}
}

// func TestDbStr(t *testing.T) {
// 	jsonput := `{"jsonrpc":"2.0","method":"db_putString","params":["testDB","myKey","myString"],"id":64}`
// 	jsonget := `{"jsonrpc":"2.0","method":"db_getString","params":["testDB","myKey"],"id":64}`
// 	expected := "myString"

// 	xeth := &xeth.XEth{}
// 	api := NewEthereumApi(xeth)
// 	var response interface{}

// 	var req RpcRequest
// 	json.Unmarshal([]byte(jsonput), &req)
// 	_ = api.GetRequestReply(&req, &response)

// 	json.Unmarshal([]byte(jsonget), &req)
// 	_ = api.GetRequestReply(&req, &response)

// 	if response.(string) != expected {
// 		t.Errorf("Expected %s got %s", expected, response)
// 	}
// }

// func TestDbHexStr(t *testing.T) {
// 	jsonput := `{"jsonrpc":"2.0","method":"db_putHex","params":["testDB","beefKey","0xbeef"],"id":64}`
// 	jsonget := `{"jsonrpc":"2.0","method":"db_getHex","params":["testDB","beefKey"],"id":64}`
// 	expected := "0xbeef"

// 	xeth := &xeth.XEth{}
// 	api := NewEthereumApi(xeth)
// 	defer api.db.Close()
// 	var response interface{}

// 	var req RpcRequest
// 	json.Unmarshal([]byte(jsonput), &req)
// 	_ = api.GetRequestReply(&req, &response)

// 	json.Unmarshal([]byte(jsonget), &req)
// 	_ = api.GetRequestReply(&req, &response)

// 	if response.(string) != expected {
// 		t.Errorf("Expected %s got %s", expected, response)
// 	}
// }

// func TestFilterClose(t *testing.T) {
// 	t.Skip()
// 	api := &EthereumApi{
// 		logs:     make(map[int]*logFilter),
// 		messages: make(map[int]*whisperFilter),
// 		quit:     make(chan struct{}),
// 	}

// 	filterTickerTime = 1
// 	api.logs[0] = &logFilter{}
// 	api.messages[0] = &whisperFilter{}
// 	var wg sync.WaitGroup
// 	wg.Add(1)
// 	go api.start()
// 	go func() {
// 		select {
// 		case <-time.After(500 * time.Millisecond):
// 			api.stop()
// 			wg.Done()
// 		}
// 	}()
// 	wg.Wait()
// 	if len(api.logs) != 0 {
// 		t.Error("expected logs to be empty")
// 	}

// 	if len(api.messages) != 0 {
// 		t.Error("expected messages to be empty")
// 	}
// }
