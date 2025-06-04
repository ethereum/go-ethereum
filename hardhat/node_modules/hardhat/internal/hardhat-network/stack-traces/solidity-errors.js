"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.SolidityError = exports.encodeSolidityStackTrace = exports.wrapWithSolidityErrorsCorrection = exports.getCurrentStack = void 0;
const util_1 = require("@ethereumjs/util");
const return_data_1 = require("../provider/return-data");
const panic_errors_1 = require("./panic-errors");
const solidity_stack_trace_1 = require("./solidity-stack-trace");
const inspect = Symbol.for("nodejs.util.inspect.custom");
function getCurrentStack() {
    const previousPrepareStackTrace = Error.prepareStackTrace;
    Error.prepareStackTrace = (e, s) => s;
    const error = new Error();
    const stack = error.stack;
    Error.prepareStackTrace = previousPrepareStackTrace;
    return stack;
}
exports.getCurrentStack = getCurrentStack;
async function wrapWithSolidityErrorsCorrection(f, stackFramesToRemove) {
    const stackTraceAtCall = getCurrentStack().slice(stackFramesToRemove);
    try {
        return await f();
    }
    catch (error) {
        if (error.stackTrace === undefined) {
            // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
            throw error;
        }
        // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
        throw encodeSolidityStackTrace(error.message, error.stackTrace, stackTraceAtCall);
    }
}
exports.wrapWithSolidityErrorsCorrection = wrapWithSolidityErrorsCorrection;
function encodeSolidityStackTrace(fallbackMessage, stackTrace, previousStack) {
    if (Error.prepareStackTrace === undefined) {
        // Node 12 doesn't have a default Error.prepareStackTrace
        require("source-map-support/register");
    }
    const previousPrepareStackTrace = Error.prepareStackTrace;
    Error.prepareStackTrace = (error, stack) => {
        if (previousStack !== undefined) {
            stack = previousStack;
        }
        else {
            // We remove error management related stack traces
            stack.splice(0, 1);
        }
        for (const entry of stackTrace) {
            const callsite = encodeStackTraceEntry(entry);
            if (callsite === undefined) {
                continue;
            }
            stack.unshift(callsite);
        }
        return previousPrepareStackTrace(error, stack);
    };
    const msg = getMessageFromLastStackTraceEntry(stackTrace[stackTrace.length - 1]);
    const solidityError = new SolidityError(msg !== undefined ? msg : fallbackMessage, stackTrace);
    // This hack is here because prepare stack is lazy
    solidityError.stack = solidityError.stack;
    Error.prepareStackTrace = previousPrepareStackTrace;
    return solidityError;
}
exports.encodeSolidityStackTrace = encodeSolidityStackTrace;
function encodeStackTraceEntry(stackTraceEntry) {
    switch (stackTraceEntry.type) {
        case solidity_stack_trace_1.StackTraceEntryType.UNRECOGNIZED_FUNCTION_WITHOUT_FALLBACK_ERROR:
        case solidity_stack_trace_1.StackTraceEntryType.MISSING_FALLBACK_OR_RECEIVE_ERROR:
            return sourceReferenceToSolidityCallsite({
                ...stackTraceEntry.sourceReference,
                function: solidity_stack_trace_1.UNRECOGNIZED_FUNCTION_NAME,
            });
        case solidity_stack_trace_1.StackTraceEntryType.CALLSTACK_ENTRY:
        case solidity_stack_trace_1.StackTraceEntryType.REVERT_ERROR:
        case solidity_stack_trace_1.StackTraceEntryType.CUSTOM_ERROR:
        case solidity_stack_trace_1.StackTraceEntryType.FUNCTION_NOT_PAYABLE_ERROR:
        case solidity_stack_trace_1.StackTraceEntryType.INVALID_PARAMS_ERROR:
        case solidity_stack_trace_1.StackTraceEntryType.FALLBACK_NOT_PAYABLE_ERROR:
        case solidity_stack_trace_1.StackTraceEntryType.FALLBACK_NOT_PAYABLE_AND_NO_RECEIVE_ERROR:
        case solidity_stack_trace_1.StackTraceEntryType.RETURNDATA_SIZE_ERROR:
        case solidity_stack_trace_1.StackTraceEntryType.NONCONTRACT_ACCOUNT_CALLED_ERROR:
        case solidity_stack_trace_1.StackTraceEntryType.CALL_FAILED_ERROR:
        case solidity_stack_trace_1.StackTraceEntryType.DIRECT_LIBRARY_CALL_ERROR:
            return sourceReferenceToSolidityCallsite(stackTraceEntry.sourceReference);
        case solidity_stack_trace_1.StackTraceEntryType.UNRECOGNIZED_CREATE_CALLSTACK_ENTRY:
            return new SolidityCallSite(undefined, solidity_stack_trace_1.UNRECOGNIZED_CONTRACT_NAME, solidity_stack_trace_1.CONSTRUCTOR_FUNCTION_NAME, undefined);
        case solidity_stack_trace_1.StackTraceEntryType.UNRECOGNIZED_CONTRACT_CALLSTACK_ENTRY:
            return new SolidityCallSite((0, util_1.bytesToHex)(stackTraceEntry.address), solidity_stack_trace_1.UNRECOGNIZED_CONTRACT_NAME, solidity_stack_trace_1.UNKNOWN_FUNCTION_NAME, undefined);
        case solidity_stack_trace_1.StackTraceEntryType.PRECOMPILE_ERROR:
            return new SolidityCallSite(undefined, `<PrecompileContract ${stackTraceEntry.precompile}>`, solidity_stack_trace_1.PRECOMPILE_FUNCTION_NAME, undefined);
        case solidity_stack_trace_1.StackTraceEntryType.UNRECOGNIZED_CREATE_ERROR:
            return new SolidityCallSite(undefined, solidity_stack_trace_1.UNRECOGNIZED_CONTRACT_NAME, solidity_stack_trace_1.CONSTRUCTOR_FUNCTION_NAME, undefined);
        case solidity_stack_trace_1.StackTraceEntryType.UNRECOGNIZED_CONTRACT_ERROR:
            return new SolidityCallSite((0, util_1.bytesToHex)(stackTraceEntry.address), solidity_stack_trace_1.UNRECOGNIZED_CONTRACT_NAME, solidity_stack_trace_1.UNKNOWN_FUNCTION_NAME, undefined);
        case solidity_stack_trace_1.StackTraceEntryType.INTERNAL_FUNCTION_CALLSTACK_ENTRY:
            return new SolidityCallSite(stackTraceEntry.sourceReference.sourceName, stackTraceEntry.sourceReference.contract, `internal@${stackTraceEntry.pc}`, undefined);
        case solidity_stack_trace_1.StackTraceEntryType.CONTRACT_CALL_RUN_OUT_OF_GAS_ERROR:
            if (stackTraceEntry.sourceReference !== undefined) {
                return sourceReferenceToSolidityCallsite(stackTraceEntry.sourceReference);
            }
            return new SolidityCallSite(undefined, solidity_stack_trace_1.UNRECOGNIZED_CONTRACT_NAME, solidity_stack_trace_1.UNKNOWN_FUNCTION_NAME, undefined);
        case solidity_stack_trace_1.StackTraceEntryType.OTHER_EXECUTION_ERROR:
        case solidity_stack_trace_1.StackTraceEntryType.CONTRACT_TOO_LARGE_ERROR:
        case solidity_stack_trace_1.StackTraceEntryType.PANIC_ERROR:
        case solidity_stack_trace_1.StackTraceEntryType.UNMAPPED_SOLC_0_6_3_REVERT_ERROR:
            if (stackTraceEntry.sourceReference === undefined) {
                return new SolidityCallSite(undefined, solidity_stack_trace_1.UNRECOGNIZED_CONTRACT_NAME, solidity_stack_trace_1.UNKNOWN_FUNCTION_NAME, undefined);
            }
            return sourceReferenceToSolidityCallsite(stackTraceEntry.sourceReference);
    }
}
function sourceReferenceToSolidityCallsite(sourceReference) {
    return new SolidityCallSite(sourceReference.sourceName, sourceReference.contract, sourceReference.function !== undefined
        ? sourceReference.function
        : solidity_stack_trace_1.UNKNOWN_FUNCTION_NAME, sourceReference.line);
}
function getMessageFromLastStackTraceEntry(stackTraceEntry) {
    switch (stackTraceEntry.type) {
        case solidity_stack_trace_1.StackTraceEntryType.PRECOMPILE_ERROR:
            return `Transaction reverted: call to precompile ${stackTraceEntry.precompile} failed`;
        case solidity_stack_trace_1.StackTraceEntryType.FUNCTION_NOT_PAYABLE_ERROR:
            return `Transaction reverted: non-payable function was called with value ${stackTraceEntry.value.toString(10)}`;
        case solidity_stack_trace_1.StackTraceEntryType.INVALID_PARAMS_ERROR:
            return `Transaction reverted: function was called with incorrect parameters`;
        case solidity_stack_trace_1.StackTraceEntryType.FALLBACK_NOT_PAYABLE_ERROR:
            return `Transaction reverted: fallback function is not payable and was called with value ${stackTraceEntry.value.toString(10)}`;
        case solidity_stack_trace_1.StackTraceEntryType.FALLBACK_NOT_PAYABLE_AND_NO_RECEIVE_ERROR:
            return `Transaction reverted: there's no receive function, fallback function is not payable and was called with value ${stackTraceEntry.value.toString(10)}`;
        case solidity_stack_trace_1.StackTraceEntryType.UNRECOGNIZED_FUNCTION_WITHOUT_FALLBACK_ERROR:
            return `Transaction reverted: function selector was not recognized and there's no fallback function`;
        case solidity_stack_trace_1.StackTraceEntryType.MISSING_FALLBACK_OR_RECEIVE_ERROR:
            return `Transaction reverted: function selector was not recognized and there's no fallback nor receive function`;
        case solidity_stack_trace_1.StackTraceEntryType.RETURNDATA_SIZE_ERROR:
            return `Transaction reverted: function returned an unexpected amount of data`;
        case solidity_stack_trace_1.StackTraceEntryType.NONCONTRACT_ACCOUNT_CALLED_ERROR:
            return `Transaction reverted: function call to a non-contract account`;
        case solidity_stack_trace_1.StackTraceEntryType.CALL_FAILED_ERROR:
            return `Transaction reverted: function call failed to execute`;
        case solidity_stack_trace_1.StackTraceEntryType.DIRECT_LIBRARY_CALL_ERROR:
            return `Transaction reverted: library was called directly`;
        case solidity_stack_trace_1.StackTraceEntryType.UNRECOGNIZED_CREATE_ERROR:
        case solidity_stack_trace_1.StackTraceEntryType.UNRECOGNIZED_CONTRACT_ERROR: {
            const returnData = new return_data_1.ReturnData(stackTraceEntry.returnData);
            if (returnData.isErrorReturnData()) {
                return `VM Exception while processing transaction: reverted with reason string '${returnData.decodeError()}'`;
            }
            if (returnData.isPanicReturnData()) {
                const message = (0, panic_errors_1.panicErrorCodeToMessage)(returnData.decodePanic());
                return `VM Exception while processing transaction: ${message}`;
            }
            if (!returnData.isEmpty()) {
                const buffer = Buffer.from(returnData.value).toString("hex");
                return `VM Exception while processing transaction: reverted with an unrecognized custom error (return data: 0x${buffer})`;
            }
            if (stackTraceEntry.isInvalidOpcodeError) {
                return "VM Exception while processing transaction: invalid opcode";
            }
            return "Transaction reverted without a reason string";
        }
        case solidity_stack_trace_1.StackTraceEntryType.REVERT_ERROR: {
            const returnData = new return_data_1.ReturnData(stackTraceEntry.returnData);
            if (returnData.isErrorReturnData()) {
                return `VM Exception while processing transaction: reverted with reason string '${returnData.decodeError()}'`;
            }
            if (stackTraceEntry.isInvalidOpcodeError) {
                return "VM Exception while processing transaction: invalid opcode";
            }
            return "Transaction reverted without a reason string";
        }
        case solidity_stack_trace_1.StackTraceEntryType.PANIC_ERROR:
            const panicMessage = (0, panic_errors_1.panicErrorCodeToMessage)(stackTraceEntry.errorCode);
            return `VM Exception while processing transaction: ${panicMessage}`;
        case solidity_stack_trace_1.StackTraceEntryType.CUSTOM_ERROR:
            return `VM Exception while processing transaction: ${stackTraceEntry.message}`;
        case solidity_stack_trace_1.StackTraceEntryType.OTHER_EXECUTION_ERROR:
            // TODO: What if there was returnData?
            return `Transaction reverted and Hardhat couldn't infer the reason.`;
        case solidity_stack_trace_1.StackTraceEntryType.UNMAPPED_SOLC_0_6_3_REVERT_ERROR:
            return "Transaction reverted without a reason string and without a valid sourcemap provided by the compiler. Some line numbers may be off. We strongly recommend upgrading solc and always using revert reasons.";
        case solidity_stack_trace_1.StackTraceEntryType.CONTRACT_TOO_LARGE_ERROR:
            return "Transaction reverted: trying to deploy a contract whose code is too large";
        case solidity_stack_trace_1.StackTraceEntryType.CONTRACT_CALL_RUN_OUT_OF_GAS_ERROR:
            return "Transaction reverted: contract call run out of gas and made the transaction revert";
    }
}
// Note: This error class MUST NOT extend ProviderError, as libraries
//   use the code property to detect if they are dealing with a JSON-RPC error,
//   and take control of errors.
class SolidityError extends Error {
    constructor(message, stackTrace) {
        super(message);
        this.stackTrace = stackTrace;
    }
    [inspect]() {
        return this.inspect();
    }
    inspect() {
        return this.stack !== undefined
            ? this.stack
            : "Internal error when encoding SolidityError";
    }
}
exports.SolidityError = SolidityError;
class SolidityCallSite {
    constructor(_sourceName, _contract, _functionName, _line) {
        this._sourceName = _sourceName;
        this._contract = _contract;
        this._functionName = _functionName;
        this._line = _line;
    }
    getColumnNumber() {
        return null;
    }
    getEvalOrigin() {
        return undefined;
    }
    getFileName() {
        return this._sourceName ?? "unknown";
    }
    getFunction() {
        return undefined;
    }
    getFunctionName() {
        // if it's a top-level function, we print its name
        if (this._contract === undefined) {
            return this._functionName ?? null;
        }
        return null;
    }
    getLineNumber() {
        return this._line !== undefined ? this._line : null;
    }
    getMethodName() {
        if (this._contract !== undefined) {
            return this._functionName ?? null;
        }
        return null;
    }
    getPosition() {
        return 0;
    }
    getPromiseIndex() {
        return 0;
    }
    getScriptNameOrSourceURL() {
        return "";
    }
    getThis() {
        return undefined;
    }
    getTypeName() {
        return this._contract ?? null;
    }
    isAsync() {
        return false;
    }
    isConstructor() {
        return false;
    }
    isEval() {
        return false;
    }
    isNative() {
        return false;
    }
    isPromiseAll() {
        return false;
    }
    isToplevel() {
        return false;
    }
    getScriptHash() {
        return "";
    }
    getEnclosingColumnNumber() {
        return 0;
    }
    getEnclosingLineNumber() {
        return 0;
    }
    toString() {
        return "[SolidityCallSite]";
    }
}
//# sourceMappingURL=solidity-errors.js.map