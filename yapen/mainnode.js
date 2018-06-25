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
    acc0 = web3.eth.accounts[0]
    acc1 = web3.eth.accounts[1]
    console.log("[mainnode]: account list: " + eth.accounts)
    console.log("[mainnode]: acc0 balance: " + web3.fromWei(web3.eth.getBalance(acc0)))
    console.log("[mainnode]: acc1 balance: " + web3.fromWei(web3.eth.getBalance(acc1)))

    miner.setEtherbase(eth.accounts[0])
    miner.start()

    personal.unlockAccount(acc0, "123456")
    eth.sendTransaction({from:acc0,to:acc1,value:web3.toWei(1,"ether")})
    while (1) {
        balance0 = web3.fromWei(web3.eth.getBalance(acc0))
        balance1 = web3.fromWei(web3.eth.getBalance(acc1))
        if (balance0 >0 && balance1 > 0) {
            miner.stop()
            break
        }
        sleep(1000)
    }
    console.log("[mainnode]: acc0 balance: " + web3.fromWei(web3.eth.getBalance(acc0)))
    console.log("[mainnode]: acc1 balance: " + web3.fromWei(web3.eth.getBalance(acc1)))

    console.log("[mainnode]: admin.nodeInfo: " + admin.nodeInfo.id)

    sleep(5000)
}

interaction()
