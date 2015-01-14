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
    var web3 = require('./web3'); // jshint ignore:line
}

// TODO: make these be actually accurate instead of falling back onto JS's doubles.
var hexToDec = function (hex) {
    return parseInt(hex, 16).toString();
};

var decToHex = function (dec) {
    return parseInt(dec).toString(16);
};

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
/// @returns right aligned string
var padLeft = function (string, chars) {
    return new Array(chars - string.length + 1).join("0") + string;
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

/// Setups input formatters for solidity types
/// @returns an array of input formatters 
var setupInputTypes = function () {
    
    /// Formats input value to byte representation of int
    /// @returns right-aligned byte representation of int
    var formatInt = function (value) {
        var padding = 32 * 2;
        if (typeof value === 'number')
            value = value.toString(16);
        else if (value.indexOf('0x') === 0)
            value = value.substr(2);
        else if (typeof value === 'string')
            value = value.toHex(value);
        else
            value = (+value).toString(16);
        return padLeft(value, padding);
    };

    /// Formats input value to byte representation of string
    /// @returns left-algined byte representation of string
    var formatString = function (value) {
        return web3.fromAscii(value, 32).substr(2);
    };

    /// Formats input value to byte representation of bool
    /// @returns right-aligned byte representation bool
    var formatBool = function (value) {
        return '000000000000000000000000000000000000000000000000000000000000000' + (value ?  '1' : '0');
    };

    return [
        { type: prefixedType('uint'), format: formatInt },
        { type: prefixedType('int'), format: formatInt },
        { type: prefixedType('hash'), format: formatInt },
        { type: prefixedType('string'), format: formatString }, 
        { type: prefixedType('real'), format: formatInt },
        { type: prefixedType('ureal'), format: formatInt },
        { type: namedType('address'), format: formatInt },
        { type: namedType('bool'), format: formatBool }
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
    var padding = 32 * 2;

    for (var i = 0; i < method.inputs.length; i++) {
        var typeMatch = false;
        for (var j = 0; j < inputTypes.length && !typeMatch; j++) {
            typeMatch = inputTypes[j].type(method.inputs[i].type, params[i]);
        }
        if (!typeMatch) {
            console.error('input parser does not support type: ' + method.inputs[i].type);
        }

        var formatter = inputTypes[j - 1].format;
        bytes += (formatter ? formatter(params[i]) : params[i]);
    }
    return bytes;
};

/// Setups output formaters for solidity types
/// @returns an array of output formatters
var setupOutputTypes = function () {

    /// Formats input right-aligned input bytes to int
    /// @returns right-aligned input bytes formatted to int
    var formatInt = function (value) {
        return value.length <= 8 ? +parseInt(value, 16) : hexToDec(value);
    };

    /// @returns right-aligned input bytes formatted to hex
    var formatHash = function (value) {
        return "0x" + value;
    };

    /// @returns right-aligned input bytes formatted to bool
    var formatBool = function (value) {
        return value === '0000000000000000000000000000000000000000000000000000000000000001' ? true : false;
    };

    /// @returns left-aligned input bytes formatted to ascii string
    var formatString = function (value) {
        return web3.toAscii(value);
    };

    /// @returns right-aligned input bytes formatted to address
    var formatAddress = function (value) {
        return "0x" + value.slice(value.length - 40, value.length);
    };

    return [
        { type: prefixedType('uint'), format: formatInt },
        { type: prefixedType('int'), format: formatInt },
        { type: prefixedType('hash'), format: formatHash },
        { type: prefixedType('string'), format: formatString },
        { type: prefixedType('real'), format: formatInt },
        { type: prefixedType('ureal'), format: formatInt },
        { type: namedType('address'), format: formatAddress },
        { type: namedType('bool'), format: formatBool }
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
    var padding = 32 * 2;
    for (var i = 0; i < method.outputs.length; i++) {
        var typeMatch = false;
        for (var j = 0; j < outputTypes.length && !typeMatch; j++) {
            typeMatch = outputTypes[j].type(method.outputs[i].type);
        }

        if (!typeMatch) {
            // not found output parsing
            console.error('output parser does not support type: ' + method.outputs[i].type);
            continue;
        }
        var res = output.slice(0, padding);
        var formatter = outputTypes[j - 1].format;
        result.push(formatter ? formatter(res) : ("0x" + res));
        output = output.slice(padding);
    }

    return result;
};

/// @param json abi for contract
/// @returns input parser object for given json abi
var inputParser = function (json) {
    var parser = {};
    json.forEach(function (method) {
        parser[method.name] = function () {
            var params = Array.prototype.slice.call(arguments);
            return toAbiInput(json, method.name, params);
        };
    });

    return parser;
};

/// @param json abi for contract
/// @returns output parser for given json abi
var outputParser = function (json) {
    var parser = {};
    json.forEach(function (method) {
        parser[method.name] = function (output) {
            return fromAbiOutput(json, method.name, output);
        };
    });

    return parser;
};

/// @param json abi for contract
/// @param method name for which we want to get method signature
/// @returns (promise) contract method signature for method with given name
var methodSignature = function (json, name) {
    var method = json[findMethodIndex(json, name)];
    var result = name + '(';
    var inputTypes = method.inputs.map(function (inp) {
        return inp.type;
    });
    result += inputTypes.join(',');
    result += ')';

    return web3.sha3(web3.fromAscii(result));
};

module.exports = {
    inputParser: inputParser,
    outputParser: outputParser,
    methodSignature: methodSignature
};

