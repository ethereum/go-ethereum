var web3 = require('./lib/main');
web3.providers.QtProvider = require('./lib/qt');
web3.abi = require('./lib/abi');

module.exports = web3;
