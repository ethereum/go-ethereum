"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.panicErrorCodeToMessage = void 0;
function panicErrorCodeToMessage(errorCode) {
    const reason = panicErrorCodeToReason(errorCode);
    if (reason !== undefined) {
        return `reverted with panic code 0x${errorCode.toString(16)} (${reason})`;
    }
    return `reverted with unknown panic code 0x${errorCode.toString(16)}`;
}
exports.panicErrorCodeToMessage = panicErrorCodeToMessage;
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
//# sourceMappingURL=panic-errors.js.map