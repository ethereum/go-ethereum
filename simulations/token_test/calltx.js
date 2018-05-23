var Web3 = require('web3')
var web3 = new Web3(new Web3.providers.HttpProvider("http://127.0.0.1:8545"));
var contract_address = process.env.ADDR

var mytokenContract = web3.eth.contract([{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"inputs":[{"name":"initialSupply","type":"uint256"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"}]).at(contract_address)

console.log(mytokenContract.balanceOf('0x43EC6d0942f7fAeF069F7F63D0384a27f529B062'))

mytokenContract.transfer.sendTransaction('0x9e602164C5826ebb5A6B68E4AFD9Cd466043dc4A', 3, {from: '0x43EC6d0942f7fAeF069F7F63D0384a27f529B062', gas: 3000000})
