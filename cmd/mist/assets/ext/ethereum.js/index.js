var web3 = require('./lib/web3');
web3.providers.HttpSyncProvider = require('./lib/httpsync');
web3.providers.QtSyncProvider = require('./lib/qtsync');
web3.eth.contract = require('./lib/contract');
web3.abi = require('./lib/abi');

module.exports = web3;
