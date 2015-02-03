var assert = require('assert');
var event = require('../lib/event.js');

describe('event', function () {
    describe('outputParser', function () {
        it('should parse basic event output object', function () {
            
            // given
            var output = {
                "address":"0x78dfc5983baecf65f73e3de3a96cee24e6b7981e",
                "data":"0x000000000000000000000000000000000000000000000000000000000000004b",
                "number":2,
                "topic":[
                    "0x6e61ef44ac2747ff8b84d353a908eb8bd5c3fb118334d57698c5cfc7041196ad",
                    "0x0000000000000000000000000000000000000000000000000000000000000001"
                ]
            };

            var e = {
                name: 'Event',
                inputs: [{"name":"a","type":"bool","indexed":true},{"name":"b","type":"uint256","indexed":false}]
            };

            // when 
            var impl = event.outputParser(e);
            var result = impl(output);

            // then
            assert.equal(result.event, 'Event');
            assert.equal(result.number, 2);
            assert.equal(Object.keys(result.args).length, 2);
            assert.equal(result.args.a, true);
            assert.equal(result.args.b, 75);
        }); 

        it('should parse event output object arguments in correct order', function () {
            
            // given
            var output = {
                "address":"0x78dfc5983baecf65f73e3de3a96cee24e6b7981e",
                "data": "0x" + 
                    "000000000000000000000000000000000000000000000000000000000000004b" + 
                    "000000000000000000000000000000000000000000000000000000000000004c" +
                    "0000000000000000000000000000000000000000000000000000000000000001",
                "number":3,
                "topic":[
                    "0x6e61ef44ac2747ff8b84d353a908eb8bd5c3fb118334d57698c5cfc7041196ad",
                    "0x0000000000000000000000000000000000000000000000000000000000000001",
                    "0x0000000000000000000000000000000000000000000000000000000000000005"
                ]
            };
            
            var e = {
                name: 'Event2',
                inputs: [
                    {"name":"a","type":"bool","indexed":true},
                    {"name":"b","type":"int","indexed":false},
                    {"name":"c","type":"int","indexed":false},
                    {"name":"d","type":"int","indexed":true},
                    {"name":"e","type":"bool","indexed":false}
                ]
            };
            
            // when 
            var impl = event.outputParser(e);
            var result = impl(output);
            
            // then
            assert.equal(result.event, 'Event2');
            assert.equal(result.number, 3);
            assert.equal(Object.keys(result.args).length, 5);
            assert.equal(result.args.a, true);
            assert.equal(result.args.b, 75);
            assert.equal(result.args.c, 76);
            assert.equal(result.args.d, 5);
            assert.equal(result.args.e, true);

        });
    });
});

