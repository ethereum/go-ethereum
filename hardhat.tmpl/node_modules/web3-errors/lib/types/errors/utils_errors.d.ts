import { InvalidValueError } from '../web3_error_base.js';
export declare class InvalidBytesError extends InvalidValueError {
    code: number;
    constructor(value: unknown);
}
export declare class InvalidNumberError extends InvalidValueError {
    code: number;
    constructor(value: unknown);
}
export declare class InvalidAddressError extends InvalidValueError {
    code: number;
    constructor(value: unknown);
}
export declare class InvalidStringError extends InvalidValueError {
    code: number;
    constructor(value: unknown);
}
export declare class InvalidUnitError extends InvalidValueError {
    code: number;
    constructor(value: unknown);
}
export declare class InvalidIntegerError extends InvalidValueError {
    code: number;
    constructor(value: unknown);
}
export declare class HexProcessingError extends InvalidValueError {
    code: number;
    constructor(value: unknown);
}
export declare class NibbleWidthError extends InvalidValueError {
    code: number;
    constructor(value: string);
}
export declare class InvalidTypeError extends InvalidValueError {
    code: number;
    constructor(value: unknown);
}
export declare class InvalidBooleanError extends InvalidValueError {
    code: number;
    constructor(value: unknown);
}
export declare class InvalidUnsignedIntegerError extends InvalidValueError {
    code: number;
    constructor(value: unknown);
}
export declare class InvalidSizeError extends InvalidValueError {
    code: number;
    constructor(value: unknown);
}
export declare class InvalidLargeValueError extends InvalidValueError {
    code: number;
    constructor(value: unknown);
}
export declare class InvalidBlockError extends InvalidValueError {
    code: number;
    constructor(value: string);
}
export declare class InvalidTypeAbiInputError extends InvalidValueError {
    code: number;
    constructor(value: string);
}
//# sourceMappingURL=utils_errors.d.ts.map