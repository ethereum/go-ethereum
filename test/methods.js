require('es6-promise').polyfill();

var assert = require('assert');
var web3 = require('../index.js');
web3.setProvider(new web3.providers.WebSocketProvider('http://localhost:8080')); // TODO: create some mock provider

var methodExists = function (object, method) {
    assert.equal('function', typeof object[method], 'method ' + method + ' is not implemented');
};

var propertyExists = function (object, property) {
    assert.equal('object', typeof object[property], 'property ' + property + ' is not implemented');
};

describe('web3', function() {
    describe('eth', function() {
        it('should have all methods implemented', function() {
            methodExists(web3.eth, 'balanceAt');
            methodExists(web3.eth, 'stateAt');
            methodExists(web3.eth, 'storageAt');
            methodExists(web3.eth, 'countAt');
            methodExists(web3.eth, 'codeAt');
            methodExists(web3.eth, 'transact');
            methodExists(web3.eth, 'call');
            methodExists(web3.eth, 'block');
            methodExists(web3.eth, 'transaction');
            methodExists(web3.eth, 'uncle');
            methodExists(web3.eth, 'compilers');
            methodExists(web3.eth, 'lll');
            methodExists(web3.eth, 'solidity');
            methodExists(web3.eth, 'serpent');
            methodExists(web3.eth, 'logs');
        });

        it('should have all properties implemented', function () {
            propertyExists(web3.eth, 'coinbase');
            propertyExists(web3.eth, 'listening');
            propertyExists(web3.eth, 'mining');
            propertyExists(web3.eth, 'gasPrice');
            propertyExists(web3.eth, 'account');
            propertyExists(web3.eth, 'accounts');
            propertyExists(web3.eth, 'peerCount');
            propertyExists(web3.eth, 'defaultBlock');
            propertyExists(web3.eth, 'number');
        });
    });
})


