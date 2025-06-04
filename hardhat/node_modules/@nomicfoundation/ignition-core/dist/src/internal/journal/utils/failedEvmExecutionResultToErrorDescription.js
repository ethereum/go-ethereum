"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.failedEvmExecutionResultToErrorDescription = void 0;
const evm_execution_1 = require("../../execution/types/evm-execution");
function failedEvmExecutionResultToErrorDescription(result) {
    switch (result.type) {
        case evm_execution_1.EvmExecutionResultTypes.INVALID_RESULT_ERROR: {
            return `Transaction appears to have succeeded, but has returned invalid data: '${result.data}'`;
        }
        case evm_execution_1.EvmExecutionResultTypes.REVERT_WITHOUT_REASON: {
            return `Transaction reverted`;
        }
        case evm_execution_1.EvmExecutionResultTypes.REVERT_WITH_REASON: {
            return `Transaction reverted with reason: '${result.message}'`;
        }
        case evm_execution_1.EvmExecutionResultTypes.REVERT_WITH_PANIC_CODE: {
            return `Transaction reverted with panic code (${result.panicCode}): '${result.panicName}'`;
        }
        case evm_execution_1.EvmExecutionResultTypes.REVERT_WITH_CUSTOM_ERROR: {
            return `Transaction reverted with custom error: '${result.errorName}' args: ${JSON.stringify(result.args.positional)}`;
        }
        case evm_execution_1.EvmExecutionResultTypes.REVERT_WITH_UNKNOWN_CUSTOM_ERROR: {
            return `Transaction reverted with unknown custom error. Error signature: '${result.signature}' data: '${result.data}'`;
        }
        case evm_execution_1.EvmExecutionResultTypes.REVERT_WITH_INVALID_DATA: {
            return `Transaction reverted with invalid error data: '${result.data}'`;
        }
        case evm_execution_1.EvmExecutionResultTypes.REVERT_WITH_INVALID_DATA_OR_UNKNOWN_CUSTOM_ERROR: {
            return `Transaction reverted with unknown error. Error signature: '${result.signature}' data: '${result.data}'`;
        }
    }
}
exports.failedEvmExecutionResultToErrorDescription = failedEvmExecutionResultToErrorDescription;
//# sourceMappingURL=failedEvmExecutionResultToErrorDescription.js.map