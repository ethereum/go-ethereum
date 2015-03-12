require=(function e(t,n,r){function s(o,u){if(!n[o]){if(!t[o]){var a=typeof require=="function"&&require;if(!u&&a)return a(o,!0);if(i)return i(o,!0);var f=new Error("Cannot find module '"+o+"'");throw f.code="MODULE_NOT_FOUND",f}var l=n[o]={exports:{}};t[o][0].call(l.exports,function(e){var n=t[o][1][e];return s(n?n:e)},l,l.exports,e,t,n,r)}return n[o].exports}var i=typeof require=="function"&&require;for(var o=0;o<r.length;o++)s(r[o]);return s})({1:[function(require,module,exports){
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

var utils = require('../utils/utils');
var c = require('../utils/config');
var types = require('./types');
var f = require('./formatters');

/**
 * throw incorrect type error
 *
 * @method throwTypeError
 * @param {String} type
 * @throws incorrect type error
 */
var throwTypeError = function (type) {
    throw new Error('parser does not support type: ' + type);
};

/** This method should be called if we want to check if givent type is an array type
 *
 * @method isArrayType
 * @param {String} type name
 * @returns {Boolean} true if it is, otherwise false
 */
var isArrayType = function (type) {
    return type.slice(-2) === '[]';
};

/**
 * This method should be called to return dynamic type length in hex
 *
 * @method dynamicTypeBytes
 * @param {String} type
 * @param {String|Array} dynamic type
 * @return {String} length of dynamic type in hex or empty string if type is not dynamic
 */
var dynamicTypeBytes = function (type, value) {
    // TODO: decide what to do with array of strings
    if (isArrayType(type) || type === 'string')    // only string itself that is dynamic; stringX is static length.
        return f.formatInputInt(value.length);
    return "";
};

var inputTypes = types.inputTypes();

/**
 * Formats input params to bytes
 *
 * @method formatInput
 * @param {Array} abi inputs of method
 * @param {Array} params that will be formatted to bytes
 * @returns bytes representation of input params
 */
var formatInput = function (inputs, params) {
    var bytes = "";
    var toAppendConstant = "";
    var toAppendArrayContent = "";

    /// first we iterate in search for dynamic
    inputs.forEach(function (input, index) {
        bytes += dynamicTypeBytes(input.type, params[index]);
    });

    inputs.forEach(function (input, i) {
        /*jshint maxcomplexity:5 */
        var typeMatch = false;
        for (var j = 0; j < inputTypes.length && !typeMatch; j++) {
            typeMatch = inputTypes[j].type(inputs[i].type, params[i]);
        }
        if (!typeMatch) {
            throwTypeError(inputs[i].type);
        }

        var formatter = inputTypes[j - 1].format;

        if (isArrayType(inputs[i].type))
            toAppendArrayContent += params[i].reduce(function (acc, curr) {
                return acc + formatter(curr);
            }, "");
        else if (inputs[i].type === 'string')
            toAppendArrayContent += formatter(params[i]);
        else
            toAppendConstant += formatter(params[i]);
    });

    bytes += toAppendConstant + toAppendArrayContent;

    return bytes;
};

/**
 * This method should be called to predict the length of dynamic type
 *
 * @method dynamicBytesLength
 * @param {String} type
 * @returns {Number} length of dynamic type, 0 or multiplication of ETH_PADDING (32)
 */
var dynamicBytesLength = function (type) {
    if (isArrayType(type) || type === 'string')   // only string itself that is dynamic; stringX is static length.
        return c.ETH_PADDING * 2;
    return 0;
};

var outputTypes = types.outputTypes();

/** 
 * Formats output bytes back to param list
 *
 * @method formatOutput
 * @param {Array} abi outputs of method
 * @param {String} bytes represention of output
 * @returns {Array} output params
 */
var formatOutput = function (outs, output) {

    output = output.slice(2);
    var result = [];
    var padding = c.ETH_PADDING * 2;

    var dynamicPartLength = outs.reduce(function (acc, curr) {
        return acc + dynamicBytesLength(curr.type);
    }, 0);

    var dynamicPart = output.slice(0, dynamicPartLength);
    output = output.slice(dynamicPartLength);

    outs.forEach(function (out, i) {
        /*jshint maxcomplexity:6 */
        var typeMatch = false;
        for (var j = 0; j < outputTypes.length && !typeMatch; j++) {
            typeMatch = outputTypes[j].type(outs[i].type);
        }

        if (!typeMatch) {
            throwTypeError(outs[i].type);
        }

        var formatter = outputTypes[j - 1].format;
        if (isArrayType(outs[i].type)) {
            var size = f.formatOutputUInt(dynamicPart.slice(0, padding));
            dynamicPart = dynamicPart.slice(padding);
            var array = [];
            for (var k = 0; k < size; k++) {
                array.push(formatter(output.slice(0, padding)));
                output = output.slice(padding);
            }
            result.push(array);
        }
        else if (types.prefixedType('string')(outs[i].type)) {
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

/**
 * Should be called to create input parser for contract with given abi
 *
 * @method inputParser
 * @param {Array} contract abi
 * @returns {Object} input parser object for given json abi
 * TODO: refactor creating the parser, do not double logic from contract
 */
var inputParser = function (json) {
    var parser = {};
    json.forEach(function (method) {
        var displayName = utils.extractDisplayName(method.name);
        var typeName = utils.extractTypeName(method.name);

        var impl = function () {
            var params = Array.prototype.slice.call(arguments);
            return formatInput(method.inputs, params);
        };

        if (parser[displayName] === undefined) {
            parser[displayName] = impl;
        }

        parser[displayName][typeName] = impl;
    });

    return parser;
};

/**
 * Should be called to create output parser for contract with given abi
 *
 * @method outputParser
 * @param {Array} contract abi
 * @returns {Object} output parser for given json abi
 */
var outputParser = function (json) {
    var parser = {};
    json.forEach(function (method) {

        var displayName = utils.extractDisplayName(method.name);
        var typeName = utils.extractTypeName(method.name);

        var impl = function (output) {
            return formatOutput(method.outputs, output);
        };

        if (parser[displayName] === undefined) {
            parser[displayName] = impl;
        }

        parser[displayName][typeName] = impl;
    });

    return parser;
};

module.exports = {
    inputParser: inputParser,
    outputParser: outputParser,
    formatInput: formatInput,
    formatOutput: formatOutput
};

},{"../utils/config":4,"../utils/utils":5,"./formatters":2,"./types":3}],2:[function(require,module,exports){
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
/** @file formatters.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2015
 */

if ("build" !== 'build') {/*
    var BigNumber = require('bignumber.js'); // jshint ignore:line
*/}

var utils = require('../utils/utils');
var c = require('../utils/config');

/**
 * Should be called to pad string to expected length
 *
 * @method padLeft
 * @param {String} string to be padded
 * @param {Number} characters that result string should have
 * @param {String} sign, by default 0
 * @returns {String} right aligned string
 */
var padLeft = function (string, chars, sign) {
    return new Array(chars - string.length + 1).join(sign ? sign : "0") + string;
};

/**
 * Formats input value to byte representation of int
 * If value is negative, return it's two's complement
 * If the value is floating point, round it down
 *
 * @method formatInputInt
 * @param {String|Number|BigNumber} value that needs to be formatted
 * @returns {String} right-aligned byte representation of int
 */
var formatInputInt = function (value) {
    var padding = c.ETH_PADDING * 2;
    BigNumber.config(c.ETH_BIGNUMBER_ROUNDING_MODE);
    return padLeft(utils.toTwosComplement(value).round().toString(16), padding);
};

/**
 * Formats input value to byte representation of string
 *
 * @method formatInputString
 * @param {String}
 * @returns {String} left-algined byte representation of string
 */
var formatInputString = function (value) {
    return utils.fromAscii(value, c.ETH_PADDING).substr(2);
};

/**
 * Formats input value to byte representation of bool
 *
 * @method formatInputBool
 * @param {Boolean}
 * @returns {String} right-aligned byte representation bool
 */
var formatInputBool = function (value) {
    return '000000000000000000000000000000000000000000000000000000000000000' + (value ?  '1' : '0');
};

/**
 * Formats input value to byte representation of real
 * Values are multiplied by 2^m and encoded as integers
 *
 * @method formatInputReal
 * @param {String|Number|BigNumber}
 * @returns {String} byte representation of real
 */
var formatInputReal = function (value) {
    return formatInputInt(new BigNumber(value).times(new BigNumber(2).pow(128))); 
};

/**
 * Check if input value is negative
 *
 * @method signedIsNegative
 * @param {String} value is hex format
 * @returns {Boolean} true if it is negative, otherwise false
 */
var signedIsNegative = function (value) {
    return (new BigNumber(value.substr(0, 1), 16).toString(2).substr(0, 1)) === '1';
};

/**
 * Formats right-aligned output bytes to int
 *
 * @method formatOutputInt
 * @param {String} bytes
 * @returns {BigNumber} right-aligned output bytes formatted to big number
 */
var formatOutputInt = function (value) {

    value = value || "0";

    // check if it's negative number
    // it it is, return two's complement
    if (signedIsNegative(value)) {
        return new BigNumber(value, 16).minus(new BigNumber('ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff', 16)).minus(1);
    }
    return new BigNumber(value, 16);
};

/**
 * Formats right-aligned output bytes to uint
 *
 * @method formatOutputUInt
 * @param {String} bytes
 * @returns {BigNumeber} right-aligned output bytes formatted to uint
 */
var formatOutputUInt = function (value) {
    value = value || "0";
    return new BigNumber(value, 16);
};

/**
 * Formats right-aligned output bytes to real
 *
 * @method formatOutputReal
 * @param {String}
 * @returns {BigNumber} input bytes formatted to real
 */
var formatOutputReal = function (value) {
    return formatOutputInt(value).dividedBy(new BigNumber(2).pow(128)); 
};

/**
 * Formats right-aligned output bytes to ureal
 *
 * @method formatOutputUReal
 * @param {String}
 * @returns {BigNumber} input bytes formatted to ureal
 */
var formatOutputUReal = function (value) {
    return formatOutputUInt(value).dividedBy(new BigNumber(2).pow(128)); 
};

/**
 * Should be used to format output hash
 *
 * @method formatOutputHash
 * @param {String}
 * @returns {String} right-aligned output bytes formatted to hex
 */
var formatOutputHash = function (value) {
    return "0x" + value;
};

/**
 * Should be used to format output bool
 *
 * @method formatOutputBool
 * @param {String}
 * @returns {Boolean} right-aligned input bytes formatted to bool
 */
var formatOutputBool = function (value) {
    return value === '0000000000000000000000000000000000000000000000000000000000000001' ? true : false;
};

/**
 * Should be used to format output string
 *
 * @method formatOutputString
 * @param {Sttring} left-aligned hex representation of string
 * @returns {String} ascii string
 */
var formatOutputString = function (value) {
    return utils.toAscii(value);
};

/**
 * Should be used to format output address
 *
 * @method formatOutputAddress
 * @param {String} right-aligned input bytes
 * @returns {String} address
 */
var formatOutputAddress = function (value) {
    return "0x" + value.slice(value.length - 40, value.length);
};

module.exports = {
    formatInputInt: formatInputInt,
    formatInputString: formatInputString,
    formatInputBool: formatInputBool,
    formatInputReal: formatInputReal,
    formatOutputInt: formatOutputInt,
    formatOutputUInt: formatOutputUInt,
    formatOutputReal: formatOutputReal,
    formatOutputUReal: formatOutputUReal,
    formatOutputHash: formatOutputHash,
    formatOutputBool: formatOutputBool,
    formatOutputString: formatOutputString,
    formatOutputAddress: formatOutputAddress
};


},{"../utils/config":4,"../utils/utils":5}],3:[function(require,module,exports){
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
/** @file types.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2015
 */

var f = require('./formatters');

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
var inputTypes = function () {
    
    return [
        { type: prefixedType('uint'), format: f.formatInputInt },
        { type: prefixedType('int'), format: f.formatInputInt },
        { type: prefixedType('hash'), format: f.formatInputInt },
        { type: prefixedType('string'), format: f.formatInputString }, 
        { type: prefixedType('real'), format: f.formatInputReal },
        { type: prefixedType('ureal'), format: f.formatInputReal },
        { type: namedType('address'), format: f.formatInputInt },
        { type: namedType('bool'), format: f.formatInputBool }
    ];
};

/// Setups output formaters for solidity types
/// @returns an array of output formatters
var outputTypes = function () {

    return [
        { type: prefixedType('uint'), format: f.formatOutputUInt },
        { type: prefixedType('int'), format: f.formatOutputInt },
        { type: prefixedType('hash'), format: f.formatOutputHash },
        { type: prefixedType('string'), format: f.formatOutputString },
        { type: prefixedType('real'), format: f.formatOutputReal },
        { type: prefixedType('ureal'), format: f.formatOutputUReal },
        { type: namedType('address'), format: f.formatOutputAddress },
        { type: namedType('bool'), format: f.formatOutputBool }
    ];
};

module.exports = {
    prefixedType: prefixedType,
    namedType: namedType,
    inputTypes: inputTypes,
    outputTypes: outputTypes
};


},{"./formatters":2}],4:[function(require,module,exports){
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
/** @file config.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2015
 */

/**
 * Utils
 * 
 * @module utils
 */

/**
 * Utility functions
 * 
 * @class [utils] config
 * @constructor
 */

/// required to define ETH_BIGNUMBER_ROUNDING_MODE
if ("build" !== 'build') {/*
    var BigNumber = require('bignumber.js'); // jshint ignore:line
*/}

var ETH_UNITS = [ 
    'wei', 
    'Kwei', 
    'Mwei', 
    'Gwei', 
    'szabo', 
    'finney', 
    'ether', 
    'grand', 
    'Mether', 
    'Gether', 
    'Tether', 
    'Pether', 
    'Eether', 
    'Zether', 
    'Yether', 
    'Nether', 
    'Dether', 
    'Vether', 
    'Uether' 
];

module.exports = {
    ETH_PADDING: 32,
    ETH_SIGNATURE_LENGTH: 4,
    ETH_UNITS: ETH_UNITS,
    ETH_BIGNUMBER_ROUNDING_MODE: { ROUNDING_MODE: BigNumber.ROUND_DOWN },
    ETH_POLLING_TIMEOUT: 1000,
    ETH_DEFAULTBLOCK: 'latest'
};


},{}],5:[function(require,module,exports){
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
/** @file utils.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2015
 */

/**
 * Utils
 * 
 * @module utils
 */

/**
 * Utility functions
 * 
 * @class [utils] utils
 * @constructor
 */

if ("build" !== 'build') {/*
    var BigNumber = require('bignumber.js'); // jshint ignore:line
*/}

var unitMap = {
    'wei':      '1',
    'kwei':     '1000',
    'ada':      '1000',
    'mwei':     '1000000',
    'babbage':  '1000000',
    'gwei':     '1000000000',
    'shannon':  '1000000000',
    'szabo':    '1000000000000',
    'finney':   '1000000000000000',
    'ether':    '1000000000000000000',
    'kether':   '1000000000000000000000',
    'grand':    '1000000000000000000000',
    'einstein': '1000000000000000000000',
    'mether':   '1000000000000000000000000',
    'gether':   '1000000000000000000000000000',
    'tether':   '1000000000000000000000000000000'
};


/** Finds first index of array element matching pattern
 *
 * @method findIndex
 * @param {Array}
 * @param {Function} pattern
 * @returns {Number} index of element
 */
var findIndex = function (array, callback) {
    var end = false;
    var i = 0;
    for (; i < array.length && !end; i++) {
        end = callback(array[i]);
    }
    return end ? i - 1 : -1;
};

/** 
 * Should be called to get sting from it's hex representation
 *
 * @method toAscii
 * @param {String} string in hex
 * @returns {String} ascii string representation of hex value
 */
var toAscii = function(hex) {
// Find termination
    var str = "";
    var i = 0, l = hex.length;
    if (hex.substring(0, 2) === '0x') {
        i = 2;
    }
    for (; i < l; i+=2) {
        var code = parseInt(hex.substr(i, 2), 16);
        if (code === 0) {
            break;
        }

        str += String.fromCharCode(code);
    }

    return str;
};
    
/**
 * Shold be called to get hex representation (prefixed by 0x) of ascii string 
 *
 * @method fromAscii
 * @param {String} string
 * @returns {String} hex representation of input string
 */
var toHexNative = function(str) {
    var hex = "";
    for(var i = 0; i < str.length; i++) {
        var n = str.charCodeAt(i).toString(16);
        hex += n.length < 2 ? '0' + n : n;
    }

    return hex;
};

/**
 * Shold be called to get hex representation (prefixed by 0x) of ascii string 
 *
 * @method fromAscii
 * @param {String} string
 * @param {Number} optional padding
 * @returns {String} hex representation of input string
 */
var fromAscii = function(str, pad) {
    pad = pad === undefined ? 0 : pad;
    var hex = toHexNative(str);
    while (hex.length < pad*2)
        hex += "00";
    return "0x" + hex;
};

/**
 * Should be called to get display name of contract function
 * 
 * @method extractDisplayName
 * @param {String} name of function/event
 * @returns {String} display name for function/event eg. multiply(uint256) -> multiply
 */
var extractDisplayName = function (name) {
    var length = name.indexOf('('); 
    return length !== -1 ? name.substr(0, length) : name;
};

/// @returns overloaded part of function/event name
var extractTypeName = function (name) {
    /// TODO: make it invulnerable
    var length = name.indexOf('(');
    return length !== -1 ? name.substr(length + 1, name.length - 1 - (length + 1)).replace(' ', '') : "";
};

/**
 * Filters all functions from input abi
 *
 * @method filterFunctions
 * @param {Array} abi
 * @returns {Array} abi array with filtered objects of type 'function'
 */
var filterFunctions = function (json) {
    return json.filter(function (current) {
        return current.type === 'function'; 
    }); 
};

/**
 * Filters all events from input abi
 *
 * @method filterEvents
 * @param {Array} abi
 * @returns {Array} abi array with filtered objects of type 'event'
 */
var filterEvents = function (json) {
    return json.filter(function (current) {
        return current.type === 'event';
    });
};

/**
 * Converts value to it's decimal representation in string
 *
 * @method toDecimal
 * @param {String|Number|BigNumber}
 * @return {String}
 */
var toDecimal = function (value) {
    return toBigNumber(value).toNumber();
};

/**
 * Converts value to it's hex representation
 *
 * @method fromDecimal
 * @param {String|Number|BigNumber}
 * @return {String}
 */
var fromDecimal = function (value) {
    var number = toBigNumber(value);
    var result = number.toString(16);

    return (number.lessThan(0))
        ? '-0x' + result.substr(1)
        : '0x' + result;
};

/**
 * Auto converts any given value into it's hex representation.
 *
 * And even stringifys objects before.
 *
 * @method toHex
 * @param {String|Number|BigNumber|Object}
 * @return {String}
 */
var toHex = function (val) {
    /*jshint maxcomplexity:5 */

    if(typeof val === 'boolean')
        return val;

    if(isBigNumber(val))
        return fromDecimal(val);

    if(typeof val === 'object')
        return fromAscii(JSON.stringify(val));

    if(isString(val) && val.indexOf('0x') === 0)
        return val;
    // if its a negative number, pass it through fromDecimal
    if(isString(val) && val.indexOf('-0x') === 0)
        return fromDecimal(val);

    if(isString(val) && !isFinite(val))
        return fromAscii(val);

    if(isFinite(val))
        return fromDecimal(val);

    return val;
};

/**
 * Returns value of unit in Wei
 *
 * @method getValueOfUnit
 * @param {String} unit the unit to convert to, default ether
 * @returns {BigNumber} value of the unit (in Wei)
 * @throws error if the unit is not correct:w
 */
var getValueOfUnit = function (unit) {
    unit = unit ? unit.toLowerCase() : 'ether';
    var unitValue = unitMap[unit];
    if (unitValue === undefined) {
        throw new Error('This unit doesn\'t exists, please use the one of the following units' + JSON.stringify(unitMap, null, 2));
    }
    return new BigNumber(unitValue, 10);
};

/**
 * Takes a number of wei and converts it to any other ether unit.
 *
 * Possible units are:
 * - kwei/ada
 * - mwei/babbage
 * - gwei/shannon
 * - szabo
 * - finney
 * - ether
 * - kether/grand/einstein
 * - mether
 * - gether
 * - tether
 *
 * @method fromWei
 * @param {Number|String} number can be a number, number string or a HEX of a decimal
 * @param {String} unit the unit to convert to, default ether
 * @return {String|Object} When given a BigNumber object it returns one as well, otherwise a number
*/
var fromWei = function(number, unit) {
    var returnValue = toBigNumber(number).dividedBy(getValueOfUnit(unit));

    return (isBigNumber(number))
        ? returnValue : returnValue.toString(10); 
};

/**
 * Takes a number of a unit and converts it to wei.
 *
 * Possible units are:
 * - kwei/ada
 * - mwei/babbage
 * - gwei/shannon
 * - szabo
 * - finney
 * - ether
 * - kether/grand/einstein
 * - mether
 * - gether
 * - tether
 *
 * @method toWei
 * @param {Number|String|BigNumber} number can be a number, number string or a HEX of a decimal
 * @param {String} unit the unit to convert from, default ether
 * @return {String|Object} When given a BigNumber object it returns one as well, otherwise a number
*/
var toWei = function(number, unit) {
    var returnValue = toBigNumber(number).times(getValueOfUnit(unit));

    return (isBigNumber(number))
        ? returnValue : returnValue.toString(10); 
};

/**
 * Takes an input and transforms it into an bignumber
 *
 * @method toBigNumber
 * @param {Number|String|BigNumber} a number, string, HEX string or BigNumber
 * @return {BigNumber} BigNumber
*/
var toBigNumber = function(number) {
    number = number || 0;
    if (isBigNumber(number))
        return number;

    return (isString(number) && (number.indexOf('0x') === 0 || number.indexOf('-0x') === 0))
        ? new BigNumber(number.replace('0x',''), 16)
        : new BigNumber(number.toString(10), 10);
};

/**
 * Takes and input transforms it into bignumber and if it is negative value, into two's complement
 *
 * @method toTwosComplement
 * @param {Number|String|BigNumber}
 * @return {BigNumber}
 */
var toTwosComplement = function (number) {
    var bigNumber = toBigNumber(number);
    if (bigNumber.lessThan(0)) {
        return new BigNumber("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16).plus(bigNumber).plus(1);
    }
    return bigNumber;
};

/**
 * Checks if the given string has proper length
 *
 * @method isAddress
 * @param {String} address the given HEX adress
 * @return {Boolean}
*/
var isAddress = function(address) {
    if (!isString(address)) {
        return false;
    }

    return ((address.indexOf('0x') === 0 && address.length === 42) ||
            (address.indexOf('0x') === -1 && address.length === 40));
};

/**
 * Returns true if object is BigNumber, otherwise false
 *
 * @method isBigNumber
 * @param {Object}
 * @return {Boolean} 
 */
var isBigNumber = function (object) {
    return object instanceof BigNumber ||
        (object && object.constructor && object.constructor.name === 'BigNumber');
};

/**
 * Returns true if object is string, otherwise false
 * 
 * @method isString
 * @param {Object}
 * @return {Boolean}
 */
var isString = function (object) {
    return typeof object === 'string' ||
        (object && object.constructor && object.constructor.name === 'String');
};

/**
 * Returns true if object is function, otherwise false
 *
 * @method isFunction
 * @param {Object}
 * @return {Boolean}
 */
var isFunction = function (object) {
    return typeof object === 'function';
};

module.exports = {
    findIndex: findIndex,
    toHex: toHex,
    toDecimal: toDecimal,
    fromDecimal: fromDecimal,
    toAscii: toAscii,
    fromAscii: fromAscii,
    extractDisplayName: extractDisplayName,
    extractTypeName: extractTypeName,
    filterFunctions: filterFunctions,
    filterEvents: filterEvents,
    toWei: toWei,
    fromWei: fromWei,
    toBigNumber: toBigNumber,
    toTwosComplement: toTwosComplement,
    isBigNumber: isBigNumber,
    isAddress: isAddress,
    isFunction: isFunction,
    isString: isString
};


},{}],6:[function(require,module,exports){
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
/** @file web3.js
 * @authors:
 *   Jeffrey Wilcke <jeff@ethdev.com>
 *   Marek Kotewicz <marek@ethdev.com>
 *   Marian Oancea <marian@ethdev.com>
 *   Gav Wood <g@ethdev.com>
 * @date 2014
 */

var net = require('./web3/net');
var eth = require('./web3/eth');
var db = require('./web3/db');
var shh = require('./web3/shh');
var watches = require('./web3/watches');
var filter = require('./web3/filter');
var utils = require('./utils/utils');
var formatters = require('./solidity/formatters');
var requestManager = require('./web3/requestmanager');
var c = require('./utils/config');

/// @returns an array of objects describing web3 api methods
var web3Methods = function () {
    return [
    { name: 'sha3', call: 'web3_sha3' }
    ];
};

/// creates methods in a given object based on method description on input
/// setups api calls for these methods
var setupMethods = function (obj, methods) {
    methods.forEach(function (method) {
        // allow for object methods 'myObject.method'
        var objectMethods = method.name.split('.'),
            callFunction = function () {
                /*jshint maxcomplexity:8 */
                
                var callback = null,
                    args = Array.prototype.slice.call(arguments),
                    call = typeof method.call === 'function' ? method.call(args) : method.call;

                // get the callback if one is available
                if(typeof args[args.length-1] === 'function'){
                    callback = args[args.length-1];
                    Array.prototype.pop.call(args);
                }

                // add the defaultBlock if not given
                if(method.addDefaultblock) {
                    if(args.length !== method.addDefaultblock)
                        Array.prototype.push.call(args, (isFinite(c.ETH_DEFAULTBLOCK) ? utils.fromDecimal(c.ETH_DEFAULTBLOCK) : c.ETH_DEFAULTBLOCK));
                    else
                        args[args.length-1] = isFinite(args[args.length-1]) ? utils.fromDecimal(args[args.length-1]) : args[args.length-1];
                }

                // show deprecated warning
                if(method.newMethod)
                    console.warn('This method is deprecated please use web3.'+ method.newMethod +'() instead.');

                return web3.manager.send({
                    method: call,
                    params: args,
                    outputFormatter: method.outputFormatter,
                    inputFormatter: method.inputFormatter,
                    addDefaultblock: method.addDefaultblock
                }, callback);
            };

        if(objectMethods.length > 1) {
            if(!obj[objectMethods[0]])
                obj[objectMethods[0]] = {};

            obj[objectMethods[0]][objectMethods[1]] = callFunction;
        
        } else {

            obj[objectMethods[0]] = callFunction;
        }

    });
};

/// creates properties in a given object based on properties description on input
/// setups api calls for these properties
var setupProperties = function (obj, properties) {
    properties.forEach(function (property) {
        var proto = {};
        proto.get = function () {

            // show deprecated warning
            if(property.newProperty)
                console.warn('This property is deprecated please use web3.'+ property.newProperty +' instead.');


            return web3.manager.send({
                method: property.getter,
                outputFormatter: property.outputFormatter
            });
        };

        if (property.setter) {
            proto.set = function (val) {

                // show deprecated warning
                if(property.newProperty)
                    console.warn('This property is deprecated please use web3.'+ property.newProperty +' instead.');

                return web3.manager.send({
                    method: property.setter,
                    params: [val],
                    inputFormatter: property.inputFormatter
                });
            };
        }

        proto.enumerable = !property.newProperty;
        Object.defineProperty(obj, property.name, proto);

    });
};

/*jshint maxparams:4 */
var startPolling = function (method, id, callback, uninstall) {
    web3.manager.startPolling({
        method: method, 
        params: [id]
    }, id,  callback, uninstall); 
};
/*jshint maxparams:3 */

var stopPolling = function (id) {
    web3.manager.stopPolling(id);
};

var ethWatch = {
    startPolling: startPolling.bind(null, 'eth_getFilterChanges'), 
    stopPolling: stopPolling
};

var shhWatch = {
    startPolling: startPolling.bind(null, 'shh_getFilterChanges'), 
    stopPolling: stopPolling
};

/// setups web3 object, and it's in-browser executed methods
var web3 = {
    manager: requestManager(),
    providers: {},

    setProvider: function (provider) {
        web3.manager.setProvider(provider);
    },
    
    /// Should be called to reset state of web3 object
    /// Resets everything except manager
    reset: function () {
        web3.manager.reset(); 
    },

    /// @returns hex string of the input
    toHex: utils.toHex,

    /// @returns ascii string representation of hex value prefixed with 0x
    toAscii: utils.toAscii,

    /// @returns hex representation (prefixed by 0x) of ascii string
    fromAscii: utils.fromAscii,

    /// @returns decimal representaton of hex value prefixed by 0x
    toDecimal: utils.toDecimal,

    /// @returns hex representation (prefixed by 0x) of decimal value
    fromDecimal: utils.fromDecimal,

    /// @returns a BigNumber object
    toBigNumber: utils.toBigNumber,

    toWei: utils.toWei,
    fromWei: utils.fromWei,
    isAddress: utils.isAddress,

    // provide network information
    net: {
        // peerCount: 
    },


    /// eth object prototype
    eth: {
        // DEPRECATED
        contractFromAbi: function (abi) {
            console.warn('Initiating a contract like this is deprecated please use var MyContract = eth.contract(abi); new MyContract(address); instead.');

            return function(addr) {
                // Default to address of Config. TODO: rremove prior to genesis.
                addr = addr || '0xc6d9d2cd449a754c494264e1809c50e34d64562b';
                var ret = web3.eth.contract(addr, abi);
                ret.address = addr;
                return ret;
            };
        },

        /// @param filter may be a string, object or event
        /// @param eventParams is optional, this is an object with optional event eventParams params
        /// @param options is optional, this is an object with optional event options ('max'...)
        /*jshint maxparams:4 */
        filter: function (fil, eventParams, options) {

            // if its event, treat it differently
            if (fil._isEvent)
                return fil(eventParams, options);

            return filter(fil, ethWatch, formatters.outputLogFormatter);
        },
        // DEPRECATED
        watch: function (fil, eventParams, options) {
            console.warn('eth.watch() is deprecated please use eth.filter() instead.');
            return this.filter(fil, eventParams, options);
        }
        /*jshint maxparams:3 */
    },

    /// db object prototype
    db: {},

    /// shh object prototype
    shh: {
        /// @param filter may be a string, object or event
        filter: function (fil) {
            return filter(fil, shhWatch, formatters.outputPostFormatter);
        },
        // DEPRECATED
        watch: function (fil) {
            console.warn('shh.watch() is deprecated please use shh.filter() instead.');
            return this.filter(fil);
        }
    }
};


// ADD defaultblock
Object.defineProperty(web3.eth, 'defaultBlock', {
    get: function () {
        return c.ETH_DEFAULTBLOCK;
    },
    set: function (val) {
        c.ETH_DEFAULTBLOCK = val;
        return c.ETH_DEFAULTBLOCK;
    }
});


/// setups all api methods
setupMethods(web3, web3Methods());
setupMethods(web3.net, net.methods);
setupProperties(web3.net, net.properties);
setupMethods(web3.eth, eth.methods);
setupProperties(web3.eth, eth.properties);
setupMethods(web3.db, db.methods());
setupMethods(web3.shh, shh.methods());
setupMethods(ethWatch, watches.eth());
setupMethods(shhWatch, watches.shh());

module.exports = web3;


},{"./solidity/formatters":2,"./utils/config":4,"./utils/utils":5,"./web3/db":8,"./web3/eth":9,"./web3/filter":11,"./web3/net":15,"./web3/requestmanager":17,"./web3/shh":18,"./web3/watches":20}],7:[function(require,module,exports){
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

var web3 = require('../web3'); 
var abi = require('../solidity/abi');
var utils = require('../utils/utils');
var eventImpl = require('./event');
var signature = require('./signature');

var exportNatspecGlobals = function (vars) {
    // it's used byt natspec.js
    // TODO: figure out better way to solve this
    web3._currentContractAbi = vars.abi;
    web3._currentContractAddress = vars.address;
    web3._currentContractMethodName = vars.method;
    web3._currentContractMethodParams = vars.params;
};

var addFunctionRelatedPropertiesToContract = function (contract) {
    
    contract.call = function (options) {
        contract._isTransaction = false;
        contract._options = options;
        return contract;
    };


    contract.sendTransaction = function (options) {
        contract._isTransaction = true;
        contract._options = options;
        return contract;
    };
    // DEPRECATED
    contract.transact = function (options) {

        console.warn('myContract.transact() is deprecated please use myContract.sendTransaction() instead.');

        return contract.sendTransaction(options);
    };

    contract._options = {};
    ['gas', 'gasPrice', 'value', 'from'].forEach(function(p) {
        contract[p] = function (v) {
            contract._options[p] = v;
            return contract;
        };
    });

};

var addFunctionsToContract = function (contract, desc, address) {
    var inputParser = abi.inputParser(desc);
    var outputParser = abi.outputParser(desc);

    // create contract functions
    utils.filterFunctions(desc).forEach(function (method) {

        var displayName = utils.extractDisplayName(method.name);
        var typeName = utils.extractTypeName(method.name);

        var impl = function () {
            /*jshint maxcomplexity:7 */
            var params = Array.prototype.slice.call(arguments);
            var sign = signature.functionSignatureFromAscii(method.name);
            var parsed = inputParser[displayName][typeName].apply(null, params);

            var options = contract._options || {};
            options.to = address;
            options.data = sign + parsed;
            
            var isTransaction = contract._isTransaction === true || (contract._isTransaction !== false && !method.constant);
            var collapse = options.collapse !== false;
            
            // reset
            contract._options = {};
            contract._isTransaction = null;

            if (isTransaction) {
                
                exportNatspecGlobals({
                    abi: desc,
                    address: address,
                    method: method.name,
                    params: params
                });

                // transactions do not have any output, cause we do not know, when they will be processed
                web3.eth.sendTransaction(options);
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

        if (contract[displayName] === undefined) {
            contract[displayName] = impl;
        }

        contract[displayName][typeName] = impl;
    });
};

var addEventRelatedPropertiesToContract = function (contract, desc, address) {
    contract.address = address;
    contract._onWatchEventResult = function (data) {
        var matchingEvent = event.getMatchingEvent(utils.filterEvents(desc));
        var parser = eventImpl.outputParser(matchingEvent);
        return parser(data);
    };
    
    Object.defineProperty(contract, 'topic', {
        get: function() {
            return utils.filterEvents(desc).map(function (e) {
                return signature.eventSignatureFromAscii(e.name);
            });
        }
    });

};

var addEventsToContract = function (contract, desc, address) {
    // create contract events
    utils.filterEvents(desc).forEach(function (e) {

        var impl = function () {
            var params = Array.prototype.slice.call(arguments);
            var sign = signature.eventSignatureFromAscii(e.name);
            var event = eventImpl.inputParser(address, sign, e);
            var o = event.apply(null, params);
            var outputFormatter = function (data) {
                var parser = eventImpl.outputParser(e);
                return parser(data);
            };
            return web3.eth.filter(o, undefined, undefined, outputFormatter);
        };
        
        // this property should be used by eth.filter to check if object is an event
        impl._isEvent = true;

        var displayName = utils.extractDisplayName(e.name);
        var typeName = utils.extractTypeName(e.name);

        if (contract[displayName] === undefined) {
            contract[displayName] = impl;
        }

        contract[displayName][typeName] = impl;

    });
};


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
 * var MyContract = web3.eth.contract(abi); // creation of contract prototype
 *
 * var contractInstance = new MyContract('0x0123123121');
 *
 * contractInstance.myMethod('this is test string param for call'); // myMethod call (implicit, default)
 * contractInstance.call().myMethod('this is test string param for call'); // myMethod call (explicit)
 * contractInstance.sendTransaction().myMethod('this is test string param for transact'); // myMethod sendTransaction
 *
 * @param abi - abi json description of the contract, which is being created
 * @returns contract object
 */
var contract = function (abi) {

    // return prototype
    if(abi instanceof Array && arguments.length === 1) {
        return Contract.bind(null, abi);

    // deprecated: auto initiate contract
    } else {

        console.warn('Initiating a contract like this is deprecated please use var MyContract = eth.contract(abi); new MyContract(address); instead.');

        return new Contract(arguments[1], arguments[0]);
    }

};

function Contract(abi, address) {

    // workaround for invalid assumption that method.name is the full anonymous prototype of the method.
    // it's not. it's just the name. the rest of the code assumes it's actually the anonymous
    // prototype, so we make it so as a workaround.
    // TODO: we may not want to modify input params, maybe use copy instead?
    abi.forEach(function (method) {
        if (method.name.indexOf('(') === -1) {
            var displayName = method.name;
            var typeName = method.inputs.map(function(i){return i.type; }).join();
            method.name = displayName + '(' + typeName + ')';
        }
    });

    var result = {};
    addFunctionRelatedPropertiesToContract(result);
    addFunctionsToContract(result, abi, address);
    addEventRelatedPropertiesToContract(result, abi, address);
    addEventsToContract(result, abi, address);

    return result;
}

module.exports = contract;


},{"../solidity/abi":1,"../utils/utils":5,"../web3":6,"./event":10,"./signature":19}],8:[function(require,module,exports){
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
/** @file db.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2015
 */

/// @returns an array of objects describing web3.db api methods
var methods = function () {
    return [
    { name: 'put', call: 'db_put' },
    { name: 'get', call: 'db_get' },
    { name: 'putString', call: 'db_putString' },
    { name: 'getString', call: 'db_getString' }
    ];
};

module.exports = {
    methods: methods
};

},{}],9:[function(require,module,exports){
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
/** @file eth.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2015
 */

/**
 * Web3
 * 
 * @module web3
 */

/**
 * Eth methods and properties
 *
 * An example method object can look as follows:
 *
 *      {
 *      name: 'getBlock',
 *      call: blockCall,
 *      outputFormatter: formatters.outputBlockFormatter,
 *      inputFormatter: [ // can be a formatter funciton or an array of functions. Where each item in the array will be used for one parameter
 *           utils.toHex, // formats paramter 1
 *           function(param){ if(!param) return false; } // formats paramter 2
 *         ]
 *       },
 *
 * @class [web3] eth
 * @constructor
 */


var formatters = require('./formatters');
var utils = require('../utils/utils');


var blockCall = function (args) {
    return (utils.isString(args[0]) && args[0].indexOf('0x') === 0) ? "eth_getBlockByHash" : "eth_getBlockByNumber";
};

var transactionFromBlockCall = function (args) {
    return (utils.isString(args[0]) && args[0].indexOf('0x') === 0) ? 'eth_getTransactionByBlockHashAndIndex' : 'eth_getTransactionByBlockNumberAndIndex';
};

var uncleCall = function (args) {
    return (utils.isString(args[0]) && args[0].indexOf('0x') === 0) ? 'eth_getUncleByBlockHashAndIndex' : 'eth_getUncleByBlockNumberAndIndex';
};

var getBlockTransactionCountCall = function (args) {
    return (utils.isString(args[0]) && args[0].indexOf('0x') === 0) ? 'eth_getBlockTransactionCountByHash' : 'eth_getBlockTransactionCountByNumber';
};

var uncleCountCall = function (args) {
    return (utils.isString(args[0]) && args[0].indexOf('0x') === 0) ? 'eth_getUncleCountByBlockHash' : 'eth_getUncleCountByBlockNumber';
};

/// @returns an array of objects describing web3.eth api methods
var methods = [
    { name: 'getBalance', call: 'eth_getBalance', addDefaultblock: 2,
        outputFormatter: formatters.convertToBigNumber},
    { name: 'getStorage', call: 'eth_getStorage', addDefaultblock: 2},
    { name: 'getStorageAt', call: 'eth_getStorageAt', addDefaultblock: 3,
        inputFormatter: utils.toHex},
    { name: 'getData', call: 'eth_getData', addDefaultblock: 2},
    { name: 'getBlock', call: blockCall,
        outputFormatter: formatters.outputBlockFormatter,
        inputFormatter: [utils.toHex, function(param){ return (!param) ? false : true; }]},
    { name: 'getUncle', call: uncleCall,
        outputFormatter: formatters.outputBlockFormatter,
        inputFormatter: [utils.toHex, utils.toHex, function(param){ return (!param) ? false : true; }]},
    { name: 'getCompilers', call: 'eth_getCompilers' },
    { name: 'getBlockTransactionCount', call: getBlockTransactionCountCall,
        outputFormatter: utils.toDecimal,
        inputFormatter: utils.toHex },
    { name: 'getBlockUncleCount', call: uncleCountCall,
        outputFormatter: utils.toDecimal,
        inputFormatter: utils.toHex },
    { name: 'getTransaction', call: 'eth_getTransactionByHash',
        outputFormatter: formatters.outputTransactionFormatter },
    { name: 'getTransactionFromBlock', call: transactionFromBlockCall,
        outputFormatter: formatters.outputTransactionFormatter,
        inputFormatter: utils.toHex },
    { name: 'getTransactionCount', call: 'eth_getTransactionCount', addDefaultblock: 2,
        outputFormatter: utils.toDecimal},
    { name: 'sendTransaction', call: 'eth_sendTransaction',
        inputFormatter: formatters.inputTransactionFormatter },
    { name: 'call', call: 'eth_call', addDefaultblock: 2,
        inputFormatter: formatters.inputCallFormatter },
    { name: 'compile.solidity', call: 'eth_compileSolidity', inputFormatter: utils.toHex },
    { name: 'compile.lll', call: 'eth_compileLLL', inputFormatter: utils.toHex },
    { name: 'compile.serpent', call: 'eth_compileSerpent', inputFormatter: utils.toHex },
    { name: 'flush', call: 'eth_flush' },

    // deprecated methods
    { name: 'balanceAt', call: 'eth_balanceAt', newMethod: 'eth.getBalance' },
    { name: 'stateAt', call: 'eth_stateAt', newMethod: 'eth.getStorageAt' },
    { name: 'storageAt', call: 'eth_storageAt', newMethod: 'eth.getStorage' },
    { name: 'countAt', call: 'eth_countAt', newMethod: 'eth.getTransactionCount' },
    { name: 'codeAt', call: 'eth_codeAt', newMethod: 'eth.getData' },
    { name: 'transact', call: 'eth_transact', newMethod: 'eth.sendTransaction' },
    { name: 'block', call: blockCall, newMethod: 'eth.getBlock' },
    { name: 'transaction', call: transactionFromBlockCall, newMethod: 'eth.getTransaction' },
    { name: 'uncle', call: uncleCall, newMethod: 'eth.getUncle' },
    { name: 'compilers', call: 'eth_compilers', newMethod: 'eth.getCompilers' },
    { name: 'solidity', call: 'eth_solidity', newMethod: 'eth.compile.solidity' },
    { name: 'lll', call: 'eth_lll', newMethod: 'eth.compile.lll' },
    { name: 'serpent', call: 'eth_serpent', newMethod: 'eth.compile.serpent' },
    { name: 'transactionCount', call: getBlockTransactionCountCall, newMethod: 'eth.getBlockTransactionCount' },
    { name: 'uncleCount', call: uncleCountCall, newMethod: 'eth.getBlockUncleCount' },
    { name: 'logs', call: 'eth_logs' }
];

/// @returns an array of objects describing web3.eth api properties
var properties = [
    { name: 'coinbase', getter: 'eth_coinbase'},
    { name: 'mining', getter: 'eth_mining'},
    { name: 'gasPrice', getter: 'eth_gasPrice', outputFormatter: formatters.convertToBigNumber},
    { name: 'accounts', getter: 'eth_accounts' },
    { name: 'blockNumber', getter: 'eth_blockNumber', outputFormatter: utils.toDecimal},

    // deprecated properties
    { name: 'listening', getter: 'net_listening', setter: 'eth_setListening', newProperty: 'net.listening'},
    { name: 'peerCount', getter: 'net_peerCount', newProperty: 'net.peerCount'},
    { name: 'number', getter: 'eth_number', newProperty: 'eth.blockNumber'}
];


module.exports = {
    methods: methods,
    properties: properties
};


},{"../utils/utils":5,"./formatters":12}],10:[function(require,module,exports){
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

var abi = require('../solidity/abi');
var utils = require('../utils/utils');
var signature = require('./signature');

/// filter inputs array && returns only indexed (or not) inputs
/// @param inputs array
/// @param bool if result should be an array of indexed params on not
/// @returns array of (not?) indexed params
var filterInputs = function (inputs, indexed) {
    return inputs.filter(function (current) {
        return current.indexed === indexed;
    });
};

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
        var inputs = [inputWithName(filterInputs(event.inputs, true), key)];

        var value = indexed[key];
        if (value instanceof Array) {
            return value.map(function (v) {
                return abi.formatInput(inputs, [v]);
            }); 
        }
        return abi.formatInput(inputs, [value]);
    });
};

var inputParser = function (address, sign, event) {
    
    // valid options are 'earliest', 'latest', 'offset' and 'max', as defined for 'eth.filter'
    return function (indexed, options) {
        var o = options || {};
        o.address = address;
        o.topic = [];
        o.topic.push(sign);
        if (indexed) {
            o.topic = o.topic.concat(indexedParamsToTopics(event, indexed));
        }
        return o;
    };
};

var getArgumentsObject = function (inputs, indexed, notIndexed) {
    var indexedCopy = indexed.slice();
    var notIndexedCopy = notIndexed.slice();
    return inputs.reduce(function (acc, current) {
        var value;
        if (current.indexed)
            value = indexedCopy.splice(0, 1)[0];
        else
            value = notIndexedCopy.splice(0, 1)[0];

        acc[current.name] = value;
        return acc;
    }, {}); 
};
 
var outputParser = function (event) {
    
    return function (output) {
        var result = {
            event: utils.extractDisplayName(event.name),
            number: output.number,
            hash: output.hash,
            args: {}
        };

        output.topics = output.topic; // fallback for go-ethereum
        if (!output.topic) {
            return result;
        }
       
        var indexedOutputs = filterInputs(event.inputs, true);
        var indexedData = "0x" + output.topic.slice(1, output.topic.length).map(function (topic) { return topic.slice(2); }).join("");
        var indexedRes = abi.formatOutput(indexedOutputs, indexedData);

        var notIndexedOutputs = filterInputs(event.inputs, false);
        var notIndexedRes = abi.formatOutput(notIndexedOutputs, output.data);

        result.args = getArgumentsObject(event.inputs, indexedRes, notIndexedRes);

        return result;
    };
};

var getMatchingEvent = function (events, payload) {
    for (var i = 0; i < events.length; i++) {
        var sign = signature.eventSignatureFromAscii(events[i].name); 
        if (sign === payload.topic[0]) {
            return events[i];
        }
    }
    return undefined;
};


module.exports = {
    inputParser: inputParser,
    outputParser: outputParser,
    getMatchingEvent: getMatchingEvent
};


},{"../solidity/abi":1,"../utils/utils":5,"./signature":19}],11:[function(require,module,exports){
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
/** @file filter.js
 * @authors:
 *   Jeffrey Wilcke <jeff@ethdev.com>
 *   Marek Kotewicz <marek@ethdev.com>
 *   Marian Oancea <marian@ethdev.com>
 *   Gav Wood <g@ethdev.com>
 * @date 2014
 */

var utils = require('../utils/utils');

/// Should be called to check if filter implementation is valid
/// @returns true if it is, otherwise false
var implementationIsValid = function (i) {
    return !!i && 
        typeof i.newFilter === 'function' && 
        typeof i.getLogs === 'function' && 
        typeof i.uninstallFilter === 'function' &&
        typeof i.startPolling === 'function' &&
        typeof i.stopPolling === 'function';
};

/// This method should be called on options object, to verify deprecated properties && lazy load dynamic ones
/// @param should be string or object
/// @returns options string or object
var getOptions = function (options) {
    /*jshint maxcomplexity:5 */

    if (typeof options === 'string') {
        return options;
    } 

    options = options || {};

    if (options.topic) {
        console.warn('"topic" is deprecated, is "topics" instead');
        options.topics = options.topic;
    }

    if (options.earliest) {
        console.warn('"earliest" is deprecated, is "fromBlock" instead');
        options.fromBlock = options.earliest;
    }

    if (options.latest) {
        console.warn('"latest" is deprecated, is "toBlock" instead');
        options.toBlock = options.latest;
    }

    if (options.skip) {
        console.warn('"skip" is deprecated, is "offset" instead');
        options.offset = options.skip;
    }

    if (options.max) {
        console.warn('"max" is deprecated, is "limit" instead');
        options.limit = options.max;
    }

    // make sure topics, get converted to hex
    if(options.topics instanceof Array) {
        options.topics = options.topics.map(function(topic){
            return utils.toHex(topic);
        });
    }


    // evaluate lazy properties
    return {
        fromBlock: utils.toHex(options.fromBlock),
        toBlock: utils.toHex(options.toBlock),
        limit: utils.toHex(options.limit),
        offset: utils.toHex(options.offset),
        to: options.to,
        address: options.address,
        topics: options.topics
    };
};

/// Should be used when we want to watch something
/// it's using inner polling mechanism and is notified about changes
/// @param options are filter options
/// @param implementation, an abstract polling implementation
/// @param formatter (optional), callback function which formats output before 'real' callback 
var filter = function(options, implementation, formatter) {
    if (!implementationIsValid(implementation)) {
        console.error('filter implemenation is invalid');
        return;
    }

    options = getOptions(options);
    var callbacks = [];
    var filterId = implementation.newFilter(options);

    // call the callbacks
    var onMessages = function (messages) {
        messages.forEach(function (message) {
            message = formatter ? formatter(message) : message;
            callbacks.forEach(function (callback) {
                callback(message);
            });
        });
    };

    implementation.startPolling(filterId, onMessages, implementation.uninstallFilter);

    var watch = function(callback) {
        callbacks.push(callback);
    };

    var stopWatching = function() {
        implementation.stopPolling(filterId);
        implementation.uninstallFilter(filterId);
        callbacks = [];
    };

    var get = function () {
        return implementation.getLogs(filterId);
    };
    
    return {
        watch: watch,
        stopWatching: stopWatching,
        get: get,

        // DEPRECATED methods
        changed:  function(){
            console.warn('watch().changed() is deprecated please use filter().watch() instead.');
            return watch.apply(this, arguments);
        },
        arrived:  function(){
            console.warn('watch().arrived() is deprecated please use filter().watch() instead.');
            return watch.apply(this, arguments);
        },
        happened:  function(){
            console.warn('watch().happened() is deprecated please use filter().watch() instead.');
            return watch.apply(this, arguments);
        },
        uninstall: function(){
            console.warn('watch().uninstall() is deprecated please use filter().stopWatching() instead.');
            return stopWatching.apply(this, arguments);
        },
        messages: function(){
            console.warn('watch().messages() is deprecated please use filter().get() instead.');
            return get.apply(this, arguments);
        },
        logs: function(){
            console.warn('watch().logs() is deprecated please use filter().get() instead.');
            return get.apply(this, arguments);
        }
    };
};

module.exports = filter;


},{"../utils/utils":5}],12:[function(require,module,exports){
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
/** @file formatters.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 *   Fabian Vogelsteller <fabian@ethdev.com>
 * @date 2015
 */

var utils = require('../utils/utils');

/**
 * Should the input to a big number
 *
 * @method convertToBigNumber
 * @param {String|Number|BigNumber}
 * @returns {BigNumber} object
 */
var convertToBigNumber = function (value) {
    return utils.toBigNumber(value);
};

/**
 * Formats the input of a transaction and converts all values to HEX
 *
 * @method inputTransactionFormatter
 * @param {Object} transaction options
 * @returns object
*/
var inputTransactionFormatter = function (options){

    // make code -> data
    if (options.code) {
        options.data = options.code;
        delete options.code;
    }

    ['gasPrice', 'gas', 'value'].forEach(function(key){
        options[key] = utils.fromDecimal(options[key]);
    });

    return options;
};

/**
 * Formats the output of a transaction to its proper values
 * 
 * @method outputTransactionFormatter
 * @param {Object} transaction
 * @returns {Object} transaction
*/
var outputTransactionFormatter = function (tx){
    tx.gas = utils.toDecimal(tx.gas);
    tx.gasPrice = utils.toBigNumber(tx.gasPrice);
    tx.value = utils.toBigNumber(tx.value);
    return tx;
};

/**
 * Formats the input of a call and converts all values to HEX
 *
 * @method inputCallFormatter
 * @param {Object} transaction options
 * @returns object
*/
var inputCallFormatter = function (options){

    // make code -> data
    if (options.code) {
        options.data = options.code;
        delete options.code;
    }

    return options;
};


/**
 * Formats the output of a block to its proper values
 *
 * @method outputBlockFormatter
 * @param {Object} block object 
 * @returns {Object} block object
*/
var outputBlockFormatter = function(block){

    // transform to number
    block.gasLimit = utils.toDecimal(block.gasLimit);
    block.gasUsed = utils.toDecimal(block.gasUsed);
    block.size = utils.toDecimal(block.size);
    block.timestamp = utils.toDecimal(block.timestamp);
    block.number = utils.toDecimal(block.number);

    block.minGasPrice = utils.toBigNumber(block.minGasPrice);
    block.difficulty = utils.toBigNumber(block.difficulty);
    block.totalDifficulty = utils.toBigNumber(block.totalDifficulty);

    if(block.transactions instanceof Array) {
        block.transactions.forEach(function(item){
            if(!utils.isString(item))
                return outputTransactionFormatter(item);
        });
    }

    return block;
};

/**
 * Formats the output of a log
 * 
 * @method outputLogFormatter
 * @param {Object} log object
 * @returns {Object} log
*/
var outputLogFormatter = function(log){
    log.number = utils.toDecimal(log.number);
    return log;
};


/**
 * Formats the input of a whisper post and converts all values to HEX
 *
 * @method inputPostFormatter
 * @param {Object} transaction object
 * @returns {Object}
*/
var inputPostFormatter = function(post){

    post.payload = utils.toHex(post.payload);
    post.ttl = utils.fromDecimal(post.ttl);
    post.workToProve = utils.fromDecimal(post.workToProve);

    if(!(post.topic instanceof Array))
        post.topic = [post.topic];


    // format the following options
    post.topic = post.topic.map(function(topic){
        return utils.fromAscii(topic);
    });

    return post;
};

/**
 * Formats the output of a received post message
 *
 * @method outputPostFormatter
 * @param {Object}
 * @returns {Object}
 */
var outputPostFormatter = function(post){

    post.expiry = utils.toDecimal(post.expiry);
    post.sent = utils.toDecimal(post.sent);
    post.ttl = utils.toDecimal(post.ttl);
    post.payloadRaw = post.payload;
    post.payload = utils.toAscii(post.payload);

    if(post.payload.indexOf('{') === 0 || post.payload.indexOf('[') === 0) {
        try {
            post.payload = JSON.parse(post.payload);
        } catch (e) { }
    }

    // format the following options
    post.topic = post.topic.map(function(topic){
        return utils.toAscii(topic);
    });

    return post;
};

module.exports = {
    convertToBigNumber: convertToBigNumber,
    inputTransactionFormatter: inputTransactionFormatter,
    outputTransactionFormatter: outputTransactionFormatter,
    inputCallFormatter: inputCallFormatter,
    outputBlockFormatter: outputBlockFormatter,
    outputLogFormatter: outputLogFormatter,
    inputPostFormatter: inputPostFormatter,
    outputPostFormatter: outputPostFormatter
};


},{"../utils/utils":5}],13:[function(require,module,exports){
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
/** @file httpprovider.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 *   Marian Oancea <marian@ethdev.com>
 * @date 2014
 */

if ("build" !== 'build') {/*
        var XMLHttpRequest = require('xmlhttprequest').XMLHttpRequest; // jshint ignore:line
*/}

var HttpProvider = function (host) {
    this.name  = 'HTTP';
    this.handlers = [];
    this.host = host || 'http://localhost:8080';
};

HttpProvider.prototype.send = function (payload, callback) {
    var request = new XMLHttpRequest();
    request.open('POST', this.host, false);

    // ASYNC
    if(typeof callback === 'function') {
        request.onreadystatechange = function() {
            if(request.readyState === 4) {
                var result = '';
                try {
                    result = JSON.parse(request.responseText)
                } catch(error) {
                    result = error;
                }
                callback(result, request.status);
            }
        };

        request.open('POST', this.host, true);
        request.send(JSON.stringify(payload));

    // SYNC
    } else {
        request.open('POST', this.host, false);
        request.send(JSON.stringify(payload));

        // check request.status
        if(request.status !== 200)
            return;
        return JSON.parse(request.responseText);
        
    }
};

module.exports = HttpProvider;


},{}],14:[function(require,module,exports){
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
/** @file jsonrpc.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2015
 */

var messageId = 1;

/// Should be called to valid json create payload object
/// @param method of jsonrpc call, required
/// @param params, an array of method params, optional
/// @returns valid jsonrpc payload object
var toPayload = function (method, params) {
    if (!method)
        console.error('jsonrpc method should be specified!');

    return {
        jsonrpc: '2.0',
        method: method,
        params: params || [],
        id: messageId++
    }; 
};

/// Should be called to check if jsonrpc response is valid
/// @returns true if response is valid, otherwise false 
var isValidResponse = function (response) {
    return !!response &&
        !response.error &&
        response.jsonrpc === '2.0' &&
        typeof response.id === 'number' &&
        response.result !== undefined; // only undefined is not valid json object
};

/// Should be called to create batch payload object
/// @param messages, an array of objects with method (required) and params (optional) fields
var toBatchPayload = function (messages) {
    return messages.map(function (message) {
        return toPayload(message.method, message.params);
    }); 
};

module.exports = {
    toPayload: toPayload,
    isValidResponse: isValidResponse,
    toBatchPayload: toBatchPayload
};



},{}],15:[function(require,module,exports){
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
/** @file eth.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2015
 */

var utils = require('../utils/utils');

/// @returns an array of objects describing web3.eth api methods
var methods = [
    // { name: 'getBalance', call: 'eth_balanceAt', outputFormatter: formatters.convertToBigNumber},
];

/// @returns an array of objects describing web3.eth api properties
var properties = [
    { name: 'listening', getter: 'net_listening'},
    { name: 'peerCount', getter: 'net_peerCount', outputFormatter: utils.toDecimal },
];


module.exports = {
    methods: methods,
    properties: properties
};


},{"../utils/utils":5}],16:[function(require,module,exports){
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
/** @file qtsync.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 *   Marian Oancea <marian@ethdev.com>
 * @date 2014
 */

var QtSyncProvider = function () {
};

QtSyncProvider.prototype.send = function (payload) {
    var result = navigator.qt.callMethod(JSON.stringify(payload));
    return JSON.parse(result);
};

module.exports = QtSyncProvider;


},{}],17:[function(require,module,exports){
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
/** @file requestmanager.js
 * @authors:
 *   Jeffrey Wilcke <jeff@ethdev.com>
 *   Marek Kotewicz <marek@ethdev.com>
 *   Marian Oancea <marian@ethdev.com>
 *   Gav Wood <g@ethdev.com>
 * @date 2014
 */

var jsonrpc = require('./jsonrpc');
var c = require('../utils/config');

/**
 * It's responsible for passing messages to providers
 * It's also responsible for polling the ethereum node for incoming messages
 * Default poll timeout is 1 second
 */
var requestManager = function() {
    var polls = [];
    var timeout = null;
    var provider;

    var send = function (data, callback) {
        /*jshint maxcomplexity: 7 */

        // FORMAT BASED ON ONE FORMATTER function
        if(typeof data.inputFormatter === 'function') {
            data.params = Array.prototype.map.call(data.params, function(item, index){
                // format everything besides the defaultblock, which is already formated
                return (!data.addDefaultblock || index+1 < data.addDefaultblock)
                    ? data.inputFormatter(item)
                    : item;
            });

        // FORMAT BASED ON the input FORMATTER ARRAY
        } else if(data.inputFormatter instanceof Array) {
            data.params = Array.prototype.map.call(data.inputFormatter, function(formatter, index){
                // format everything besides the defaultblock, which is already formated
                return (!data.addDefaultblock || index+1 < data.addDefaultblock)
                    ? formatter(data.params[index])
                    : data.params[index];
            });
        }


        var payload = jsonrpc.toPayload(data.method, data.params);
        
        if (!provider) {
            console.error('provider is not set');
            return null;
        }

        // HTTP ASYNC (only when callback is given, and it a HttpProvidor)
        if(typeof callback === 'function' && provider.name === 'HTTP'){
            provider.send(payload, function(result, status){

                if (!jsonrpc.isValidResponse(result)) {
                    if(typeof result === 'object' && result.error && result.error.message) {
                        console.error(result.error.message);
                        callback(result.error);
                    } else {
                        callback(new Error({
                            status: status,
                            error: result,
                            message: 'Bad Request'
                        }));
                    }
                    return null;
                }

                // format the output
                callback(null, (typeof data.outputFormatter === 'function') ? data.outputFormatter(result.result) : result.result);
            });

        // SYNC
        } else {
            var result = provider.send(payload);

            if (!jsonrpc.isValidResponse(result)) {
                console.log(result);
                if(typeof result === 'object' && result.error && result.error.message)
                    console.error(result.error.message);
                return null;
            }

            // format the output
            return (typeof data.outputFormatter === 'function') ? data.outputFormatter(result.result) : result.result;
        }
        
    };

    var setProvider = function (p) {
        provider = p;
    };

    /*jshint maxparams:4 */
    var startPolling = function (data, pollId, callback, uninstall) {
        polls.push({data: data, id: pollId, callback: callback, uninstall: uninstall});
    };
    /*jshint maxparams:3 */

    var stopPolling = function (pollId) {
        for (var i = polls.length; i--;) {
            var poll = polls[i];
            if (poll.id === pollId) {
                polls.splice(i, 1);
            }
        }
    };

    var reset = function () {
        polls.forEach(function (poll) {
            poll.uninstall(poll.id); 
        });
        polls = [];

        if (timeout) {
            clearTimeout(timeout);
            timeout = null;
        }
        poll();
    };

    var poll = function () {
        polls.forEach(function (data) {
            // send async
            send(data.data, function(result){
                if (!(result instanceof Array) || result.length === 0) {
                    return;
                }
                data.callback(result);
            });
        });
        timeout = setTimeout(poll, c.ETH_POLLING_TIMEOUT);
    };
    
    poll();

    return {
        send: send,
        setProvider: setProvider,
        startPolling: startPolling,
        stopPolling: stopPolling,
        reset: reset
    };
};

module.exports = requestManager;


},{"../utils/config":4,"./jsonrpc":14}],18:[function(require,module,exports){
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
/** @file shh.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2015
 */

var formatters = require('./formatters');

/// @returns an array of objects describing web3.shh api methods
var methods = function () {
    return [
    { name: 'post', call: 'shh_post', inputFormatter: formatters.inputPostFormatter },
    { name: 'newIdentity', call: 'shh_newIdentity' },
    { name: 'hasIdentity', call: 'shh_hasIdentity' },
    { name: 'newGroup', call: 'shh_newGroup' },
    { name: 'addToGroup', call: 'shh_addToGroup' },

    // deprecated
    { name: 'haveIdentity', call: 'shh_haveIdentity', newMethod: 'shh.hasIdentity' },
    ];
};

module.exports = {
    methods: methods
};


},{"./formatters":12}],19:[function(require,module,exports){
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
/** @file signature.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2015
 */

var web3 = require('../web3'); 
var c = require('../utils/config');

/// @param function name for which we want to get signature
/// @returns signature of function with given name
var functionSignatureFromAscii = function (name) {
    return web3.sha3(web3.fromAscii(name)).slice(0, 2 + c.ETH_SIGNATURE_LENGTH * 2);
};

/// @param event name for which we want to get signature
/// @returns signature of event with given name
var eventSignatureFromAscii = function (name) {
    return web3.sha3(web3.fromAscii(name));
};

module.exports = {
    functionSignatureFromAscii: functionSignatureFromAscii,
    eventSignatureFromAscii: eventSignatureFromAscii
};


},{"../utils/config":4,"../web3":6}],20:[function(require,module,exports){
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
/** @file watches.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2015
 */

/// @returns an array of objects describing web3.eth.filter api methods
var eth = function () {
    var newFilter = function (args) {
        return typeof args[0] === 'string' ? 'eth_newBlockFilter' : 'eth_newFilter';
    };

    return [
    { name: 'newFilter', call: newFilter },
    { name: 'uninstallFilter', call: 'eth_uninstallFilter' },
    { name: 'getLogs', call: 'eth_getFilterLogs' }
    ];
};

/// @returns an array of objects describing web3.shh.watch api methods
var shh = function () {
    return [
    { name: 'newFilter', call: 'shh_newFilter' },
    { name: 'uninstallFilter', call: 'shh_uninstallFilter' },
    { name: 'getLogs', call: 'shh_getMessages' }
    ];
};

module.exports = {
    eth: eth,
    shh: shh
};


},{}],"web3":[function(require,module,exports){
var web3 = require('./lib/web3');
web3.providers.HttpProvider = require('./lib/web3/httpprovider');
web3.providers.QtSyncProvider = require('./lib/web3/qtsync');
web3.eth.contract = require('./lib/web3/contract');
web3.abi = require('./lib/solidity/abi');

module.exports = web3;

},{"./lib/solidity/abi":1,"./lib/web3":6,"./lib/web3/contract":7,"./lib/web3/httpprovider":13,"./lib/web3/qtsync":16}]},{},["web3"])


//# sourceMappingURL=ethereum.js.map
