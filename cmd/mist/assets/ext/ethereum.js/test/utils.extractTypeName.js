var assert = require('assert');
var utils = require('../lib/utils.js');

describe('utils', function () {
    describe('extractTypeName', function () {
        it('should extract type name from method with no params', function () {
            
            // given
            var test = 'helloworld()';

            // when
            var typeName = utils.extractTypeName(test); 

            // then
            assert.equal(typeName, '');
        });

        it('should extract type name from method with one param', function () {
            
            // given
            var test = 'helloworld1(int)';

            // when
            var typeName = utils.extractTypeName(test);

            // then
            assert.equal(typeName, 'int');
        });
        
        it('should extract type name from method with two params', function () {
            
            // given
            var test = 'helloworld2(int,string)';

            // when
            var typeName = utils.extractTypeName(test);

            // then
            assert.equal(typeName, 'int,string');
        });
        
        it('should extract type name from method with spaces between params', function () {
            
            // given
            var test = 'helloworld3(int, string)';

            // when
            var typeName = utils.extractTypeName(test);

            // then
            assert.equal(typeName, 'int,string');
        });

    });
});
