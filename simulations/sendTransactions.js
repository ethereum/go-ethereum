var firstAccount = web3.eth.accounts[0]
var secondAccount = web3.eth.accounts[1] 
var thirdAccount = web3.eth.accounts[2]

console.log(firstAccount)

web3.eth.sendTransaction({
  from: web3.eth.accounts[0],
  to: web3.eth.accounts[1],
  value: 623,
  gas: 50000,
  gasPrice: 20
})

web3.eth.sendTransaction({
  from: web3.eth.accounts[0],
  to: web3.eth.accounts[2],
  value: 291,
  gas: 50000,
  gasPrice: 20
})

web3.eth.sendTransaction({
  from: web3.eth.accounts[1],
  to: web3.eth.accounts[3],
  value: 53039,
  gas: 50000,
  gasPrice: 20
})
