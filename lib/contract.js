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

/// method signature length in bytes
var ETH_METHOD_SIGNATURE_LENGTH = 4;

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
 * myContract.myMethod('this is test string param for call').call(); // myMethod call
 * myContract.myMethod('this is test string param for transact').transact() // myMethod transact
 *
 * @param address - address of the contract, which should be called
 * @param desc - abi json description of the contract, which is being created
 * @returns contract object
 */
var contract = function (address, desc) {
    var inputParser = abi.inputParser(desc);
    var outputParser = abi.outputParser(desc);

    var contract = {};

    desc.forEach(function (method) {

        var displayName = abi.methodDisplayName(method.name);
        var typeName = abi.methodTypeName(method.name);

        var impl = function () {
            var params = Array.prototype.slice.call(arguments);
            var parsed = inputParser[displayName][typeName].apply(null, params);

            var onSuccess = function (result) {
                return outputParser[displayName][typeName](result);
            };

            return {
                call: function (extra) {
                    extra = extra || {};
                    extra.to = address;
                    return abi.methodSignature(desc, method.name).then(function (signature) {
                        extra.data = signature.slice(0, 2 + ETH_METHOD_SIGNATURE_LENGTH * 2) + parsed;
                        return web3.eth.call(extra).then(onSuccess);
                    });
                },
                transact: function (extra) {
                    extra = extra || {};
                    extra.to = address;
                    return abi.methodSignature(desc, method.name).then(function (signature) {
                        extra.data = signature.slice(0, 2 + ETH_METHOD_SIGNATURE_LENGTH * 2) + parsed;
                        return web3.eth.transact(extra).then(onSuccess);
                    });
                }
            };
        };

        if (contract[displayName] === undefined) {
            contract[displayName] = impl;
        }

        contract[displayName][typeName] = impl;
    });

    return contract;
};

module.exports = contract;

