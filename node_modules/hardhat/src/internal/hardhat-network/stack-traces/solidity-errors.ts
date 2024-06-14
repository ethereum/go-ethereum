import { bytesToHex as bufferToHex } from "@nomicfoundation/ethereumjs-util";

import { panicErrorCodeToMessage } from "./panic-errors";
import {
  CONSTRUCTOR_FUNCTION_NAME,
  PRECOMPILE_FUNCTION_NAME,
  SolidityStackTrace,
  SolidityStackTraceEntry,
  SourceReference,
  StackTraceEntryType,
  UNKNOWN_FUNCTION_NAME,
  UNRECOGNIZED_CONTRACT_NAME,
  UNRECOGNIZED_FUNCTION_NAME,
} from "./solidity-stack-trace";

const inspect = Symbol.for("nodejs.util.inspect.custom");

export function getCurrentStack(): NodeJS.CallSite[] {
  const previousPrepareStackTrace = Error.prepareStackTrace;

  Error.prepareStackTrace = (e, s) => s;

  const error = new Error();
  const stack: NodeJS.CallSite[] = error.stack as any;

  Error.prepareStackTrace = previousPrepareStackTrace;

  return stack;
}

export async function wrapWithSolidityErrorsCorrection(
  f: () => Promise<any>,
  stackFramesToRemove: number
) {
  const stackTraceAtCall = getCurrentStack().slice(stackFramesToRemove);

  try {
    return await f();
  } catch (error: any) {
    if (error.stackTrace === undefined) {
      // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
      throw error;
    }

    // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
    throw encodeSolidityStackTrace(
      error.message,
      error.stackTrace,
      stackTraceAtCall
    );
  }
}

export function encodeSolidityStackTrace(
  fallbackMessage: string,
  stackTrace: SolidityStackTrace,
  previousStack?: NodeJS.CallSite[]
): SolidityError {
  if (Error.prepareStackTrace === undefined) {
    // Node 12 doesn't have a default Error.prepareStackTrace
    require("source-map-support/register");
  }

  const previousPrepareStackTrace = Error.prepareStackTrace;
  Error.prepareStackTrace = (error, stack) => {
    if (previousStack !== undefined) {
      stack = previousStack;
    } else {
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

    return previousPrepareStackTrace!(error, stack);
  };

  const msg = getMessageFromLastStackTraceEntry(
    stackTrace[stackTrace.length - 1]
  );

  const solidityError = new SolidityError(
    msg !== undefined ? msg : fallbackMessage,
    stackTrace
  );

  // This hack is here because prepare stack is lazy
  solidityError.stack = solidityError.stack;

  Error.prepareStackTrace = previousPrepareStackTrace;

  return solidityError;
}

function encodeStackTraceEntry(
  stackTraceEntry: SolidityStackTraceEntry
): SolidityCallSite {
  switch (stackTraceEntry.type) {
    case StackTraceEntryType.UNRECOGNIZED_FUNCTION_WITHOUT_FALLBACK_ERROR:
    case StackTraceEntryType.MISSING_FALLBACK_OR_RECEIVE_ERROR:
      return sourceReferenceToSolidityCallsite({
        ...stackTraceEntry.sourceReference,
        function: UNRECOGNIZED_FUNCTION_NAME,
      });

    case StackTraceEntryType.CALLSTACK_ENTRY:
    case StackTraceEntryType.REVERT_ERROR:
    case StackTraceEntryType.CUSTOM_ERROR:
    case StackTraceEntryType.FUNCTION_NOT_PAYABLE_ERROR:
    case StackTraceEntryType.INVALID_PARAMS_ERROR:
    case StackTraceEntryType.FALLBACK_NOT_PAYABLE_ERROR:
    case StackTraceEntryType.FALLBACK_NOT_PAYABLE_AND_NO_RECEIVE_ERROR:
    case StackTraceEntryType.RETURNDATA_SIZE_ERROR:
    case StackTraceEntryType.NONCONTRACT_ACCOUNT_CALLED_ERROR:
    case StackTraceEntryType.CALL_FAILED_ERROR:
    case StackTraceEntryType.DIRECT_LIBRARY_CALL_ERROR:
      return sourceReferenceToSolidityCallsite(stackTraceEntry.sourceReference);

    case StackTraceEntryType.UNRECOGNIZED_CREATE_CALLSTACK_ENTRY:
      return new SolidityCallSite(
        undefined,
        UNRECOGNIZED_CONTRACT_NAME,
        CONSTRUCTOR_FUNCTION_NAME,
        undefined
      );

    case StackTraceEntryType.UNRECOGNIZED_CONTRACT_CALLSTACK_ENTRY:
      return new SolidityCallSite(
        bufferToHex(stackTraceEntry.address),
        UNRECOGNIZED_CONTRACT_NAME,
        UNKNOWN_FUNCTION_NAME,
        undefined
      );

    case StackTraceEntryType.PRECOMPILE_ERROR:
      return new SolidityCallSite(
        undefined,
        `<PrecompileContract ${stackTraceEntry.precompile}>`,
        PRECOMPILE_FUNCTION_NAME,
        undefined
      );

    case StackTraceEntryType.UNRECOGNIZED_CREATE_ERROR:
      return new SolidityCallSite(
        undefined,
        UNRECOGNIZED_CONTRACT_NAME,
        CONSTRUCTOR_FUNCTION_NAME,
        undefined
      );

    case StackTraceEntryType.UNRECOGNIZED_CONTRACT_ERROR:
      return new SolidityCallSite(
        bufferToHex(stackTraceEntry.address),
        UNRECOGNIZED_CONTRACT_NAME,
        UNKNOWN_FUNCTION_NAME,
        undefined
      );

    case StackTraceEntryType.INTERNAL_FUNCTION_CALLSTACK_ENTRY:
      return new SolidityCallSite(
        stackTraceEntry.sourceReference.sourceName,
        stackTraceEntry.sourceReference.contract,
        `internal@${stackTraceEntry.pc}`,
        undefined
      );
    case StackTraceEntryType.CONTRACT_CALL_RUN_OUT_OF_GAS_ERROR:
      if (stackTraceEntry.sourceReference !== undefined) {
        return sourceReferenceToSolidityCallsite(
          stackTraceEntry.sourceReference
        );
      }

      return new SolidityCallSite(
        undefined,
        UNRECOGNIZED_CONTRACT_NAME,
        UNKNOWN_FUNCTION_NAME,
        undefined
      );

    case StackTraceEntryType.OTHER_EXECUTION_ERROR:
    case StackTraceEntryType.CONTRACT_TOO_LARGE_ERROR:
    case StackTraceEntryType.PANIC_ERROR:
    case StackTraceEntryType.UNMAPPED_SOLC_0_6_3_REVERT_ERROR:
      if (stackTraceEntry.sourceReference === undefined) {
        return new SolidityCallSite(
          undefined,
          UNRECOGNIZED_CONTRACT_NAME,
          UNKNOWN_FUNCTION_NAME,
          undefined
        );
      }

      return sourceReferenceToSolidityCallsite(stackTraceEntry.sourceReference);
  }
}

function sourceReferenceToSolidityCallsite(
  sourceReference: SourceReference
): SolidityCallSite {
  return new SolidityCallSite(
    sourceReference.sourceName,
    sourceReference.contract,
    sourceReference.function !== undefined
      ? sourceReference.function
      : UNKNOWN_FUNCTION_NAME,
    sourceReference.line
  );
}

function getMessageFromLastStackTraceEntry(
  stackTraceEntry: SolidityStackTraceEntry
): string | undefined {
  switch (stackTraceEntry.type) {
    case StackTraceEntryType.PRECOMPILE_ERROR:
      return `Transaction reverted: call to precompile ${stackTraceEntry.precompile} failed`;

    case StackTraceEntryType.FUNCTION_NOT_PAYABLE_ERROR:
      return `Transaction reverted: non-payable function was called with value ${stackTraceEntry.value.toString(
        10
      )}`;

    case StackTraceEntryType.INVALID_PARAMS_ERROR:
      return `Transaction reverted: function was called with incorrect parameters`;

    case StackTraceEntryType.FALLBACK_NOT_PAYABLE_ERROR:
      return `Transaction reverted: fallback function is not payable and was called with value ${stackTraceEntry.value.toString(
        10
      )}`;

    case StackTraceEntryType.FALLBACK_NOT_PAYABLE_AND_NO_RECEIVE_ERROR:
      return `Transaction reverted: there's no receive function, fallback function is not payable and was called with value ${stackTraceEntry.value.toString(
        10
      )}`;

    case StackTraceEntryType.UNRECOGNIZED_FUNCTION_WITHOUT_FALLBACK_ERROR:
      return `Transaction reverted: function selector was not recognized and there's no fallback function`;

    case StackTraceEntryType.MISSING_FALLBACK_OR_RECEIVE_ERROR:
      return `Transaction reverted: function selector was not recognized and there's no fallback nor receive function`;

    case StackTraceEntryType.RETURNDATA_SIZE_ERROR:
      return `Transaction reverted: function returned an unexpected amount of data`;

    case StackTraceEntryType.NONCONTRACT_ACCOUNT_CALLED_ERROR:
      return `Transaction reverted: function call to a non-contract account`;

    case StackTraceEntryType.CALL_FAILED_ERROR:
      return `Transaction reverted: function call failed to execute`;

    case StackTraceEntryType.DIRECT_LIBRARY_CALL_ERROR:
      return `Transaction reverted: library was called directly`;

    case StackTraceEntryType.UNRECOGNIZED_CREATE_ERROR:
    case StackTraceEntryType.UNRECOGNIZED_CONTRACT_ERROR:
      if (stackTraceEntry.message.isErrorReturnData()) {
        return `VM Exception while processing transaction: reverted with reason string '${stackTraceEntry.message.decodeError()}'`;
      }

      if (stackTraceEntry.message.isPanicReturnData()) {
        const message = panicErrorCodeToMessage(
          stackTraceEntry.message.decodePanic()
        );
        return `VM Exception while processing transaction: ${message}`;
      }

      if (!stackTraceEntry.message.isEmpty()) {
        const returnData = Buffer.from(stackTraceEntry.message.value).toString(
          "hex"
        );

        return `VM Exception while processing transaction: reverted with an unrecognized custom error (return data: 0x${returnData})`;
      }

      if (stackTraceEntry.isInvalidOpcodeError) {
        return "VM Exception while processing transaction: invalid opcode";
      }

      return "Transaction reverted without a reason string";

    case StackTraceEntryType.REVERT_ERROR:
      if (stackTraceEntry.message.isErrorReturnData()) {
        return `VM Exception while processing transaction: reverted with reason string '${stackTraceEntry.message.decodeError()}'`;
      }

      if (stackTraceEntry.isInvalidOpcodeError) {
        return "VM Exception while processing transaction: invalid opcode";
      }

      return "Transaction reverted without a reason string";

    case StackTraceEntryType.PANIC_ERROR:
      const panicMessage = panicErrorCodeToMessage(stackTraceEntry.errorCode);
      return `VM Exception while processing transaction: ${panicMessage}`;

    case StackTraceEntryType.CUSTOM_ERROR:
      return `VM Exception while processing transaction: ${stackTraceEntry.message}`;

    case StackTraceEntryType.OTHER_EXECUTION_ERROR:
      // TODO: What if there was returnData?
      return `Transaction reverted and Hardhat couldn't infer the reason.`;

    case StackTraceEntryType.UNMAPPED_SOLC_0_6_3_REVERT_ERROR:
      return "Transaction reverted without a reason string and without a valid sourcemap provided by the compiler. Some line numbers may be off. We strongly recommend upgrading solc and always using revert reasons.";

    case StackTraceEntryType.CONTRACT_TOO_LARGE_ERROR:
      return "Transaction reverted: trying to deploy a contract whose code is too large";

    case StackTraceEntryType.CONTRACT_CALL_RUN_OUT_OF_GAS_ERROR:
      return "Transaction reverted: contract call run out of gas and made the transaction revert";
  }
}

// Note: This error class MUST NOT extend ProviderError, as libraries
//   use the code property to detect if they are dealing with a JSON-RPC error,
//   and take control of errors.
export class SolidityError extends Error {
  public readonly stackTrace: SolidityStackTrace;

  constructor(message: string, stackTrace: SolidityStackTrace) {
    super(message);
    this.stackTrace = stackTrace;
  }

  public [inspect](): string {
    return this.inspect();
  }

  public inspect(): string {
    return this.stack !== undefined
      ? this.stack
      : "Internal error when encoding SolidityError";
  }
}

class SolidityCallSite implements NodeJS.CallSite {
  constructor(
    private _sourceName: string | undefined,
    private _contract: string | undefined,
    private _functionName: string | undefined,
    private _line: number | undefined
  ) {}

  public getColumnNumber() {
    return null;
  }

  public getEvalOrigin() {
    return undefined;
  }

  public getFileName() {
    return this._sourceName ?? "unknown";
  }

  public getFunction() {
    return undefined;
  }

  public getFunctionName() {
    // if it's a top-level function, we print its name
    if (this._contract === undefined) {
      return this._functionName ?? null;
    }

    return null;
  }

  public getLineNumber() {
    return this._line !== undefined ? this._line : null;
  }

  public getMethodName() {
    if (this._contract !== undefined) {
      return this._functionName ?? null;
    }

    return null;
  }

  public getPosition() {
    return 0;
  }

  public getPromiseIndex() {
    return 0;
  }

  public getScriptNameOrSourceURL() {
    return "";
  }

  public getThis() {
    return undefined;
  }

  public getTypeName() {
    return this._contract ?? null;
  }

  public isAsync() {
    return false;
  }

  public isConstructor() {
    return false;
  }

  public isEval() {
    return false;
  }

  public isNative() {
    return false;
  }

  public isPromiseAll() {
    return false;
  }

  public isToplevel() {
    return false;
  }

  public getScriptHash(): string {
    return "";
  }

  public getEnclosingColumnNumber(): number {
    return 0;
  }

  public getEnclosingLineNumber(): number {
    return 0;
  }

  public toString(): string {
    return "[SolidityCallSite]";
  }
}
