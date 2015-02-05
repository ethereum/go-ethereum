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

if (process.env.NODE_ENV !== 'build') {
    var BigNumber = require('bignumber.js'); // jshint ignore:line
}

var utils = require('./utils');
var c = require('./const');

/// @param string string to be padded
/// @param number of characters that result string should have
/// @param sign, by default 0
/// @returns right aligned string
var padLeft = function (string, chars, sign) {
    return new Array(chars - string.length + 1).join(sign ? sign : "0") + string;
};

/// Formats input value to byte representation of int
/// If value is negative, return it's two's complement
/// If the value is floating point, round it down
/// @returns right-aligned byte representation of int
var formatInputInt = function (value) {
    var padding = c.ETH_PADDING * 2;
    if (value instanceof BigNumber || typeof value === 'number') {
        if (typeof value === 'number')
            value = new BigNumber(value);
        BigNumber.config(c.ETH_BIGNUMBER_ROUNDING_MODE);
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
    return utils.fromAscii(value, c.ETH_PADDING).substr(2);
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


/// Check if input value is negative
/// @param value is hex format
/// @returns true if it is negative, otherwise false
var signedIsNegative = function (value) {
    return (new BigNumber(value.substr(0, 1), 16).toString(2).substr(0, 1)) === '1';
};

/// Formats input right-aligned input bytes to int
/// @returns right-aligned input bytes formatted to int
var formatOutputInt = function (value) {
    value = value || "0";
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
    value = value || "0";
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
    return utils.toAscii(value);
};

/// @returns right-aligned input bytes formatted to address
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

