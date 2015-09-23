// get all balances (node and console)
var getBalances = function() {
  var accountBalances = [];
  var batch = web3.createBatch();
  var getBalanceCallback = function(err, res) { 
    var address = this.params[0].toString().substring(2,42);
    var eth = web3.fromWei(res, 'ether').toFixed();
    var wei = res.toFixed();
    console.log(address + ": " + eth + " ETH " + wei + " WEI");
  }
  web3.eth.accounts.forEach(function(account) { batch.add(web3.eth.getBalance.request(account, getBalanceCallback)); });
  batch.execute();
}

// Unlock all accounts (console only)

var unlockAccounts = function(password) {
  var unlockBatch = web3.createBatch();
  var unlockAccountCallback = function(err, res) {}
  web3.eth.accounts.forEach(function(account) { unlockBatch.add(personal.unlockAccount.request(account, 'test123', unlockAccountCallback))});
  unlockBatch.execute();
  return true;
}

var distributeFromEtherbase = function(amount) {
  var etherbase = web3.eth.coinbase;
  var amount = web3.toWei(amount, 'ether');
  var accounts = web3.eth.accounts;
  var sendBatch = web3.createBatch();
  var txids = []
  var sendCallback = function(err, res) { 
    if(!err) txids.push(res);
  }
  accounts.forEach(function(account) {
    if(account != etherbase) sendBatch.add(web3.eth.sendTransaction.request({from: etherbase, to: account, value: amount}, sendCallback));
  });
  sendBatch.execute();
  return txids;
}

var getLastBlocks = function(count) {
  var last = web3.eth.blockNumber;
  var first = last - count;
  var blocks = [];
  var getBlockBatch = web3.createBatch();
  var numbers = Array.apply(null, Array(count)).map(function (_, i) {return i + first;});
  var getBlockCallback = function(err, res) { blocks.push(res)};
  numbers.forEach(function(number) { getBlockBatch.add(web3.eth.getBlock.request(number, true, getBlockCallback))});
  getBlockBatch.execute();
  return blocks;
}
