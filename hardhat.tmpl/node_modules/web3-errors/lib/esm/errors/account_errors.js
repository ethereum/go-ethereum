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
import { ERR_PRIVATE_KEY_LENGTH, ERR_INVALID_PRIVATE_KEY, ERR_INVALID_SIGNATURE, ERR_UNSUPPORTED_KDF, ERR_KEY_DERIVATION_FAIL, ERR_KEY_VERSION_UNSUPPORTED, ERR_INVALID_PASSWORD, ERR_IV_LENGTH, ERR_PBKDF2_ITERATIONS, } from '../error_codes.js';
import { BaseWeb3Error } from '../web3_error_base.js';
export class PrivateKeyLengthError extends BaseWeb3Error {
    constructor() {
        super(`Private key must be 32 bytes.`);
        this.code = ERR_PRIVATE_KEY_LENGTH;
    }
}
export class InvalidPrivateKeyError extends BaseWeb3Error {
    constructor() {
        super(`Invalid Private Key, Not a valid string or uint8Array`);
        this.code = ERR_INVALID_PRIVATE_KEY;
    }
}
export class InvalidSignatureError extends BaseWeb3Error {
    constructor(errorDetails) {
        super(`"${errorDetails}"`);
        this.code = ERR_INVALID_SIGNATURE;
    }
}
export class InvalidKdfError extends BaseWeb3Error {
    constructor() {
        super(`Invalid key derivation function`);
        this.code = ERR_UNSUPPORTED_KDF;
    }
}
export class KeyDerivationError extends BaseWeb3Error {
    constructor() {
        super(`Key derivation failed - possibly wrong password`);
        this.code = ERR_KEY_DERIVATION_FAIL;
    }
}
export class KeyStoreVersionError extends BaseWeb3Error {
    constructor() {
        super('Unsupported key store version');
        this.code = ERR_KEY_VERSION_UNSUPPORTED;
    }
}
export class InvalidPasswordError extends BaseWeb3Error {
    constructor() {
        super('Password cannot be empty');
        this.code = ERR_INVALID_PASSWORD;
    }
}
export class IVLengthError extends BaseWeb3Error {
    constructor() {
        super('Initialization vector must be 16 bytes');
        this.code = ERR_IV_LENGTH;
    }
}
export class PBKDF2IterationsError extends BaseWeb3Error {
    constructor() {
        super('c > 1000, pbkdf2 is less secure with less iterations');
        this.code = ERR_PBKDF2_ITERATIONS;
    }
}
//# sourceMappingURL=account_errors.js.map