"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.formatSolidityParameter = exports.formatCustomError = exports.formatFailedEvmExecutionResult = exports.formatExecutionError = void 0;
const evm_execution_1 = require("./execution/types/evm-execution");
const execution_result_1 = require("./execution/types/execution-result");
const convert_evm_tuple_to_solidity_param_1 = require("./execution/utils/convert-evm-tuple-to-solidity-param");
/**
 * Formats an execution error result into a human-readable string.
 */
function formatExecutionError(result) {
    switch (result.type) {
        case execution_result_1.ExecutionResultType.SIMULATION_ERROR:
            return `Simulating the transaction failed with error: ${formatFailedEvmExecutionResult(result.error)}`;
        case execution_result_1.ExecutionResultType.STRATEGY_SIMULATION_ERROR:
            return `Simulating the transaction failed with error: ${result.error}`;
        case execution_result_1.ExecutionResultType.REVERTED_TRANSACTION:
            return `Transaction ${result.txHash} reverted`;
        case execution_result_1.ExecutionResultType.STATIC_CALL_ERROR:
            return `Static call failed with error: ${formatFailedEvmExecutionResult(result.error)}`;
        case execution_result_1.ExecutionResultType.STRATEGY_ERROR:
            return `Execution failed with error: ${result.error}`;
    }
}
exports.formatExecutionError = formatExecutionError;
/**
 * Formats a failed EVM execution result into a human-readable string.
 */
function formatFailedEvmExecutionResult(result) {
    switch (result.type) {
        case evm_execution_1.EvmExecutionResultTypes.INVALID_RESULT_ERROR:
            return `Invalid data returned`;
        case evm_execution_1.EvmExecutionResultTypes.REVERT_WITHOUT_REASON:
            return `Reverted without reason`;
        case evm_execution_1.EvmExecutionResultTypes.REVERT_WITH_REASON:
            return `Reverted with reason "${result.message}"`;
        case evm_execution_1.EvmExecutionResultTypes.REVERT_WITH_PANIC_CODE:
            return `Reverted with panic code ${result.panicCode} (${result.panicName}))`;
        case evm_execution_1.EvmExecutionResultTypes.REVERT_WITH_CUSTOM_ERROR:
            return `Reverted with custom error ${formatCustomError(result.errorName, result.args)}`;
        case evm_execution_1.EvmExecutionResultTypes.REVERT_WITH_UNKNOWN_CUSTOM_ERROR:
            return `Reverted with unknown custom error (signature ${result.signature})`;
        case evm_execution_1.EvmExecutionResultTypes.REVERT_WITH_INVALID_DATA:
            return `Reverted with invalid return data`;
        case evm_execution_1.EvmExecutionResultTypes.REVERT_WITH_INVALID_DATA_OR_UNKNOWN_CUSTOM_ERROR:
            return `Reverted with invalid return data or unknown custom error (signature ${result.signature})`;
    }
}
exports.formatFailedEvmExecutionResult = formatFailedEvmExecutionResult;
/**
 * Formats a custom error into a human-readable string.
 */
function formatCustomError(errorName, args) {
    const transformedArgs = (0, convert_evm_tuple_to_solidity_param_1.convertEvmTupleToSolidityParam)(args);
    return `${errorName}(${transformedArgs
        .map(formatSolidityParameter)
        .join(", ")})`;
}
exports.formatCustomError = formatCustomError;
/**
 * Formats a Solidity parameter into a human-readable string.
 *
 * @beta
 */
function formatSolidityParameter(param) {
    if (Array.isArray(param)) {
        const values = param.map(formatSolidityParameter);
        return `[${values.join(", ")}]`;
    }
    if (typeof param === "object") {
        const values = Object.entries(param).map(([key, value]) => `"${key}": ${formatSolidityParameter(value)}`);
        return `{${values.join(", ")}}`;
    }
    if (typeof param === "string") {
        return `"${param}"`;
    }
    return param.toString();
}
exports.formatSolidityParameter = formatSolidityParameter;
//# sourceMappingURL=formatters.js.map