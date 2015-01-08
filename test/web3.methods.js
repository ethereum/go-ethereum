require('es6-promise').polyfill();

var assert = require('assert');
var web3 = require('../index.js');
var u = require('./utils.js');
web3.setProvider(new web3.providers.WebSocketProvider('http://localhost:8080')); // TODO: create some mock provider

describe('web3', function() {
    it('should have all methods implemented', function() {
        u.methodExists(web3, 'sha3');
        u.methodExists(web3, 'toAscii');
        u.methodExists(web3, 'fromAscii');
        u.methodExists(web3, 'toFixed');
        u.methodExists(web3, 'fromFixed');
        u.methodExists(web3, 'offset');
    });
});

