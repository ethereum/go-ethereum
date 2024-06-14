import type { ExceptionalHalt, SuccessReason } from "@nomicfoundation/edr";

import { requireNapiRsModule } from "../../../../common/napi-rs";

export enum ExitCode {
  SUCCESS,
  REVERT,
  OUT_OF_GAS,
  INTERNAL_ERROR,
  INVALID_OPCODE,
  STACK_UNDERFLOW,
  CODESIZE_EXCEEDS_MAXIMUM,
  CREATE_COLLISION,
  STATIC_STATE_CHANGE,
}

export class Exit {
  public static fromEdrSuccessReason(reason: SuccessReason): Exit {
    const { SuccessReason } = requireNapiRsModule(
      "@nomicfoundation/edr"
    ) as typeof import("@nomicfoundation/edr");

    switch (reason) {
      case SuccessReason.Stop:
      case SuccessReason.Return:
      case SuccessReason.SelfDestruct:
        return new Exit(ExitCode.SUCCESS);
    }

    const _exhaustiveCheck: never = reason;
  }

  public static fromEdrExceptionalHalt(halt: ExceptionalHalt): Exit {
    const { ExceptionalHalt } = requireNapiRsModule(
      "@nomicfoundation/edr"
    ) as typeof import("@nomicfoundation/edr");

    switch (halt) {
      case ExceptionalHalt.OutOfGas:
        return new Exit(ExitCode.OUT_OF_GAS);

      case ExceptionalHalt.OpcodeNotFound:
      case ExceptionalHalt.InvalidFEOpcode:
      // Returned when an opcode is not implemented for the hardfork
      case ExceptionalHalt.NotActivated:
        return new Exit(ExitCode.INVALID_OPCODE);

      case ExceptionalHalt.StackUnderflow:
        return new Exit(ExitCode.STACK_UNDERFLOW);

      case ExceptionalHalt.CreateCollision:
        return new Exit(ExitCode.CREATE_COLLISION);

      case ExceptionalHalt.CreateContractSizeLimit:
        return new Exit(ExitCode.CODESIZE_EXCEEDS_MAXIMUM);

      default: {
        // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
        throw new Error(`Unmatched EDR exceptional halt: ${halt}`);
      }
    }
  }

  constructor(public kind: ExitCode) {}

  public isError(): boolean {
    return this.kind !== ExitCode.SUCCESS;
  }

  public getReason(): string {
    switch (this.kind) {
      case ExitCode.SUCCESS:
        return "Success";
      case ExitCode.REVERT:
        return "Reverted";
      case ExitCode.OUT_OF_GAS:
        return "Out of gas";
      case ExitCode.INTERNAL_ERROR:
        return "Internal error";
      case ExitCode.INVALID_OPCODE:
        return "Invalid opcode";
      case ExitCode.STACK_UNDERFLOW:
        return "Stack underflow";
      case ExitCode.CODESIZE_EXCEEDS_MAXIMUM:
        return "Codesize exceeds maximum";
      case ExitCode.CREATE_COLLISION:
        return "Create collision";
      case ExitCode.STATIC_STATE_CHANGE:
        return "Static state change";
    }

    const _exhaustiveCheck: never = this.kind;
  }

  public getEdrExceptionalHalt(): ExceptionalHalt {
    const { ExceptionalHalt } = requireNapiRsModule(
      "@nomicfoundation/edr"
    ) as typeof import("@nomicfoundation/edr");

    switch (this.kind) {
      case ExitCode.OUT_OF_GAS:
        return ExceptionalHalt.OutOfGas;
      case ExitCode.INVALID_OPCODE:
        return ExceptionalHalt.OpcodeNotFound;
      case ExitCode.CODESIZE_EXCEEDS_MAXIMUM:
        return ExceptionalHalt.CreateContractSizeLimit;
      case ExitCode.CREATE_COLLISION:
        return ExceptionalHalt.CreateCollision;

      default:
        // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
        throw new Error(`Unmatched exit code: ${this.kind}`);
    }
  }
}
