var assert = require('assert');
var event = require('../lib/event.js');
var f = require('../lib/formatters.js');

describe('event', function () {
    describe('inputParser', function () {
        it('should create basic filter input object', function () {
            
            // given
            var address = '0x012345'; 
            var signature = '0x987654';
            var e = {
                name: 'Event',
                inputs: [{"name":"a","type":"uint256","indexed":true},{"name":"b","type":"hash256","indexed":false}]
            };

            // when
            var impl = event.inputParser(address, signature, e);
            var result = impl();

            // then
            assert.equal(result.address, address); 
            assert.equal(result.topic.length, 1);
            assert.equal(result.topic[0], signature);

        });

        it('should create filter input object with options', function () {
            
            // given
            var address = '0x012345';
            var signature = '0x987654';
            var options = {
                earliest: 1,
                latest: 2,
                offset: 3,
                max: 4
            };
            var e = {
                name: 'Event',
                inputs: [{"name":"a","type":"uint256","indexed":true},{"name":"b","type":"hash256","indexed":false}]
            };

            // when
            var impl = event.inputParser(address, signature, e); 
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

        it('should create filter input object with indexed params', function () {
        
            // given
            var address = '0x012345';
            var signature = '0x987654';
            var options = {
                earliest: 1,
                latest: 2,
                offset: 3,
                max: 4
            };
            var e = {
                name: 'Event',
                inputs: [{"name":"a","type":"uint256","indexed":true},{"name":"b","type":"hash256","indexed":false}]
            };

            // when
            var impl = event.inputParser(address, signature, e); 
            var result = impl({a: 4}, options);

            // then
            assert.equal(result.address, address);
            assert.equal(result.topic.length, 2);
            assert.equal(result.topic[0], signature);
            assert.equal(result.topic[1], f.formatInputInt(4));
            assert.equal(result.earliest, options.earliest);
            assert.equal(result.latest, options.latest);
            assert.equal(result.offset, options.offset);
            assert.equal(result.max, options.max);

        });

        it('should create filter input object with an array of indexed params', function () {
        
            // given
            var address = '0x012345';
            var signature = '0x987654';
            var options = {
                earliest: 1,
                latest: 2,
                offset: 3,
                max: 4
            };
            var e = {
                name: 'Event',
                inputs: [{"name":"a","type":"uint256","indexed":true},{"name":"b","type":"hash256","indexed":false}]
            };

            // when
            var impl = event.inputParser(address, signature, e); 
            var result = impl({a: [4, 69]}, options);

            // then
            assert.equal(result.address, address);
            assert.equal(result.topic.length, 2);
            assert.equal(result.topic[0], signature);
            assert.equal(result.topic[1][0], f.formatInputInt(4));
            assert.equal(result.topic[1][1], f.formatInputInt(69));
            assert.equal(result.earliest, options.earliest);
            assert.equal(result.latest, options.latest);
            assert.equal(result.offset, options.offset);
            assert.equal(result.max, options.max);

        });
    });
});

