/*
    This file is part of ethereum.js.

    ethereum.js is free software: you can redistribute it and/or modify
    it under the terms of the GNU Lesser General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    ethereum.js is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Lesser General Public License for more details.

    You should have received a copy of the GNU Lesser General Public License
    along with ethereum.js.  If not, see <http://www.gnu.org/licenses/>.
*/
/** @file contract.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2014
 */

var web3 = require('./web3'); // jshint ignore:line
var abi = require('./abi');

/**
 * This method should be called when we want to call / transact some solidity method from javascript
 * it returns an object which has same methods available as solidity contract description
 * usage example: 
 *
 * var abi = [{
 *      name: 'myMethod',
 *      inputs: [{ name: 'a', type: 'string' }],
 *      outputs: [{name: 'd', type: 'string' }]
 * }];  // contract abi
 *
 * var myContract = web3.eth.contract('0x0123123121', abi); // creation of contract object
 *
 * myContract.myMethod('this is test string param for call'); // myMethod call (implicit, default)
 * myContract.call().myMethod('this is test string param for call'); // myMethod call (explicit)
 * myContract.transact().myMethod('this is test string param for transact'); // myMethod transact
 *
 * @param address - address of the contract, which should be called
 * @param desc - abi json description of the contract, which is being created
 * @returns contract object
 */

var contract = function (address, desc) {

    desc.forEach(function (method) {
        // workaround for invalid assumption that method.name is the full anonymous prototype of the method.
        // it's not. it's just the name. the rest of the code assumes it's actually the anonymous
        // prototype, so we make it so as a workaround.
        if (method.name.indexOf('(') === -1) {
            var displayName = method.name;
            var typeName = method.inputs.map(function(i){return i.type; }).join();
            method.name = displayName + '(' + typeName + ')';
        }
    });

    var inputParser = abi.inputParser(desc);
    var outputParser = abi.outputParser(desc);

    var result = {};

    result.call = function (options) {
        result._isTransact = false;
        result._options = options;
        return result;
    };

    result.transact = function (options) {
        result._isTransact = true;
        result._options = options;
        return result;
    };

    result._options = {};
    ['gas', 'gasPrice', 'value', 'from'].forEach(function(p) {
        result[p] = function (v) {
            result._options[p] = v;
            return result;
        };
    });


    desc.forEach(function (method) {

        var displayName = abi.methodDisplayName(method.name);
        var typeName = abi.methodTypeName(method.name);

        var impl = function () {
            var params = Array.prototype.slice.call(arguments);
            var signature = abi.methodSignature(method.name);
            var parsed = inputParser[displayName][typeName].apply(null, params);

            var options = result._options || {};
            options.to = address;
            options.data = signature + parsed;
            
            var isTransact = result._isTransact === true || (result._isTransact !== false && !method.constant);
            var collapse = options.collapse !== false;
            
            // reset
            result._options = {};
            result._isTransact = null;

            if (isTransact) {
                // it's used byt natspec.js
                // TODO: figure out better way to solve this
                web3._currentContractAbi = desc;
                web3._currentContractAddress = address;

                // transactions do not have any output, cause we do not know, when they will be processed
                web3.eth.transact(options);
                return;
            }
            
            var output = web3.eth.call(options);
            var ret = outputParser[displayName][typeName](output);
            if (collapse)
            {
                if (ret.length === 1)
                    ret = ret[0];
                else if (ret.length === 0)
                    ret = null;
            }
            return ret;
        };

        if (result[displayName] === undefined) {
            result[displayName] = impl;
        }

        result[displayName][typeName] = impl;

    });

    return result;
};

module.exports = contract;

