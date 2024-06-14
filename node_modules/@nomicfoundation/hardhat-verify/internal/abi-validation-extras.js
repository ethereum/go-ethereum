"use strict";
// This file contains helpers to detect and handle various
// errors that may be thrown by @ethersproject/abi
Object.defineProperty(exports, "__esModule", { value: true });
exports.isABIArgumentOverflowError = exports.isABIArgumentTypeError = exports.isABIArgumentLengthError = void 0;
function isABIArgumentLengthError(error) {
    return (error.code === "INVALID_ARGUMENT" &&
        error.count !== undefined &&
        typeof error.count.types === "number" &&
        typeof error.count.values === "number" &&
        error.value !== undefined &&
        typeof error.value.types === "object" &&
        typeof error.value.values === "object" &&
        error instanceof Error);
}
exports.isABIArgumentLengthError = isABIArgumentLengthError;
function isABIArgumentTypeError(error) {
    return (error.code === "INVALID_ARGUMENT" &&
        typeof error.argument === "string" &&
        "value" in error &&
        error instanceof Error);
}
exports.isABIArgumentTypeError = isABIArgumentTypeError;
function isABIArgumentOverflowError(error) {
    return (error.code === "NUMERIC_FAULT" &&
        error.fault === "overflow" &&
        typeof error.operation === "string" &&
        "value" in error &&
        error instanceof Error);
}
exports.isABIArgumentOverflowError = isABIArgumentOverflowError;
//# sourceMappingURL=abi-validation-extras.js.map