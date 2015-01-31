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
/** @file event.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2014
 */

var abi = require('./abi');
var utils = require('./utils');

var inputWithName = function (inputs, name) {
    var index = utils.findIndex(inputs, function (input) {
        return input.name === name; 
    });
    if (index === -1) {
       console.error('indexed param ' + name + ' not found in the abi');
       return undefined;
    }
    return inputs[index];
};

var indexedParamsToTopics = function (inputs, indexed) {
    Object.keys(indexed).map(function (key) {
        var inp = inputWithName(key); 
        var value = indexed[key];
        if (value instanceof Array) {
            
        }
    });
};

var implementationOfEvent = function (address, signature, event) {

    
    // valid options are 'earliest', 'latest', 'offset' and 'max', as defined for 'eth.watch'
    return function (indexed, options) {
        var o = options || {};
        o.address = address;
        o.topic = [];
        o.topic.push(signature);
        return o;
    };
};

module.exports = implementationOfEvent;

