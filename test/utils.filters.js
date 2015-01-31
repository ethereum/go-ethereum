var assert = require('assert');
var utils = require('../lib/utils.js');

describe('utils', function() {
    it('should filter functions and events from input array properly', function () {

        // given
        var description = [{
            "name": "test",
            "type": "function",
            "inputs": [{
                "name": "a",
                "type": "uint256"
            }
            ],
            "outputs": [
            {
                "name": "d",
                "type": "uint256"
            }
            ],
        }, {
            "name": "test2",
            "type": "event",
            "inputs": [{
                "name": "a",
                "type": "uint256"
            }
            ],
            "outputs": [
            {
                "name": "d",
                "type": "uint256"
            }
            ]
        }];
        
        // when
        var events = utils.filterEvents(description);
        var functions = utils.filterFunctions(description);

        // then
        assert.equal(events.length, 1);
        assert.equal(events[0].name, 'test2');
        assert.equal(functions.length, 1);
        assert.equal(functions[0].name, 'test');
        
    });
});
