/**
 * The basic types that we use to represent EVM values
 */
export type BaseEvmValue = number | bigint | string | boolean;
/**
 * A tuple of values, where each value can have a name or not.
 *
 * This are used to encode structs, tuples, and list of arguments returned
 * by a function, or used as arguments for a custom error or event.
 */
export interface EvmTuple {
    /**
     * The values in defintion order.
     */
    positional: EvmValue[];
    /**
     * A mapping from the return/param name to their value.
     *
     * Note that not every value will be named, so this mapping may
     * have less values than the `positional` array.
     */
    named: Record<string, EvmValue>;
}
/**
 * A value used in the EVM. Either accepted as an argument or returned as a result.
 */
export type EvmValue = BaseEvmValue | EvmValue[] | EvmTuple;
/**
 * The result of executing a contract function/constructor.
 */
export type EvmExecutionResult = SuccessfulEvmExecutionResult | FailedEvmExecutionResult;
/**
 * The result of executing a contract function/constructor that failed.
 */
export type FailedEvmExecutionResult = InvalidResultError | RevertWithoutReason | RevertWithReason | RevertWithPanicCode | RevertWithCustomError | RevertWithUnknownCustomError | RevertWithInvalidData | RevertWithInvalidDataOrUnknownCustomError;
/**
 * Each of the possible contract execution results that Ignition can handle.
 */
export declare enum EvmExecutionResultTypes {
    SUCESSFUL_RESULT = "SUCESSFUL_RESULT",
    INVALID_RESULT_ERROR = "INVALID_RESULT_ERROR",
    REVERT_WITHOUT_REASON = "REVERT_WITHOUT_REASON",
    REVERT_WITH_REASON = "REVERT_WITH_REASON",
    REVERT_WITH_PANIC_CODE = "REVERT_WITH_PANIC_CODE",
    REVERT_WITH_CUSTOM_ERROR = "REVERT_WITH_CUSTOM_ERROR",
    REVERT_WITH_UNKNOWN_CUSTOM_ERROR = "REVERT_WITH_UNKNOWN_CUSTOM_ERROR",
    REVERT_WITH_INVALID_DATA = "REVERT_WITH_INVALID_DATA",
    REVERT_WITH_INVALID_DATA_OR_UNKNOWN_CUSTOM_ERROR = "REVERT_WITH_INVALID_DATA_OR_UNKNOWN_CUSTOM_ERROR"
}
/**
 * The results returned by Solidity either as a function result, or as
 * custom error parameters.
 */
export interface SuccessfulEvmExecutionResult {
    type: EvmExecutionResultTypes.SUCESSFUL_RESULT;
    /**
     * The values returned by the execution.
     */
    values: EvmTuple;
}
/**
 * The execution was seemingly successful, but the data returned by it was invalid.
 */
export interface InvalidResultError {
    type: EvmExecutionResultTypes.INVALID_RESULT_ERROR;
    data: string;
}
/**
 * The execution reverted without a reason string nor any other kind of error.
 */
export interface RevertWithoutReason {
    type: EvmExecutionResultTypes.REVERT_WITHOUT_REASON;
}
/**
 * The execution reverted with a reason string by calling `revert("reason")`.
 */
export interface RevertWithReason {
    type: EvmExecutionResultTypes.REVERT_WITH_REASON;
    message: string;
}
/**
 * The execution reverted with a panic code due to some error that solc handled.
 */
export interface RevertWithPanicCode {
    type: EvmExecutionResultTypes.REVERT_WITH_PANIC_CODE;
    panicCode: number;
    panicName: string;
}
/**
 * The execution reverted with a custom error that was defined by the contract.
 */
export interface RevertWithCustomError {
    type: EvmExecutionResultTypes.REVERT_WITH_CUSTOM_ERROR;
    errorName: string;
    args: EvmTuple;
}
/**
 * This error is used when the JSON-RPC server indicated that the error was due to
 * a custom error, yet Ignition can't decode its data.
 *
 * Note that this only happens in development networks like Hardhat Network. They
 * can recognize that the data they are returning is a custom error, and inform that
 * to the user.
 *
 * We could treat this situation as RevertWithInvalidDataOrUnknownCustomError but
 * that would be loosing information.
 */
export interface RevertWithUnknownCustomError {
    type: EvmExecutionResultTypes.REVERT_WITH_UNKNOWN_CUSTOM_ERROR;
    signature: string;
    data: string;
}
/**
 * The execution failed due to some error whose kind we can recognize, but that
 * we can't decode because its data is invalid. This happens when the ABI decoding
 * of the error fails, or when a panic code is invalid.
 */
export interface RevertWithInvalidData {
    type: EvmExecutionResultTypes.REVERT_WITH_INVALID_DATA;
    data: string;
}
/**
 * If this error is returned the execution either returned completely invalid/unrecognizable
 * data, or a custom error that we can't recognize and the JSON-RPC server can't recognize either.
 */
export interface RevertWithInvalidDataOrUnknownCustomError {
    type: EvmExecutionResultTypes.REVERT_WITH_INVALID_DATA_OR_UNKNOWN_CUSTOM_ERROR;
    signature: string;
    data: string;
}
//# sourceMappingURL=evm-execution.d.ts.map