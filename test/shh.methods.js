require('es6-promise').polyfill();

var assert = require('assert');
var web3 = require('../index.js');
var u = require('./utils.js');

describe('web3', function() {
    describe('shh', function() {
        u.methodExists(web3.shh, 'post');
        u.methodExists(web3.shh, 'newIdentity');
        u.methodExists(web3.shh, 'haveIdentity');
        u.methodExists(web3.shh, 'newGroup');
        u.methodExists(web3.shh, 'addToGroup');
    });
});

