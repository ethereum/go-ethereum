"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.instructionToCallstackStackTraceEntry = exports.ErrorInferrer = void 0;
/* eslint "@typescript-eslint/no-non-null-assertion": "error" */
const abi_1 = require("@ethersproject/abi");
const ethereumjs_util_1 = require("@nomicfoundation/ethereumjs-util");
const semver_1 = __importDefault(require("semver"));
const errors_1 = require("../../core/errors");
const abi_helpers_1 = require("../../util/abi-helpers");
const return_data_1 = require("../provider/return-data");
const exit_1 = require("../provider/vm/exit");
const message_trace_1 = require("./message-trace");
const model_1 = require("./model");
const opcodes_1 = require("./opcodes");
const solidity_stack_trace_1 = require("./solidity-stack-trace");
const FIRST_SOLC_VERSION_CREATE_PARAMS_VALIDATION = "0.5.9";
const FIRST_SOLC_VERSION_RECEIVE_FUNCTION = "0.6.0";
const FIRST_SOLC_VERSION_WITH_UNMAPPED_REVERTS = "0.6.3";
/* eslint-disable @nomicfoundation/hardhat-internal-rules/only-hardhat-error */
class ErrorInferrer {
    inferBeforeTracingCallMessage(trace) {
        if (this._isDirectLibraryCall(trace)) {
            return this._getDirectLibraryCallErrorStackTrace(trace);
        }
        const calledFunction = trace.bytecode.contract.getFunctionFromSelector(trace.calldata.slice(0, 4));
        if (calledFunction !== undefined &&
            this._isFunctionNotPayableError(trace, calledFunction)) {
            return [
                {
                    type: solidity_stack_trace_1.StackTraceEntryType.FUNCTION_NOT_PAYABLE_ERROR,
                    sourceReference: this._getFunctionStartSourceReference(trace, calledFunction),
                    value: trace.value,
                },
            ];
        }
        if (this._isMissingFunctionAndFallbackError(trace, calledFunction)) {
            if (this._emptyCalldataAndNoReceive(trace)) {
                return [
                    {
                        type: solidity_stack_trace_1.StackTraceEntryType.MISSING_FALLBACK_OR_RECEIVE_ERROR,
                        sourceReference: this._getContractStartWithoutFunctionSourceReference(trace),
                    },
                ];
            }
            return [
                {
                    type: solidity_stack_trace_1.StackTraceEntryType.UNRECOGNIZED_FUNCTION_WITHOUT_FALLBACK_ERROR,
                    sourceReference: this._getContractStartWithoutFunctionSourceReference(trace),
                },
            ];
        }
        if (this._isFallbackNotPayableError(trace, calledFunction)) {
            if (this._emptyCalldataAndNoReceive(trace)) {
                return [
                    {
                        type: solidity_stack_trace_1.StackTraceEntryType.FALLBACK_NOT_PAYABLE_AND_NO_RECEIVE_ERROR,
                        sourceReference: this._getFallbackStartSourceReference(trace),
                        value: trace.value,
                    },
                ];
            }
            return [
                {
                    type: solidity_stack_trace_1.StackTraceEntryType.FALLBACK_NOT_PAYABLE_ERROR,
                    sourceReference: this._getFallbackStartSourceReference(trace),
                    value: trace.value,
                },
            ];
        }
    }
    inferBeforeTracingCreateMessage(trace) {
        if (this._isConstructorNotPayableError(trace)) {
            return [
                {
                    type: solidity_stack_trace_1.StackTraceEntryType.FUNCTION_NOT_PAYABLE_ERROR,
                    sourceReference: this._getConstructorStartSourceReference(trace),
                    value: trace.value,
                },
            ];
        }
        if (this._isConstructorInvalidArgumentsError(trace)) {
            return [
                {
                    type: solidity_stack_trace_1.StackTraceEntryType.INVALID_PARAMS_ERROR,
                    sourceReference: this._getConstructorStartSourceReference(trace),
                },
            ];
        }
    }
    inferAfterTracing(trace, stacktrace, functionJumpdests, jumpedIntoFunction, lastSubmessageData) {
        return (this._checkLastSubmessage(trace, stacktrace, lastSubmessageData) ??
            this._checkFailedLastCall(trace, stacktrace) ??
            this._checkLastInstruction(trace, stacktrace, functionJumpdests, jumpedIntoFunction) ??
            this._checkNonContractCalled(trace, stacktrace) ??
            this._checkSolidity063UnmappedRevert(trace, stacktrace) ??
            this._checkContractTooLarge(trace) ??
            this._otherExecutionErrorStacktrace(trace, stacktrace));
    }
    filterRedundantFrames(stacktrace) {
        return stacktrace.filter((frame, i) => {
            if (i + 1 === stacktrace.length) {
                return true;
            }
            const nextFrame = stacktrace[i + 1];
            // we can only filter frames if we know their sourceReference
            // and the one from the next frame
            if (frame.sourceReference === undefined ||
                nextFrame.sourceReference === undefined) {
                return true;
            }
            // look TWO frames ahead to determine if this is a specific occurrence of
            // a redundant CALLSTACK_ENTRY frame observed when using Solidity 0.8.5:
            if (frame.type === solidity_stack_trace_1.StackTraceEntryType.CALLSTACK_ENTRY &&
                i + 2 < stacktrace.length &&
                stacktrace[i + 2].sourceReference !== undefined &&
                stacktrace[i + 2].type === solidity_stack_trace_1.StackTraceEntryType.RETURNDATA_SIZE_ERROR) {
                // ! below for tsc. we confirmed existence in the enclosing conditional.
                const thatSrcRef = stacktrace[i + 2].sourceReference;
                if (thatSrcRef !== undefined &&
                    frame.sourceReference.range[0] === thatSrcRef.range[0] &&
                    frame.sourceReference.range[1] === thatSrcRef.range[1] &&
                    frame.sourceReference.line === thatSrcRef.line) {
                    return false;
                }
            }
            // constructors contain the whole contract, so we ignore them
            if (frame.sourceReference.function === "constructor" &&
                nextFrame.sourceReference.function !== "constructor") {
                return true;
            }
            // this is probably a recursive call
            if (i > 0 &&
                frame.type === nextFrame.type &&
                frame.sourceReference.range[0] === nextFrame.sourceReference.range[0] &&
                frame.sourceReference.range[1] === nextFrame.sourceReference.range[1] &&
                frame.sourceReference.line === nextFrame.sourceReference.line) {
                return true;
            }
            if (frame.sourceReference.range[0] <= nextFrame.sourceReference.range[0] &&
                frame.sourceReference.range[1] >= nextFrame.sourceReference.range[1]) {
                return false;
            }
            return true;
        });
    }
    // Heuristics
    /**
     * Check if the last submessage can be used to generate the stack trace.
     */
    _checkLastSubmessage(trace, stacktrace, lastSubmessageData) {
        if (lastSubmessageData === undefined) {
            return undefined;
        }
        const inferredStacktrace = [...stacktrace];
        // get the instruction before the submessage and add it to the stack trace
        const callStep = trace.steps[lastSubmessageData.stepIndex - 1];
        if (!(0, message_trace_1.isEvmStep)(callStep)) {
            throw new Error("This should not happen: MessageTrace should be preceded by a EVM step");
        }
        const callInst = trace.bytecode.getInstruction(callStep.pc);
        const callStackFrame = instructionToCallstackStackTraceEntry(trace.bytecode, callInst);
        const lastMessageFailed = lastSubmessageData.messageTrace.exit.isError();
        if (lastMessageFailed) {
            // add the call/create that generated the message to the stack trace
            inferredStacktrace.push(callStackFrame);
            if (this._isSubtraceErrorPropagated(trace, lastSubmessageData.stepIndex) ||
                this._isProxyErrorPropagated(trace, lastSubmessageData.stepIndex)) {
                inferredStacktrace.push(...lastSubmessageData.stacktrace);
                if (this._isContractCallRunOutOfGasError(trace, lastSubmessageData.stepIndex)) {
                    const lastFrame = inferredStacktrace.pop();
                    (0, errors_1.assertHardhatInvariant)(lastFrame !== undefined, "Expected inferred stack trace to have at least one frame");
                    inferredStacktrace.push({
                        type: solidity_stack_trace_1.StackTraceEntryType.CONTRACT_CALL_RUN_OUT_OF_GAS_ERROR,
                        sourceReference: lastFrame.sourceReference,
                    });
                }
                return this._fixInitialModifier(trace, inferredStacktrace);
            }
        }
        else {
            const isReturnDataSizeError = this._failsRightAfterCall(trace, lastSubmessageData.stepIndex);
            if (isReturnDataSizeError) {
                inferredStacktrace.push({
                    type: solidity_stack_trace_1.StackTraceEntryType.RETURNDATA_SIZE_ERROR,
                    sourceReference: callStackFrame.sourceReference,
                });
                return this._fixInitialModifier(trace, inferredStacktrace);
            }
        }
    }
    /**
     * Check if the last call/create that was done failed.
     */
    _checkFailedLastCall(trace, stacktrace) {
        for (let stepIndex = trace.steps.length - 2; stepIndex >= 0; stepIndex--) {
            const step = trace.steps[stepIndex];
            const nextStep = trace.steps[stepIndex + 1];
            if (!(0, message_trace_1.isEvmStep)(step)) {
                return;
            }
            const inst = trace.bytecode.getInstruction(step.pc);
            const isCallOrCreate = (0, opcodes_1.isCall)(inst.opcode) || (0, opcodes_1.isCreate)(inst.opcode);
            if (isCallOrCreate && (0, message_trace_1.isEvmStep)(nextStep)) {
                if (this._isCallFailedError(trace, stepIndex, inst)) {
                    const inferredStacktrace = [
                        ...stacktrace,
                        this._callInstructionToCallFailedToExecuteStackTraceEntry(trace.bytecode, inst),
                    ];
                    return this._fixInitialModifier(trace, inferredStacktrace);
                }
            }
        }
    }
    /**
     * Check if the execution stopped with a revert or an invalid opcode.
     */
    _checkRevertOrInvalidOpcode(trace, stacktrace, lastInstruction, functionJumpdests, jumpedIntoFunction) {
        if (lastInstruction.opcode !== opcodes_1.Opcode.REVERT &&
            lastInstruction.opcode !== opcodes_1.Opcode.INVALID) {
            return;
        }
        const inferredStacktrace = [...stacktrace];
        if (lastInstruction.location !== undefined &&
            (!(0, message_trace_1.isDecodedCallTrace)(trace) || jumpedIntoFunction)) {
            // There should always be a function here, but that's not the case with optimizations.
            //
            // If this is a create trace, we already checked args and nonpayable failures before
            // calling this function.
            //
            // If it's a call trace, we already jumped into a function. But optimizations can happen.
            const failingFunction = lastInstruction.location.getContainingFunction();
            // If the failure is in a modifier we add an entry with the function/constructor
            if (failingFunction !== undefined &&
                failingFunction.type === model_1.ContractFunctionType.MODIFIER) {
                inferredStacktrace.push(this._getEntryBeforeFailureInModifier(trace, functionJumpdests));
            }
        }
        const panicStacktrace = this._checkPanic(trace, inferredStacktrace, lastInstruction);
        if (panicStacktrace !== undefined) {
            return panicStacktrace;
        }
        const customErrorStacktrace = this._checkCustomErrors(trace, inferredStacktrace, lastInstruction);
        if (customErrorStacktrace !== undefined) {
            return customErrorStacktrace;
        }
        if (lastInstruction.location !== undefined &&
            (!(0, message_trace_1.isDecodedCallTrace)(trace) || jumpedIntoFunction)) {
            const failingFunction = lastInstruction.location.getContainingFunction();
            if (failingFunction !== undefined) {
                inferredStacktrace.push(this._instructionWithinFunctionToRevertStackTraceEntry(trace, lastInstruction));
            }
            else if ((0, message_trace_1.isDecodedCallTrace)(trace)) {
                // This is here because of the optimizations
                const functionSelector = trace.bytecode.contract.getFunctionFromSelector(trace.calldata.slice(0, 4));
                // in general this shouldn't happen, but it does when viaIR is enabled,
                // "optimizerSteps": "u" is used, and the called function is fallback or
                // receive
                if (functionSelector === undefined) {
                    return;
                }
                inferredStacktrace.push({
                    type: solidity_stack_trace_1.StackTraceEntryType.REVERT_ERROR,
                    sourceReference: this._getFunctionStartSourceReference(trace, functionSelector),
                    message: new return_data_1.ReturnData(trace.returnData),
                    isInvalidOpcodeError: lastInstruction.opcode === opcodes_1.Opcode.INVALID,
                });
            }
            else {
                // This is here because of the optimizations
                inferredStacktrace.push({
                    type: solidity_stack_trace_1.StackTraceEntryType.REVERT_ERROR,
                    sourceReference: this._getConstructorStartSourceReference(trace),
                    message: new return_data_1.ReturnData(trace.returnData),
                    isInvalidOpcodeError: lastInstruction.opcode === opcodes_1.Opcode.INVALID,
                });
            }
            return this._fixInitialModifier(trace, inferredStacktrace);
        }
        // If the revert instruction is not mapped but there is return data,
        // we add the frame anyway, sith the best sourceReference we can get
        if (lastInstruction.location === undefined && trace.returnData.length > 0) {
            const revertFrame = {
                type: solidity_stack_trace_1.StackTraceEntryType.REVERT_ERROR,
                sourceReference: this._getLastSourceReference(trace) ??
                    this._getContractStartWithoutFunctionSourceReference(trace),
                message: new return_data_1.ReturnData(trace.returnData),
                isInvalidOpcodeError: lastInstruction.opcode === opcodes_1.Opcode.INVALID,
            };
            inferredStacktrace.push(revertFrame);
            return this._fixInitialModifier(trace, inferredStacktrace);
        }
    }
    /**
     * Check if the trace reverted with a panic error.
     */
    _checkPanic(trace, stacktrace, lastInstruction) {
        if (!this._isPanicReturnData(trace.returnData)) {
            return;
        }
        // If the last frame is an internal function, it means that the trace
        // jumped there to return the panic. If that's the case, we remove that
        // frame.
        const lastFrame = stacktrace[stacktrace.length - 1];
        if (lastFrame?.type === solidity_stack_trace_1.StackTraceEntryType.INTERNAL_FUNCTION_CALLSTACK_ENTRY) {
            stacktrace.splice(-1);
        }
        const panicReturnData = new return_data_1.ReturnData(trace.returnData);
        const errorCode = panicReturnData.decodePanic();
        // if the error comes from a call to a zero-initialized function,
        // we remove the last frame, which represents the call, to avoid
        // having duplicated frames
        if (errorCode === 0x51n) {
            stacktrace.splice(-1);
        }
        const inferredStacktrace = [...stacktrace];
        inferredStacktrace.push(this._instructionWithinFunctionToPanicStackTraceEntry(trace, lastInstruction, errorCode));
        return this._fixInitialModifier(trace, inferredStacktrace);
    }
    _checkCustomErrors(trace, stacktrace, lastInstruction) {
        const returnData = new return_data_1.ReturnData(trace.returnData);
        if (returnData.isEmpty() || returnData.isErrorReturnData()) {
            // if there is no return data, or if it's a Error(string),
            // then it can't be a custom error
            return;
        }
        const rawReturnData = Buffer.from(returnData.value).toString("hex");
        let errorMessage = `reverted with an unrecognized custom error (return data: 0x${rawReturnData})`;
        for (const customError of trace.bytecode.contract.customErrors) {
            if (returnData.matchesSelector(customError.selector)) {
                // if the return data matches a custom error in the called contract,
                // we format the message using the returnData and the custom error instance
                const decodedValues = abi_1.defaultAbiCoder.decode(customError.paramTypes, returnData.value.slice(4));
                const params = abi_helpers_1.AbiHelpers.formatValues([...decodedValues]);
                errorMessage = `reverted with custom error '${customError.name}(${params})'`;
                break;
            }
        }
        const inferredStacktrace = [...stacktrace];
        inferredStacktrace.push(this._instructionWithinFunctionToCustomErrorStackTraceEntry(trace, lastInstruction, errorMessage));
        return this._fixInitialModifier(trace, inferredStacktrace);
    }
    /**
     * Check last instruction to try to infer the error.
     */
    _checkLastInstruction(trace, stacktrace, functionJumpdests, jumpedIntoFunction) {
        if (trace.steps.length === 0) {
            return;
        }
        const lastStep = trace.steps[trace.steps.length - 1];
        if (!(0, message_trace_1.isEvmStep)(lastStep)) {
            throw new Error("This should not happen: MessageTrace ends with a subtrace");
        }
        const lastInstruction = trace.bytecode.getInstruction(lastStep.pc);
        const revertOrInvalidStacktrace = this._checkRevertOrInvalidOpcode(trace, stacktrace, lastInstruction, functionJumpdests, jumpedIntoFunction);
        if (revertOrInvalidStacktrace !== undefined) {
            return revertOrInvalidStacktrace;
        }
        if ((0, message_trace_1.isDecodedCallTrace)(trace) && !jumpedIntoFunction) {
            if (this._hasFailedInsideTheFallbackFunction(trace) ||
                this._hasFailedInsideTheReceiveFunction(trace)) {
                return [
                    this._instructionWithinFunctionToRevertStackTraceEntry(trace, lastInstruction),
                ];
            }
            // Sometimes we do fail inside of a function but there's no jump into
            if (lastInstruction.location !== undefined) {
                const failingFunction = lastInstruction.location.getContainingFunction();
                if (failingFunction !== undefined) {
                    return [
                        {
                            type: solidity_stack_trace_1.StackTraceEntryType.REVERT_ERROR,
                            sourceReference: this._getFunctionStartSourceReference(trace, failingFunction),
                            message: new return_data_1.ReturnData(trace.returnData),
                            isInvalidOpcodeError: lastInstruction.opcode === opcodes_1.Opcode.INVALID,
                        },
                    ];
                }
            }
            const calledFunction = trace.bytecode.contract.getFunctionFromSelector(trace.calldata.slice(0, 4));
            if (calledFunction !== undefined) {
                const isValidCalldata = calledFunction.isValidCalldata(trace.calldata.slice(4));
                if (!isValidCalldata) {
                    return [
                        {
                            type: solidity_stack_trace_1.StackTraceEntryType.INVALID_PARAMS_ERROR,
                            sourceReference: this._getFunctionStartSourceReference(trace, calledFunction),
                        },
                    ];
                }
            }
            if (this._solidity063MaybeUnmappedRevert(trace)) {
                const revertFrame = this._solidity063GetFrameForUnmappedRevertBeforeFunction(trace);
                if (revertFrame !== undefined) {
                    return [revertFrame];
                }
            }
            return [this._getOtherErrorBeforeCalledFunctionStackTraceEntry(trace)];
        }
    }
    _checkNonContractCalled(trace, stacktrace) {
        if (this._isCalledNonContractAccountError(trace)) {
            const sourceReference = this._getLastSourceReference(trace);
            // We are sure this is not undefined because there was at least a call instruction
            (0, errors_1.assertHardhatInvariant)(sourceReference !== undefined, "Expected source reference to be defined");
            const nonContractCalledFrame = {
                type: solidity_stack_trace_1.StackTraceEntryType.NONCONTRACT_ACCOUNT_CALLED_ERROR,
                sourceReference,
            };
            return [...stacktrace, nonContractCalledFrame];
        }
    }
    _checkSolidity063UnmappedRevert(trace, stacktrace) {
        if (this._solidity063MaybeUnmappedRevert(trace)) {
            const revertFrame = this._solidity063GetFrameForUnmappedRevertWithinFunction(trace);
            if (revertFrame !== undefined) {
                return [...stacktrace, revertFrame];
            }
        }
    }
    _checkContractTooLarge(trace) {
        if ((0, message_trace_1.isCreateTrace)(trace) && this._isContractTooLargeError(trace)) {
            return [
                {
                    type: solidity_stack_trace_1.StackTraceEntryType.CONTRACT_TOO_LARGE_ERROR,
                    sourceReference: this._getConstructorStartSourceReference(trace),
                },
            ];
        }
    }
    _otherExecutionErrorStacktrace(trace, stacktrace) {
        const otherExecutionErrorFrame = {
            type: solidity_stack_trace_1.StackTraceEntryType.OTHER_EXECUTION_ERROR,
            sourceReference: this._getLastSourceReference(trace),
        };
        return [...stacktrace, otherExecutionErrorFrame];
    }
    // Helpers
    _fixInitialModifier(trace, stacktrace) {
        const firstEntry = stacktrace[0];
        if (firstEntry !== undefined &&
            firstEntry.type === solidity_stack_trace_1.StackTraceEntryType.CALLSTACK_ENTRY &&
            firstEntry.functionType === model_1.ContractFunctionType.MODIFIER) {
            return [
                this._getEntryBeforeInitialModifierCallstackEntry(trace),
                ...stacktrace,
            ];
        }
        return stacktrace;
    }
    _isDirectLibraryCall(trace) {
        return (trace.depth === 0 && trace.bytecode.contract.type === model_1.ContractType.LIBRARY);
    }
    _getDirectLibraryCallErrorStackTrace(trace) {
        const func = trace.bytecode.contract.getFunctionFromSelector(trace.calldata.slice(0, 4));
        if (func !== undefined) {
            return [
                {
                    type: solidity_stack_trace_1.StackTraceEntryType.DIRECT_LIBRARY_CALL_ERROR,
                    sourceReference: this._getFunctionStartSourceReference(trace, func),
                },
            ];
        }
        return [
            {
                type: solidity_stack_trace_1.StackTraceEntryType.DIRECT_LIBRARY_CALL_ERROR,
                sourceReference: this._getContractStartWithoutFunctionSourceReference(trace),
            },
        ];
    }
    _isFunctionNotPayableError(trace, calledFunction) {
        // This error doesn't return data
        if (trace.returnData.length > 0) {
            return false;
        }
        if (trace.value <= 0n) {
            return false;
        }
        // Libraries don't have a nonpayable check
        if (trace.bytecode.contract.type === model_1.ContractType.LIBRARY) {
            return false;
        }
        return calledFunction.isPayable === undefined || !calledFunction.isPayable;
    }
    _getFunctionStartSourceReference(trace, func) {
        return {
            sourceName: func.location.file.sourceName,
            sourceContent: func.location.file.content,
            contract: trace.bytecode.contract.name,
            function: func.name,
            line: func.location.getStartingLineNumber(),
            range: [
                func.location.offset,
                func.location.offset + func.location.length,
            ],
        };
    }
    _isMissingFunctionAndFallbackError(trace, calledFunction) {
        // This error doesn't return data
        if (trace.returnData.length > 0) {
            return false;
        }
        // the called function exists in the contract
        if (calledFunction !== undefined) {
            return false;
        }
        // there's a receive function and no calldata
        if (trace.calldata.length === 0 &&
            trace.bytecode.contract.receive !== undefined) {
            return false;
        }
        return trace.bytecode.contract.fallback === undefined;
    }
    _emptyCalldataAndNoReceive(trace) {
        // this only makes sense when receive functions are available
        if (semver_1.default.lt(trace.bytecode.compilerVersion, FIRST_SOLC_VERSION_RECEIVE_FUNCTION)) {
            return false;
        }
        return (trace.calldata.length === 0 &&
            trace.bytecode.contract.receive === undefined);
    }
    _getContractStartWithoutFunctionSourceReference(trace) {
        const location = trace.bytecode.contract.location;
        return {
            sourceName: location.file.sourceName,
            sourceContent: location.file.content,
            contract: trace.bytecode.contract.name,
            line: location.getStartingLineNumber(),
            range: [location.offset, location.offset + location.length],
        };
    }
    _isFallbackNotPayableError(trace, calledFunction) {
        if (calledFunction !== undefined) {
            return false;
        }
        // This error doesn't return data
        if (trace.returnData.length > 0) {
            return false;
        }
        if (trace.value <= 0n) {
            return false;
        }
        if (trace.bytecode.contract.fallback === undefined) {
            return false;
        }
        const isPayable = trace.bytecode.contract.fallback.isPayable;
        return isPayable === undefined || !isPayable;
    }
    _getFallbackStartSourceReference(trace) {
        const func = trace.bytecode.contract.fallback;
        if (func === undefined) {
            throw new Error("This shouldn't happen: trying to get fallback source reference from a contract without fallback");
        }
        return {
            sourceName: func.location.file.sourceName,
            sourceContent: func.location.file.content,
            contract: trace.bytecode.contract.name,
            function: solidity_stack_trace_1.FALLBACK_FUNCTION_NAME,
            line: func.location.getStartingLineNumber(),
            range: [
                func.location.offset,
                func.location.offset + func.location.length,
            ],
        };
    }
    _isConstructorNotPayableError(trace) {
        // This error doesn't return data
        if (trace.returnData.length > 0) {
            return false;
        }
        const constructor = trace.bytecode.contract.constructorFunction;
        // This function is only matters with contracts that have constructors defined. The ones that
        // don't are abstract contracts, or their constructor doesn't take any argument.
        if (constructor === undefined) {
            return false;
        }
        return (trace.value > 0n &&
            (constructor.isPayable === undefined || !constructor.isPayable));
    }
    /**
     * Returns a source reference pointing to the constructor if it exists, or to the contract
     * otherwise.
     */
    _getConstructorStartSourceReference(trace) {
        const contract = trace.bytecode.contract;
        const constructor = contract.constructorFunction;
        const line = constructor !== undefined
            ? constructor.location.getStartingLineNumber()
            : contract.location.getStartingLineNumber();
        return {
            sourceName: contract.location.file.sourceName,
            sourceContent: contract.location.file.content,
            contract: contract.name,
            function: solidity_stack_trace_1.CONSTRUCTOR_FUNCTION_NAME,
            line,
            range: [
                contract.location.offset,
                contract.location.offset + contract.location.length,
            ],
        };
    }
    _isConstructorInvalidArgumentsError(trace) {
        // This error doesn't return data
        if (trace.returnData.length > 0) {
            return false;
        }
        const contract = trace.bytecode.contract;
        const constructor = contract.constructorFunction;
        // This function is only matters with contracts that have constructors defined. The ones that
        // don't are abstract contracts, or their constructor doesn't take any argument.
        if (constructor === undefined) {
            return false;
        }
        if (semver_1.default.lt(trace.bytecode.compilerVersion, FIRST_SOLC_VERSION_CREATE_PARAMS_VALIDATION)) {
            return false;
        }
        const lastStep = trace.steps[trace.steps.length - 1];
        if (!(0, message_trace_1.isEvmStep)(lastStep)) {
            return false;
        }
        const lastInst = trace.bytecode.getInstruction(lastStep.pc);
        if (lastInst.opcode !== opcodes_1.Opcode.REVERT || lastInst.location !== undefined) {
            return false;
        }
        let hasReadDeploymentCodeSize = false;
        // eslint-disable-next-line @typescript-eslint/prefer-for-of
        for (let stepIndex = 0; stepIndex < trace.steps.length; stepIndex++) {
            const step = trace.steps[stepIndex];
            if (!(0, message_trace_1.isEvmStep)(step)) {
                return false;
            }
            const inst = trace.bytecode.getInstruction(step.pc);
            if (inst.location !== undefined &&
                !contract.location.equals(inst.location) &&
                !constructor.location.equals(inst.location)) {
                return false;
            }
            if (inst.opcode === opcodes_1.Opcode.CODESIZE && (0, message_trace_1.isCreateTrace)(trace)) {
                hasReadDeploymentCodeSize = true;
            }
        }
        return hasReadDeploymentCodeSize;
    }
    _getEntryBeforeInitialModifierCallstackEntry(trace) {
        if ((0, message_trace_1.isDecodedCreateTrace)(trace)) {
            return {
                type: solidity_stack_trace_1.StackTraceEntryType.CALLSTACK_ENTRY,
                sourceReference: this._getConstructorStartSourceReference(trace),
                functionType: model_1.ContractFunctionType.CONSTRUCTOR,
            };
        }
        const calledFunction = trace.bytecode.contract.getFunctionFromSelector(trace.calldata.slice(0, 4));
        if (calledFunction !== undefined) {
            return {
                type: solidity_stack_trace_1.StackTraceEntryType.CALLSTACK_ENTRY,
                sourceReference: this._getFunctionStartSourceReference(trace, calledFunction),
                functionType: model_1.ContractFunctionType.FUNCTION,
            };
        }
        // If it failed or made a call from within a modifier, and the selector doesn't match
        // any function, it must have a fallback.
        return {
            type: solidity_stack_trace_1.StackTraceEntryType.CALLSTACK_ENTRY,
            sourceReference: this._getFallbackStartSourceReference(trace),
            functionType: model_1.ContractFunctionType.FALLBACK,
        };
    }
    _getLastSourceReference(trace) {
        for (let i = trace.steps.length - 1; i >= 0; i--) {
            const step = trace.steps[i];
            if (!(0, message_trace_1.isEvmStep)(step)) {
                continue;
            }
            const inst = trace.bytecode.getInstruction(step.pc);
            if (inst.location === undefined) {
                continue;
            }
            const sourceReference = sourceLocationToSourceReference(trace.bytecode, inst.location);
            if (sourceReference !== undefined) {
                return sourceReference;
            }
        }
        return undefined;
    }
    _hasFailedInsideTheFallbackFunction(trace) {
        const contract = trace.bytecode.contract;
        if (contract.fallback === undefined) {
            return false;
        }
        return this._hasFailedInsideFunction(trace, contract.fallback);
    }
    _hasFailedInsideTheReceiveFunction(trace) {
        const contract = trace.bytecode.contract;
        if (contract.receive === undefined) {
            return false;
        }
        return this._hasFailedInsideFunction(trace, contract.receive);
    }
    _hasFailedInsideFunction(trace, func) {
        const lastStep = trace.steps[trace.steps.length - 1];
        const lastInstruction = trace.bytecode.getInstruction(lastStep.pc);
        return (lastInstruction.location !== undefined &&
            lastInstruction.opcode === opcodes_1.Opcode.REVERT &&
            func.location.contains(lastInstruction.location));
    }
    _instructionWithinFunctionToRevertStackTraceEntry(trace, inst) {
        const sourceReference = sourceLocationToSourceReference(trace.bytecode, inst.location);
        (0, errors_1.assertHardhatInvariant)(sourceReference !== undefined, "Expected source reference to be defined");
        return {
            type: solidity_stack_trace_1.StackTraceEntryType.REVERT_ERROR,
            sourceReference,
            message: new return_data_1.ReturnData(trace.returnData),
            isInvalidOpcodeError: inst.opcode === opcodes_1.Opcode.INVALID,
        };
    }
    _instructionWithinFunctionToUnmappedSolc063RevertErrorStackTraceEntry(trace, inst) {
        const sourceReference = sourceLocationToSourceReference(trace.bytecode, inst.location);
        return {
            type: solidity_stack_trace_1.StackTraceEntryType.UNMAPPED_SOLC_0_6_3_REVERT_ERROR,
            sourceReference,
        };
    }
    _instructionWithinFunctionToPanicStackTraceEntry(trace, inst, errorCode) {
        const lastSourceReference = this._getLastSourceReference(trace);
        return {
            type: solidity_stack_trace_1.StackTraceEntryType.PANIC_ERROR,
            sourceReference: sourceLocationToSourceReference(trace.bytecode, inst.location) ??
                lastSourceReference,
            errorCode,
        };
    }
    _instructionWithinFunctionToCustomErrorStackTraceEntry(trace, inst, message) {
        const lastSourceReference = this._getLastSourceReference(trace);
        (0, errors_1.assertHardhatInvariant)(lastSourceReference !== undefined, "Expected last source reference to be defined");
        return {
            type: solidity_stack_trace_1.StackTraceEntryType.CUSTOM_ERROR,
            sourceReference: sourceLocationToSourceReference(trace.bytecode, inst.location) ??
                lastSourceReference,
            message,
        };
    }
    _solidity063MaybeUnmappedRevert(trace) {
        if (trace.steps.length === 0) {
            return false;
        }
        const lastStep = trace.steps[trace.steps.length - 1];
        if (!(0, message_trace_1.isEvmStep)(lastStep)) {
            return false;
        }
        const lastInst = trace.bytecode.getInstruction(lastStep.pc);
        return (semver_1.default.satisfies(trace.bytecode.compilerVersion, `^${FIRST_SOLC_VERSION_WITH_UNMAPPED_REVERTS}`) && lastInst.opcode === opcodes_1.Opcode.REVERT);
    }
    // Solidity 0.6.3 unmapped reverts special handling
    // For more info: https://github.com/ethereum/solidity/issues/9006
    _solidity063GetFrameForUnmappedRevertBeforeFunction(trace) {
        let revertFrame = this._solidity063GetFrameForUnmappedRevertWithinFunction(trace);
        if (revertFrame === undefined ||
            revertFrame.sourceReference === undefined) {
            if (trace.bytecode.contract.receive === undefined ||
                trace.calldata.length > 0) {
                if (trace.bytecode.contract.fallback !== undefined) {
                    // Failed within the fallback
                    const location = trace.bytecode.contract.fallback.location;
                    revertFrame = {
                        type: solidity_stack_trace_1.StackTraceEntryType.UNMAPPED_SOLC_0_6_3_REVERT_ERROR,
                        sourceReference: {
                            contract: trace.bytecode.contract.name,
                            function: solidity_stack_trace_1.FALLBACK_FUNCTION_NAME,
                            sourceName: location.file.sourceName,
                            sourceContent: location.file.content,
                            line: location.getStartingLineNumber(),
                            range: [location.offset, location.offset + location.length],
                        },
                    };
                    this._solidity063CorrectLineNumber(revertFrame);
                }
            }
            else {
                // Failed within the receive function
                const location = trace.bytecode.contract.receive.location;
                revertFrame = {
                    type: solidity_stack_trace_1.StackTraceEntryType.UNMAPPED_SOLC_0_6_3_REVERT_ERROR,
                    sourceReference: {
                        contract: trace.bytecode.contract.name,
                        function: solidity_stack_trace_1.RECEIVE_FUNCTION_NAME,
                        sourceName: location.file.sourceName,
                        sourceContent: location.file.content,
                        line: location.getStartingLineNumber(),
                        range: [location.offset, location.offset + location.length],
                    },
                };
                this._solidity063CorrectLineNumber(revertFrame);
            }
        }
        return revertFrame;
    }
    _getOtherErrorBeforeCalledFunctionStackTraceEntry(trace) {
        return {
            type: solidity_stack_trace_1.StackTraceEntryType.OTHER_EXECUTION_ERROR,
            sourceReference: this._getContractStartWithoutFunctionSourceReference(trace),
        };
    }
    _isCalledNonContractAccountError(trace) {
        // We could change this to checking that the last valid location maps to a call, but
        // it's way more complex as we need to get the ast node from that location.
        const lastIndex = this._getLastInstructionWithValidLocationStepIndex(trace);
        if (lastIndex === undefined || lastIndex === 0) {
            return false;
        }
        const lastStep = trace.steps[lastIndex]; // We know this is an EVM step
        const lastInst = trace.bytecode.getInstruction(lastStep.pc);
        if (lastInst.opcode !== opcodes_1.Opcode.ISZERO) {
            return false;
        }
        const prevStep = trace.steps[lastIndex - 1]; // We know this is an EVM step
        const prevInst = trace.bytecode.getInstruction(prevStep.pc);
        return prevInst.opcode === opcodes_1.Opcode.EXTCODESIZE;
    }
    _solidity063GetFrameForUnmappedRevertWithinFunction(trace) {
        // If we are within a function there's a last valid location. It may
        // be the entire contract.
        const prevInst = this._getLastInstructionWithValidLocation(trace);
        const lastStep = trace.steps[trace.steps.length - 1];
        const nextInstPc = lastStep.pc + 1;
        const hasNextInst = trace.bytecode.hasInstruction(nextInstPc);
        if (hasNextInst) {
            const nextInst = trace.bytecode.getInstruction(nextInstPc);
            const prevLoc = prevInst?.location;
            const nextLoc = nextInst.location;
            const prevFunc = prevLoc?.getContainingFunction();
            const nextFunc = nextLoc?.getContainingFunction();
            // This is probably a require. This means that we have the exact
            // line, but the stack trace may be degraded (e.g. missing our
            // synthetic call frames when failing in a modifier) so we still
            // add this frame as UNMAPPED_SOLC_0_6_3_REVERT_ERROR
            if (prevFunc !== undefined &&
                nextLoc !== undefined &&
                prevLoc !== undefined &&
                prevLoc.equals(nextLoc)) {
                return this._instructionWithinFunctionToUnmappedSolc063RevertErrorStackTraceEntry(trace, nextInst);
            }
            let revertFrame;
            // If the previous and next location don't match, we try to use the
            // previous one if it's inside a function, otherwise we use the next one
            if (prevFunc !== undefined && prevInst !== undefined) {
                revertFrame =
                    this._instructionWithinFunctionToUnmappedSolc063RevertErrorStackTraceEntry(trace, prevInst);
            }
            else if (nextFunc !== undefined) {
                revertFrame =
                    this._instructionWithinFunctionToUnmappedSolc063RevertErrorStackTraceEntry(trace, nextInst);
            }
            if (revertFrame !== undefined) {
                this._solidity063CorrectLineNumber(revertFrame);
            }
            return revertFrame;
        }
        if ((0, message_trace_1.isCreateTrace)(trace) && prevInst !== undefined) {
            // Solidity is smart enough to stop emitting extra instructions after
            // an unconditional revert happens in a constructor. If this is the case
            // we just return a special error.
            const constructorRevertFrame = this._instructionWithinFunctionToUnmappedSolc063RevertErrorStackTraceEntry(trace, prevInst);
            // When the latest instruction is not within a function we need
            // some default sourceReference to show to the user
            if (constructorRevertFrame.sourceReference === undefined) {
                const location = trace.bytecode.contract.location;
                const defaultSourceReference = {
                    function: solidity_stack_trace_1.CONSTRUCTOR_FUNCTION_NAME,
                    contract: trace.bytecode.contract.name,
                    sourceName: location.file.sourceName,
                    sourceContent: location.file.content,
                    line: location.getStartingLineNumber(),
                    range: [location.offset, location.offset + location.length],
                };
                if (trace.bytecode.contract.constructorFunction !== undefined) {
                    defaultSourceReference.line =
                        trace.bytecode.contract.constructorFunction.location.getStartingLineNumber();
                }
                constructorRevertFrame.sourceReference = defaultSourceReference;
            }
            else {
                this._solidity063CorrectLineNumber(constructorRevertFrame);
            }
            return constructorRevertFrame;
        }
        if (prevInst !== undefined) {
            // We may as well just be in a function or modifier and just happen
            // to be at the last instruction of the runtime bytecode.
            // In this case we just return whatever the last mapped intruction
            // points to.
            const latestInstructionRevertFrame = this._instructionWithinFunctionToUnmappedSolc063RevertErrorStackTraceEntry(trace, prevInst);
            if (latestInstructionRevertFrame.sourceReference !== undefined) {
                this._solidity063CorrectLineNumber(latestInstructionRevertFrame);
            }
            return latestInstructionRevertFrame;
        }
    }
    _isContractTooLargeError(trace) {
        return trace.exit.kind === exit_1.ExitCode.CODESIZE_EXCEEDS_MAXIMUM;
    }
    _solidity063CorrectLineNumber(revertFrame) {
        if (revertFrame.sourceReference === undefined) {
            return;
        }
        const lines = revertFrame.sourceReference.sourceContent.split("\n");
        const currentLine = lines[revertFrame.sourceReference.line - 1];
        if (currentLine.includes("require") || currentLine.includes("revert")) {
            return;
        }
        const nextLines = lines.slice(revertFrame.sourceReference.line);
        const firstNonEmptyLine = nextLines.findIndex((l) => l.trim() !== "");
        if (firstNonEmptyLine === -1) {
            return;
        }
        const nextLine = nextLines[firstNonEmptyLine];
        if (nextLine.includes("require") || nextLine.includes("revert")) {
            revertFrame.sourceReference.line += 1 + firstNonEmptyLine;
        }
    }
    _getLastInstructionWithValidLocationStepIndex(trace) {
        for (let i = trace.steps.length - 1; i >= 0; i--) {
            const step = trace.steps[i];
            if (!(0, message_trace_1.isEvmStep)(step)) {
                return undefined;
            }
            const inst = trace.bytecode.getInstruction(step.pc);
            if (inst.location !== undefined) {
                return i;
            }
        }
        return undefined;
    }
    _getLastInstructionWithValidLocation(trace) {
        const lastLocationIndex = this._getLastInstructionWithValidLocationStepIndex(trace);
        if (lastLocationIndex === undefined) {
            return undefined;
        }
        const lastLocationStep = trace.steps[lastLocationIndex];
        if ((0, message_trace_1.isEvmStep)(lastLocationStep)) {
            const lastInstructionWithLocation = trace.bytecode.getInstruction(lastLocationStep.pc);
            return lastInstructionWithLocation;
        }
        return undefined;
    }
    _callInstructionToCallFailedToExecuteStackTraceEntry(bytecode, callInst) {
        const sourceReference = sourceLocationToSourceReference(bytecode, callInst.location);
        (0, errors_1.assertHardhatInvariant)(sourceReference !== undefined, "Expected source reference to be defined");
        // Calls only happen within functions
        return {
            type: solidity_stack_trace_1.StackTraceEntryType.CALL_FAILED_ERROR,
            sourceReference,
        };
    }
    _getEntryBeforeFailureInModifier(trace, functionJumpdests) {
        // If there's a jumpdest, this modifier belongs to the last function that it represents
        if (functionJumpdests.length > 0) {
            return instructionToCallstackStackTraceEntry(trace.bytecode, functionJumpdests[functionJumpdests.length - 1]);
        }
        // This function is only called after we jumped into the initial function in call traces, so
        // there should always be at least a function jumpdest.
        if (!(0, message_trace_1.isDecodedCreateTrace)(trace)) {
            throw new Error("This shouldn't happen: a call trace has no functionJumpdest but has already jumped into a function");
        }
        // If there's no jump dest, we point to the constructor.
        return {
            type: solidity_stack_trace_1.StackTraceEntryType.CALLSTACK_ENTRY,
            sourceReference: this._getConstructorStartSourceReference(trace),
            functionType: model_1.ContractFunctionType.CONSTRUCTOR,
        };
    }
    _failsRightAfterCall(trace, callSubtraceStepIndex) {
        const lastStep = trace.steps[trace.steps.length - 1];
        if (!(0, message_trace_1.isEvmStep)(lastStep)) {
            return false;
        }
        const lastInst = trace.bytecode.getInstruction(lastStep.pc);
        if (lastInst.opcode !== opcodes_1.Opcode.REVERT) {
            return false;
        }
        const callOpcodeStep = trace.steps[callSubtraceStepIndex - 1];
        const callInst = trace.bytecode.getInstruction(callOpcodeStep.pc);
        // Calls are always made from within functions
        (0, errors_1.assertHardhatInvariant)(callInst.location !== undefined, "Expected call instruction location to be defined");
        return this._isLastLocation(trace, callSubtraceStepIndex + 1, callInst.location);
    }
    _isCallFailedError(trace, instIndex, callInstruction) {
        const callLocation = callInstruction.location;
        // Calls are always made from within functions
        (0, errors_1.assertHardhatInvariant)(callLocation !== undefined, "Expected call location to be defined");
        return this._isLastLocation(trace, instIndex, callLocation);
    }
    _isLastLocation(trace, fromStep, location) {
        for (let i = fromStep; i < trace.steps.length; i++) {
            const step = trace.steps[i];
            if (!(0, message_trace_1.isEvmStep)(step)) {
                return false;
            }
            const stepInst = trace.bytecode.getInstruction(step.pc);
            if (stepInst.location === undefined) {
                continue;
            }
            if (!location.equals(stepInst.location)) {
                return false;
            }
        }
        return true;
    }
    _isSubtraceErrorPropagated(trace, callSubtraceStepIndex) {
        const call = trace.steps[callSubtraceStepIndex];
        if (!(0, ethereumjs_util_1.equalsBytes)(trace.returnData, call.returnData)) {
            return false;
        }
        if (trace.exit.kind === exit_1.ExitCode.OUT_OF_GAS &&
            call.exit.kind === exit_1.ExitCode.OUT_OF_GAS) {
            return true;
        }
        // If the return data is not empty, and it's still the same, we assume it
        // is being propagated
        if (trace.returnData.length > 0) {
            return true;
        }
        return this._failsRightAfterCall(trace, callSubtraceStepIndex);
    }
    _isProxyErrorPropagated(trace, callSubtraceStepIndex) {
        if (!(0, message_trace_1.isDecodedCallTrace)(trace)) {
            return false;
        }
        const callStep = trace.steps[callSubtraceStepIndex - 1];
        if (!(0, message_trace_1.isEvmStep)(callStep)) {
            return false;
        }
        const callInst = trace.bytecode.getInstruction(callStep.pc);
        if (callInst.opcode !== opcodes_1.Opcode.DELEGATECALL) {
            return false;
        }
        const subtrace = trace.steps[callSubtraceStepIndex];
        if ((0, message_trace_1.isEvmStep)(subtrace)) {
            return false;
        }
        if ((0, message_trace_1.isPrecompileTrace)(subtrace)) {
            return false;
        }
        // If we can't recognize the implementation we'd better don't consider it as such
        if (subtrace.bytecode === undefined) {
            return false;
        }
        if (subtrace.bytecode.contract.type === model_1.ContractType.LIBRARY) {
            return false;
        }
        if (!(0, ethereumjs_util_1.equalsBytes)(trace.returnData, subtrace.returnData)) {
            return false;
        }
        for (let i = callSubtraceStepIndex + 1; i < trace.steps.length; i++) {
            const step = trace.steps[i];
            if (!(0, message_trace_1.isEvmStep)(step)) {
                return false;
            }
            const inst = trace.bytecode.getInstruction(step.pc);
            // All the remaining locations should be valid, as they are part of the inline asm
            if (inst.location === undefined) {
                return false;
            }
            if (inst.jumpType === model_1.JumpType.INTO_FUNCTION ||
                inst.jumpType === model_1.JumpType.OUTOF_FUNCTION) {
                return false;
            }
        }
        const lastStep = trace.steps[trace.steps.length - 1];
        const lastInst = trace.bytecode.getInstruction(lastStep.pc);
        return lastInst.opcode === opcodes_1.Opcode.REVERT;
    }
    _isContractCallRunOutOfGasError(trace, callStepIndex) {
        if (trace.returnData.length > 0) {
            return false;
        }
        if (trace.exit.kind !== exit_1.ExitCode.REVERT) {
            return false;
        }
        const call = trace.steps[callStepIndex];
        if (call.exit.kind !== exit_1.ExitCode.OUT_OF_GAS) {
            return false;
        }
        return this._failsRightAfterCall(trace, callStepIndex);
    }
    _isPanicReturnData(returnData) {
        return new return_data_1.ReturnData(returnData).isPanicReturnData();
    }
}
exports.ErrorInferrer = ErrorInferrer;
function instructionToCallstackStackTraceEntry(bytecode, inst) {
    // This means that a jump is made from within an internal solc function.
    // These are normally made from yul code, so they don't map to any Solidity
    // function
    if (inst.location === undefined) {
        const location = bytecode.contract.location;
        return {
            type: solidity_stack_trace_1.StackTraceEntryType.INTERNAL_FUNCTION_CALLSTACK_ENTRY,
            pc: inst.pc,
            sourceReference: {
                sourceName: bytecode.contract.location.file.sourceName,
                sourceContent: bytecode.contract.location.file.content,
                contract: bytecode.contract.name,
                function: undefined,
                line: bytecode.contract.location.getStartingLineNumber(),
                range: [location.offset, location.offset + location.length],
            },
        };
    }
    const func = inst.location?.getContainingFunction();
    if (func !== undefined) {
        const sourceReference = sourceLocationToSourceReference(bytecode, inst.location);
        (0, errors_1.assertHardhatInvariant)(sourceReference !== undefined, "Expected source reference to be defined");
        return {
            type: solidity_stack_trace_1.StackTraceEntryType.CALLSTACK_ENTRY,
            sourceReference,
            functionType: func.type,
        };
    }
    (0, errors_1.assertHardhatInvariant)(inst.location !== undefined, "Expected instruction location to be defined");
    return {
        type: solidity_stack_trace_1.StackTraceEntryType.CALLSTACK_ENTRY,
        sourceReference: {
            function: undefined,
            contract: bytecode.contract.name,
            sourceName: inst.location.file.sourceName,
            sourceContent: inst.location.file.content,
            line: inst.location.getStartingLineNumber(),
            range: [
                inst.location.offset,
                inst.location.offset + inst.location.length,
            ],
        },
        functionType: model_1.ContractFunctionType.FUNCTION,
    };
}
exports.instructionToCallstackStackTraceEntry = instructionToCallstackStackTraceEntry;
function sourceLocationToSourceReference(bytecode, location) {
    if (location === undefined) {
        return undefined;
    }
    const func = location.getContainingFunction();
    if (func === undefined) {
        return undefined;
    }
    let funcName = func.name;
    if (func.type === model_1.ContractFunctionType.CONSTRUCTOR) {
        funcName = solidity_stack_trace_1.CONSTRUCTOR_FUNCTION_NAME;
    }
    else if (func.type === model_1.ContractFunctionType.FALLBACK) {
        funcName = solidity_stack_trace_1.FALLBACK_FUNCTION_NAME;
    }
    else if (func.type === model_1.ContractFunctionType.RECEIVE) {
        funcName = solidity_stack_trace_1.RECEIVE_FUNCTION_NAME;
    }
    return {
        function: funcName,
        contract: func.type === model_1.ContractFunctionType.FREE_FUNCTION
            ? undefined
            : bytecode.contract.name,
        sourceName: func.location.file.sourceName,
        sourceContent: func.location.file.content,
        line: location.getStartingLineNumber(),
        range: [location.offset, location.offset + location.length],
    };
}
//# sourceMappingURL=error-inferrer.js.map