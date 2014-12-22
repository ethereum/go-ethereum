var web3 = require('./lib/main');
web3.providers.QtProvider = require('./lib/qt');
web3.contract = require('./lib/contract');

module.exports = web3;
