"use strict";
/*
This file is part of web3.js.

web3.js is free software: you can redistribute it and/or modify
it under the terms of the GNU Lesser General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

web3.js is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public License
along with web3.js.  If not, see <http://www.gnu.org/licenses/>.
*/
Object.defineProperty(exports, "__esModule", { value: true });
exports.decodeArray = exports.encodeArray = exports.decodeTuple = exports.encodeTuple = exports.decodeString = exports.encodeString = exports.decodeNumber = exports.encodeNumber = exports.decodeBytes = exports.encodeBytes = exports.decodeBool = exports.encodeBoolean = exports.decodeAddress = exports.encodeAddress = void 0;
exports.encodeParamFromAbiParameter = encodeParamFromAbiParameter;
exports.decodeParamFromAbiParameter = decodeParamFromAbiParameter;
const web3_errors_1 = require("web3-errors");
const address_js_1 = require("./address.js");
const bool_js_1 = require("./bool.js");
const bytes_js_1 = require("./bytes.js");
const number_js_1 = require("./number.js");
const string_js_1 = require("./string.js");
// eslint-disable-next-line import/no-cycle
const tuple_js_1 = require("./tuple.js");
// eslint-disable-next-line import/no-cycle
const array_js_1 = require("./array.js");
var address_js_2 = require("./address.js");
Object.defineProperty(exports, "encodeAddress", { enumerable: true, get: function () { return address_js_2.encodeAddress; } });
Object.defineProperty(exports, "decodeAddress", { enumerable: true, get: function () { return address_js_2.decodeAddress; } });
var bool_js_2 = require("./bool.js");
Object.defineProperty(exports, "encodeBoolean", { enumerable: true, get: function () { return bool_js_2.encodeBoolean; } });
Object.defineProperty(exports, "decodeBool", { enumerable: true, get: function () { return bool_js_2.decodeBool; } });
var bytes_js_2 = require("./bytes.js");
Object.defineProperty(exports, "encodeBytes", { enumerable: true, get: function () { return bytes_js_2.encodeBytes; } });
Object.defineProperty(exports, "decodeBytes", { enumerable: true, get: function () { return bytes_js_2.decodeBytes; } });
var number_js_2 = require("./number.js");
Object.defineProperty(exports, "encodeNumber", { enumerable: true, get: function () { return number_js_2.encodeNumber; } });
Object.defineProperty(exports, "decodeNumber", { enumerable: true, get: function () { return number_js_2.decodeNumber; } });
var string_js_2 = require("./string.js");
Object.defineProperty(exports, "encodeString", { enumerable: true, get: function () { return string_js_2.encodeString; } });
Object.defineProperty(exports, "decodeString", { enumerable: true, get: function () { return string_js_2.decodeString; } });
// eslint-disable-next-line import/no-cycle
var tuple_js_2 = require("./tuple.js");
Object.defineProperty(exports, "encodeTuple", { enumerable: true, get: function () { return tuple_js_2.encodeTuple; } });
Object.defineProperty(exports, "decodeTuple", { enumerable: true, get: function () { return tuple_js_2.decodeTuple; } });
// eslint-disable-next-line import/no-cycle
var array_js_2 = require("./array.js");
Object.defineProperty(exports, "encodeArray", { enumerable: true, get: function () { return array_js_2.encodeArray; } });
Object.defineProperty(exports, "decodeArray", { enumerable: true, get: function () { return array_js_2.decodeArray; } });
function encodeParamFromAbiParameter(param, value) {
    if (param.type === 'string') {
        return (0, string_js_1.encodeString)(param, value);
    }
    if (param.type === 'bool') {
        return (0, bool_js_1.encodeBoolean)(param, value);
    }
    if (param.type === 'address') {
        return (0, address_js_1.encodeAddress)(param, value);
    }
    if (param.type === 'tuple') {
        return (0, tuple_js_1.encodeTuple)(param, value);
    }
    if (param.type.endsWith(']')) {
        return (0, array_js_1.encodeArray)(param, value);
    }
    if (param.type.startsWith('bytes')) {
        return (0, bytes_js_1.encodeBytes)(param, value);
    }
    if (param.type.startsWith('uint') || param.type.startsWith('int')) {
        return (0, number_js_1.encodeNumber)(param, value);
    }
    throw new web3_errors_1.AbiError('Unsupported', {
        param,
        value,
    });
}
function decodeParamFromAbiParameter(param, bytes) {
    if (param.type === 'string') {
        return (0, string_js_1.decodeString)(param, bytes);
    }
    if (param.type === 'bool') {
        return (0, bool_js_1.decodeBool)(param, bytes);
    }
    if (param.type === 'address') {
        return (0, address_js_1.decodeAddress)(param, bytes);
    }
    if (param.type === 'tuple') {
        return (0, tuple_js_1.decodeTuple)(param, bytes);
    }
    if (param.type.endsWith(']')) {
        return (0, array_js_1.decodeArray)(param, bytes);
    }
    if (param.type.startsWith('bytes')) {
        return (0, bytes_js_1.decodeBytes)(param, bytes);
    }
    if (param.type.startsWith('uint') || param.type.startsWith('int')) {
        return (0, number_js_1.decodeNumber)(param, bytes);
    }
    throw new web3_errors_1.AbiError('Unsupported', {
        param,
        bytes,
    });
}
//# sourceMappingURL=index.js.map