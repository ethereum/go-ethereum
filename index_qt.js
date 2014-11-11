var web3 = require('./lib/main');
web3.providers.QtProvider = require('./lib/qt');

module.exports = web3;
