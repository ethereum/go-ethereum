var assert = require('assert');
var abi = require('../lib/abi.js');
var clone = function (object) { return JSON.parse(JSON.stringify(object)); };

var description =  [{
    "name": "test",
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

describe('abi', function() {
    describe('inputParser', function() {
        it('should parse input uint', function() {

            var d = clone(description);

            d[0].inputs = [
                { type: "uint256" }
            ];
            
            var parser = abi.inputParser(d);
            assert.equal(parser.test(1), "0000000000000000000000000000000000000000000000000000000000000001");
            assert.equal(parser.test(10), "000000000000000000000000000000000000000000000000000000000000000a");

            d[0].inputs = [
                { type: "uint128" }
            ];

            var parser = abi.inputParser(d);
            assert.equal(parser.test(1), "0000000000000000000000000000000000000000000000000000000000000001");
            assert.equal(parser.test(10), "000000000000000000000000000000000000000000000000000000000000000a");

            d[0].inputs = [
                { type: "uint" }
            ];

            var parser = abi.inputParser(d);
            assert.equal(parser.test(1), "0000000000000000000000000000000000000000000000000000000000000001");
            assert.equal(parser.test(10), "000000000000000000000000000000000000000000000000000000000000000a");
            
        });

        it('should parse input int', function() {

            var d = clone(description);

            d[0].inputs = [
                { type: "int256" }
            ];
            
            var parser = abi.inputParser(d);
            assert.equal(parser.test(1), "0000000000000000000000000000000000000000000000000000000000000001");
            assert.equal(parser.test(10), "000000000000000000000000000000000000000000000000000000000000000a");

            d[0].inputs = [
                { type: "int128" }
            ];

            var parser = abi.inputParser(d);
            assert.equal(parser.test(1), "0000000000000000000000000000000000000000000000000000000000000001");
            assert.equal(parser.test(10), "000000000000000000000000000000000000000000000000000000000000000a");

            d[0].inputs = [
                { type: "int" }
            ];

            var parser = abi.inputParser(d);
            assert.equal(parser.test(1), "0000000000000000000000000000000000000000000000000000000000000001");
            assert.equal(parser.test(10), "000000000000000000000000000000000000000000000000000000000000000a");
            
        });

        it('should parse input hash', function() {

            var d = clone(description);

            d[0].inputs = [
                { type: "hash256" }
            ];
            
            var parser = abi.inputParser(d);
            assert.equal(parser.test(1), "0000000000000000000000000000000000000000000000000000000000000001");

            d[0].inputs = [
                { type: "hash128" }
            ];

            var parser = abi.inputParser(d);
            assert.equal(parser.test(1), "0000000000000000000000000000000000000000000000000000000000000001");

            d[0].inputs = [
                { type: "hash" }
            ];

            var parser = abi.inputParser(d);
            assert.equal(parser.test(1), "0000000000000000000000000000000000000000000000000000000000000001");
            
        });

        it('should parse input string', function() {
            
            var d = clone(description);

            d[0].inputs = [
                { type: "string" }
            ];

            var parser = abi.inputParser(d);
            assert.equal(parser.test('hello'), "68656c6c6f000000000000000000000000000000000000000000000000000000");
            assert.equal(parser.test('world'), "776f726c64000000000000000000000000000000000000000000000000000000");
        });

    });


    describe('outputParser', function() {
        it('parse ...', function() {
            
            var d = clone(description);

            d[0].outputs = [
                { type: "string" }
            ];

            var parser = abi.outputParser(d);
            assert.equal(parser.test("0x68656c6c6f00000000000000000000000000000000000000000000000000000")[0], 'hello');
            assert.equal(parser.test("0x776f726c6400000000000000000000000000000000000000000000000000000")[0], 'world');

        });
    });
});

