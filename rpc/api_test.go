package rpc

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common/compiler"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/xeth"
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

const solcVersion = "0.9.23"

func TestCompileSolidity(t *testing.T) {

	solc, err := compiler.New("")
	if solc == nil {
		t.Skip("no solc found: skip")
	} else if solc.Version() != solcVersion {
		t.Skip("WARNING: skipping test because of solc different version (%v, test written for %v, may need to update)", solc.Version(), solcVersion)
	}
	source := `contract test {\n` +
		"   /// @notice Will multiply `a` by 7." + `\n` +
		`   function multiply(uint a) returns(uint d) {\n` +
		`       return a * 7;\n` +
		`   }\n` +
		`}\n`

	jsonstr := `{"jsonrpc":"2.0","method":"eth_compileSolidity","params":["` + source + `"],"id":64}`

	expCode := "0x605880600c6000396000f3006000357c010000000000000000000000000000000000000000000000000000000090048063c6888fa114602e57005b603d6004803590602001506047565b8060005260206000f35b60006007820290506053565b91905056"
	expAbiDefinition := `[{"constant":false,"inputs":[{"name":"a","type":"uint256"}],"name":"multiply","outputs":[{"name":"d","type":"uint256"}],"type":"function"}]`
	expUserDoc := `{"methods":{"multiply(uint256)":{"notice":"Will multiply ` + "`a`" + ` by 7."}}}`
	expDeveloperDoc := `{"methods":{}}`
	expCompilerVersion := solc.Version()
	expLanguage := "Solidity"
	expLanguageVersion := "0"
	expSource := source

	api := NewEthereumApi(xeth.NewTest(&eth.Ethereum{}, nil))

	var req RpcRequest
	json.Unmarshal([]byte(jsonstr), &req)

	var response interface{}
	err = api.GetRequestReply(&req, &response)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	respjson, err := json.Marshal(response)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	var contracts = make(map[string]*compiler.Contract)
	err = json.Unmarshal(respjson, &contracts)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(contracts) != 1 {
		t.Errorf("expected one contract, got %v", len(contracts))
	}

	contract := contracts["test"]

	if contract.Code != expCode {
		t.Errorf("Expected \n%s got \n%s", expCode, contract.Code)
	}

	if strconv.Quote(contract.Info.Source) != `"`+expSource+`"` {
		t.Errorf("Expected \n'%s' got \n'%s'", expSource, strconv.Quote(contract.Info.Source))
	}

	if contract.Info.Language != expLanguage {
		t.Errorf("Expected %s got %s", expLanguage, contract.Info.Language)
	}

	if contract.Info.LanguageVersion != expLanguageVersion {
		t.Errorf("Expected %s got %s", expLanguageVersion, contract.Info.LanguageVersion)
	}

	if contract.Info.CompilerVersion != expCompilerVersion {
		t.Errorf("Expected %s got %s", expCompilerVersion, contract.Info.CompilerVersion)
	}

	userdoc, err := json.Marshal(contract.Info.UserDoc)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	devdoc, err := json.Marshal(contract.Info.DeveloperDoc)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	abidef, err := json.Marshal(contract.Info.AbiDefinition)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if string(abidef) != expAbiDefinition {
		t.Errorf("Expected \n'%s' got \n'%s'", expAbiDefinition, string(abidef))
	}

	if string(userdoc) != expUserDoc {
		t.Errorf("Expected \n'%s' got \n'%s'", expUserDoc, string(userdoc))
	}

	if string(devdoc) != expDeveloperDoc {
		t.Errorf("Expected %s got %s", expDeveloperDoc, string(devdoc))
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
