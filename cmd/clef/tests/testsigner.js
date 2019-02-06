// This file is a test-utility for testing clef-functionality
// Start geth with
//
// build/bin/geth --nodiscover --maxpeers 0 --signer http://localhost:8550 console --preload=cmd/clef/tests/testsigner.js
//
// and in the console simply invoke
//
// > test()
//
// You can reload the file via `reload()`

function reload(){
	loadScript("./cmd/clef/tests/testsigner.js");
}
function init(){
    accts = eth.accounts
    console.log("Got accounts ", accts);
}
init()
function testTx(){
    if( accts && accts.length > 0) {
        var a = accts[0]
        var r = eth.signTransaction({from: a, to: a, value: 1, nonce: 1, gas: 1, gasPrice: 1})
   		console.log("signing response", r)
    }
}
function testSignText(){
    if( accts && accts.length > 0){
        var a = accts[0]
        var r = eth.sign(a, "0x68656c6c6f20776f726c64"); //hello world
        console.log("signing response",  r)
    }

}
function test(){
	try{
		testTx()
    }catch(err){
		console.log(err)
	}
    try{
        testSignText()
    }catch(err){
        console.log(err)
    }
}
