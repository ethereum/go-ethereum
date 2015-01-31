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
/** @file abi.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 *   Gav Wood <g@ethdev.com>
 * @date 2014
 */

var web3 = require('./web3'); 
var utils = require('./utils');
var types = require('./types');
var c = require('./const');
var f = require('./formatters');

/// Filters all function from input abi
/// @returns abi array with filtered objects of type 'function'
var filterFunctions = function (json) {
    return json.filter(function (current) {
        return current.type === 'function'; 
    }); 
};

/// Filters all events form input abi
/// @returns abi array with filtered objects of type 'event'
var filterEvents = function (json) {
    return json.filter(function (current) {
        return current.type === 'event';
    });
};

/// This method should be called if we want to check if givent type is an array type
/// @returns true if it is, otherwise false
var arrayType = function (type) {
    return type.slice(-2) === '[]';
};

var dynamicTypeBytes = function (type, value) {
    // TODO: decide what to do with array of strings
    if (arrayType(type) || type === 'string')    // only string itself that is dynamic; stringX is static length.
        return f.formatInputInt(value.length); 
    return "";
};

var inputTypes = types.inputTypes(); 

/// Formats input params to bytes
/// @param abi contract method
/// @param array of params that will be formatted to bytes
/// @returns bytes representation of input params
var toAbiInput = function (method, params) {
    var bytes = "";
    var padding = c.ETH_PADDING * 2;

    /// first we iterate in search for dynamic 
    method.inputs.forEach(function (input, index) {
        bytes += dynamicTypeBytes(input.type, params[index]);
    });

    method.inputs.forEach(function (input, i) {
        var typeMatch = false;
        for (var j = 0; j < inputTypes.length && !typeMatch; j++) {
            typeMatch = inputTypes[j].type(method.inputs[i].type, params[i]);
        }
        if (!typeMatch) {
            console.error('input parser does not support type: ' + method.inputs[i].type);
        }

        var formatter = inputTypes[j - 1].format;
        var toAppend = "";

        if (arrayType(method.inputs[i].type))
            toAppend = params[i].reduce(function (acc, curr) {
                return acc + formatter(curr);
            }, "");
        else
            toAppend = formatter(params[i]);

        bytes += toAppend; 
    });
    return bytes;
};

var dynamicBytesLength = function (type) {
    if (arrayType(type) || type === 'string')   // only string itself that is dynamic; stringX is static length.
        return c.ETH_PADDING * 2;
    return 0;
};

var outputTypes = types.outputTypes(); 

/// Formats output bytes back to param list
/// @param contract abi method
/// @param bytes representtion of output 
/// @returns array of output params 
var fromAbiOutput = function (method, output) {
    
    output = output.slice(2);
    var result = [];
    var padding = c.ETH_PADDING * 2;

    var dynamicPartLength = method.outputs.reduce(function (acc, curr) {
        return acc + dynamicBytesLength(curr.type);
    }, 0);
    
    var dynamicPart = output.slice(0, dynamicPartLength);
    output = output.slice(dynamicPartLength);

    method.outputs.forEach(function (out, i) {
        var typeMatch = false;
        for (var j = 0; j < outputTypes.length && !typeMatch; j++) {
            typeMatch = outputTypes[j].type(method.outputs[i].type);
        }

        if (!typeMatch) {
            console.error('output parser does not support type: ' + method.outputs[i].type);
        }

        var formatter = outputTypes[j - 1].format;
        if (arrayType(method.outputs[i].type)) {
            var size = f.formatOutputUInt(dynamicPart.slice(0, padding));
            dynamicPart = dynamicPart.slice(padding);
            var array = [];
            for (var k = 0; k < size; k++) {
                array.push(formatter(output.slice(0, padding))); 
                output = output.slice(padding);
            }
            result.push(array);
        }
        else if (types.prefixedType('string')(method.outputs[i].type)) {
            dynamicPart = dynamicPart.slice(padding); 
            result.push(formatter(output.slice(0, padding)));
            output = output.slice(padding);
        } else {
            result.push(formatter(output.slice(0, padding)));
            output = output.slice(padding);
        }
    });

    return result;
};

/// @returns display name for method eg. multiply(uint256) -> multiply
var methodDisplayName = function (method) {
    var length = method.indexOf('('); 
    return length !== -1 ? method.substr(0, length) : method;
};

/// @returns overloaded part of method's name
var methodTypeName = function (method) {
    /// TODO: make it invulnerable
    var length = method.indexOf('(');
    return length !== -1 ? method.substr(length + 1, method.length - 1 - (length + 1)) : "";
};

/// @param json abi for contract
/// @returns input parser object for given json abi
/// TODO: refactor creating the parser, do not double logic from contract
var inputParser = function (json) {
    var parser = {};
    json.forEach(function (method) {
        var displayName = methodDisplayName(method.name); 
        var typeName = methodTypeName(method.name);

        var impl = function () {
            var params = Array.prototype.slice.call(arguments);
            return toAbiInput(method, params);
        };
       
        if (parser[displayName] === undefined) {
            parser[displayName] = impl;
        }

        parser[displayName][typeName] = impl;
    });

    return parser;
};

/// @param json abi for contract
/// @returns output parser for given json abi
var outputParser = function (json) {
    var parser = {};
    json.forEach(function (method) {

        var displayName = methodDisplayName(method.name); 
        var typeName = methodTypeName(method.name);

        var impl = function (output) {
            return fromAbiOutput(method, output);
        };

        if (parser[displayName] === undefined) {
            parser[displayName] = impl;
        }

        parser[displayName][typeName] = impl;
    });

    return parser;
};

/// @param method name for which we want to get method signature
/// @returns (promise) contract method signature for method with given name
var methodSignature = function (name) {
    return web3.sha3(web3.fromAscii(name)).slice(0, 2 + c.ETH_METHOD_SIGNATURE_LENGTH * 2);
};

module.exports = {
    inputParser: inputParser,
    outputParser: outputParser,
    methodSignature: methodSignature,
    methodDisplayName: methodDisplayName,
    methodTypeName: methodTypeName,
    filterFunctions: filterFunctions,
    filterEvents: filterEvents
};

