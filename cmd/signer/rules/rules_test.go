package rules

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"math/big"
	"testing"
	"github.com/ethereum/go-ethereum/cmd/signer/core"
)

const JS = `
/**
This is an example implementation of a Javascript rule file. 

When the signer receives a request over the external API, the corresponding method is evaluated. 
Three things can happen: 

1. The method returns "Approve". This means the operation is permitted. 
2. The method returns "Reject". This means the operation is rejected. 
3. Anything else; other return values [*], method not implemented or exception occurred during processing. This means
that the operation will continue to manual processing, via the regular UI method chosen by the user. 

[*] Note: Future version of the ruleset may use more complex json-based returnvalues, making it possible to not 
only respond Approve/Reject/Manual, but also modify responses. For example, choose to list only one, but not all 
accounts in a list-request. The points above will continue to hold for non-json based responses ("Approve"/"Reject").

**/

function ApproveListing(request){
	console.log("In js approve listing");
	console.log(request.accounts[3].Address)
	console.log(request.meta.Remote)
	return "Approve"
}

function ApproveTx(request){
	console.log("test");
	console.log("from");
	return "Reject";
}

function test(thing){
	console.log(thing.String())
}

`

func hexAddr(a string) common.Address { return common.BytesToAddress(common.Hex2Bytes(a)) }
func mixAddr(a string) (*common.MixedcaseAddress, error) {
	return common.NewMixedcaseAddressFromString(a)
}

func initRuleEngine(js string) (*rulesetUi, error) {
	r, err := NewRuleEvaluator()
	if err != nil {
		return nil, fmt.Errorf("Failed to create js engine: %v", err)
	}
	if err = r.Init(js); err != nil {
		return nil, fmt.Errorf("Failed to load bootstrap js: %v", err)
	}
	return r, nil
}

func TestListRequest(t *testing.T) {
	accs := make([]core.Account, 5)

	for i, _ := range accs {
		addr := fmt.Sprintf("000000000000000000000000000000000000000%x", i)
		acc := core.Account{
			Address: common.BytesToAddress(common.Hex2Bytes(addr)),
			URL:     accounts.URL{Scheme: "test", Path: fmt.Sprintf("acc-%d", i)},
		}
		accs[i] = acc
	}

	js := `function ApproveListing(){ return "Approve" }`

	r, err := initRuleEngine(js)
	if err != nil {
		t.Errorf("Couldn't create evaluator %v", err)
		return
	}
	resp, err := r.ApproveListing(&core.ListRequest{
		accs,
		core.Metadata{
			"remoteip", "localip", "inproc",
		},
	})
	if len(resp.Accounts) != len(accs) {
		t.Errorf("Expected check to resolve to 'Approve'")
	}
}

func TestSignTxRequest(t *testing.T) {

	js := `
	function ApproveTx(jsonstr){
		console.log(jsonstr)
		r = JSON.parse(jsonstr)
		console.log("transaction.from", r.transaction.from);
		console.log("transaction.to", r.transaction.to);
		console.log("transaction.value", r.transaction.value);
		console.log("transaction.nonce", r.transaction.nonce);
		if(r.transaction.from.toLowerCase()=="0x0000000000000000000000000000000000001337"){ return "Approve"}
		if(r.transaction.from.toLowerCase()=="0x000000000000000000000000000000000000dead"){ return "Reject"}
	}`

	r, err := initRuleEngine(js)
	if err != nil {
		t.Errorf("Couldn't create evaluator %v", err)
		return
	}
	to, err := mixAddr("000000000000000000000000000000000000dead")
	if err != nil {
		t.Error(err)
		return
	}
	from, err := mixAddr("0000000000000000000000000000000000001337")

	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("to %v", to.Address().String())
	resp, err := r.ApproveTx(&core.SignTxRequest{
		Transaction: core.SendTxArgs{
			From: *from,
			To:   to},
		Callinfo: "",
		Meta:     core.Metadata{"remoteip", "localip", "inproc"},
	})
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}
	if !resp.Approved {
		t.Errorf("Expected check to resolve to 'Approve'")
	}
}

func TestMissingFunc(t *testing.T) {
	r, err := initRuleEngine(JS)
	if err != nil {
		t.Errorf("Couldn't create evaluator %v", err)
		return
	}

	_, err = r.vm.Call("MissingMethod", nil, "test")

	if err == nil {
		t.Error("Expected error")
	}

	if r.checkApproval("MissingMethod", nil, nil) == nil {
		t.Errorf("Expected error to resolve to 'Reject'")
	}

	fmt.Printf("Err %v", err)

}
func TestStorage(t *testing.T) {

	js := `
	function testStorage(){
		storage.Put("mykey", "myvalue")
		a = storage.Get("mykey")
		
		storage.Put("mykey", ["a", "list"])  	// Should result in "a,list"
		a += storage.Get("mykey")

		
		storage.Put("mykey", {"an": "object"}) 	// Should result in "[object Object]"
		a += storage.Get("mykey")

		
		storage.Put("mykey", JSON.stringify({"an": "object"})) // Should result in '{"an":"object"}'
		a += storage.Get("mykey")

		a += storage.Get("missingkey")		//Missing keys should result in empty string
		storage.Put("","missing key==noop") // Can't store with 0-length key
		a += storage.Get("")				// Should result in ''
		
		var b = new BigNumber(2)
		var c = new BigNumber(16)//"0xf0",16)
		var d = b.plus(c)
		console.log(d)
		return a
	}
`
	r, err := initRuleEngine(js)
	if err != nil {
		t.Errorf("Couldn't create evaluator %v", err)
		return
	}

	v, err := r.vm.Call("testStorage", nil, nil)

	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}

	retval, err := v.ToString()

	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}
	exp := `myvaluea,list[object Object]{"an":"object"}`
	if retval != exp {
		t.Errorf("Unexpected data, expected '%v', got '%v'", exp, retval)
	}
	fmt.Printf("Err %v", err)

}

const ExampleTxWindow = `
	function big(str){
		if(str.slice(0,2) == "0x"){ return new BigNumber(str.slice(2),16)}
		return new BigNumber(str)
	}
	
	// Time window: 1 week
	var window = 1000* 3600*24*7;

	// Limit : 1 ether
	var limit = new BigNumber("1e18");

	function isLimitOk(transaction){
		var value = big(transaction.value)
		// Start of our window function		
		var windowstart = new Date().getTime() - window;

		var txs = [];
		var stored = storage.Get('txs');

		if(stored != ""){
			txs = JSON.parse(stored)
		}
		// First, remove all that have passed out of the time-window
		var newtxs = txs.filter(function(tx){return tx.tstamp > windowstart});
		console.log(txs, newtxs.length);
	
		// Secondly, aggregate the current sum
		sum = new BigNumber(0)

		sum = newtxs.reduce(function(agg, tx){ return big(tx.value).plus(agg)}, sum);
		console.log("ApproveTx > Sum so far", sum);
		console.log("ApproveTx > Requested", value.toNumber());
		
		// Would we exceed weekly limit ?
		return sum.plus(value).lt(limit)
		
	}
	function ApproveTx(jsonstr){
		var r = JSON.parse(jsonstr)	
		if (isLimitOk(r.transaction)){
			return "Approve"
		}
		return "Nope"
	}

	/**
	* OnApprovedTx(str) is called when a transaction has been approved and signed. The parameter
 	* 'response_str' contains the return value that will be sent to the external caller. 
	* The return value from this method is ignore - the reason for having this callback is to allow the 
	* ruleset to keep track of approved transactions. 
	*
	* When implementing rate-limited rules, this callback should be used. 
	* If a rule responds with neither 'Approve' nor 'Reject' - the tx goes to manual processing. If the user
	* then accepts the transaction, this method will be called.
	* 
	* TLDR; Use this method to keep track of signed transactions, instead of using the data in ApproveTx.
	*/
 	function OnApprovedTx(response_str){
		console.log("OnApprovedTx > called with data\n\t "+response_str)
		var resp = JSON.parse(response_str)
		var value = big(resp.tx.value)
		var txs = []
		// Load stored transactions
		var stored = storage.Get('txs');
		if(stored != ""){
			txs = JSON.parse(stored)
		}
		// Add this to the storage
		txs.push({tstamp: new Date().getTime(), value: value});
		storage.Put("txs", JSON.stringify(txs));
	}

`

func dummyTx(value hexutil.Big) *core.SignTxRequest {

	to, _ := mixAddr("000000000000000000000000000000000000dead")
	from, _ := mixAddr("000000000000000000000000000000000000dead")
	n := hexutil.Uint64(3)
	gas := hexutil.Big(*big.NewInt(21000))
	gasPrice := hexutil.Big(*big.NewInt(2000000))

	return &core.SignTxRequest{
		Transaction: core.SendTxArgs{
			From:     *from,
			To:       to,
			Value:    value,
			Nonce:    n,
			GasPrice: gas,
			Gas:      gasPrice,
		},
		Callinfo: "Warning, all your base are bellong to us",
		Meta:     core.Metadata{"remoteip", "localip", "inproc"},
	}
}
func dummySigned(value *big.Int) *types.Transaction {
	to := common.HexToAddress("000000000000000000000000000000000000dead")
	gas := big.NewInt(21000)
	gasPrice := big.NewInt(2000000)
	data := make([]byte, 0)
	return types.NewTransaction(3, to, value, gas, gasPrice, data)

}
func TestLimitWindow(t *testing.T) {

	r, err := initRuleEngine(ExampleTxWindow)
	if err != nil {
		t.Errorf("Couldn't create evaluator %v", err)
		return
	}
	if err != nil {
		t.Error(err)
		return
	}
	// 0.3 ether: 429D069189E0000 wei
	v := big.NewInt(0).SetBytes(common.Hex2Bytes("0429D069189E0000"))
	h := hexutil.Big(*v)
	// The first three should succeed
	for i := 0; i < 3; i++ {
		unsigned := dummyTx(h)
		resp, err := r.ApproveTx(unsigned)
		if err != nil {
			t.Errorf("Unexpected error %v", err)
		}
		if !resp.Approved {
			t.Errorf("Expected check to resolve to 'Approve'")
		}
		// Create a dummy signed transaction

		response := ethapi.SignTransactionResult{
			Tx:  dummySigned(v),
			Raw: common.Hex2Bytes("deadbeef"),
		}
		r.OnApprovedTx(response)
	}
	// Fourth should fail
	resp, err := r.ApproveTx(dummyTx(h))
	if resp.Approved {
		t.Errorf("Expected check to resolve to 'Reject'")
	}

}
