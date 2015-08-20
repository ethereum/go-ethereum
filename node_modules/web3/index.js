var web3 = require('./lib/web3');
var namereg = require('./lib/web3/namereg');

web3.providers.HttpProvider = require('./lib/web3/httpprovider');
web3.providers.IpcProvider = require('./lib/web3/ipcprovider');

web3.eth.contract = require('./lib/web3/contract');
web3.eth.namereg = namereg.namereg;
web3.eth.ibanNamereg = namereg.ibanNamereg;
web3.eth.sendIBANTransaction = require('./lib/web3/transfer');
web3.eth.iban = require('./lib/web3/iban');

// dont override global variable
if (typeof window !== 'undefined' && typeof window.web3 === 'undefined') {
    window.web3 = web3;
}

module.exports = web3;

