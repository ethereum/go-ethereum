var assert = require('assert');
var abi = require('../lib/abi.js');

describe('abi', function() {
    describe('inputParser', function() {
        it('should parse ...', function() {

            var desc =  [{
                "name": "multiply",
                "inputs": [
                {
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

            var iParser = abi.inputParser(desc);
            assert.equal(iParser.multiply(1), "0x000000000000000000000000000000000000000000000000000000000000000001");

        });
    });


    describe('outputParser', function() {
        it('parse ...', function() {

        });
    });
});

