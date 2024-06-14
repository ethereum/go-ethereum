"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.panicErrorCodeToReason = exports.PANIC_CODES = void 0;
exports.PANIC_CODES = {
    ASSERTION_ERROR: 0x1,
    ARITHMETIC_OVERFLOW: 0x11,
    DIVISION_BY_ZERO: 0x12,
    ENUM_CONVERSION_OUT_OF_BOUNDS: 0x21,
    INCORRECTLY_ENCODED_STORAGE_BYTE_ARRAY: 0x22,
    POP_ON_EMPTY_ARRAY: 0x31,
    ARRAY_ACCESS_OUT_OF_BOUNDS: 0x32,
    TOO_MUCH_MEMORY_ALLOCATED: 0x41,
    ZERO_INITIALIZED_VARIABLE: 0x51,
};
// copied from hardhat-core
function panicErrorCodeToReason(errorCode) {
    switch (errorCode) {
        case 0x1n:
            return "Assertion error";
        case 0x11n:
            return "Arithmetic operation overflowed outside of an unchecked block";
        case 0x12n:
            return "Division or modulo division by zero";
        case 0x21n:
            return "Tried to convert a value into an enum, but the value was too big or negative";
        case 0x22n:
            return "Incorrectly encoded storage byte array";
        case 0x31n:
            return ".pop() was called on an empty array";
        case 0x32n:
            return "Array accessed at an out-of-bounds or negative index";
        case 0x41n:
            return "Too much memory was allocated, or an array was created that is too large";
        case 0x51n:
            return "Called a zero-initialized variable of internal function type";
    }
}
exports.panicErrorCodeToReason = panicErrorCodeToReason;
//# sourceMappingURL=panic.js.map