"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.UNRECOGNIZED_CONTRACT_NAME = exports.PRECOMPILE_FUNCTION_NAME = exports.UNKNOWN_FUNCTION_NAME = exports.UNRECOGNIZED_FUNCTION_NAME = exports.CONSTRUCTOR_FUNCTION_NAME = exports.RECEIVE_FUNCTION_NAME = exports.FALLBACK_FUNCTION_NAME = exports.StackTraceEntryType = void 0;
var StackTraceEntryType;
(function (StackTraceEntryType) {
    StackTraceEntryType[StackTraceEntryType["CALLSTACK_ENTRY"] = 0] = "CALLSTACK_ENTRY";
    StackTraceEntryType[StackTraceEntryType["UNRECOGNIZED_CREATE_CALLSTACK_ENTRY"] = 1] = "UNRECOGNIZED_CREATE_CALLSTACK_ENTRY";
    StackTraceEntryType[StackTraceEntryType["UNRECOGNIZED_CONTRACT_CALLSTACK_ENTRY"] = 2] = "UNRECOGNIZED_CONTRACT_CALLSTACK_ENTRY";
    StackTraceEntryType[StackTraceEntryType["PRECOMPILE_ERROR"] = 3] = "PRECOMPILE_ERROR";
    StackTraceEntryType[StackTraceEntryType["REVERT_ERROR"] = 4] = "REVERT_ERROR";
    StackTraceEntryType[StackTraceEntryType["PANIC_ERROR"] = 5] = "PANIC_ERROR";
    StackTraceEntryType[StackTraceEntryType["CUSTOM_ERROR"] = 6] = "CUSTOM_ERROR";
    StackTraceEntryType[StackTraceEntryType["FUNCTION_NOT_PAYABLE_ERROR"] = 7] = "FUNCTION_NOT_PAYABLE_ERROR";
    StackTraceEntryType[StackTraceEntryType["INVALID_PARAMS_ERROR"] = 8] = "INVALID_PARAMS_ERROR";
    StackTraceEntryType[StackTraceEntryType["FALLBACK_NOT_PAYABLE_ERROR"] = 9] = "FALLBACK_NOT_PAYABLE_ERROR";
    StackTraceEntryType[StackTraceEntryType["FALLBACK_NOT_PAYABLE_AND_NO_RECEIVE_ERROR"] = 10] = "FALLBACK_NOT_PAYABLE_AND_NO_RECEIVE_ERROR";
    StackTraceEntryType[StackTraceEntryType["UNRECOGNIZED_FUNCTION_WITHOUT_FALLBACK_ERROR"] = 11] = "UNRECOGNIZED_FUNCTION_WITHOUT_FALLBACK_ERROR";
    StackTraceEntryType[StackTraceEntryType["MISSING_FALLBACK_OR_RECEIVE_ERROR"] = 12] = "MISSING_FALLBACK_OR_RECEIVE_ERROR";
    StackTraceEntryType[StackTraceEntryType["RETURNDATA_SIZE_ERROR"] = 13] = "RETURNDATA_SIZE_ERROR";
    StackTraceEntryType[StackTraceEntryType["NONCONTRACT_ACCOUNT_CALLED_ERROR"] = 14] = "NONCONTRACT_ACCOUNT_CALLED_ERROR";
    StackTraceEntryType[StackTraceEntryType["CALL_FAILED_ERROR"] = 15] = "CALL_FAILED_ERROR";
    StackTraceEntryType[StackTraceEntryType["DIRECT_LIBRARY_CALL_ERROR"] = 16] = "DIRECT_LIBRARY_CALL_ERROR";
    StackTraceEntryType[StackTraceEntryType["UNRECOGNIZED_CREATE_ERROR"] = 17] = "UNRECOGNIZED_CREATE_ERROR";
    StackTraceEntryType[StackTraceEntryType["UNRECOGNIZED_CONTRACT_ERROR"] = 18] = "UNRECOGNIZED_CONTRACT_ERROR";
    StackTraceEntryType[StackTraceEntryType["OTHER_EXECUTION_ERROR"] = 19] = "OTHER_EXECUTION_ERROR";
    // This is a special case to handle a regression introduced in solc 0.6.3
    // For more info: https://github.com/ethereum/solidity/issues/9006
    StackTraceEntryType[StackTraceEntryType["UNMAPPED_SOLC_0_6_3_REVERT_ERROR"] = 20] = "UNMAPPED_SOLC_0_6_3_REVERT_ERROR";
    StackTraceEntryType[StackTraceEntryType["CONTRACT_TOO_LARGE_ERROR"] = 21] = "CONTRACT_TOO_LARGE_ERROR";
    StackTraceEntryType[StackTraceEntryType["INTERNAL_FUNCTION_CALLSTACK_ENTRY"] = 22] = "INTERNAL_FUNCTION_CALLSTACK_ENTRY";
    StackTraceEntryType[StackTraceEntryType["CONTRACT_CALL_RUN_OUT_OF_GAS_ERROR"] = 23] = "CONTRACT_CALL_RUN_OUT_OF_GAS_ERROR";
})(StackTraceEntryType = exports.StackTraceEntryType || (exports.StackTraceEntryType = {}));
exports.FALLBACK_FUNCTION_NAME = "<fallback>";
exports.RECEIVE_FUNCTION_NAME = "<receive>";
exports.CONSTRUCTOR_FUNCTION_NAME = "constructor";
exports.UNRECOGNIZED_FUNCTION_NAME = "<unrecognized-selector>";
exports.UNKNOWN_FUNCTION_NAME = "<unknown>";
exports.PRECOMPILE_FUNCTION_NAME = "<precompile>";
exports.UNRECOGNIZED_CONTRACT_NAME = "<UnrecognizedContract>";
//# sourceMappingURL=solidity-stack-trace.js.map