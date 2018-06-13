var abi = [{"constant":false,"inputs":[],"name":"kill","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"greet","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"inputs":[{"name":"_greeting","type":"string"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"}]

function getGreeting(address){
  var MyContract = web3.eth.contract(abi);
  var myContractInstance = MyContract.at(address);
  console.log(myContractInstance.greet())
}
