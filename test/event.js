var assert = require('assert');
var event = require('../lib/event.js');

describe('event', function () {
    it('should create basic filter input object', function () {
        
        // given
        var address = '0x012345'; 
        var signature = '0x987654';

        // when
        var impl = event(address, signature);
        var result = impl();

        // then
        assert.equal(result.address, address); 
        assert.equal(result.topic.length, 1);
        assert.equal(result.topic[0], signature);

    });

    it('should create basic filter input object', function () {
        
        // given
        var address = '0x012345';
        var signature = '0x987654';
        var options = {
            earliest: 1,
            latest: 2,
            offset: 3,
            max: 4
        };

        // when
        var impl = event(address, signature); 
        var result = impl({}, options);

        // then
        assert.equal(result.address, address);
        assert.equal(result.topic.length, 1);
        assert.equal(result.topic[0], signature);
        assert.equal(result.earliest, options.earliest);
        assert.equal(result.latest, options.latest);
        assert.equal(result.offset, options.offset);
        assert.equal(result.max, options.max);
    
    });

});

