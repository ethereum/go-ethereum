require('es6-promise').polyfill();

var assert = require('assert');
var web3 = require('../index.js');
var u = require('./utils.js');
web3.setProvider(new web3.providers.WebSocketProvider('http://localhost:8080')); // TODO: create some mock provider

describe('web3', function() {
    describe('db', function() {
        u.methodExists(web3.db, 'put');
        u.methodExists(web3.db, 'get');
        u.methodExists(web3.db, 'putString');
        u.methodExists(web3.db, 'getString');
    });
});

