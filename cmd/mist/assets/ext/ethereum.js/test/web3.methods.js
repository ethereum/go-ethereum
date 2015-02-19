var assert = require('assert');
var web3 = require('../index.js');
var u = require('./test.utils.js');

describe('web3', function() {
    u.methodExists(web3, 'sha3');
    u.methodExists(web3, 'toAscii');
    u.methodExists(web3, 'fromAscii');
});

