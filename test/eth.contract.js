var assert = require('assert');
var contract = require('../lib/contract.js');

describe('contract', function() {
    it('should create simple contract with one method from abi with explicit type name', function () {
        
        // given
        var description =  [{
            "name": "test(uint256)",
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
            ]
        }];
    
        // when
        var con = contract(null, description);

        // then
        assert.equal('function', typeof con.test); 
        assert.equal('function', typeof con.test['uint256']);
    });

    it('should create simple contract with one method from abi with implicit type name', function () {
    
        // given
        var description =  [{
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
            ]
        }];

        // when
        var con = contract(null, description);

        // then
        assert.equal('function', typeof con.test); 
        assert.equal('function', typeof con.test['uint256']);
    }); 

    it('should create contract with multiple methods', function () {
        
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
            ]
        }];
        
        // when
        var con = contract(null, description);

        // then
        assert.equal('function', typeof con.test); 
        assert.equal('function', typeof con.test['uint256']);
        assert.equal('function', typeof con.test2); 
        assert.equal('function', typeof con.test2['uint256']);
    });

    it('should create contract with overloaded methods', function () {
    
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
            "name": "test",
            "type": "function",
            "inputs": [{
                "name": "a",
                "type": "string"
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
        var con = contract(null, description);

        // then
        assert.equal('function', typeof con.test); 
        assert.equal('function', typeof con.test['uint256']);
        assert.equal('function', typeof con.test['string']); 
    });

    it('should create contract with no methods', function () {
        
        // given
        var description =  [{
            "name": "test(uint256)",
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
        var con = contract(null, description);

        // then
        assert.equal('undefined', typeof con.test); 

    });
});

