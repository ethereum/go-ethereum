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
exports.PBKDF2IterationsError = exports.IVLengthError = exports.InvalidPasswordError = exports.KeyStoreVersionError = exports.KeyDerivationError = exports.InvalidKdfError = exports.InvalidSignatureError = exports.InvalidPrivateKeyError = exports.PrivateKeyLengthError = void 0;
/* eslint-disable max-classes-per-file */
const error_codes_js_1 = require("../error_codes.js");
const web3_error_base_js_1 = require("../web3_error_base.js");
class PrivateKeyLengthError extends web3_error_base_js_1.BaseWeb3Error {
    constructor() {
        super(`Private key must be 32 bytes.`);
        this.code = error_codes_js_1.ERR_PRIVATE_KEY_LENGTH;
    }
}
exports.PrivateKeyLengthError = PrivateKeyLengthError;
class InvalidPrivateKeyError extends web3_error_base_js_1.BaseWeb3Error {
    constructor() {
        super(`Invalid Private Key, Not a valid string or uint8Array`);
        this.code = error_codes_js_1.ERR_INVALID_PRIVATE_KEY;
    }
}
exports.InvalidPrivateKeyError = InvalidPrivateKeyError;
class InvalidSignatureError extends web3_error_base_js_1.BaseWeb3Error {
    constructor(errorDetails) {
        super(`"${errorDetails}"`);
        this.code = error_codes_js_1.ERR_INVALID_SIGNATURE;
    }
}
exports.InvalidSignatureError = InvalidSignatureError;
class InvalidKdfError extends web3_error_base_js_1.BaseWeb3Error {
    constructor() {
        super(`Invalid key derivation function`);
        this.code = error_codes_js_1.ERR_UNSUPPORTED_KDF;
    }
}
exports.InvalidKdfError = InvalidKdfError;
class KeyDerivationError extends web3_error_base_js_1.BaseWeb3Error {
    constructor() {
        super(`Key derivation failed - possibly wrong password`);
        this.code = error_codes_js_1.ERR_KEY_DERIVATION_FAIL;
    }
}
exports.KeyDerivationError = KeyDerivationError;
class KeyStoreVersionError extends web3_error_base_js_1.BaseWeb3Error {
    constructor() {
        super('Unsupported key store version');
        this.code = error_codes_js_1.ERR_KEY_VERSION_UNSUPPORTED;
    }
}
exports.KeyStoreVersionError = KeyStoreVersionError;
class InvalidPasswordError extends web3_error_base_js_1.BaseWeb3Error {
    constructor() {
        super('Password cannot be empty');
        this.code = error_codes_js_1.ERR_INVALID_PASSWORD;
    }
}
exports.InvalidPasswordError = InvalidPasswordError;
class IVLengthError extends web3_error_base_js_1.BaseWeb3Error {
    constructor() {
        super('Initialization vector must be 16 bytes');
        this.code = error_codes_js_1.ERR_IV_LENGTH;
    }
}
exports.IVLengthError = IVLengthError;
class PBKDF2IterationsError extends web3_error_base_js_1.BaseWeb3Error {
    constructor() {
        super('c > 1000, pbkdf2 is less secure with less iterations');
        this.code = error_codes_js_1.ERR_PBKDF2_ITERATIONS;
    }
}
exports.PBKDF2IterationsError = PBKDF2IterationsError;
//# sourceMappingURL=account_errors.js.map