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

/// @returns ascii string representation of hex value prefixed with 0x
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
    
var toHex = function(str) {
    var hex = "";
    for(var i = 0; i < str.length; i++) {
        var n = str.charCodeAt(i).toString(16);
        hex += n.length < 2 ? '0' + n : n;
    }

    return hex;
};

/// @returns hex representation (prefixed by 0x) of ascii string
var fromAscii = function(str, pad) {
    pad = pad === undefined ? 0 : pad;
    var hex = toHex(str);
    while (hex.length < pad*2)
        hex += "00";
    return "0x" + hex;
};

/// @returns display name for function/event eg. multiply(uint256) -> multiply
var extractDisplayName = function (name) {
    var length = name.indexOf('('); 
    return length !== -1 ? name.substr(0, length) : name;
};

/// @returns overloaded part of function/event name
var extractTypeName = function (name) {
    /// TODO: make it invulnerable
    var length = name.indexOf('(');
    return length !== -1 ? name.substr(length + 1, name.length - 1 - (length + 1)) : "";
};

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

module.exports = {
    findIndex: findIndex,
    toAscii: toAscii,
    fromAscii: fromAscii,
    extractDisplayName: extractDisplayName,
    extractTypeName: extractTypeName,
    filterFunctions: filterFunctions,
    filterEvents: filterEvents
};

