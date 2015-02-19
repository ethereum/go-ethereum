var assert = require('assert');
var jsonrpc = require('../lib/jsonrpc');

describe('jsonrpc', function () {
    describe('toPayload', function () {
        it('should create basic payload', function () {
            
            // given 
            var method = 'helloworld';

            // when
            var payload = jsonrpc.toPayload(method);

            // then
            assert.equal(payload.jsonrpc, '2.0');
            assert.equal(payload.method, method);
            assert.equal(payload.params instanceof Array, true);
            assert.equal(payload.params.length, 0);
            assert.equal(typeof payload.id, 'number');
        });
        
        it('should create payload with params', function () {
            
            // given 
            var method = 'helloworld1';
            var params = [123, 'test'];

            // when
            var payload = jsonrpc.toPayload(method, params);

            // then
            assert.equal(payload.jsonrpc, '2.0');
            assert.equal(payload.method, method);
            assert.equal(payload.params.length, 2);
            assert.equal(payload.params[0], params[0]);
            assert.equal(payload.params[1], params[1]);
            assert.equal(typeof payload.id, 'number');
        });
    });
});
