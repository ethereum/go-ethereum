var assert = require('assert');
var event = require('../lib/event.js');

describe('event', function () {
    it('should create filter input object from given', function () {
        
        // given
        var address = '0x012345'; 
        var signature = '0x987654';

        // when
        var impl = event(address, signature);
        var result = impl();

        // then
        assert.equal(result.address, address); 
        assert.equal(result.topics.length, 1);
        assert.equal(result.topics[0], signature);

    });
});

