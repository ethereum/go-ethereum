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

            // given
            var d = clone(description);

            d[0].inputs = [
                { type: "uint" }
            ];
            
            // when
            var parser = abi.inputParser(d);

            // then
            assert.equal(parser.test(1), "0000000000000000000000000000000000000000000000000000000000000001");
            assert.equal(parser.test(10), "000000000000000000000000000000000000000000000000000000000000000a");

        });

        it('should parse input uint128', function() {

            // given
            var d = clone(description);

            d[0].inputs = [
                { type: "uint128" }
            ];

            // when
            var parser = abi.inputParser(d);

            // then
            assert.equal(parser.test(1), "0000000000000000000000000000000000000000000000000000000000000001");
            assert.equal(parser.test(10), "000000000000000000000000000000000000000000000000000000000000000a");

        });
        
        it('should parse input uint', function() {

            // given
            var d = clone(description);

            d[0].inputs = [
                { type: "uint256" }
            ];

            // when
            var parser = abi.inputParser(d);

            // then
            assert.equal(parser.test(1), "0000000000000000000000000000000000000000000000000000000000000001");
            assert.equal(parser.test(10), "000000000000000000000000000000000000000000000000000000000000000a");
            
        });

        it('should parse input int', function() {

            // given
            var d = clone(description);

            d[0].inputs = [
                { type: "int" }
            ];
            
            // when
            var parser = abi.inputParser(d);

            // then
            assert.equal(parser.test(1), "0000000000000000000000000000000000000000000000000000000000000001");
            assert.equal(parser.test(10), "000000000000000000000000000000000000000000000000000000000000000a");

        });

        it('should parse input int128', function() {

            // given
            var d = clone(description);

            d[0].inputs = [
                { type: "int128" }
            ];

            // when
            var parser = abi.inputParser(d);

            // then
            assert.equal(parser.test(1), "0000000000000000000000000000000000000000000000000000000000000001");
            assert.equal(parser.test(10), "000000000000000000000000000000000000000000000000000000000000000a");

        });

        it('should parse input int256', function() {
        
            // given
            var d = clone(description);

            d[0].inputs = [
                { type: "int256" }
            ];

            // when
            var parser = abi.inputParser(d);

            // then
            assert.equal(parser.test(1), "0000000000000000000000000000000000000000000000000000000000000001");
            assert.equal(parser.test(10), "000000000000000000000000000000000000000000000000000000000000000a");
            
        });

        it('should parse input bool', function() {
            
            // given
            var d = clone(description);

            d[0].inputs = [
                { type: 'bool' }
            ];

            // when
            var parser = abi.inputParser(d);

            // then
            assert.equal(parser.test(true), "0000000000000000000000000000000000000000000000000000000000000001");
            assert.equal(parser.test(false), "0000000000000000000000000000000000000000000000000000000000000000");

        });

        it('should parse input hash', function() {
/*
            // given
            var d = clone(description);

            d[0].inputs = [
                { type: "hash" }
            ];
            
            // when
            var parser = abi.inputParser(d);

            // then
            assert.equal(parser.test(1), "0000000000000000000000000000000000000000000000000000000000000001");
*/
        }); 

        it('should parse input hash128', function() {
/*
            // given
            var d = clone(description);

            d[0].inputs = [
                { type: "hash128" }
            ];

            // when
            var parser = abi.inputParser(d);

            // then
            assert.equal(parser.test(1), "0000000000000000000000000000000000000000000000000000000000000001");
*/
        });


        it('should parse input hash', function() {
/*
            // given
            var d = clone(description);

            d[0].inputs = [
                { type: "hash" }
            ];

            // when
            var parser = abi.inputParser(d);
            
            // then
            assert.equal(parser.test(1), "0000000000000000000000000000000000000000000000000000000000000001");
*/          
        });

        it('should parse input string', function() {
            
            // given
            var d = clone(description);

            d[0].inputs = [
                { type: "string" }
            ];

            // when
            var parser = abi.inputParser(d);

            // then
            assert.equal(parser.test('hello'), "68656c6c6f000000000000000000000000000000000000000000000000000000");
            assert.equal(parser.test('world'), "776f726c64000000000000000000000000000000000000000000000000000000");
        });

        it('should use proper method name', function () {
        
            // given
            var d = clone(description);
            d[0].name = 'helloworld';
            d[0].inputs = [
                { type: "int" }
            ];

            // when
            var parser = abi.inputParser(d);

            // then
            assert.equal(parser.helloworld(1), "0000000000000000000000000000000000000000000000000000000000000001");

        });
        
        it('should parse multiple methods', function () {
            
            // given
            var d =  [{
                name: "test",
                inputs: [{ type: "int" }],
                outputs: [{ type: "int" }]
            },{
                name: "test2",
                inputs: [{ type: "string" }],
                outputs: [{ type: "string" }]
            }];

            // when
            var parser = abi.inputParser(d);

            //then
            assert.equal(parser.test(1), "0000000000000000000000000000000000000000000000000000000000000001");
            assert.equal(parser.test2('hello'), "68656c6c6f000000000000000000000000000000000000000000000000000000");

        });
    });

    describe('outputParser', function() {
        it('should parse output string', function() {

            // given
            var d = clone(description);

            d[0].outputs = [
                { type: "string" }
            ];

            // when
            var parser = abi.outputParser(d);
            
            // then
            assert.equal(parser.test("0x68656c6c6f000000000000000000000000000000000000000000000000000000")[0], 'hello');
            assert.equal(parser.test("0x776f726c64000000000000000000000000000000000000000000000000000000")[0], 'world');

        });
        
        it('should parse output bool', function() {
            
            // given
            var d = clone(description);

            d[0].outputs = [
                { type: 'bool' }
            ];

            // when
            var parser = abi.outputParser(d);

            // then
            assert.equal(parser.test("000000000000000000000000000000000000000000000000000000000000000001")[0], true);
            assert.equal(parser.test("000000000000000000000000000000000000000000000000000000000000000000")[0], false);
            

        });

        it('should parse multiple output strings', function() {

            // given
            var d = clone(description);

            d[0].outputs = [
                { type: "string" },
                { type: "string" }
            ];

            // when
            var parser = abi.outputParser(d);

            // then
            assert.equal(parser.test("0x68656c6c6f000000000000000000000000000000000000000000000000000000776f726c64000000000000000000000000000000000000000000000000000000")[0], 'hello');
            assert.equal(parser.test("0x68656c6c6f000000000000000000000000000000000000000000000000000000776f726c64000000000000000000000000000000000000000000000000000000")[1], 'world');

        });
        
        it('should use proper method name', function () {
        
            // given
            var d = clone(description);
            d[0].name = 'helloworld';
            d[0].outputs = [
                { type: "int" }
            ];

            // when
            var parser = abi.outputParser(d);

            // then
            assert.equal(parser.helloworld("0x0000000000000000000000000000000000000000000000000000000000000001")[0], 1);

        });


        it('should parse multiple methods', function () {
            
            // given
            var d =  [{
                name: "test",
                inputs: [{ type: "int" }],
                outputs: [{ type: "int" }]
            },{
                name: "test2",
                inputs: [{ type: "string" }],
                outputs: [{ type: "string" }]
            }];

            // when
            var parser = abi.outputParser(d);

            //then
            assert.equal(parser.test("0000000000000000000000000000000000000000000000000000000000000001")[0], 1);
            assert.equal(parser.test2("0x68656c6c6f000000000000000000000000000000000000000000000000000000")[0], "hello");

        });

    });
});

