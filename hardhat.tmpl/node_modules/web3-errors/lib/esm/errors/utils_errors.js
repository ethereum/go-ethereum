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
/* eslint-disable max-classes-per-file */
import { ERR_INVALID_BYTES, ERR_INVALID_NUMBER, ERR_INVALID_ADDRESS, ERR_INVALID_BLOCK, ERR_INVALID_BOOLEAN, ERR_INVALID_HEX, ERR_INVALID_LARGE_VALUE, ERR_INVALID_NIBBLE_WIDTH, ERR_INVALID_SIZE, ERR_INVALID_STRING, ERR_INVALID_TYPE, ERR_INVALID_TYPE_ABI, ERR_INVALID_UNIT, ERR_INVALID_INTEGER, ERR_INVALID_UNSIGNED_INTEGER, } from '../error_codes.js';
import { InvalidValueError } from '../web3_error_base.js';
export class InvalidBytesError extends InvalidValueError {
    constructor(value) {
        super(value, 'can not parse as byte data');
        this.code = ERR_INVALID_BYTES;
    }
}
export class InvalidNumberError extends InvalidValueError {
    constructor(value) {
        super(value, 'can not parse as number data');
        this.code = ERR_INVALID_NUMBER;
    }
}
export class InvalidAddressError extends InvalidValueError {
    constructor(value) {
        super(value, 'invalid ethereum address');
        this.code = ERR_INVALID_ADDRESS;
    }
}
export class InvalidStringError extends InvalidValueError {
    constructor(value) {
        super(value, 'not a valid string');
        this.code = ERR_INVALID_STRING;
    }
}
export class InvalidUnitError extends InvalidValueError {
    constructor(value) {
        super(value, 'invalid unit');
        this.code = ERR_INVALID_UNIT;
    }
}
export class InvalidIntegerError extends InvalidValueError {
    constructor(value) {
        super(value, 'not a valid unit. Must be a positive integer');
        this.code = ERR_INVALID_INTEGER;
    }
}
export class HexProcessingError extends InvalidValueError {
    constructor(value) {
        super(value, 'can not be converted to hex');
        this.code = ERR_INVALID_HEX;
    }
}
export class NibbleWidthError extends InvalidValueError {
    constructor(value) {
        super(value, 'value greater than the nibble width');
        this.code = ERR_INVALID_NIBBLE_WIDTH;
    }
}
export class InvalidTypeError extends InvalidValueError {
    constructor(value) {
        super(value, 'invalid type, type not supported');
        this.code = ERR_INVALID_TYPE;
    }
}
export class InvalidBooleanError extends InvalidValueError {
    constructor(value) {
        super(value, 'not a valid boolean.');
        this.code = ERR_INVALID_BOOLEAN;
    }
}
export class InvalidUnsignedIntegerError extends InvalidValueError {
    constructor(value) {
        super(value, 'not a valid unsigned integer.');
        this.code = ERR_INVALID_UNSIGNED_INTEGER;
    }
}
export class InvalidSizeError extends InvalidValueError {
    constructor(value) {
        super(value, 'invalid size given.');
        this.code = ERR_INVALID_SIZE;
    }
}
export class InvalidLargeValueError extends InvalidValueError {
    constructor(value) {
        super(value, 'value is larger than size.');
        this.code = ERR_INVALID_LARGE_VALUE;
    }
}
export class InvalidBlockError extends InvalidValueError {
    constructor(value) {
        super(value, 'invalid string given');
        this.code = ERR_INVALID_BLOCK;
    }
}
export class InvalidTypeAbiInputError extends InvalidValueError {
    constructor(value) {
        super(value, 'components found but type is not tuple');
        this.code = ERR_INVALID_TYPE_ABI;
    }
}
//# sourceMappingURL=utils_errors.js.map