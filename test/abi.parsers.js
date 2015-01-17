var assert = require('assert');
var BigNumber = require('bignumber.js');
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
            assert.equal(
                parser.test("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"), 
                "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
                );
            assert.equal(
                parser.test(new BigNumber("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)),
                "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
                );
            assert.equal(parser.test(0.1), "0000000000000000000000000000000000000000000000000000000000000000");
            assert.equal(parser.test(3.9), "0000000000000000000000000000000000000000000000000000000000000003");
            assert.equal(parser.test('0.1'), "0000000000000000000000000000000000000000000000000000000000000000");
            assert.equal(parser.test('3.9'), "0000000000000000000000000000000000000000000000000000000000000003");


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
            assert.equal(
                parser.test("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"), 
                "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
                );
            assert.equal(
                parser.test(new BigNumber("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)),
                "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
                );
            assert.equal(parser.test(0.1), "0000000000000000000000000000000000000000000000000000000000000000");
            assert.equal(parser.test(3.9), "0000000000000000000000000000000000000000000000000000000000000003");
            assert.equal(parser.test('0.1'), "0000000000000000000000000000000000000000000000000000000000000000");
            assert.equal(parser.test('3.9'), "0000000000000000000000000000000000000000000000000000000000000003");

        });
        
        it('should parse input uint256', function() {

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
            assert.equal(
                parser.test("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"), 
                "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
                );
            assert.equal(
                parser.test(new BigNumber("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)),
                "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
                );
            assert.equal(parser.test(0.1), "0000000000000000000000000000000000000000000000000000000000000000");
            assert.equal(parser.test(3.9), "0000000000000000000000000000000000000000000000000000000000000003");
            assert.equal(parser.test('0.1'), "0000000000000000000000000000000000000000000000000000000000000000");
            assert.equal(parser.test('3.9'), "0000000000000000000000000000000000000000000000000000000000000003");
            
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
            assert.equal(parser.test(-1), "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff");
            assert.equal(parser.test(-2), "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe");
            assert.equal(parser.test(-16), "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0");
            assert.equal(
                parser.test("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"), 
                "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
                );
            assert.equal(
                parser.test(new BigNumber("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)),
                "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
                );
            assert.equal(parser.test(0.1), "0000000000000000000000000000000000000000000000000000000000000000");
            assert.equal(parser.test(3.9), "0000000000000000000000000000000000000000000000000000000000000003");
            assert.equal(parser.test('0.1'), "0000000000000000000000000000000000000000000000000000000000000000");
            assert.equal(parser.test('3.9'), "0000000000000000000000000000000000000000000000000000000000000003");
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
            assert.equal(parser.test(-1), "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff");
            assert.equal(parser.test(-2), "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe");
            assert.equal(parser.test(-16), "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0");
            assert.equal(
                parser.test("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"), 
                "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
                );
            assert.equal(
                parser.test(new BigNumber("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)),
                "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
                );
            assert.equal(parser.test(0.1), "0000000000000000000000000000000000000000000000000000000000000000");
            assert.equal(parser.test(3.9), "0000000000000000000000000000000000000000000000000000000000000003");
            assert.equal(parser.test('0.1'), "0000000000000000000000000000000000000000000000000000000000000000");
            assert.equal(parser.test('3.9'), "0000000000000000000000000000000000000000000000000000000000000003");

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
            assert.equal(parser.test(-1), "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff");
            assert.equal(parser.test(-2), "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe");
            assert.equal(parser.test(-16), "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0");
            assert.equal(
                parser.test("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"), 
                "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
                );
            assert.equal(
                parser.test(new BigNumber("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)),
                "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
                );
            assert.equal(parser.test(0.1), "0000000000000000000000000000000000000000000000000000000000000000");
            assert.equal(parser.test(3.9), "0000000000000000000000000000000000000000000000000000000000000003");
            assert.equal(parser.test('0.1'), "0000000000000000000000000000000000000000000000000000000000000000");
            assert.equal(parser.test('3.9'), "0000000000000000000000000000000000000000000000000000000000000003");
            
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

            // given
            var d = clone(description);

            d[0].inputs = [
                { type: "hash" }
            ];
            
            // when
            var parser = abi.inputParser(d);

            // then
            assert.equal(parser.test("0x407d73d8a49eeb85d32cf465507dd71d507100c1"), "000000000000000000000000407d73d8a49eeb85d32cf465507dd71d507100c1");

        }); 

        it('should parse input hash256', function() {

            // given
            var d = clone(description);

            d[0].inputs = [
                { type: "hash256" }
            ];

            // when
            var parser = abi.inputParser(d);

            // then
            assert.equal(parser.test("0x407d73d8a49eeb85d32cf465507dd71d507100c1"), "000000000000000000000000407d73d8a49eeb85d32cf465507dd71d507100c1");

        });


        it('should parse input hash160', function() {
            // given
            var d = clone(description);

            d[0].inputs = [
                { type: "hash160" }
            ];

            // when
            var parser = abi.inputParser(d);
            
            // then
            assert.equal(parser.test("0x407d73d8a49eeb85d32cf465507dd71d507100c1"), "000000000000000000000000407d73d8a49eeb85d32cf465507dd71d507100c1");
        });

        it('should parse input address', function () {

            // given
            var d = clone(description);

            d[0].inputs = [
                { type: "address" }
            ];
            
            // when
            var parser = abi.inputParser(d)
            
            // then
            assert.equal(parser.test("0x407d73d8a49eeb85d32cf465507dd71d507100c1"), "000000000000000000000000407d73d8a49eeb85d32cf465507dd71d507100c1");

        });

        it('should parse input string', function () {
            
            // given
            var d = clone(description);

            d[0].inputs = [
                { type: "string" }
            ];

            // when
            var parser = abi.inputParser(d);

            // then
            assert.equal(
                parser.test('hello'), 
                "000000000000000000000000000000000000000000000000000000000000000568656c6c6f000000000000000000000000000000000000000000000000000000"
                );
            assert.equal(
                parser.test('world'),
                "0000000000000000000000000000000000000000000000000000000000000005776f726c64000000000000000000000000000000000000000000000000000000"
                );
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
            assert.equal(
                parser.test2('hello'), 
                "000000000000000000000000000000000000000000000000000000000000000568656c6c6f000000000000000000000000000000000000000000000000000000"
                );

        });

        it('should parse input array of ints', function () {
            
            // given
            var d = clone(description);

            d[0].inputs = [
                { type: "int[]" }
            ];

            // when
            var parser = abi.inputParser(d);

            // then
            assert.equal(
                parser.test([5, 6]),
                "0000000000000000000000000000000000000000000000000000000000000002" + 
                "0000000000000000000000000000000000000000000000000000000000000005" + 
                "0000000000000000000000000000000000000000000000000000000000000006"
                );
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
            assert.equal(
                parser.test("0x" + 
                    "0000000000000000000000000000000000000000000000000000000000000005" +
                    "68656c6c6f000000000000000000000000000000000000000000000000000000")[0],
                'hello'
                );
            assert.equal(
                parser.test("0x" + 
                    "0000000000000000000000000000000000000000000000000000000000000005" +
                    "776f726c64000000000000000000000000000000000000000000000000000000")[0], 
                'world'
                );

        });

        it('should parse output uint', function() {

            // given
            var d = clone(description);

            d[0].outputs = [
                { type: 'uint' }
            ];

            // when
            var parser = abi.outputParser(d);

            // then
            assert.equal(parser.test("0x0000000000000000000000000000000000000000000000000000000000000001")[0], 1);
            assert.equal(parser.test("0x000000000000000000000000000000000000000000000000000000000000000a")[0], 10);
            assert.equal(
                parser.test("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")[0].toString(10), 
                new BigNumber("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16).toString(10)
                );
            assert.equal(
                parser.test("0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0")[0].toString(10), 
                new BigNumber("fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0", 16).toString(10)
                );
        });
        
        it('should parse output uint256', function() {

            // given
            var d = clone(description);

            d[0].outputs = [
                { type: 'uint256' }
            ];

            // when
            var parser = abi.outputParser(d);

            // then
            assert.equal(parser.test("0x0000000000000000000000000000000000000000000000000000000000000001")[0], 1);
            assert.equal(parser.test("0x000000000000000000000000000000000000000000000000000000000000000a")[0], 10);
            assert.equal(
                parser.test("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")[0].toString(10), 
                new BigNumber("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16).toString(10)
                );
            assert.equal(
                parser.test("0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0")[0].toString(10), 
                new BigNumber("fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0", 16).toString(10)
                );
        });

        it('should parse output uint128', function() {

            // given
            var d = clone(description);

            d[0].outputs = [
                { type: 'uint128' }
            ];

            // when
            var parser = abi.outputParser(d);

            // then
            assert.equal(parser.test("0x0000000000000000000000000000000000000000000000000000000000000001")[0], 1);
            assert.equal(parser.test("0x000000000000000000000000000000000000000000000000000000000000000a")[0], 10);
            assert.equal(
                parser.test("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")[0].toString(10), 
                new BigNumber("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16).toString(10)
                );
            assert.equal(
                parser.test("0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0")[0].toString(10), 
                new BigNumber("fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0", 16).toString(10)
                );
        });

        it('should parse output int', function() {

            // given
            var d = clone(description);

            d[0].outputs = [
                { type: 'int' }
            ];

            // when
            var parser = abi.outputParser(d);

            // then
            assert.equal(parser.test("0x0000000000000000000000000000000000000000000000000000000000000001")[0], 1);
            assert.equal(parser.test("0x000000000000000000000000000000000000000000000000000000000000000a")[0], 10);
            assert.equal(parser.test("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")[0], -1);
            assert.equal(parser.test("0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0")[0], -16);
        });
        
        it('should parse output int256', function() {

            // given
            var d = clone(description);

            d[0].outputs = [
                { type: 'int256' }
            ];

            // when
            var parser = abi.outputParser(d);

            // then
            assert.equal(parser.test("0x0000000000000000000000000000000000000000000000000000000000000001")[0], 1);
            assert.equal(parser.test("0x000000000000000000000000000000000000000000000000000000000000000a")[0], 10);
            assert.equal(parser.test("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")[0], -1);
            assert.equal(parser.test("0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0")[0], -16);
        });

        it('should parse output int128', function() {

            // given
            var d = clone(description);

            d[0].outputs = [
                { type: 'int128' }
            ];

            // when
            var parser = abi.outputParser(d);

            // then
            assert.equal(parser.test("0x0000000000000000000000000000000000000000000000000000000000000001")[0], 1);
            assert.equal(parser.test("0x000000000000000000000000000000000000000000000000000000000000000a")[0], 10);
            assert.equal(parser.test("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")[0], -1);
            assert.equal(parser.test("0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0")[0], -16);
        });

        it('should parse output hash', function() {
            
            // given
            var d = clone(description);

            d[0].outputs = [
                { type: 'hash' }
            ];

            // when
            var parser = abi.outputParser(d);

            // then
            assert.equal(
                parser.test("0x000000000000000000000000407d73d8a49eeb85d32cf465507dd71d507100c1")[0],
                "0x000000000000000000000000407d73d8a49eeb85d32cf465507dd71d507100c1"
                );
        });
        
        it('should parse output hash256', function() {
        
            // given
            var d = clone(description);

            d[0].outputs = [
                { type: 'hash256' }
            ];

            // when
            var parser = abi.outputParser(d);

            // then
            assert.equal(
                parser.test("0x000000000000000000000000407d73d8a49eeb85d32cf465507dd71d507100c1")[0],
                "0x000000000000000000000000407d73d8a49eeb85d32cf465507dd71d507100c1"
                );
        });

        it('should parse output hash160', function() {
            
            // given
            var d = clone(description);

            d[0].outputs = [
                { type: 'hash160' }
            ];

            // when
            var parser = abi.outputParser(d);

            // then
            assert.equal(
                parser.test("0x000000000000000000000000407d73d8a49eeb85d32cf465507dd71d507100c1")[0],
                "0x000000000000000000000000407d73d8a49eeb85d32cf465507dd71d507100c1"
                );
            // TODO shouldnt' the expected hash be shorter?
        });

        it('should parse output address', function() {
            
            // given
            var d = clone(description);

            d[0].outputs = [
                { type: 'address' }
            ];

            // when
            var parser = abi.outputParser(d);

            // then
            assert.equal(
                parser.test("0x000000000000000000000000407d73d8a49eeb85d32cf465507dd71d507100c1")[0],
                "0x407d73d8a49eeb85d32cf465507dd71d507100c1"
                );
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
            assert.equal(
                parser.test("0x" +
                    "0000000000000000000000000000000000000000000000000000000000000005" +
                    "0000000000000000000000000000000000000000000000000000000000000005" +
                    "68656c6c6f000000000000000000000000000000000000000000000000000000" + 
                    "776f726c64000000000000000000000000000000000000000000000000000000")[0],
                'hello'
                );
            assert.equal(
                parser.test("0x" +
                    "0000000000000000000000000000000000000000000000000000000000000005" +
                    "0000000000000000000000000000000000000000000000000000000000000005" +
                    "68656c6c6f000000000000000000000000000000000000000000000000000000" + 
                    "776f726c64000000000000000000000000000000000000000000000000000000")[1],
                'world'
                );

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
            assert.equal(parser.test2("0x" + 
                    "0000000000000000000000000000000000000000000000000000000000000005" +
                    "68656c6c6f000000000000000000000000000000000000000000000000000000")[0],
                "hello"
                );

        });

        it('should parse output array', function () {
            
            // given
            var d = clone(description);
            d[0].outputs = [
                { type: 'int[]' }
            ];

            // when
            var parser = abi.outputParser(d);

            // then
            assert.equal(parser.test("0x" +
                    "0000000000000000000000000000000000000000000000000000000000000002" + 
                    "0000000000000000000000000000000000000000000000000000000000000005" + 
                    "0000000000000000000000000000000000000000000000000000000000000006")[0][0],
                5
                );
            assert.equal(parser.test("0x" +
                    "0000000000000000000000000000000000000000000000000000000000000002" + 
                    "0000000000000000000000000000000000000000000000000000000000000005" + 
                    "0000000000000000000000000000000000000000000000000000000000000006")[0][1],
                6
                );

        });

    });
});

