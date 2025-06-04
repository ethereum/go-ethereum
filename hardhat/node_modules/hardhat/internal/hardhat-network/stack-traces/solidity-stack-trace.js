"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.UNRECOGNIZED_CONTRACT_NAME = exports.PRECOMPILE_FUNCTION_NAME = exports.UNKNOWN_FUNCTION_NAME = exports.UNRECOGNIZED_FUNCTION_NAME = exports.CONSTRUCTOR_FUNCTION_NAME = exports.RECEIVE_FUNCTION_NAME = exports.FALLBACK_FUNCTION_NAME = exports.stackTraceEntryTypeToString = exports.StackTraceEntryType = void 0;
const napi_rs_1 = require("../../../common/napi-rs");
const { StackTraceEntryType, stackTraceEntryTypeToString, FALLBACK_FUNCTION_NAME, RECEIVE_FUNCTION_NAME, CONSTRUCTOR_FUNCTION_NAME, UNRECOGNIZED_FUNCTION_NAME, UNKNOWN_FUNCTION_NAME, PRECOMPILE_FUNCTION_NAME, UNRECOGNIZED_CONTRACT_NAME, } = (0, napi_rs_1.requireNapiRsModule)("@nomicfoundation/edr");
exports.StackTraceEntryType = StackTraceEntryType;
exports.stackTraceEntryTypeToString = stackTraceEntryTypeToString;
exports.FALLBACK_FUNCTION_NAME = FALLBACK_FUNCTION_NAME;
exports.RECEIVE_FUNCTION_NAME = RECEIVE_FUNCTION_NAME;
exports.CONSTRUCTOR_FUNCTION_NAME = CONSTRUCTOR_FUNCTION_NAME;
exports.UNRECOGNIZED_FUNCTION_NAME = UNRECOGNIZED_FUNCTION_NAME;
exports.UNKNOWN_FUNCTION_NAME = UNKNOWN_FUNCTION_NAME;
exports.PRECOMPILE_FUNCTION_NAME = PRECOMPILE_FUNCTION_NAME;
exports.UNRECOGNIZED_CONTRACT_NAME = UNRECOGNIZED_CONTRACT_NAME;
//# sourceMappingURL=solidity-stack-trace.js.map