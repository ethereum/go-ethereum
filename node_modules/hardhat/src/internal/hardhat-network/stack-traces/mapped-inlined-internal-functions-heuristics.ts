/**
 * This file includes Solidity tracing heuristics for solc starting with version
 * 0.6.9.
 *
 * This solc version introduced a significant change to how sourcemaps are
 * handled for inline yul/internal functions. These were mapped to the
 * unmapped/-1 file before, which lead to many unmapped reverts. Now, they are
 * mapped to the part of the Solidity source that lead to their inlining.
 *
 * This change is a very positive change, as errors would point to the correct
 * line by default. The only problem is that we used to rely very heavily on
 * unmapped reverts to decide when our error detection heuristics were to be
 * run. In fact, this heuristics were first introduced because of unmapped
 * reverts.
 *
 * Instead of synthetically completing stack traces when unmapped reverts occur,
 * we now start from complete stack traces and adjust them if we can provide
 * more meaningful errors.
 */

import semver from "semver";

import {
  DecodedEvmMessageTrace,
  isDecodedCallTrace,
  isDecodedCreateTrace,
  isEvmStep,
} from "./message-trace";
import { Opcode } from "./opcodes";
import {
  SolidityStackTrace,
  StackTraceEntryType,
} from "./solidity-stack-trace";

const FIRST_SOLC_VERSION_WITH_MAPPED_SMALL_INTERNAL_FUNCTIONS = "0.6.9";

export function stackTraceMayRequireAdjustments(
  stackTrace: SolidityStackTrace,
  decodedTrace: DecodedEvmMessageTrace
): boolean {
  if (stackTrace.length === 0) {
    return false;
  }

  const lastFrame = stackTrace[stackTrace.length - 1];

  return (
    lastFrame.type === StackTraceEntryType.REVERT_ERROR &&
    !lastFrame.isInvalidOpcodeError &&
    lastFrame.message.isEmpty() &&
    semver.gte(
      decodedTrace.bytecode.compilerVersion,
      FIRST_SOLC_VERSION_WITH_MAPPED_SMALL_INTERNAL_FUNCTIONS
    )
  );
}

export function adjustStackTrace(
  stackTrace: SolidityStackTrace,
  decodedTrace: DecodedEvmMessageTrace
): SolidityStackTrace {
  const start = stackTrace.slice(0, -1);
  const [revert] = stackTrace.slice(-1);

  if (isNonContractAccountCalledError(decodedTrace)) {
    return [
      ...start,
      {
        type: StackTraceEntryType.NONCONTRACT_ACCOUNT_CALLED_ERROR,
        sourceReference: revert.sourceReference!,
      },
    ];
  }

  if (isConstructorInvalidParamsError(decodedTrace)) {
    return [
      ...start,
      {
        type: StackTraceEntryType.INVALID_PARAMS_ERROR,
        sourceReference: revert.sourceReference!,
      },
    ];
  }

  if (isCallInvalidParamsError(decodedTrace)) {
    return [
      ...start,
      {
        type: StackTraceEntryType.INVALID_PARAMS_ERROR,
        sourceReference: revert.sourceReference!,
      },
    ];
  }

  return stackTrace;
}

function isNonContractAccountCalledError(
  decodedTrace: DecodedEvmMessageTrace
): boolean {
  return matchOpcodes(decodedTrace, -9, [
    Opcode.EXTCODESIZE,
    Opcode.ISZERO,
    Opcode.DUP1,
    Opcode.ISZERO,
  ]);
}

function isConstructorInvalidParamsError(decodedTrace: DecodedEvmMessageTrace) {
  if (!isDecodedCreateTrace(decodedTrace)) {
    return false;
  }

  return (
    matchOpcodes(decodedTrace, -20, [Opcode.CODESIZE]) &&
    matchOpcodes(decodedTrace, -15, [Opcode.CODECOPY]) &&
    matchOpcodes(decodedTrace, -7, [Opcode.LT, Opcode.ISZERO])
  );
}

function isCallInvalidParamsError(decodedTrace: DecodedEvmMessageTrace) {
  if (!isDecodedCallTrace(decodedTrace)) {
    return false;
  }

  return (
    matchOpcodes(decodedTrace, -11, [Opcode.CALLDATASIZE]) &&
    matchOpcodes(decodedTrace, -7, [Opcode.LT, Opcode.ISZERO])
  );
}

function matchOpcode(
  decodedTrace: DecodedEvmMessageTrace,
  stepIndex: number,
  opcode: Opcode
): boolean {
  const [step] = decodedTrace.steps.slice(stepIndex, stepIndex + 1);

  if (step === undefined || !isEvmStep(step)) {
    return false;
  }

  const instruction = decodedTrace.bytecode.getInstruction(step.pc);

  return instruction.opcode === opcode;
}

function matchOpcodes(
  decodedTrace: DecodedEvmMessageTrace,
  firstStepIndex: number,
  opcodes: Opcode[]
): boolean {
  let index = firstStepIndex;
  for (const opcode of opcodes) {
    if (!matchOpcode(decodedTrace, index, opcode)) {
      return false;
    }

    index += 1;
  }

  return true;
}
