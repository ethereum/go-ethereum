require('es6-promise').polyfill();

var assert = require('assert');
var web3 = require('../index.js');
var u = require('./utils.js');
web3.setProvider(new web3.providers.WebSocketProvider('http://localhost:8080')); // TODO: create some mock provider

describe('web3', function() {
    describe('shh', function() {
        it('should have all methods implemented', function() {
            u.methodExists(web3.shh, 'post');
            u.methodExists(web3.shh, 'newIdentity');
            u.methodExists(web3.shh, 'haveIdentity');
            u.methodExists(web3.shh, 'newGroup');
            u.methodExists(web3.shh, 'addToGroup');
        });
    });
});

