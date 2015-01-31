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
        console.error('indexed param with name ' + name + ' not found');
        return undefined;
    }
    return inputs[index];
};

var indexedParamsToTopics = function (event, indexed) {
    // sort keys?
    return Object.keys(indexed).map(function (key) {
        // TODO: simplify this!
        var parser = abi.inputParser([{
            name: 'test', 
            inputs: [inputWithName(event.inputs, key)] 
        }]);

        var value = indexed[key];
        if (value instanceof Array) {
            return value.map(function (v) {
                return parser.test(v);
            }); 
        }
        return parser.test(value);
    });
};

var implementationOfEvent = function (address, signature, event) {
    
    // valid options are 'earliest', 'latest', 'offset' and 'max', as defined for 'eth.watch'
    return function (indexed, options) {
        var o = options || {};
        o.address = address;
        o.topic = [];
        o.topic.push(signature);
        if (indexed) {
            o.topic = o.topic.concat(indexedParamsToTopics(event, indexed));
        }
        return o;
    };
};

module.exports = implementationOfEvent;

