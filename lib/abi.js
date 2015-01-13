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

var padLeft = function (string, chars) {
    return new Array(chars - string.length + 1).join("0") + string;
};

var setupInputTypes = function () {
    
    var prefixedType = function (prefix) {
        return function (type, value) {
            var expected = prefix;
            if (type.indexOf(expected) !== 0) {
                return false;
            }

            var padding = 32;   // override as per the new ABI.

            if (prefix === "string")
                return web3.fromAscii(value, padding).substr(2);
            if (typeof value === "number")
                value = value.toString(16);
            else if (typeof value === "string")
                value = web3.toHex(value); 
            else if (value.indexOf('0x') === 0)
                value = value.substr(2);
            else
                value = (+value).toString(16);
            return padLeft(value, padding * 2);
        };
    };

    var namedType = function (name, formatter) {
        return function (type, value) {
            if (type !== name) {
                return false;
            }

            var padding = 32;   //override as per the new ABI.

            return padLeft(formatter ? formatter(value) : value, padding * 2);
        };
    };

    var formatBool = function (value) {
        return value ? '0x1' : '0x0';
    };

    return [
        prefixedType('uint'),
        prefixedType('int'),
        prefixedType('hash'),
        prefixedType('string'),
        prefixedType('real'),
        prefixedType('ureal'),
        namedType('address'),
        namedType('bool', formatBool),
    ];
};

var inputTypes = setupInputTypes();

var toAbiInput = function (json, methodName, params) {
    var bytes = "";
    var index = findMethodIndex(json, methodName);

    if (index === -1) {
        return;
    }

    var method = json[index];

    for (var i = 0; i < method.inputs.length; i++) {
        var found = false;
        for (var j = 0; j < inputTypes.length && !found; j++) {
            found = inputTypes[j](method.inputs[i].type, params[i]);
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
            var padding = 32;  // override as per the new ABI.
            return padding * 2;
        };
    };

    var namedType = function (name) {
        return function (type) {
            var padding = 32;  // override as per the new ABI.
            return name === type ? padding * 2 : -1;
        };
    };

    var formatInt = function (value) {
        return value.length <= 8 ? +parseInt(value, 16) : hexToDec(value);
    };

    var formatHash = function (value) {
        return "0x" + value;
    };

    var formatBool = function (value) {
        return value === '1' ? true : false;
    };

    var formatString = function (value) {
        return web3.toAscii(value);
    };

    return [
    { padding: prefixedType('uint'), format: formatInt },
    { padding: prefixedType('int'), format: formatInt },
    { padding: prefixedType('hash'), format: formatHash },
    { padding: prefixedType('string'), format: formatString },
    { padding: prefixedType('real'), format: formatInt },
    { padding: prefixedType('ureal'), format: formatInt },
    { padding: namedType('address') },
    { padding: namedType('bool'), format: formatBool }
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
        result.push(formatter ? formatter(res) : ("0x" + res));
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

