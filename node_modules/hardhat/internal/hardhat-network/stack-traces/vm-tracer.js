"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.VMTracer = void 0;
const ethereumjs_util_1 = require("@nomicfoundation/ethereumjs-util");
const errors_1 = require("../../core/errors");
const exit_1 = require("../provider/vm/exit");
const message_trace_1 = require("./message-trace");
/* eslint-disable @nomicfoundation/hardhat-internal-rules/only-hardhat-error */
const DUMMY_RETURN_DATA = Buffer.from([]);
const DUMMY_GAS_USED = 0n;
class VMTracer {
    constructor(_throwErrors = true) {
        this._throwErrors = _throwErrors;
        this.tracingSteps = [];
        this._messageTraces = [];
        // TODO: temporarily hardcoded to remove the need of using ethereumjs' common and evm here
        this._maxPrecompileNumber = 10;
    }
    getLastTopLevelMessageTrace() {
        return this._messageTraces[0];
    }
    getLastError() {
        return this._lastError;
    }
    clearLastError() {
        this._lastError = undefined;
    }
    _shouldKeepTracing() {
        return this._throwErrors || this._lastError === undefined;
    }
    async addBeforeMessage(message) {
        if (!this._shouldKeepTracing()) {
            return;
        }
        try {
            let trace;
            if (message.depth === 0) {
                this._messageTraces = [];
                this.tracingSteps = [];
            }
            if (message.to === undefined) {
                const createTrace = {
                    code: message.data,
                    steps: [],
                    value: message.value,
                    exit: new exit_1.Exit(exit_1.ExitCode.SUCCESS),
                    returnData: DUMMY_RETURN_DATA,
                    numberOfSubtraces: 0,
                    depth: message.depth,
                    deployedContract: undefined,
                    gasUsed: DUMMY_GAS_USED,
                };
                trace = createTrace;
            }
            else {
                const toAsBigInt = (0, ethereumjs_util_1.bytesToBigInt)(message.to);
                if (toAsBigInt > 0 && toAsBigInt <= this._maxPrecompileNumber) {
                    const precompileTrace = {
                        precompile: Number(toAsBigInt),
                        calldata: message.data,
                        value: message.value,
                        exit: new exit_1.Exit(exit_1.ExitCode.SUCCESS),
                        returnData: DUMMY_RETURN_DATA,
                        depth: message.depth,
                        gasUsed: DUMMY_GAS_USED,
                    };
                    trace = precompileTrace;
                }
                else {
                    const codeAddress = message.codeAddress;
                    // if we enter here, then `to` is not undefined, therefore
                    // `codeAddress` and `code` should be defined
                    (0, errors_1.assertHardhatInvariant)(codeAddress !== undefined, "codeAddress should be defined");
                    (0, errors_1.assertHardhatInvariant)(message.code !== undefined, "code should be defined");
                    const callTrace = {
                        code: message.code,
                        calldata: message.data,
                        steps: [],
                        value: message.value,
                        exit: new exit_1.Exit(exit_1.ExitCode.SUCCESS),
                        returnData: DUMMY_RETURN_DATA,
                        address: message.to,
                        numberOfSubtraces: 0,
                        depth: message.depth,
                        gasUsed: DUMMY_GAS_USED,
                        codeAddress,
                    };
                    trace = callTrace;
                }
            }
            if (this._messageTraces.length > 0) {
                const parentTrace = this._messageTraces[this._messageTraces.length - 1];
                if ((0, message_trace_1.isPrecompileTrace)(parentTrace)) {
                    throw new Error("This should not happen: message execution started while a precompile was executing");
                }
                parentTrace.steps.push(trace);
                parentTrace.numberOfSubtraces += 1;
            }
            this._messageTraces.push(trace);
        }
        catch (error) {
            if (this._throwErrors) {
                throw error;
            }
            else {
                this._lastError = error;
            }
        }
    }
    async addStep(step) {
        if (!this._shouldKeepTracing()) {
            return;
        }
        this.tracingSteps.push(step);
        try {
            const trace = this._messageTraces[this._messageTraces.length - 1];
            if ((0, message_trace_1.isPrecompileTrace)(trace)) {
                throw new Error("This should not happen: step event fired while a precompile was executing");
            }
            trace.steps.push({ pc: Number(step.pc) });
        }
        catch (error) {
            if (this._throwErrors) {
                throw error;
            }
            else {
                this._lastError = error;
            }
        }
    }
    async addAfterMessage(result, haltOverride) {
        if (!this._shouldKeepTracing()) {
            return;
        }
        try {
            const trace = this._messageTraces[this._messageTraces.length - 1];
            trace.gasUsed = result.result.gasUsed;
            const executionResult = result.result;
            if ((0, message_trace_1.isSuccessResult)(executionResult)) {
                trace.exit = exit_1.Exit.fromEdrSuccessReason(executionResult.reason);
                trace.returnData = executionResult.output.returnValue;
                if ((0, message_trace_1.isCreateTrace)(trace)) {
                    trace.deployedContract = executionResult.output.address;
                }
            }
            else if ((0, message_trace_1.isHaltResult)(executionResult)) {
                trace.exit =
                    haltOverride ?? exit_1.Exit.fromEdrExceptionalHalt(executionResult.reason);
                trace.returnData = Buffer.from([]);
            }
            else {
                trace.exit = new exit_1.Exit(exit_1.ExitCode.REVERT);
                trace.returnData = executionResult.output;
            }
            if (this._messageTraces.length > 1) {
                this._messageTraces.pop();
            }
        }
        catch (error) {
            if (this._throwErrors) {
                throw error;
            }
            else {
                this._lastError = error;
            }
        }
    }
}
exports.VMTracer = VMTracer;
//# sourceMappingURL=vm-tracer.js.map