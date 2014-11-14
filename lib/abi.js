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
 * @date 2014
 */

var findIndex = function (array, callback) {
    var end = false;
    var i = 0;
    for (; i < array.length && !end; i++) {
        end = callback(array[i]);
    }
    return end ? i - 1 : -1;
};

var findMethodIndex = function (json, methodName) {
    return findIndex(json, function (method) {
        return method.name === methodName;
    });
};

var padLeft = function (number, n) {
    return (new Array(n * 2 - number.toString().length + 1)).join("0") + number;
};

var setupInputTypes = function () {
    var prefixedType = function (prefix) {
        return function (type, value) {
            var expected = prefix;
            if (type.indexOf(expected) !== 0) {
                return false;
            }

            var padding = parseInt(type.slice(expected.length)) / 8;
            return padLeft(value, padding);
        };
    };

    var namedType = function (name, padding, formatter) {
        return function (type, value) {
            if (type !== name) {
                return false; 
            }

            return padLeft(formatter ? value : formatter(value), padding);
        };
    };

    var formatBool = function (value) {
        return value ? '1' : '0';
    };

    return [
        prefixedType('uint'),
        prefixedType('int'),
        namedType('address', 20),
        namedType('bool', 1, formatBool),
    ];
};

var inputTypes = setupInputTypes();

var toAbiInput = function (json, methodName, params) {
    var bytes = "";
    var index = findMethodIndex(json, methodName);
    
    if (index === -1) {
        return;
    }

    // it needs to be checked in WebThreeStubServer 
    // something wrong might be with this additional zero
    bytes = bytes + index + 'x' + '0';
    var method = json[index];
    
    for (var i = 0; i < method.inputs.length; i++) {
        var found = false;
        for (var j = 0; j < inputTypes.length && !found; j++) {
            var val = parseInt(params[i]).toString(16);
            found = inputTypes[j](method.inputs[i].type, val);
        }
        if (!found) {
            console.error('unsupported json type: ' + method.inputs[i].type);
        }
        bytes += found;
    }
    return bytes;
};

var setupOutputTypes = function () {
    var prefixedType = function (prefix) {
        return function (type) {
            var expected = prefix;
            if (type.indexOf(expected) !== 0) {
                return -1;
            }
            
            var padding = parseInt(type.slice(expected.length)) / 8;
            return padding * 2;
        };
    };

    var namedType = function (name, padding) {
        return function (type) {
            return name === type ? padding * 2: -1;
        };
    };

    var formatInt = function (value) {
        return parseInt(value, 16);
    };

    var formatBool = function (value) {
        return value === '1' ? true : false;
    };

    return [
    { padding: prefixedType('uint'), format: formatInt },
    { padding: prefixedType('int'), format: formatInt },
    { padding: namedType('address', 20) },
    { padding: namedType('bool', 1), format: formatBool }
    ];
};

var outputTypes = setupOutputTypes();

var fromAbiOutput = function (json, methodName, output) {
    var index = findMethodIndex(json, methodName);

    if (index === -1) {
        return;
    }
    
    output = output.slice(2);

    var result = [];
    var method = json[index];
    for (var i = 0; i < method.outputs.length; i++) {
        var padding = -1;
        for (var j = 0; j < outputTypes.length && padding === -1; j++) {
            padding = outputTypes[j].padding(method.outputs[i].type);
        }

        if (padding === -1) {
            // not found output parsing
            continue;
        }
        var res = output.slice(0, padding);
        var formatter = outputTypes[j - 1].format;
        result.push(formatter ? formatter(res): res);
        output = output.slice(padding);
    }

    return result;
};

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

var outputParser = function (json) {
    var parser = {};
    json.forEach(function (method) {
        parser[method.name] = function (output) {
            return fromAbiOutput(json, method.name, output);
        };
    });

    return parser;
};

module.exports = {
    inputParser: inputParser,
    outputParser: outputParser
};

