/* eslint "@typescript-eslint/no-non-null-assertion": "error" */
import { defaultAbiCoder as abi } from "@ethersproject/abi";
import { equalsBytes } from "@nomicfoundation/ethereumjs-util";
import semver from "semver";

import { assertHardhatInvariant } from "../../core/errors";
import { AbiHelpers } from "../../util/abi-helpers";
import { ReturnData } from "../provider/return-data";
import { ExitCode } from "../provider/vm/exit";

import {
  DecodedCallMessageTrace,
  DecodedCreateMessageTrace,
  DecodedEvmMessageTrace,
  EvmStep,
  isCreateTrace,
  isDecodedCallTrace,
  isDecodedCreateTrace,
  isEvmStep,
  isPrecompileTrace,
  MessageTrace,
} from "./message-trace";
import {
  Bytecode,
  ContractFunction,
  ContractFunctionType,
  ContractType,
  Instruction,
  JumpType,
  SourceLocation,
} from "./model";
import { isCall, isCreate, Opcode } from "./opcodes";
import {
  CallFailedErrorStackTraceEntry,
  CallstackEntryStackTraceEntry,
  CONSTRUCTOR_FUNCTION_NAME,
  CustomErrorStackTraceEntry,
  FALLBACK_FUNCTION_NAME,
  InternalFunctionCallStackEntry,
  OtherExecutionErrorStackTraceEntry,
  PanicErrorStackTraceEntry,
  RECEIVE_FUNCTION_NAME,
  RevertErrorStackTraceEntry,
  SolidityStackTrace,
  SolidityStackTraceEntry,
  SourceReference,
  StackTraceEntryType,
  UnmappedSolc063RevertErrorStackTraceEntry,
} from "./solidity-stack-trace";

const FIRST_SOLC_VERSION_CREATE_PARAMS_VALIDATION = "0.5.9";
const FIRST_SOLC_VERSION_RECEIVE_FUNCTION = "0.6.0";
const FIRST_SOLC_VERSION_WITH_UNMAPPED_REVERTS = "0.6.3";

export interface SubmessageData {
  messageTrace: MessageTrace;
  stacktrace: SolidityStackTrace;
  stepIndex: number;
}

/* eslint-disable @nomicfoundation/hardhat-internal-rules/only-hardhat-error */

export class ErrorInferrer {
  public inferBeforeTracingCallMessage(
    trace: DecodedCallMessageTrace
  ): SolidityStackTrace | undefined {
    if (this._isDirectLibraryCall(trace)) {
      return this._getDirectLibraryCallErrorStackTrace(trace);
    }

    const calledFunction = trace.bytecode.contract.getFunctionFromSelector(
      trace.calldata.slice(0, 4)
    );

    if (
      calledFunction !== undefined &&
      this._isFunctionNotPayableError(trace, calledFunction)
    ) {
      return [
        {
          type: StackTraceEntryType.FUNCTION_NOT_PAYABLE_ERROR,
          sourceReference: this._getFunctionStartSourceReference(
            trace,
            calledFunction
          ),
          value: trace.value,
        },
      ];
    }

    if (this._isMissingFunctionAndFallbackError(trace, calledFunction)) {
      if (this._emptyCalldataAndNoReceive(trace)) {
        return [
          {
            type: StackTraceEntryType.MISSING_FALLBACK_OR_RECEIVE_ERROR,
            sourceReference:
              this._getContractStartWithoutFunctionSourceReference(trace),
          },
        ];
      }

      return [
        {
          type: StackTraceEntryType.UNRECOGNIZED_FUNCTION_WITHOUT_FALLBACK_ERROR,
          sourceReference:
            this._getContractStartWithoutFunctionSourceReference(trace),
        },
      ];
    }

    if (this._isFallbackNotPayableError(trace, calledFunction)) {
      if (this._emptyCalldataAndNoReceive(trace)) {
        return [
          {
            type: StackTraceEntryType.FALLBACK_NOT_PAYABLE_AND_NO_RECEIVE_ERROR,
            sourceReference: this._getFallbackStartSourceReference(trace),
            value: trace.value,
          },
        ];
      }

      return [
        {
          type: StackTraceEntryType.FALLBACK_NOT_PAYABLE_ERROR,
          sourceReference: this._getFallbackStartSourceReference(trace),
          value: trace.value,
        },
      ];
    }
  }

  public inferBeforeTracingCreateMessage(
    trace: DecodedCreateMessageTrace
  ): SolidityStackTrace | undefined {
    if (this._isConstructorNotPayableError(trace)) {
      return [
        {
          type: StackTraceEntryType.FUNCTION_NOT_PAYABLE_ERROR,
          sourceReference: this._getConstructorStartSourceReference(trace),
          value: trace.value,
        },
      ];
    }

    if (this._isConstructorInvalidArgumentsError(trace)) {
      return [
        {
          type: StackTraceEntryType.INVALID_PARAMS_ERROR,
          sourceReference: this._getConstructorStartSourceReference(trace),
        },
      ];
    }
  }

  public inferAfterTracing(
    trace: DecodedEvmMessageTrace,
    stacktrace: SolidityStackTrace,
    functionJumpdests: Instruction[],
    jumpedIntoFunction: boolean,
    lastSubmessageData: SubmessageData | undefined
  ): SolidityStackTrace {
    return (
      this._checkLastSubmessage(trace, stacktrace, lastSubmessageData) ??
      this._checkFailedLastCall(trace, stacktrace) ??
      this._checkLastInstruction(
        trace,
        stacktrace,
        functionJumpdests,
        jumpedIntoFunction
      ) ??
      this._checkNonContractCalled(trace, stacktrace) ??
      this._checkSolidity063UnmappedRevert(trace, stacktrace) ??
      this._checkContractTooLarge(trace) ??
      this._otherExecutionErrorStacktrace(trace, stacktrace)
    );
  }

  public filterRedundantFrames(
    stacktrace: SolidityStackTrace
  ): SolidityStackTrace {
    return stacktrace.filter((frame, i) => {
      if (i + 1 === stacktrace.length) {
        return true;
      }

      const nextFrame = stacktrace[i + 1];

      // we can only filter frames if we know their sourceReference
      // and the one from the next frame
      if (
        frame.sourceReference === undefined ||
        nextFrame.sourceReference === undefined
      ) {
        return true;
      }

      // look TWO frames ahead to determine if this is a specific occurrence of
      // a redundant CALLSTACK_ENTRY frame observed when using Solidity 0.8.5:
      if (
        frame.type === StackTraceEntryType.CALLSTACK_ENTRY &&
        i + 2 < stacktrace.length &&
        stacktrace[i + 2].sourceReference !== undefined &&
        stacktrace[i + 2].type === StackTraceEntryType.RETURNDATA_SIZE_ERROR
      ) {
        // ! below for tsc. we confirmed existence in the enclosing conditional.
        const thatSrcRef = stacktrace[i + 2].sourceReference;
        if (
          thatSrcRef !== undefined &&
          frame.sourceReference.range[0] === thatSrcRef.range[0] &&
          frame.sourceReference.range[1] === thatSrcRef.range[1] &&
          frame.sourceReference.line === thatSrcRef.line
        ) {
          return false;
        }
      }

      // constructors contain the whole contract, so we ignore them
      if (
        frame.sourceReference.function === "constructor" &&
        nextFrame.sourceReference.function !== "constructor"
      ) {
        return true;
      }

      // this is probably a recursive call
      if (
        i > 0 &&
        frame.type === nextFrame.type &&
        frame.sourceReference.range[0] === nextFrame.sourceReference.range[0] &&
        frame.sourceReference.range[1] === nextFrame.sourceReference.range[1] &&
        frame.sourceReference.line === nextFrame.sourceReference.line
      ) {
        return true;
      }

      if (
        frame.sourceReference.range[0] <= nextFrame.sourceReference.range[0] &&
        frame.sourceReference.range[1] >= nextFrame.sourceReference.range[1]
      ) {
        return false;
      }

      return true;
    });
  }

  // Heuristics

  /**
   * Check if the last submessage can be used to generate the stack trace.
   */
  private _checkLastSubmessage(
    trace: DecodedEvmMessageTrace,
    stacktrace: SolidityStackTrace,
    lastSubmessageData: SubmessageData | undefined
  ): SolidityStackTrace | undefined {
    if (lastSubmessageData === undefined) {
      return undefined;
    }

    const inferredStacktrace = [...stacktrace];

    // get the instruction before the submessage and add it to the stack trace
    const callStep = trace.steps[lastSubmessageData.stepIndex - 1];

    if (!isEvmStep(callStep)) {
      throw new Error(
        "This should not happen: MessageTrace should be preceded by a EVM step"
      );
    }

    const callInst = trace.bytecode.getInstruction(callStep.pc);
    const callStackFrame = instructionToCallstackStackTraceEntry(
      trace.bytecode,
      callInst
    );

    const lastMessageFailed = lastSubmessageData.messageTrace.exit.isError();
    if (lastMessageFailed) {
      // add the call/create that generated the message to the stack trace
      inferredStacktrace.push(callStackFrame);

      if (
        this._isSubtraceErrorPropagated(trace, lastSubmessageData.stepIndex) ||
        this._isProxyErrorPropagated(trace, lastSubmessageData.stepIndex)
      ) {
        inferredStacktrace.push(...lastSubmessageData.stacktrace);

        if (
          this._isContractCallRunOutOfGasError(
            trace,
            lastSubmessageData.stepIndex
          )
        ) {
          const lastFrame = inferredStacktrace.pop();
          assertHardhatInvariant(
            lastFrame !== undefined,
            "Expected inferred stack trace to have at least one frame"
          );
          inferredStacktrace.push({
            type: StackTraceEntryType.CONTRACT_CALL_RUN_OUT_OF_GAS_ERROR,
            sourceReference: lastFrame.sourceReference,
          });
        }

        return this._fixInitialModifier(trace, inferredStacktrace);
      }
    } else {
      const isReturnDataSizeError = this._failsRightAfterCall(
        trace,
        lastSubmessageData.stepIndex
      );
      if (isReturnDataSizeError) {
        inferredStacktrace.push({
          type: StackTraceEntryType.RETURNDATA_SIZE_ERROR,
          sourceReference: callStackFrame.sourceReference,
        });

        return this._fixInitialModifier(trace, inferredStacktrace);
      }
    }
  }

  /**
   * Check if the last call/create that was done failed.
   */
  private _checkFailedLastCall(
    trace: DecodedEvmMessageTrace,
    stacktrace: SolidityStackTrace
  ): SolidityStackTrace | undefined {
    for (let stepIndex = trace.steps.length - 2; stepIndex >= 0; stepIndex--) {
      const step = trace.steps[stepIndex];
      const nextStep = trace.steps[stepIndex + 1];

      if (!isEvmStep(step)) {
        return;
      }

      const inst = trace.bytecode.getInstruction(step.pc);

      const isCallOrCreate = isCall(inst.opcode) || isCreate(inst.opcode);

      if (isCallOrCreate && isEvmStep(nextStep)) {
        if (this._isCallFailedError(trace, stepIndex, inst)) {
          const inferredStacktrace = [
            ...stacktrace,
            this._callInstructionToCallFailedToExecuteStackTraceEntry(
              trace.bytecode,
              inst
            ),
          ];

          return this._fixInitialModifier(trace, inferredStacktrace);
        }
      }
    }
  }

  /**
   * Check if the execution stopped with a revert or an invalid opcode.
   */
  private _checkRevertOrInvalidOpcode(
    trace: DecodedEvmMessageTrace,
    stacktrace: SolidityStackTrace,
    lastInstruction: Instruction,
    functionJumpdests: Instruction[],
    jumpedIntoFunction: boolean
  ): SolidityStackTrace | undefined {
    if (
      lastInstruction.opcode !== Opcode.REVERT &&
      lastInstruction.opcode !== Opcode.INVALID
    ) {
      return;
    }

    const inferredStacktrace = [...stacktrace];

    if (
      lastInstruction.location !== undefined &&
      (!isDecodedCallTrace(trace) || jumpedIntoFunction)
    ) {
      // There should always be a function here, but that's not the case with optimizations.
      //
      // If this is a create trace, we already checked args and nonpayable failures before
      // calling this function.
      //
      // If it's a call trace, we already jumped into a function. But optimizations can happen.
      const failingFunction = lastInstruction.location.getContainingFunction();

      // If the failure is in a modifier we add an entry with the function/constructor
      if (
        failingFunction !== undefined &&
        failingFunction.type === ContractFunctionType.MODIFIER
      ) {
        inferredStacktrace.push(
          this._getEntryBeforeFailureInModifier(trace, functionJumpdests)
        );
      }
    }

    const panicStacktrace = this._checkPanic(
      trace,
      inferredStacktrace,
      lastInstruction
    );
    if (panicStacktrace !== undefined) {
      return panicStacktrace;
    }

    const customErrorStacktrace = this._checkCustomErrors(
      trace,
      inferredStacktrace,
      lastInstruction
    );
    if (customErrorStacktrace !== undefined) {
      return customErrorStacktrace;
    }

    if (
      lastInstruction.location !== undefined &&
      (!isDecodedCallTrace(trace) || jumpedIntoFunction)
    ) {
      const failingFunction = lastInstruction.location.getContainingFunction();

      if (failingFunction !== undefined) {
        inferredStacktrace.push(
          this._instructionWithinFunctionToRevertStackTraceEntry(
            trace,
            lastInstruction
          )
        );
      } else if (isDecodedCallTrace(trace)) {
        // This is here because of the optimizations
        const functionSelector =
          trace.bytecode.contract.getFunctionFromSelector(
            trace.calldata.slice(0, 4)
          );

        // in general this shouldn't happen, but it does when viaIR is enabled,
        // "optimizerSteps": "u" is used, and the called function is fallback or
        // receive
        if (functionSelector === undefined) {
          return;
        }

        inferredStacktrace.push({
          type: StackTraceEntryType.REVERT_ERROR,
          sourceReference: this._getFunctionStartSourceReference(
            trace,
            functionSelector
          ),
          message: new ReturnData(trace.returnData),
          isInvalidOpcodeError: lastInstruction.opcode === Opcode.INVALID,
        });
      } else {
        // This is here because of the optimizations
        inferredStacktrace.push({
          type: StackTraceEntryType.REVERT_ERROR,
          sourceReference: this._getConstructorStartSourceReference(trace),
          message: new ReturnData(trace.returnData),
          isInvalidOpcodeError: lastInstruction.opcode === Opcode.INVALID,
        });
      }

      return this._fixInitialModifier(trace, inferredStacktrace);
    }

    // If the revert instruction is not mapped but there is return data,
    // we add the frame anyway, sith the best sourceReference we can get
    if (lastInstruction.location === undefined && trace.returnData.length > 0) {
      const revertFrame: RevertErrorStackTraceEntry = {
        type: StackTraceEntryType.REVERT_ERROR,
        sourceReference:
          this._getLastSourceReference(trace) ??
          this._getContractStartWithoutFunctionSourceReference(trace),
        message: new ReturnData(trace.returnData),
        isInvalidOpcodeError: lastInstruction.opcode === Opcode.INVALID,
      };
      inferredStacktrace.push(revertFrame);

      return this._fixInitialModifier(trace, inferredStacktrace);
    }
  }

  /**
   * Check if the trace reverted with a panic error.
   */
  private _checkPanic(
    trace: DecodedEvmMessageTrace,
    stacktrace: SolidityStackTrace,
    lastInstruction: Instruction
  ): SolidityStackTrace | undefined {
    if (!this._isPanicReturnData(trace.returnData)) {
      return;
    }

    // If the last frame is an internal function, it means that the trace
    // jumped there to return the panic. If that's the case, we remove that
    // frame.
    const lastFrame = stacktrace[stacktrace.length - 1];
    if (
      lastFrame?.type === StackTraceEntryType.INTERNAL_FUNCTION_CALLSTACK_ENTRY
    ) {
      stacktrace.splice(-1);
    }

    const panicReturnData = new ReturnData(trace.returnData);
    const errorCode = panicReturnData.decodePanic();

    // if the error comes from a call to a zero-initialized function,
    // we remove the last frame, which represents the call, to avoid
    // having duplicated frames
    if (errorCode === 0x51n) {
      stacktrace.splice(-1);
    }

    const inferredStacktrace = [...stacktrace];
    inferredStacktrace.push(
      this._instructionWithinFunctionToPanicStackTraceEntry(
        trace,
        lastInstruction,
        errorCode
      )
    );

    return this._fixInitialModifier(trace, inferredStacktrace);
  }

  private _checkCustomErrors(
    trace: DecodedEvmMessageTrace,
    stacktrace: SolidityStackTrace,
    lastInstruction: Instruction
  ): SolidityStackTrace | undefined {
    const returnData = new ReturnData(trace.returnData);

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
        const decodedValues = abi.decode(
          customError.paramTypes,
          returnData.value.slice(4)
        );

        const params = AbiHelpers.formatValues([...decodedValues]);
        errorMessage = `reverted with custom error '${customError.name}(${params})'`;
        break;
      }
    }

    const inferredStacktrace = [...stacktrace];
    inferredStacktrace.push(
      this._instructionWithinFunctionToCustomErrorStackTraceEntry(
        trace,
        lastInstruction,
        errorMessage
      )
    );

    return this._fixInitialModifier(trace, inferredStacktrace);
  }

  /**
   * Check last instruction to try to infer the error.
   */
  private _checkLastInstruction(
    trace: DecodedEvmMessageTrace,
    stacktrace: SolidityStackTrace,
    functionJumpdests: Instruction[],
    jumpedIntoFunction: boolean
  ): SolidityStackTrace | undefined {
    if (trace.steps.length === 0) {
      return;
    }

    const lastStep = trace.steps[trace.steps.length - 1];

    if (!isEvmStep(lastStep)) {
      throw new Error(
        "This should not happen: MessageTrace ends with a subtrace"
      );
    }

    const lastInstruction = trace.bytecode.getInstruction(lastStep.pc);

    const revertOrInvalidStacktrace = this._checkRevertOrInvalidOpcode(
      trace,
      stacktrace,
      lastInstruction,
      functionJumpdests,
      jumpedIntoFunction
    );

    if (revertOrInvalidStacktrace !== undefined) {
      return revertOrInvalidStacktrace;
    }

    if (isDecodedCallTrace(trace) && !jumpedIntoFunction) {
      if (
        this._hasFailedInsideTheFallbackFunction(trace) ||
        this._hasFailedInsideTheReceiveFunction(trace)
      ) {
        return [
          this._instructionWithinFunctionToRevertStackTraceEntry(
            trace,
            lastInstruction
          ),
        ];
      }

      // Sometimes we do fail inside of a function but there's no jump into
      if (lastInstruction.location !== undefined) {
        const failingFunction =
          lastInstruction.location.getContainingFunction();
        if (failingFunction !== undefined) {
          return [
            {
              type: StackTraceEntryType.REVERT_ERROR,
              sourceReference: this._getFunctionStartSourceReference(
                trace,
                failingFunction
              ),
              message: new ReturnData(trace.returnData),
              isInvalidOpcodeError: lastInstruction.opcode === Opcode.INVALID,
            },
          ];
        }
      }

      const calledFunction = trace.bytecode.contract.getFunctionFromSelector(
        trace.calldata.slice(0, 4)
      );

      if (calledFunction !== undefined) {
        const isValidCalldata = calledFunction.isValidCalldata(
          trace.calldata.slice(4)
        );

        if (!isValidCalldata) {
          return [
            {
              type: StackTraceEntryType.INVALID_PARAMS_ERROR,
              sourceReference: this._getFunctionStartSourceReference(
                trace,
                calledFunction
              ),
            },
          ];
        }
      }

      if (this._solidity063MaybeUnmappedRevert(trace)) {
        const revertFrame =
          this._solidity063GetFrameForUnmappedRevertBeforeFunction(trace);

        if (revertFrame !== undefined) {
          return [revertFrame];
        }
      }

      return [this._getOtherErrorBeforeCalledFunctionStackTraceEntry(trace)];
    }
  }

  private _checkNonContractCalled(
    trace: DecodedEvmMessageTrace,
    stacktrace: SolidityStackTrace
  ): SolidityStackTrace | undefined {
    if (this._isCalledNonContractAccountError(trace)) {
      const sourceReference = this._getLastSourceReference(trace);

      // We are sure this is not undefined because there was at least a call instruction
      assertHardhatInvariant(
        sourceReference !== undefined,
        "Expected source reference to be defined"
      );

      const nonContractCalledFrame: SolidityStackTraceEntry = {
        type: StackTraceEntryType.NONCONTRACT_ACCOUNT_CALLED_ERROR,
        sourceReference,
      };

      return [...stacktrace, nonContractCalledFrame];
    }
  }

  private _checkSolidity063UnmappedRevert(
    trace: DecodedEvmMessageTrace,
    stacktrace: SolidityStackTrace
  ): SolidityStackTrace | undefined {
    if (this._solidity063MaybeUnmappedRevert(trace)) {
      const revertFrame =
        this._solidity063GetFrameForUnmappedRevertWithinFunction(trace);

      if (revertFrame !== undefined) {
        return [...stacktrace, revertFrame];
      }
    }
  }

  private _checkContractTooLarge(
    trace: DecodedEvmMessageTrace
  ): SolidityStackTrace | undefined {
    if (isCreateTrace(trace) && this._isContractTooLargeError(trace)) {
      return [
        {
          type: StackTraceEntryType.CONTRACT_TOO_LARGE_ERROR,
          sourceReference: this._getConstructorStartSourceReference(trace),
        },
      ];
    }
  }

  private _otherExecutionErrorStacktrace(
    trace: DecodedEvmMessageTrace,
    stacktrace: SolidityStackTrace
  ): SolidityStackTrace {
    const otherExecutionErrorFrame: SolidityStackTraceEntry = {
      type: StackTraceEntryType.OTHER_EXECUTION_ERROR,
      sourceReference: this._getLastSourceReference(trace),
    };

    return [...stacktrace, otherExecutionErrorFrame];
  }

  // Helpers

  private _fixInitialModifier(
    trace: DecodedEvmMessageTrace,
    stacktrace: SolidityStackTrace
  ): SolidityStackTrace {
    const firstEntry = stacktrace[0];
    if (
      firstEntry !== undefined &&
      firstEntry.type === StackTraceEntryType.CALLSTACK_ENTRY &&
      firstEntry.functionType === ContractFunctionType.MODIFIER
    ) {
      return [
        this._getEntryBeforeInitialModifierCallstackEntry(trace),
        ...stacktrace,
      ];
    }

    return stacktrace;
  }

  private _isDirectLibraryCall(trace: DecodedCallMessageTrace): boolean {
    return (
      trace.depth === 0 && trace.bytecode.contract.type === ContractType.LIBRARY
    );
  }

  private _getDirectLibraryCallErrorStackTrace(
    trace: DecodedCallMessageTrace
  ): SolidityStackTrace {
    const func = trace.bytecode.contract.getFunctionFromSelector(
      trace.calldata.slice(0, 4)
    );

    if (func !== undefined) {
      return [
        {
          type: StackTraceEntryType.DIRECT_LIBRARY_CALL_ERROR,
          sourceReference: this._getFunctionStartSourceReference(trace, func),
        },
      ];
    }

    return [
      {
        type: StackTraceEntryType.DIRECT_LIBRARY_CALL_ERROR,
        sourceReference:
          this._getContractStartWithoutFunctionSourceReference(trace),
      },
    ];
  }

  private _isFunctionNotPayableError(
    trace: DecodedCallMessageTrace,
    calledFunction: ContractFunction
  ): boolean {
    // This error doesn't return data
    if (trace.returnData.length > 0) {
      return false;
    }

    if (trace.value <= 0n) {
      return false;
    }

    // Libraries don't have a nonpayable check
    if (trace.bytecode.contract.type === ContractType.LIBRARY) {
      return false;
    }

    return calledFunction.isPayable === undefined || !calledFunction.isPayable;
  }

  private _getFunctionStartSourceReference(
    trace: DecodedEvmMessageTrace,
    func: ContractFunction
  ): SourceReference {
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

  private _isMissingFunctionAndFallbackError(
    trace: DecodedCallMessageTrace,
    calledFunction: ContractFunction | undefined
  ): boolean {
    // This error doesn't return data
    if (trace.returnData.length > 0) {
      return false;
    }

    // the called function exists in the contract
    if (calledFunction !== undefined) {
      return false;
    }

    // there's a receive function and no calldata
    if (
      trace.calldata.length === 0 &&
      trace.bytecode.contract.receive !== undefined
    ) {
      return false;
    }

    return trace.bytecode.contract.fallback === undefined;
  }

  private _emptyCalldataAndNoReceive(trace: DecodedCallMessageTrace): boolean {
    // this only makes sense when receive functions are available
    if (
      semver.lt(
        trace.bytecode.compilerVersion,
        FIRST_SOLC_VERSION_RECEIVE_FUNCTION
      )
    ) {
      return false;
    }

    return (
      trace.calldata.length === 0 &&
      trace.bytecode.contract.receive === undefined
    );
  }

  private _getContractStartWithoutFunctionSourceReference(
    trace: DecodedEvmMessageTrace
  ): SourceReference {
    const location = trace.bytecode.contract.location;
    return {
      sourceName: location.file.sourceName,
      sourceContent: location.file.content,
      contract: trace.bytecode.contract.name,
      line: location.getStartingLineNumber(),
      range: [location.offset, location.offset + location.length],
    };
  }

  private _isFallbackNotPayableError(
    trace: DecodedCallMessageTrace,
    calledFunction: ContractFunction | undefined
  ): boolean {
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

  private _getFallbackStartSourceReference(
    trace: DecodedCallMessageTrace
  ): SourceReference {
    const func = trace.bytecode.contract.fallback;

    if (func === undefined) {
      throw new Error(
        "This shouldn't happen: trying to get fallback source reference from a contract without fallback"
      );
    }

    return {
      sourceName: func.location.file.sourceName,
      sourceContent: func.location.file.content,
      contract: trace.bytecode.contract.name,
      function: FALLBACK_FUNCTION_NAME,
      line: func.location.getStartingLineNumber(),
      range: [
        func.location.offset,
        func.location.offset + func.location.length,
      ],
    };
  }

  private _isConstructorNotPayableError(
    trace: DecodedCreateMessageTrace
  ): boolean {
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

    return (
      trace.value > 0n &&
      (constructor.isPayable === undefined || !constructor.isPayable)
    );
  }

  /**
   * Returns a source reference pointing to the constructor if it exists, or to the contract
   * otherwise.
   */
  private _getConstructorStartSourceReference(
    trace: DecodedCreateMessageTrace
  ): SourceReference {
    const contract = trace.bytecode.contract;
    const constructor = contract.constructorFunction;

    const line =
      constructor !== undefined
        ? constructor.location.getStartingLineNumber()
        : contract.location.getStartingLineNumber();

    return {
      sourceName: contract.location.file.sourceName,
      sourceContent: contract.location.file.content,
      contract: contract.name,
      function: CONSTRUCTOR_FUNCTION_NAME,
      line,
      range: [
        contract.location.offset,
        contract.location.offset + contract.location.length,
      ],
    };
  }

  private _isConstructorInvalidArgumentsError(
    trace: DecodedCreateMessageTrace
  ): boolean {
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

    if (
      semver.lt(
        trace.bytecode.compilerVersion,
        FIRST_SOLC_VERSION_CREATE_PARAMS_VALIDATION
      )
    ) {
      return false;
    }

    const lastStep = trace.steps[trace.steps.length - 1];
    if (!isEvmStep(lastStep)) {
      return false;
    }

    const lastInst = trace.bytecode.getInstruction(lastStep.pc);
    if (lastInst.opcode !== Opcode.REVERT || lastInst.location !== undefined) {
      return false;
    }

    let hasReadDeploymentCodeSize = false;

    // eslint-disable-next-line @typescript-eslint/prefer-for-of
    for (let stepIndex = 0; stepIndex < trace.steps.length; stepIndex++) {
      const step = trace.steps[stepIndex];
      if (!isEvmStep(step)) {
        return false;
      }

      const inst = trace.bytecode.getInstruction(step.pc);

      if (
        inst.location !== undefined &&
        !contract.location.equals(inst.location) &&
        !constructor.location.equals(inst.location)
      ) {
        return false;
      }

      if (inst.opcode === Opcode.CODESIZE && isCreateTrace(trace)) {
        hasReadDeploymentCodeSize = true;
      }
    }

    return hasReadDeploymentCodeSize;
  }

  private _getEntryBeforeInitialModifierCallstackEntry(
    trace: DecodedEvmMessageTrace
  ): SolidityStackTraceEntry {
    if (isDecodedCreateTrace(trace)) {
      return {
        type: StackTraceEntryType.CALLSTACK_ENTRY,
        sourceReference: this._getConstructorStartSourceReference(trace),
        functionType: ContractFunctionType.CONSTRUCTOR,
      };
    }

    const calledFunction = trace.bytecode.contract.getFunctionFromSelector(
      trace.calldata.slice(0, 4)
    );

    if (calledFunction !== undefined) {
      return {
        type: StackTraceEntryType.CALLSTACK_ENTRY,
        sourceReference: this._getFunctionStartSourceReference(
          trace,
          calledFunction
        ),
        functionType: ContractFunctionType.FUNCTION,
      };
    }

    // If it failed or made a call from within a modifier, and the selector doesn't match
    // any function, it must have a fallback.
    return {
      type: StackTraceEntryType.CALLSTACK_ENTRY,
      sourceReference: this._getFallbackStartSourceReference(trace),
      functionType: ContractFunctionType.FALLBACK,
    };
  }

  private _getLastSourceReference(
    trace: DecodedEvmMessageTrace
  ): SourceReference | undefined {
    for (let i = trace.steps.length - 1; i >= 0; i--) {
      const step = trace.steps[i];
      if (!isEvmStep(step)) {
        continue;
      }

      const inst = trace.bytecode.getInstruction(step.pc);

      if (inst.location === undefined) {
        continue;
      }

      const sourceReference = sourceLocationToSourceReference(
        trace.bytecode,
        inst.location
      );

      if (sourceReference !== undefined) {
        return sourceReference;
      }
    }

    return undefined;
  }

  private _hasFailedInsideTheFallbackFunction(
    trace: DecodedCallMessageTrace
  ): boolean {
    const contract = trace.bytecode.contract;

    if (contract.fallback === undefined) {
      return false;
    }

    return this._hasFailedInsideFunction(trace, contract.fallback);
  }

  private _hasFailedInsideTheReceiveFunction(
    trace: DecodedCallMessageTrace
  ): boolean {
    const contract = trace.bytecode.contract;

    if (contract.receive === undefined) {
      return false;
    }

    return this._hasFailedInsideFunction(trace, contract.receive);
  }

  private _hasFailedInsideFunction(
    trace: DecodedCallMessageTrace,
    func: ContractFunction
  ) {
    const lastStep = trace.steps[trace.steps.length - 1] as EvmStep;
    const lastInstruction = trace.bytecode.getInstruction(lastStep.pc);

    return (
      lastInstruction.location !== undefined &&
      lastInstruction.opcode === Opcode.REVERT &&
      func.location.contains(lastInstruction.location)
    );
  }

  private _instructionWithinFunctionToRevertStackTraceEntry(
    trace: DecodedEvmMessageTrace,
    inst: Instruction
  ): RevertErrorStackTraceEntry {
    const sourceReference = sourceLocationToSourceReference(
      trace.bytecode,
      inst.location
    );
    assertHardhatInvariant(
      sourceReference !== undefined,
      "Expected source reference to be defined"
    );

    return {
      type: StackTraceEntryType.REVERT_ERROR,
      sourceReference,
      message: new ReturnData(trace.returnData),
      isInvalidOpcodeError: inst.opcode === Opcode.INVALID,
    };
  }

  private _instructionWithinFunctionToUnmappedSolc063RevertErrorStackTraceEntry(
    trace: DecodedEvmMessageTrace,
    inst: Instruction
  ): UnmappedSolc063RevertErrorStackTraceEntry {
    const sourceReference = sourceLocationToSourceReference(
      trace.bytecode,
      inst.location
    );

    return {
      type: StackTraceEntryType.UNMAPPED_SOLC_0_6_3_REVERT_ERROR,
      sourceReference,
    };
  }

  private _instructionWithinFunctionToPanicStackTraceEntry(
    trace: DecodedEvmMessageTrace,
    inst: Instruction,
    errorCode: bigint
  ): PanicErrorStackTraceEntry {
    const lastSourceReference = this._getLastSourceReference(trace);
    return {
      type: StackTraceEntryType.PANIC_ERROR,
      sourceReference:
        sourceLocationToSourceReference(trace.bytecode, inst.location) ??
        lastSourceReference,
      errorCode,
    };
  }

  private _instructionWithinFunctionToCustomErrorStackTraceEntry(
    trace: DecodedEvmMessageTrace,
    inst: Instruction,
    message: string
  ): CustomErrorStackTraceEntry {
    const lastSourceReference = this._getLastSourceReference(trace);

    assertHardhatInvariant(
      lastSourceReference !== undefined,
      "Expected last source reference to be defined"
    );

    return {
      type: StackTraceEntryType.CUSTOM_ERROR,
      sourceReference:
        sourceLocationToSourceReference(trace.bytecode, inst.location) ??
        lastSourceReference,
      message,
    };
  }

  private _solidity063MaybeUnmappedRevert(trace: DecodedEvmMessageTrace) {
    if (trace.steps.length === 0) {
      return false;
    }

    const lastStep = trace.steps[trace.steps.length - 1];
    if (!isEvmStep(lastStep)) {
      return false;
    }

    const lastInst = trace.bytecode.getInstruction(lastStep.pc);

    return (
      semver.satisfies(
        trace.bytecode.compilerVersion,
        `^${FIRST_SOLC_VERSION_WITH_UNMAPPED_REVERTS}`
      ) && lastInst.opcode === Opcode.REVERT
    );
  }

  // Solidity 0.6.3 unmapped reverts special handling
  // For more info: https://github.com/ethereum/solidity/issues/9006
  private _solidity063GetFrameForUnmappedRevertBeforeFunction(
    trace: DecodedCallMessageTrace
  ) {
    let revertFrame =
      this._solidity063GetFrameForUnmappedRevertWithinFunction(trace);

    if (
      revertFrame === undefined ||
      revertFrame.sourceReference === undefined
    ) {
      if (
        trace.bytecode.contract.receive === undefined ||
        trace.calldata.length > 0
      ) {
        if (trace.bytecode.contract.fallback !== undefined) {
          // Failed within the fallback
          const location = trace.bytecode.contract.fallback.location;
          revertFrame = {
            type: StackTraceEntryType.UNMAPPED_SOLC_0_6_3_REVERT_ERROR,
            sourceReference: {
              contract: trace.bytecode.contract.name,
              function: FALLBACK_FUNCTION_NAME,
              sourceName: location.file.sourceName,
              sourceContent: location.file.content,
              line: location.getStartingLineNumber(),
              range: [location.offset, location.offset + location.length],
            },
          };

          this._solidity063CorrectLineNumber(revertFrame);
        }
      } else {
        // Failed within the receive function
        const location = trace.bytecode.contract.receive.location;
        revertFrame = {
          type: StackTraceEntryType.UNMAPPED_SOLC_0_6_3_REVERT_ERROR,
          sourceReference: {
            contract: trace.bytecode.contract.name,
            function: RECEIVE_FUNCTION_NAME,
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

  private _getOtherErrorBeforeCalledFunctionStackTraceEntry(
    trace: DecodedCallMessageTrace
  ): OtherExecutionErrorStackTraceEntry {
    return {
      type: StackTraceEntryType.OTHER_EXECUTION_ERROR,
      sourceReference:
        this._getContractStartWithoutFunctionSourceReference(trace),
    };
  }

  private _isCalledNonContractAccountError(
    trace: DecodedEvmMessageTrace
  ): boolean {
    // We could change this to checking that the last valid location maps to a call, but
    // it's way more complex as we need to get the ast node from that location.

    const lastIndex = this._getLastInstructionWithValidLocationStepIndex(trace);
    if (lastIndex === undefined || lastIndex === 0) {
      return false;
    }

    const lastStep = trace.steps[lastIndex] as EvmStep; // We know this is an EVM step
    const lastInst = trace.bytecode.getInstruction(lastStep.pc);
    if (lastInst.opcode !== Opcode.ISZERO) {
      return false;
    }

    const prevStep = trace.steps[lastIndex - 1] as EvmStep; // We know this is an EVM step
    const prevInst = trace.bytecode.getInstruction(prevStep.pc);
    return prevInst.opcode === Opcode.EXTCODESIZE;
  }

  private _solidity063GetFrameForUnmappedRevertWithinFunction(
    trace: DecodedEvmMessageTrace
  ): UnmappedSolc063RevertErrorStackTraceEntry | undefined {
    // If we are within a function there's a last valid location. It may
    // be the entire contract.
    const prevInst = this._getLastInstructionWithValidLocation(trace);
    const lastStep = trace.steps[trace.steps.length - 1] as EvmStep;
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
      if (
        prevFunc !== undefined &&
        nextLoc !== undefined &&
        prevLoc !== undefined &&
        prevLoc.equals(nextLoc)
      ) {
        return this._instructionWithinFunctionToUnmappedSolc063RevertErrorStackTraceEntry(
          trace,
          nextInst
        );
      }

      let revertFrame: UnmappedSolc063RevertErrorStackTraceEntry | undefined;

      // If the previous and next location don't match, we try to use the
      // previous one if it's inside a function, otherwise we use the next one
      if (prevFunc !== undefined && prevInst !== undefined) {
        revertFrame =
          this._instructionWithinFunctionToUnmappedSolc063RevertErrorStackTraceEntry(
            trace,
            prevInst
          );
      } else if (nextFunc !== undefined) {
        revertFrame =
          this._instructionWithinFunctionToUnmappedSolc063RevertErrorStackTraceEntry(
            trace,
            nextInst
          );
      }

      if (revertFrame !== undefined) {
        this._solidity063CorrectLineNumber(revertFrame);
      }

      return revertFrame;
    }

    if (isCreateTrace(trace) && prevInst !== undefined) {
      // Solidity is smart enough to stop emitting extra instructions after
      // an unconditional revert happens in a constructor. If this is the case
      // we just return a special error.
      const constructorRevertFrame: UnmappedSolc063RevertErrorStackTraceEntry =
        this._instructionWithinFunctionToUnmappedSolc063RevertErrorStackTraceEntry(
          trace,
          prevInst
        );

      // When the latest instruction is not within a function we need
      // some default sourceReference to show to the user
      if (constructorRevertFrame.sourceReference === undefined) {
        const location = trace.bytecode.contract.location;
        const defaultSourceReference: SourceReference = {
          function: CONSTRUCTOR_FUNCTION_NAME,
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
      } else {
        this._solidity063CorrectLineNumber(constructorRevertFrame);
      }

      return constructorRevertFrame;
    }

    if (prevInst !== undefined) {
      // We may as well just be in a function or modifier and just happen
      // to be at the last instruction of the runtime bytecode.
      // In this case we just return whatever the last mapped intruction
      // points to.
      const latestInstructionRevertFrame: UnmappedSolc063RevertErrorStackTraceEntry =
        this._instructionWithinFunctionToUnmappedSolc063RevertErrorStackTraceEntry(
          trace,
          prevInst
        );

      if (latestInstructionRevertFrame.sourceReference !== undefined) {
        this._solidity063CorrectLineNumber(latestInstructionRevertFrame);
      }
      return latestInstructionRevertFrame;
    }
  }

  private _isContractTooLargeError(trace: DecodedCreateMessageTrace) {
    return trace.exit.kind === ExitCode.CODESIZE_EXCEEDS_MAXIMUM;
  }

  private _solidity063CorrectLineNumber(
    revertFrame: UnmappedSolc063RevertErrorStackTraceEntry
  ) {
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

  private _getLastInstructionWithValidLocationStepIndex(
    trace: DecodedEvmMessageTrace
  ): number | undefined {
    for (let i = trace.steps.length - 1; i >= 0; i--) {
      const step = trace.steps[i];

      if (!isEvmStep(step)) {
        return undefined;
      }

      const inst = trace.bytecode.getInstruction(step.pc);

      if (inst.location !== undefined) {
        return i;
      }
    }

    return undefined;
  }

  private _getLastInstructionWithValidLocation(
    trace: DecodedEvmMessageTrace
  ): Instruction | undefined {
    const lastLocationIndex =
      this._getLastInstructionWithValidLocationStepIndex(trace);

    if (lastLocationIndex === undefined) {
      return undefined;
    }

    const lastLocationStep = trace.steps[lastLocationIndex];
    if (isEvmStep(lastLocationStep)) {
      const lastInstructionWithLocation = trace.bytecode.getInstruction(
        lastLocationStep.pc
      );
      return lastInstructionWithLocation;
    }

    return undefined;
  }

  private _callInstructionToCallFailedToExecuteStackTraceEntry(
    bytecode: Bytecode,
    callInst: Instruction
  ): CallFailedErrorStackTraceEntry {
    const sourceReference = sourceLocationToSourceReference(
      bytecode,
      callInst.location
    );
    assertHardhatInvariant(
      sourceReference !== undefined,
      "Expected source reference to be defined"
    );

    // Calls only happen within functions
    return {
      type: StackTraceEntryType.CALL_FAILED_ERROR,
      sourceReference,
    };
  }

  private _getEntryBeforeFailureInModifier(
    trace: DecodedEvmMessageTrace,
    functionJumpdests: Instruction[]
  ): CallstackEntryStackTraceEntry | InternalFunctionCallStackEntry {
    // If there's a jumpdest, this modifier belongs to the last function that it represents
    if (functionJumpdests.length > 0) {
      return instructionToCallstackStackTraceEntry(
        trace.bytecode,
        functionJumpdests[functionJumpdests.length - 1]
      );
    }

    // This function is only called after we jumped into the initial function in call traces, so
    // there should always be at least a function jumpdest.
    if (!isDecodedCreateTrace(trace)) {
      throw new Error(
        "This shouldn't happen: a call trace has no functionJumpdest but has already jumped into a function"
      );
    }

    // If there's no jump dest, we point to the constructor.
    return {
      type: StackTraceEntryType.CALLSTACK_ENTRY,
      sourceReference: this._getConstructorStartSourceReference(trace),
      functionType: ContractFunctionType.CONSTRUCTOR,
    };
  }

  private _failsRightAfterCall(
    trace: DecodedEvmMessageTrace,
    callSubtraceStepIndex: number
  ): boolean {
    const lastStep = trace.steps[trace.steps.length - 1];
    if (!isEvmStep(lastStep)) {
      return false;
    }

    const lastInst = trace.bytecode.getInstruction(lastStep.pc);
    if (lastInst.opcode !== Opcode.REVERT) {
      return false;
    }

    const callOpcodeStep = trace.steps[callSubtraceStepIndex - 1] as EvmStep;
    const callInst = trace.bytecode.getInstruction(callOpcodeStep.pc);

    // Calls are always made from within functions
    assertHardhatInvariant(
      callInst.location !== undefined,
      "Expected call instruction location to be defined"
    );

    return this._isLastLocation(
      trace,
      callSubtraceStepIndex + 1,
      callInst.location
    );
  }

  private _isCallFailedError(
    trace: DecodedEvmMessageTrace,
    instIndex: number,
    callInstruction: Instruction
  ): boolean {
    const callLocation = callInstruction.location;

    // Calls are always made from within functions
    assertHardhatInvariant(
      callLocation !== undefined,
      "Expected call location to be defined"
    );

    return this._isLastLocation(trace, instIndex, callLocation);
  }

  private _isLastLocation(
    trace: DecodedEvmMessageTrace,
    fromStep: number,
    location: SourceLocation
  ): boolean {
    for (let i = fromStep; i < trace.steps.length; i++) {
      const step = trace.steps[i];

      if (!isEvmStep(step)) {
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

  private _isSubtraceErrorPropagated(
    trace: DecodedEvmMessageTrace,
    callSubtraceStepIndex: number
  ): boolean {
    const call = trace.steps[callSubtraceStepIndex] as MessageTrace;

    if (!equalsBytes(trace.returnData, call.returnData)) {
      return false;
    }

    if (
      trace.exit.kind === ExitCode.OUT_OF_GAS &&
      call.exit.kind === ExitCode.OUT_OF_GAS
    ) {
      return true;
    }

    // If the return data is not empty, and it's still the same, we assume it
    // is being propagated
    if (trace.returnData.length > 0) {
      return true;
    }

    return this._failsRightAfterCall(trace, callSubtraceStepIndex);
  }

  private _isProxyErrorPropagated(
    trace: DecodedEvmMessageTrace,
    callSubtraceStepIndex: number
  ): boolean {
    if (!isDecodedCallTrace(trace)) {
      return false;
    }

    const callStep = trace.steps[callSubtraceStepIndex - 1];
    if (!isEvmStep(callStep)) {
      return false;
    }

    const callInst = trace.bytecode.getInstruction(callStep.pc);
    if (callInst.opcode !== Opcode.DELEGATECALL) {
      return false;
    }

    const subtrace = trace.steps[callSubtraceStepIndex];
    if (isEvmStep(subtrace)) {
      return false;
    }

    if (isPrecompileTrace(subtrace)) {
      return false;
    }

    // If we can't recognize the implementation we'd better don't consider it as such
    if (subtrace.bytecode === undefined) {
      return false;
    }

    if (subtrace.bytecode.contract.type === ContractType.LIBRARY) {
      return false;
    }

    if (!equalsBytes(trace.returnData, subtrace.returnData)) {
      return false;
    }

    for (let i = callSubtraceStepIndex + 1; i < trace.steps.length; i++) {
      const step = trace.steps[i];
      if (!isEvmStep(step)) {
        return false;
      }

      const inst = trace.bytecode.getInstruction(step.pc);

      // All the remaining locations should be valid, as they are part of the inline asm
      if (inst.location === undefined) {
        return false;
      }

      if (
        inst.jumpType === JumpType.INTO_FUNCTION ||
        inst.jumpType === JumpType.OUTOF_FUNCTION
      ) {
        return false;
      }
    }

    const lastStep = trace.steps[trace.steps.length - 1] as EvmStep;
    const lastInst = trace.bytecode.getInstruction(lastStep.pc);

    return lastInst.opcode === Opcode.REVERT;
  }

  private _isContractCallRunOutOfGasError(
    trace: DecodedEvmMessageTrace,
    callStepIndex: number
  ): boolean {
    if (trace.returnData.length > 0) {
      return false;
    }

    if (trace.exit.kind !== ExitCode.REVERT) {
      return false;
    }

    const call = trace.steps[callStepIndex] as MessageTrace;
    if (call.exit.kind !== ExitCode.OUT_OF_GAS) {
      return false;
    }

    return this._failsRightAfterCall(trace, callStepIndex);
  }

  private _isPanicReturnData(returnData: Uint8Array): boolean {
    return new ReturnData(returnData).isPanicReturnData();
  }
}

export function instructionToCallstackStackTraceEntry(
  bytecode: Bytecode,
  inst: Instruction
): CallstackEntryStackTraceEntry | InternalFunctionCallStackEntry {
  // This means that a jump is made from within an internal solc function.
  // These are normally made from yul code, so they don't map to any Solidity
  // function
  if (inst.location === undefined) {
    const location = bytecode.contract.location;
    return {
      type: StackTraceEntryType.INTERNAL_FUNCTION_CALLSTACK_ENTRY,
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
    const sourceReference = sourceLocationToSourceReference(
      bytecode,
      inst.location
    );
    assertHardhatInvariant(
      sourceReference !== undefined,
      "Expected source reference to be defined"
    );

    return {
      type: StackTraceEntryType.CALLSTACK_ENTRY,
      sourceReference,
      functionType: func.type,
    };
  }

  assertHardhatInvariant(
    inst.location !== undefined,
    "Expected instruction location to be defined"
  );

  return {
    type: StackTraceEntryType.CALLSTACK_ENTRY,
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
    functionType: ContractFunctionType.FUNCTION,
  };
}

function sourceLocationToSourceReference(
  bytecode: Bytecode,
  location?: SourceLocation
): SourceReference | undefined {
  if (location === undefined) {
    return undefined;
  }

  const func = location.getContainingFunction();

  if (func === undefined) {
    return undefined;
  }

  let funcName = func.name;

  if (func.type === ContractFunctionType.CONSTRUCTOR) {
    funcName = CONSTRUCTOR_FUNCTION_NAME;
  } else if (func.type === ContractFunctionType.FALLBACK) {
    funcName = FALLBACK_FUNCTION_NAME;
  } else if (func.type === ContractFunctionType.RECEIVE) {
    funcName = RECEIVE_FUNCTION_NAME;
  }

  return {
    function: funcName,
    contract:
      func.type === ContractFunctionType.FREE_FUNCTION
        ? undefined
        : bytecode.contract.name,
    sourceName: func.location.file.sourceName,
    sourceContent: func.location.file.content,
    line: location.getStartingLineNumber(),
    range: [location.offset, location.offset + location.length],
  };
}
