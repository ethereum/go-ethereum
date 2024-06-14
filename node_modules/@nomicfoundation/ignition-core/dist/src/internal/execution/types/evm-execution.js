"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.EvmExecutionResultTypes = void 0;
/**
 * Each of the possible contract execution results that Ignition can handle.
 */
var EvmExecutionResultTypes;
(function (EvmExecutionResultTypes) {
    EvmExecutionResultTypes["SUCESSFUL_RESULT"] = "SUCESSFUL_RESULT";
    EvmExecutionResultTypes["INVALID_RESULT_ERROR"] = "INVALID_RESULT_ERROR";
    EvmExecutionResultTypes["REVERT_WITHOUT_REASON"] = "REVERT_WITHOUT_REASON";
    EvmExecutionResultTypes["REVERT_WITH_REASON"] = "REVERT_WITH_REASON";
    EvmExecutionResultTypes["REVERT_WITH_PANIC_CODE"] = "REVERT_WITH_PANIC_CODE";
    EvmExecutionResultTypes["REVERT_WITH_CUSTOM_ERROR"] = "REVERT_WITH_CUSTOM_ERROR";
    EvmExecutionResultTypes["REVERT_WITH_UNKNOWN_CUSTOM_ERROR"] = "REVERT_WITH_UNKNOWN_CUSTOM_ERROR";
    EvmExecutionResultTypes["REVERT_WITH_INVALID_DATA"] = "REVERT_WITH_INVALID_DATA";
    EvmExecutionResultTypes["REVERT_WITH_INVALID_DATA_OR_UNKNOWN_CUSTOM_ERROR"] = "REVERT_WITH_INVALID_DATA_OR_UNKNOWN_CUSTOM_ERROR";
})(EvmExecutionResultTypes || (exports.EvmExecutionResultTypes = EvmExecutionResultTypes = {}));
//# sourceMappingURL=evm-execution.js.map