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

// TODO: is these line is supposed to be here? 
if (process.env.NODE_ENV !== 'build') {
    var web3 = require('./web3'); // jshint ignore:line
}

var abi = require('./abi');

var contract = function (address, desc) {
    var inputParser = abi.inputParser(desc);
    var outputParser = abi.outputParser(desc);

    var contract = {};

    desc.forEach(function (method) {
        contract[method.name] = function () {
            var params = Array.prototype.slice.call(arguments);
            var parsed = inputParser[method.name].apply(null, params);

            var onSuccess = function (result) {
                return outputParser[method.name](result);
            };

            return {
                call: function (extra) {
                    extra = extra || {};
                    extra.to = address;
                    extra.data = parsed;
                    return web3.eth.call(extra).then(onSuccess);
                },
                transact: function (extra) {
                    extra = extra || {};
                    extra.to = address;
                    extra.data = parsed;
                    return web3.eth.transact(extra).then(onSuccess);
                }
            };
        };
    });

    return contract;
};

if (typeof(module) !== "undefined")
	module.exports = contract;
