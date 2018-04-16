console.log("Checking balances...")
for(i = 0; i < web3.eth.accounts.length; i++){
  console.log(web3.eth.accounts[i] + ":" + web3.eth.getBalance(web3.eth.accounts[i]))
}
