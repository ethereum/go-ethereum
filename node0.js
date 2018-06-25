function sleep(milliseconds) {
  var start = new Date().getTime();
  for (var i = 0; i < 1e7; i++) {
    if ((new Date().getTime() - start) > milliseconds){
      break;
    }
  }
}

function interaction() {
   personal.newAccount("123456")
   personal.newAccount("123456")
   acc0 = web3.eth.accounts[0]
   acc1 = web3.eth.accounts[1]
   console.log("account list: " + eth.accounts)
   console.log("acc0 balance: " + web3.fromWei(web3.eth.getBalance(acc0)))
   console.log("acc1 balance: " + web3.fromWei(web3.eth.getBalance(acc1)))

   miner.setEtherbase(eth.accounts[0])
   miner.start()

   while (1) {
     balance = web3.fromWei(web3.eth.getBalance(acc0)) 
     if (balance > 0) {
        miner.stop()
        personal.unlockAccount(acc0, "123456")
        eth.sendTransaction({from:acc0,to:acc1,value:web3.toWei(1,"ether")})
        miner.start()
       // sleep(10000)
         //  if (balance1 = balance - 1) {
	       while (1) {
	  	  balance1 = web3.fromWei(web3.eth.getBalance(acc1))
		  if (balance1 > 0) {
		     miner.stop()
                      break
		  }
               }  
               console.log("acc0 balance: " + web3.fromWei(web3.eth.getBalance(acc0)))
               console.log("acc1 balance: " + web3.fromWei(web3.eth.getBalance(acc1)))
	 //  }
        break 
     }
  }
  console.log(admin.nodeInfo.id)
}

interaction()
