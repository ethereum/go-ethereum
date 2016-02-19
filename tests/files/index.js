var tests = module.exports = {};

Object.defineProperties(tests, {
  blockchainTests: {
    get: require('require-all').bind(this, __dirname + '/BlockchainTests')
  },
  basicTests: {
    get: require('require-all').bind(this, __dirname + '/BasicTests/')
  },
  trieTests: {
    get: require('require-all').bind(this, __dirname + '/TrieTests/')
  },
  stateTests: {
    get: require('require-all').bind(this, __dirname + '/StateTests/')
  },
  transactionTests: {
    get: require('require-all').bind(this, __dirname + '/TransactionTests/')
  },
  vmTests: {
    get: require('require-all').bind(this, __dirname + '/VMTests')
  },
  powTests: {
    get: require('require-all').bind(this, __dirname + '/PoWTests')
  }
});
