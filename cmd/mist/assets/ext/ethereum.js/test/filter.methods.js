var assert = require('assert');
var filter = require('../lib/filter');
var u = require('./test.utils.js');

var empty = function () {};
var implementation = {
    newFilter: empty,
    getMessages: empty,
    uninstallFilter: empty,
    startPolling: empty,
    stopPolling: empty,
};

describe('web3', function () {
    describe('eth', function () {
        describe('filter', function () {
            var f = filter({}, implementation);

            u.methodExists(f, 'arrived');
            u.methodExists(f, 'happened');
            u.methodExists(f, 'changed');
            u.methodExists(f, 'messages');
            u.methodExists(f, 'logs');
            u.methodExists(f, 'uninstall');
        });
    });
});
