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

// TODO: is these line is supposed to be here? 
if (process.env.NODE_ENV !== 'build') {
    var BigNumber = require('bignumber.js'); // jshint ignore:line
}

var web3 = require('./web3'); // jshint ignore:line

BigNumber.config({ ROUNDING_MODE: BigNumber.ROUND_DOWN });

var ETH_PADDING = 32;

/// method signature length in bytes
var ETH_METHOD_SIGNATURE_LENGTH = 4;

/// Finds first index of array element matching pattern
/// @param array
/// @param callback pattern
/// @returns index of element
var findIndex = function (array, callback) {
    var end = false;
    var i = 0;
    for (; i < array.length && !end; i++) {
        end = callback(array[i]);
    }
    return end ? i - 1 : -1;
};

/// @returns a function that is used as a pattern for 'findIndex'
var findMethodIndex = function (json, methodName) {
    return findIndex(json, function (method) {
        return method.name === methodName;
    });
};

/// @param string string to be padded
/// @param number of characters that result string should have
/// @param sign, by default 0
/// @returns right aligned string
var padLeft = function (string, chars, sign) {
    return new Array(chars - string.length + 1).join(sign ? sign : "0") + string;
};

/// @param expected type prefix (string)
/// @returns function which checks if type has matching prefix. if yes, returns true, otherwise false
var prefixedType = function (prefix) {
    return function (type) {
        return type.indexOf(prefix) === 0;
    };
};

/// @param expected type name (string)
/// @returns function which checks if type is matching expected one. if yes, returns true, otherwise false
var namedType = function (name) {
    return function (type) {
        return name === type;
    };
};

var arrayType = function (type) {
    return type.slice(-2) === '[]';
};

/// Formats input value to byte representation of int
/// If value is negative, return it's two's complement
/// If the value is floating point, round it down
/// @returns right-aligned byte representation of int
var formatInputInt = function (value) {
    var padding = ETH_PADDING * 2;
    if (value instanceof BigNumber || typeof value === 'number') {
        if (typeof value === 'number')
            value = new BigNumber(value);
        value = value.round();

        if (value.lessThan(0)) 
            value = new BigNumber("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16).plus(value).plus(1);
        value = value.toString(16);
    }
    else if (value.indexOf('0x') === 0)
        value = value.substr(2);
    else if (typeof value === 'string')
        value = formatInputInt(new BigNumber(value));
    else
        value = (+value).toString(16);
    return padLeft(value, padding);
};

/// Formats input value to byte representation of string
/// @returns left-algined byte representation of string
var formatInputString = function (value) {
    return web3.fromAscii(value, ETH_PADDING).substr(2);
};

/// Formats input value to byte representation of bool
/// @returns right-aligned byte representation bool
var formatInputBool = function (value) {
    return '000000000000000000000000000000000000000000000000000000000000000' + (value ?  '1' : '0');
};

/// Formats input value to byte representation of real
/// Values are multiplied by 2^m and encoded as integers
/// @returns byte representation of real
var formatInputReal = function (value) {
    return formatInputInt(new BigNumber(value).times(new BigNumber(2).pow(128))); 
};

var dynamicTypeBytes = function (type, value) {
    // TODO: decide what to do with array of strings
    if (arrayType(type) || type == 'string')    // only string itself that is dynamic; stringX is static length.
        return formatInputInt(value.length); 
    return "";
};

/// Setups input formatters for solidity types
/// @returns an array of input formatters 
var setupInputTypes = function () {
    
    return [
        { type: prefixedType('uint'), format: formatInputInt },
        { type: prefixedType('int'), format: formatInputInt },
        { type: prefixedType('hash'), format: formatInputInt },
        { type: prefixedType('string'), format: formatInputString }, 
        { type: prefixedType('real'), format: formatInputReal },
        { type: prefixedType('ureal'), format: formatInputReal },
        { type: namedType('address'), format: formatInputInt },
        { type: namedType('bool'), format: formatInputBool }
    ];
};

var inputTypes = setupInputTypes();

/// Formats input params to bytes
/// @param contract json abi
/// @param name of the method that we want to use
/// @param array of params that will be formatted to bytes
/// @returns bytes representation of input params
var toAbiInput = function (json, methodName, params) {
    var bytes = "";
    var index = findMethodIndex(json, methodName);

    if (index === -1) {
        return;
    }

    var method = json[index];
    var padding = ETH_PADDING * 2;

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

/// Check if input value is negative
/// @param value is hex format
/// @returns true if it is negative, otherwise false
var signedIsNegative = function (value) {
    return (new BigNumber(value.substr(0, 1), 16).toString(2).substr(0, 1)) === '1';
};

/// Formats input right-aligned input bytes to int
/// @returns right-aligned input bytes formatted to int
var formatOutputInt = function (value) {
    // check if it's negative number
    // it it is, return two's complement
    if (signedIsNegative(value)) {
        return new BigNumber(value, 16).minus(new BigNumber('ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff', 16)).minus(1);
    }
    return new BigNumber(value, 16);
};

/// Formats big right-aligned input bytes to uint
/// @returns right-aligned input bytes formatted to uint
var formatOutputUInt = function (value) {
    return new BigNumber(value, 16);
};

/// @returns input bytes formatted to real
var formatOutputReal = function (value) {
    return formatOutputInt(value).dividedBy(new BigNumber(2).pow(128)); 
};

/// @returns input bytes formatted to ureal
var formatOutputUReal = function (value) {
    return formatOutputUInt(value).dividedBy(new BigNumber(2).pow(128)); 
};

/// @returns right-aligned input bytes formatted to hex
var formatOutputHash = function (value) {
    return "0x" + value;
};

/// @returns right-aligned input bytes formatted to bool
var formatOutputBool = function (value) {
    return value === '0000000000000000000000000000000000000000000000000000000000000001' ? true : false;
};

/// @returns left-aligned input bytes formatted to ascii string
var formatOutputString = function (value) {
    return web3.toAscii(value);
};

/// @returns right-aligned input bytes formatted to address
var formatOutputAddress = function (value) {
    return "0x" + value.slice(value.length - 40, value.length);
};

var dynamicBytesLength = function (type) {
    if (arrayType(type) || type == 'string')   // only string itself that is dynamic; stringX is static length.
        return ETH_PADDING * 2;
    return 0;
};

/// Setups output formaters for solidity types
/// @returns an array of output formatters
var setupOutputTypes = function () {

    return [
        { type: prefixedType('uint'), format: formatOutputUInt },
        { type: prefixedType('int'), format: formatOutputInt },
        { type: prefixedType('hash'), format: formatOutputHash },
        { type: prefixedType('string'), format: formatOutputString },
        { type: prefixedType('real'), format: formatOutputReal },
        { type: prefixedType('ureal'), format: formatOutputUReal },
        { type: namedType('address'), format: formatOutputAddress },
        { type: namedType('bool'), format: formatOutputBool }
    ];
};

var outputTypes = setupOutputTypes();

/// Formats output bytes back to param list
/// @param contract json abi
/// @param name of the method that we want to use
/// @param bytes representtion of output 
/// @returns array of output params 
var fromAbiOutput = function (json, methodName, output) {
    var index = findMethodIndex(json, methodName);

    if (index === -1) {
        return;
    }

    output = output.slice(2);

    var result = [];
    var method = json[index];
    var padding = ETH_PADDING * 2;

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
            var size = formatOutputUInt(dynamicPart.slice(0, padding));
            dynamicPart = dynamicPart.slice(padding);
            var array = [];
            for (var k = 0; k < size; k++) {
                array.push(formatter(output.slice(0, padding))); 
                output = output.slice(padding);
            }
            result.push(array);
        }
        else if (prefixedType('string')(method.outputs[i].type)) {
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
    /// TODO: make it not vulnerable
    var length = method.indexOf('(');
    return length !== -1 ? method.substr(length + 1, method.length - 1 - (length + 1)) : "";
};

/// @param json abi for contract
/// @returns input parser object for given json abi
var inputParser = function (json) {
    var parser = {};
    json.forEach(function (method) {
        var displayName = methodDisplayName(method.name); 
        var typeName = methodTypeName(method.name);

        var impl = function () {
            var params = Array.prototype.slice.call(arguments);
            return toAbiInput(json, method.name, params);
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
            return fromAbiOutput(json, method.name, output);
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
    return web3.sha3(web3.fromAscii(name)).slice(0, 2 + ETH_METHOD_SIGNATURE_LENGTH * 2);
};

module.exports = {
    inputParser: inputParser,
    outputParser: outputParser,
    methodSignature: methodSignature,
    methodDisplayName: methodDisplayName,
    methodTypeName: methodTypeName
};

