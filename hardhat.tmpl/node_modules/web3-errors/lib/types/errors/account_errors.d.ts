import { BaseWeb3Error } from '../web3_error_base.js';
export declare class PrivateKeyLengthError extends BaseWeb3Error {
    code: number;
    constructor();
}
export declare class InvalidPrivateKeyError extends BaseWeb3Error {
    code: number;
    constructor();
}
export declare class InvalidSignatureError extends BaseWeb3Error {
    code: number;
    constructor(errorDetails: string);
}
export declare class InvalidKdfError extends BaseWeb3Error {
    code: number;
    constructor();
}
export declare class KeyDerivationError extends BaseWeb3Error {
    code: number;
    constructor();
}
export declare class KeyStoreVersionError extends BaseWeb3Error {
    code: number;
    constructor();
}
export declare class InvalidPasswordError extends BaseWeb3Error {
    code: number;
    constructor();
}
export declare class IVLengthError extends BaseWeb3Error {
    code: number;
    constructor();
}
export declare class PBKDF2IterationsError extends BaseWeb3Error {
    code: number;
    constructor();
}
//# sourceMappingURL=account_errors.d.ts.map