import type { ExceptionalHalt, SuccessReason } from "@nomicfoundation/edr";
import type { Address } from "@ethereumjs/util";

/**
 * These types are minimal versions of the values returned by ethereumjs
 * in the event listeners.
 */

export interface MinimalInterpreterStep {
  pc: number;
  depth: number;
  opcode: {
    name: string;
  };
  stack: bigint[];
  memory?: Uint8Array;
}

export interface MinimalExecResult {
  success: boolean;
  executionGasUsed: bigint;
  contractAddress?: Address;
  reason?: SuccessReason | ExceptionalHalt;
  output?: Buffer;
}

export interface MinimalEVMResult {
  execResult: MinimalExecResult;
}

export interface MinimalMessage {
  to?: Address;
  codeAddress?: Address;
  value: bigint;
  data: Uint8Array;
  caller: Address;
  gasLimit: bigint;
  isStaticCall: boolean;
}
