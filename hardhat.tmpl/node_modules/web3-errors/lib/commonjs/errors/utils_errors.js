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
exports.InvalidTypeAbiInputError = exports.InvalidBlockError = exports.InvalidLargeValueError = exports.InvalidSizeError = exports.InvalidUnsignedIntegerError = exports.InvalidBooleanError = exports.InvalidTypeError = exports.NibbleWidthError = exports.HexProcessingError = exports.InvalidIntegerError = exports.InvalidUnitError = exports.InvalidStringError = exports.InvalidAddressError = exports.InvalidNumberError = exports.InvalidBytesError = void 0;
/* eslint-disable max-classes-per-file */
const error_codes_js_1 = require("../error_codes.js");
const web3_error_base_js_1 = require("../web3_error_base.js");
class InvalidBytesError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        super(value, 'can not parse as byte data');
        this.code = error_codes_js_1.ERR_INVALID_BYTES;
    }
}
exports.InvalidBytesError = InvalidBytesError;
class InvalidNumberError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        super(value, 'can not parse as number data');
        this.code = error_codes_js_1.ERR_INVALID_NUMBER;
    }
}
exports.InvalidNumberError = InvalidNumberError;
class InvalidAddressError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        super(value, 'invalid ethereum address');
        this.code = error_codes_js_1.ERR_INVALID_ADDRESS;
    }
}
exports.InvalidAddressError = InvalidAddressError;
class InvalidStringError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        super(value, 'not a valid string');
        this.code = error_codes_js_1.ERR_INVALID_STRING;
    }
}
exports.InvalidStringError = InvalidStringError;
class InvalidUnitError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        super(value, 'invalid unit');
        this.code = error_codes_js_1.ERR_INVALID_UNIT;
    }
}
exports.InvalidUnitError = InvalidUnitError;
class InvalidIntegerError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        super(value, 'not a valid unit. Must be a positive integer');
        this.code = error_codes_js_1.ERR_INVALID_INTEGER;
    }
}
exports.InvalidIntegerError = InvalidIntegerError;
class HexProcessingError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        super(value, 'can not be converted to hex');
        this.code = error_codes_js_1.ERR_INVALID_HEX;
    }
}
exports.HexProcessingError = HexProcessingError;
class NibbleWidthError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        super(value, 'value greater than the nibble width');
        this.code = error_codes_js_1.ERR_INVALID_NIBBLE_WIDTH;
    }
}
exports.NibbleWidthError = NibbleWidthError;
class InvalidTypeError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        super(value, 'invalid type, type not supported');
        this.code = error_codes_js_1.ERR_INVALID_TYPE;
    }
}
exports.InvalidTypeError = InvalidTypeError;
class InvalidBooleanError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        super(value, 'not a valid boolean.');
        this.code = error_codes_js_1.ERR_INVALID_BOOLEAN;
    }
}
exports.InvalidBooleanError = InvalidBooleanError;
class InvalidUnsignedIntegerError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        super(value, 'not a valid unsigned integer.');
        this.code = error_codes_js_1.ERR_INVALID_UNSIGNED_INTEGER;
    }
}
exports.InvalidUnsignedIntegerError = InvalidUnsignedIntegerError;
class InvalidSizeError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        super(value, 'invalid size given.');
        this.code = error_codes_js_1.ERR_INVALID_SIZE;
    }
}
exports.InvalidSizeError = InvalidSizeError;
class InvalidLargeValueError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        super(value, 'value is larger than size.');
        this.code = error_codes_js_1.ERR_INVALID_LARGE_VALUE;
    }
}
exports.InvalidLargeValueError = InvalidLargeValueError;
class InvalidBlockError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        super(value, 'invalid string given');
        this.code = error_codes_js_1.ERR_INVALID_BLOCK;
    }
}
exports.InvalidBlockError = InvalidBlockError;
class InvalidTypeAbiInputError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        super(value, 'components found but type is not tuple');
        this.code = error_codes_js_1.ERR_INVALID_TYPE_ABI;
    }
}
exports.InvalidTypeAbiInputError = InvalidTypeAbiInputError;
//# sourceMappingURL=utils_errors.js.map