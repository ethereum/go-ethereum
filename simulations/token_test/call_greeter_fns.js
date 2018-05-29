var Web3 = require('web3')
var web3 = new Web3(new Web3.providers.HttpProvider("http://127.0.0.1:8545"));
var greeter_addr = process.env.GREETER
var proxy_greeter_addr = process.env.PROXYGREETER

var greeterContract = web3.eth.contract([{"constant":false,"inputs":[{"name":"_greeting","type":"string"}],"name":"setGreeting","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"greet","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"greeting","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"inputs":[{"name":"_greeting","type":"string"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"}]).at(greeter_addr)
var proxygreeterContract = web3.eth.contract([{"constant":false,"inputs":[{"name":"_greeting","type":"string"}],"name":"proxySetGreeting","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"inputs":[{"name":"_address","type":"address"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"}]).at(proxy_greeter_addr)

var greetings = ["My new greeting", "greeting 3", "greeting four", "greeting five"]

for(i = 0; i < greetings.length;i++){

}

sendTxes = async () => {
  for(i = 0; i < greetings.length;i++){
    //var hash = await proxygreeterContract.proxySetGreeting.sendTransaction.call(greetings[i], {from: '0x43EC6d0942f7fAeF069F7F63D0384a27f529B062', gas: 3000000})
    console.log(greetings[i])
    var res = await proxygreeterContract.proxySetGreeting.sendTransaction(greetings[i], {from: '0x43EC6d0942f7fAeF069F7F63D0384a27f529B062', gas: 3000000})
    console.log(res)
    await sleep(10000)
    //setTimeout(() => console.log('timeout'), 3000)
    console.log(greeterContract.greet())
  }
}

sendTxes()

function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}