var assert = require('assert');
var jsonrpc = require('../lib/jsonrpc');

describe('jsonrpc', function () {
    describe('toBatchPayload', function () {
        it('should create basic batch payload', function () {
            
            // given 
            var messages = [{
                method: 'helloworld'
            }, {
                method: 'test2',
                params: [1]
            }];

            // when
            var payload = jsonrpc.toBatchPayload(messages);

            // then
            assert.equal(payload instanceof Array, true);
            assert.equal(payload.length, 2);
            assert.equal(payload[0].jsonrpc, '2.0');
            assert.equal(payload[1].jsonrpc, '2.0');
            assert.equal(payload[0].method, 'helloworld');
            assert.equal(payload[1].method, 'test2');
            assert.equal(payload[0].params instanceof Array, true);
            assert.equal(payload[1].params.length, 1);
            assert.equal(payload[1].params[0], 1);
            assert.equal(typeof payload[0].id, 'number');
            assert.equal(typeof payload[1].id, 'number');
            assert.equal(payload[0].id + 1, payload[1].id);
        });
        
        it('should create batch payload for empty input array', function () {
            
            // given 
            var messages = [];

            // when
            var payload = jsonrpc.toBatchPayload(messages);

            // then
            assert.equal(payload instanceof Array, true);
            assert.equal(payload.length, 0);
        });
    });
});
