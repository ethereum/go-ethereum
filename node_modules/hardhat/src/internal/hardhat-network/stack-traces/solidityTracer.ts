import { equalsBytes } from "@nomicfoundation/ethereumjs-util";

import { ReturnData } from "../provider/return-data";
import { ExitCode } from "../provider/vm/exit";

import {
  ErrorInferrer,
  instructionToCallstackStackTraceEntry,
  SubmessageData,
} from "./error-inferrer";
import {
  adjustStackTrace,
  stackTraceMayRequireAdjustments,
} from "./mapped-inlined-internal-functions-heuristics";
import {
  DecodedCallMessageTrace,
  DecodedCreateMessageTrace,
  DecodedEvmMessageTrace,
  EvmMessageTrace,
  EvmStep,
  isCreateTrace,
  isDecodedCallTrace,
  isDecodedCreateTrace,
  isEvmStep,
  isPrecompileTrace,
  MessageTrace,
  PrecompileMessageTrace,
} from "./message-trace";
import { Instruction, JumpType } from "./model";
import { Opcode } from "./opcodes";
import {
  SolidityStackTrace,
  SolidityStackTraceEntry,
  StackTraceEntryType,
} from "./solidity-stack-trace";

export class SolidityTracer {
  private _errorInferrer = new ErrorInferrer();

  public getStackTrace(
    maybeDecodedMessageTrace: MessageTrace
  ): SolidityStackTrace {
    if (!maybeDecodedMessageTrace.exit.isError()) {
      return [];
    }

    if (isPrecompileTrace(maybeDecodedMessageTrace)) {
      return this._getPrecompileMessageStackTrace(maybeDecodedMessageTrace);
    }

    if (isDecodedCreateTrace(maybeDecodedMessageTrace)) {
      return this._getCreateMessageStackTrace(maybeDecodedMessageTrace);
    }

    if (isDecodedCallTrace(maybeDecodedMessageTrace)) {
      return this._getCallMessageStackTrace(maybeDecodedMessageTrace);
    }

    return this._getUnrecognizedMessageStackTrace(maybeDecodedMessageTrace);
  }

  private _getCallMessageStackTrace(
    trace: DecodedCallMessageTrace
  ): SolidityStackTrace {
    const inferredError =
      this._errorInferrer.inferBeforeTracingCallMessage(trace);

    if (inferredError !== undefined) {
      return inferredError;
    }

    return this._traceEvmExecution(trace);
  }

  private _getUnrecognizedMessageStackTrace(
    trace: EvmMessageTrace
  ): SolidityStackTrace {
    const subtrace = this._getLastSubtrace(trace);

    if (subtrace !== undefined) {
      // This is not a very exact heuristic, but most of the time it will be right, as solidity
      // reverts if a call fails, and most contracts are in solidity
      if (
        subtrace.exit.isError() &&
        equalsBytes(trace.returnData, subtrace.returnData)
      ) {
        let unrecognizedEntry: SolidityStackTraceEntry;

        if (isCreateTrace(trace)) {
          unrecognizedEntry = {
            type: StackTraceEntryType.UNRECOGNIZED_CREATE_CALLSTACK_ENTRY,
          };
        } else {
          unrecognizedEntry = {
            type: StackTraceEntryType.UNRECOGNIZED_CONTRACT_CALLSTACK_ENTRY,
            address: trace.address,
          };
        }

        return [unrecognizedEntry, ...this.getStackTrace(subtrace)];
      }
    }

    if (trace.exit.kind === ExitCode.CODESIZE_EXCEEDS_MAXIMUM) {
      return [
        {
          type: StackTraceEntryType.CONTRACT_TOO_LARGE_ERROR,
        },
      ];
    }

    const isInvalidOpcodeError = trace.exit.kind === ExitCode.INVALID_OPCODE;

    if (isCreateTrace(trace)) {
      return [
        {
          type: StackTraceEntryType.UNRECOGNIZED_CREATE_ERROR,
          message: new ReturnData(trace.returnData),
          isInvalidOpcodeError,
        },
      ];
    }

    return [
      {
        type: StackTraceEntryType.UNRECOGNIZED_CONTRACT_ERROR,
        address: trace.address,
        message: new ReturnData(trace.returnData),
        isInvalidOpcodeError,
      },
    ];
  }

  private _getCreateMessageStackTrace(
    trace: DecodedCreateMessageTrace
  ): SolidityStackTrace {
    const inferredError =
      this._errorInferrer.inferBeforeTracingCreateMessage(trace);

    if (inferredError !== undefined) {
      return inferredError;
    }

    return this._traceEvmExecution(trace);
  }

  private _getPrecompileMessageStackTrace(
    trace: PrecompileMessageTrace
  ): SolidityStackTrace {
    return [
      {
        type: StackTraceEntryType.PRECOMPILE_ERROR,
        precompile: trace.precompile,
      },
    ];
  }

  private _traceEvmExecution(
    trace: DecodedEvmMessageTrace
  ): SolidityStackTrace {
    const stackTrace = this._rawTraceEvmExecution(trace);

    if (stackTraceMayRequireAdjustments(stackTrace, trace)) {
      return adjustStackTrace(stackTrace, trace);
    }

    return stackTrace;
  }

  private _rawTraceEvmExecution(
    trace: DecodedEvmMessageTrace
  ): SolidityStackTrace {
    const stacktrace: SolidityStackTrace = [];

    let subtracesSeen = 0;

    // There was a jump into a function according to the sourcemaps
    let jumpedIntoFunction = false;

    const functionJumpdests: Instruction[] = [];

    let lastSubmessageData: SubmessageData | undefined;

    for (let stepIndex = 0; stepIndex < trace.steps.length; stepIndex++) {
      const step = trace.steps[stepIndex];
      const nextStep = trace.steps[stepIndex + 1];

      if (isEvmStep(step)) {
        const inst = trace.bytecode.getInstruction(step.pc);

        if (
          inst.jumpType === JumpType.INTO_FUNCTION &&
          nextStep !== undefined
        ) {
          const nextEvmStep = nextStep as EvmStep; // A jump can't be followed by a subtrace
          const nextInst = trace.bytecode.getInstruction(nextEvmStep.pc);

          if (nextInst !== undefined && nextInst.opcode === Opcode.JUMPDEST) {
            stacktrace.push(
              instructionToCallstackStackTraceEntry(trace.bytecode, inst)
            );
            if (nextInst.location !== undefined) {
              jumpedIntoFunction = true;
            }
            functionJumpdests.push(nextInst);
          }
        } else if (inst.jumpType === JumpType.OUTOF_FUNCTION) {
          stacktrace.pop();
          functionJumpdests.pop();
        }
      } else {
        subtracesSeen += 1;

        // If there are more subtraces, this one didn't terminate the execution
        if (subtracesSeen < trace.numberOfSubtraces) {
          continue;
        }

        const submessageTrace = this.getStackTrace(step);

        lastSubmessageData = {
          messageTrace: step,
          stepIndex,
          stacktrace: submessageTrace,
        };
      }
    }

    const stacktraceWithInferredError = this._errorInferrer.inferAfterTracing(
      trace,
      stacktrace,
      functionJumpdests,
      jumpedIntoFunction,
      lastSubmessageData
    );

    return this._errorInferrer.filterRedundantFrames(
      stacktraceWithInferredError
    );
  }

  private _getLastSubtrace(trace: EvmMessageTrace): MessageTrace | undefined {
    if (trace.numberOfSubtraces < 1) {
      return undefined;
    }

    let i = trace.steps.length - 1;

    while (isEvmStep(trace.steps[i])) {
      i -= 1;
    }

    return trace.steps[i] as MessageTrace;
  }
}
