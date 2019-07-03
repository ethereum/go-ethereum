// This file is a test-utility for testing clef-functionality
//
// Start clef with
//
// build/bin/clef --4bytedb=./cmd/clef/4byte.json --rpc
//
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
    if (typeof accts == 'undefined' || accts.length == 0){
        accts = eth.accounts
        console.log("Got accounts ", accts);
    }
}
init()
function testTx(){
    if( accts && accts.length > 0) {
        var a = accts[0]
        var txdata = eth.signTransaction({from: a, to: a, value: 1, nonce: 1, gas: 1, gasPrice: 1})
        var v = parseInt(txdata.tx.v)
        console.log("V value: ", v)
        if (v == 37 || v == 38){
            console.log("Mainnet 155-protected chainid was used")
        }
        if (v == 27 || v == 28){
            throw new Error("Mainnet chainid was used, but without replay protection!")
        }
    }
}
function testSignText(){
    if( accts && accts.length > 0){
        var a = accts[0]
        var r = eth.sign(a, "0x68656c6c6f20776f726c64"); //hello world
        console.log("signing response",  r)
    }
}
function testClique(){
    if( accts && accts.length > 0){
        var a = accts[0]
        var r = debug.testSignCliqueBlock(a, 0); // Sign genesis
        console.log("signing response",  r)
        if( a != r){
            throw new Error("Requested signing by "+a+ " but got sealer "+r)
        }
    }
}

function test(){
    var tests = [
        testTx,
        testSignText,
        testClique,
    ]
    for( i in tests){
        try{
            tests[i]()
        }catch(err){
            console.log(err)
        }
    }
 }
