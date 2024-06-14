import { ReturnData } from "../provider/return-data";

import { ContractFunctionType } from "./model";

export enum StackTraceEntryType {
  CALLSTACK_ENTRY,
  UNRECOGNIZED_CREATE_CALLSTACK_ENTRY,
  UNRECOGNIZED_CONTRACT_CALLSTACK_ENTRY,
  PRECOMPILE_ERROR,
  REVERT_ERROR,
  PANIC_ERROR,
  CUSTOM_ERROR,
  FUNCTION_NOT_PAYABLE_ERROR,
  INVALID_PARAMS_ERROR,
  FALLBACK_NOT_PAYABLE_ERROR,
  FALLBACK_NOT_PAYABLE_AND_NO_RECEIVE_ERROR,
  UNRECOGNIZED_FUNCTION_WITHOUT_FALLBACK_ERROR, // TODO: Should trying to call a private/internal be a special case of this?
  MISSING_FALLBACK_OR_RECEIVE_ERROR,
  RETURNDATA_SIZE_ERROR,
  NONCONTRACT_ACCOUNT_CALLED_ERROR,
  CALL_FAILED_ERROR,
  DIRECT_LIBRARY_CALL_ERROR,
  UNRECOGNIZED_CREATE_ERROR,
  UNRECOGNIZED_CONTRACT_ERROR,
  OTHER_EXECUTION_ERROR,
  // This is a special case to handle a regression introduced in solc 0.6.3
  // For more info: https://github.com/ethereum/solidity/issues/9006
  UNMAPPED_SOLC_0_6_3_REVERT_ERROR,
  CONTRACT_TOO_LARGE_ERROR,
  INTERNAL_FUNCTION_CALLSTACK_ENTRY,
  CONTRACT_CALL_RUN_OUT_OF_GAS_ERROR,
}

export const FALLBACK_FUNCTION_NAME = "<fallback>";
export const RECEIVE_FUNCTION_NAME = "<receive>";
export const CONSTRUCTOR_FUNCTION_NAME = "constructor";
export const UNRECOGNIZED_FUNCTION_NAME = "<unrecognized-selector>";
export const UNKNOWN_FUNCTION_NAME = "<unknown>";
export const PRECOMPILE_FUNCTION_NAME = "<precompile>";
export const UNRECOGNIZED_CONTRACT_NAME = "<UnrecognizedContract>";

export interface SourceReference {
  sourceName: string;
  sourceContent: string;
  contract?: string;
  function?: string;
  line: number;
  range: [number, number];
}

export interface CallstackEntryStackTraceEntry {
  type: StackTraceEntryType.CALLSTACK_ENTRY;
  sourceReference: SourceReference;
  functionType: ContractFunctionType;
}

export interface UnrecognizedCreateCallstackEntryStackTraceEntry {
  type: StackTraceEntryType.UNRECOGNIZED_CREATE_CALLSTACK_ENTRY;
  sourceReference?: undefined;
}

export interface UnrecognizedContractCallstackEntryStackTraceEntry {
  type: StackTraceEntryType.UNRECOGNIZED_CONTRACT_CALLSTACK_ENTRY;
  address: Uint8Array;
  sourceReference?: undefined;
}

export interface PrecompileErrorStackTraceEntry {
  type: StackTraceEntryType.PRECOMPILE_ERROR;
  precompile: number;
  sourceReference?: undefined;
}

export interface RevertErrorStackTraceEntry {
  type: StackTraceEntryType.REVERT_ERROR;
  message: ReturnData;
  sourceReference: SourceReference;
  isInvalidOpcodeError: boolean;
}

export interface PanicErrorStackTraceEntry {
  type: StackTraceEntryType.PANIC_ERROR;
  errorCode: bigint;
  sourceReference?: SourceReference;
}

export interface CustomErrorStackTraceEntry {
  type: StackTraceEntryType.CUSTOM_ERROR;
  // unlike RevertErrorStackTraceEntry, this includes the message already parsed
  message: string;
  sourceReference: SourceReference;
}

export interface UnmappedSolc063RevertErrorStackTraceEntry {
  type: StackTraceEntryType.UNMAPPED_SOLC_0_6_3_REVERT_ERROR;
  sourceReference?: SourceReference;
}

export interface FunctionNotPayableErrorStackTraceEntry {
  type: StackTraceEntryType.FUNCTION_NOT_PAYABLE_ERROR;
  value: bigint;
  sourceReference: SourceReference;
}

export interface InvalidParamsErrorStackTraceEntry {
  type: StackTraceEntryType.INVALID_PARAMS_ERROR;
  sourceReference: SourceReference;
}

export interface FallbackNotPayableErrorStackTraceEntry {
  type: StackTraceEntryType.FALLBACK_NOT_PAYABLE_ERROR;
  value: bigint;
  sourceReference: SourceReference;
}

export interface FallbackNotPayableAndNoReceiveErrorStackTraceEntry {
  type: StackTraceEntryType.FALLBACK_NOT_PAYABLE_AND_NO_RECEIVE_ERROR;
  value: bigint;
  sourceReference: SourceReference;
}

export interface UnrecognizedFunctionWithoutFallbackErrorStackTraceEntry {
  type: StackTraceEntryType.UNRECOGNIZED_FUNCTION_WITHOUT_FALLBACK_ERROR;
  sourceReference: SourceReference;
}

export interface MissingFallbackOrReceiveErrorStackTraceEntry {
  type: StackTraceEntryType.MISSING_FALLBACK_OR_RECEIVE_ERROR;
  sourceReference: SourceReference;
}

export interface ReturndataSizeErrorStackTraceEntry {
  type: StackTraceEntryType.RETURNDATA_SIZE_ERROR;
  sourceReference: SourceReference;
}

export interface NonContractAccountCalledErrorStackTraceEntry {
  type: StackTraceEntryType.NONCONTRACT_ACCOUNT_CALLED_ERROR;
  sourceReference: SourceReference;
}

export interface CallFailedErrorStackTraceEntry {
  type: StackTraceEntryType.CALL_FAILED_ERROR;
  sourceReference: SourceReference;
}

export interface DirectLibraryCallErrorStackTraceEntry {
  type: StackTraceEntryType.DIRECT_LIBRARY_CALL_ERROR;
  sourceReference: SourceReference;
}

export interface UnrecognizedCreateErrorStackTraceEntry {
  type: StackTraceEntryType.UNRECOGNIZED_CREATE_ERROR;
  message: ReturnData;
  sourceReference?: undefined;
  isInvalidOpcodeError: boolean;
}

export interface UnrecognizedContractErrorStackTraceEntry {
  type: StackTraceEntryType.UNRECOGNIZED_CONTRACT_ERROR;
  address: Uint8Array;
  message: ReturnData;
  sourceReference?: undefined;
  isInvalidOpcodeError: boolean;
}

export interface OtherExecutionErrorStackTraceEntry {
  type: StackTraceEntryType.OTHER_EXECUTION_ERROR;
  sourceReference?: SourceReference;
}

export interface ContractTooLargeErrorStackTraceEntry {
  type: StackTraceEntryType.CONTRACT_TOO_LARGE_ERROR;
  sourceReference?: SourceReference;
}

export interface InternalFunctionCallStackEntry {
  type: StackTraceEntryType.INTERNAL_FUNCTION_CALLSTACK_ENTRY;
  pc: number;
  sourceReference: SourceReference;
}

export interface ContractCallRunOutOfGasError {
  type: StackTraceEntryType.CONTRACT_CALL_RUN_OUT_OF_GAS_ERROR;
  sourceReference?: SourceReference;
}

export type SolidityStackTraceEntry =
  | CallstackEntryStackTraceEntry
  | UnrecognizedCreateCallstackEntryStackTraceEntry
  | UnrecognizedContractCallstackEntryStackTraceEntry
  | PrecompileErrorStackTraceEntry
  | RevertErrorStackTraceEntry
  | PanicErrorStackTraceEntry
  | CustomErrorStackTraceEntry
  | FunctionNotPayableErrorStackTraceEntry
  | InvalidParamsErrorStackTraceEntry
  | FallbackNotPayableErrorStackTraceEntry
  | FallbackNotPayableAndNoReceiveErrorStackTraceEntry
  | UnrecognizedFunctionWithoutFallbackErrorStackTraceEntry
  | MissingFallbackOrReceiveErrorStackTraceEntry
  | ReturndataSizeErrorStackTraceEntry
  | NonContractAccountCalledErrorStackTraceEntry
  | CallFailedErrorStackTraceEntry
  | DirectLibraryCallErrorStackTraceEntry
  | UnrecognizedCreateErrorStackTraceEntry
  | UnrecognizedContractErrorStackTraceEntry
  | OtherExecutionErrorStackTraceEntry
  | UnmappedSolc063RevertErrorStackTraceEntry
  | ContractTooLargeErrorStackTraceEntry
  | InternalFunctionCallStackEntry
  | ContractCallRunOutOfGasError;

export type SolidityStackTrace = SolidityStackTraceEntry[];
