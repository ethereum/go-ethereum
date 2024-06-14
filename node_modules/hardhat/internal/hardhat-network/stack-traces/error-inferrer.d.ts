import { DecodedCallMessageTrace, DecodedCreateMessageTrace, DecodedEvmMessageTrace, MessageTrace } from "./message-trace";
import { Bytecode, Instruction } from "./model";
import { CallstackEntryStackTraceEntry, InternalFunctionCallStackEntry, SolidityStackTrace } from "./solidity-stack-trace";
export interface SubmessageData {
    messageTrace: MessageTrace;
    stacktrace: SolidityStackTrace;
    stepIndex: number;
}
export declare class ErrorInferrer {
    inferBeforeTracingCallMessage(trace: DecodedCallMessageTrace): SolidityStackTrace | undefined;
    inferBeforeTracingCreateMessage(trace: DecodedCreateMessageTrace): SolidityStackTrace | undefined;
    inferAfterTracing(trace: DecodedEvmMessageTrace, stacktrace: SolidityStackTrace, functionJumpdests: Instruction[], jumpedIntoFunction: boolean, lastSubmessageData: SubmessageData | undefined): SolidityStackTrace;
    filterRedundantFrames(stacktrace: SolidityStackTrace): SolidityStackTrace;
    /**
     * Check if the last submessage can be used to generate the stack trace.
     */
    private _checkLastSubmessage;
    /**
     * Check if the last call/create that was done failed.
     */
    private _checkFailedLastCall;
    /**
     * Check if the execution stopped with a revert or an invalid opcode.
     */
    private _checkRevertOrInvalidOpcode;
    /**
     * Check if the trace reverted with a panic error.
     */
    private _checkPanic;
    private _checkCustomErrors;
    /**
     * Check last instruction to try to infer the error.
     */
    private _checkLastInstruction;
    private _checkNonContractCalled;
    private _checkSolidity063UnmappedRevert;
    private _checkContractTooLarge;
    private _otherExecutionErrorStacktrace;
    private _fixInitialModifier;
    private _isDirectLibraryCall;
    private _getDirectLibraryCallErrorStackTrace;
    private _isFunctionNotPayableError;
    private _getFunctionStartSourceReference;
    private _isMissingFunctionAndFallbackError;
    private _emptyCalldataAndNoReceive;
    private _getContractStartWithoutFunctionSourceReference;
    private _isFallbackNotPayableError;
    private _getFallbackStartSourceReference;
    private _isConstructorNotPayableError;
    /**
     * Returns a source reference pointing to the constructor if it exists, or to the contract
     * otherwise.
     */
    private _getConstructorStartSourceReference;
    private _isConstructorInvalidArgumentsError;
    private _getEntryBeforeInitialModifierCallstackEntry;
    private _getLastSourceReference;
    private _hasFailedInsideTheFallbackFunction;
    private _hasFailedInsideTheReceiveFunction;
    private _hasFailedInsideFunction;
    private _instructionWithinFunctionToRevertStackTraceEntry;
    private _instructionWithinFunctionToUnmappedSolc063RevertErrorStackTraceEntry;
    private _instructionWithinFunctionToPanicStackTraceEntry;
    private _instructionWithinFunctionToCustomErrorStackTraceEntry;
    private _solidity063MaybeUnmappedRevert;
    private _solidity063GetFrameForUnmappedRevertBeforeFunction;
    private _getOtherErrorBeforeCalledFunctionStackTraceEntry;
    private _isCalledNonContractAccountError;
    private _solidity063GetFrameForUnmappedRevertWithinFunction;
    private _isContractTooLargeError;
    private _solidity063CorrectLineNumber;
    private _getLastInstructionWithValidLocationStepIndex;
    private _getLastInstructionWithValidLocation;
    private _callInstructionToCallFailedToExecuteStackTraceEntry;
    private _getEntryBeforeFailureInModifier;
    private _failsRightAfterCall;
    private _isCallFailedError;
    private _isLastLocation;
    private _isSubtraceErrorPropagated;
    private _isProxyErrorPropagated;
    private _isContractCallRunOutOfGasError;
    private _isPanicReturnData;
}
export declare function instructionToCallstackStackTraceEntry(bytecode: Bytecode, inst: Instruction): CallstackEntryStackTraceEntry | InternalFunctionCallStackEntry;
//# sourceMappingURL=error-inferrer.d.ts.map