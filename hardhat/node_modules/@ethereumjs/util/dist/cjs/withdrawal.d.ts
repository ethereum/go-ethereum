import { Address } from './address.js';
import type { AddressLike, BigIntLike, PrefixedHexString } from './types.js';
/**
 * Flexible input data type for EIP-4895 withdrawal data with amount in Gwei to
 * match CL representation and for eventual ssz withdrawalsRoot
 */
export declare type WithdrawalData = {
    index: BigIntLike;
    validatorIndex: BigIntLike;
    address: AddressLike;
    amount: BigIntLike;
};
/**
 * JSON RPC interface for EIP-4895 withdrawal data with amount in Gwei to
 * match CL representation and for eventual ssz withdrawalsRoot
 */
export interface JsonRpcWithdrawal {
    index: PrefixedHexString;
    validatorIndex: PrefixedHexString;
    address: PrefixedHexString;
    amount: PrefixedHexString;
}
export declare type WithdrawalBytes = [Uint8Array, Uint8Array, Uint8Array, Uint8Array];
/**
 * Representation of EIP-4895 withdrawal data
 */
export declare class Withdrawal {
    readonly index: bigint;
    readonly validatorIndex: bigint;
    readonly address: Address;
    /**
     * withdrawal amount in Gwei to match the CL repesentation and eventually ssz withdrawalsRoot
     */
    readonly amount: bigint;
    /**
     * This constructor assigns and validates the values.
     * Use the static factory methods to assist in creating a Withdrawal object from varying data types.
     * Its amount is in Gwei to match CL representation and for eventual ssz withdrawalsRoot
     */
    constructor(index: bigint, validatorIndex: bigint, address: Address, 
    /**
     * withdrawal amount in Gwei to match the CL repesentation and eventually ssz withdrawalsRoot
     */
    amount: bigint);
    static fromWithdrawalData(withdrawalData: WithdrawalData): Withdrawal;
    static fromValuesArray(withdrawalArray: WithdrawalBytes): Withdrawal;
    /**
     * Convert a withdrawal to a buffer array
     * @param withdrawal the withdrawal to convert
     * @returns buffer array of the withdrawal
     */
    static toBytesArray(withdrawal: Withdrawal | WithdrawalData): WithdrawalBytes;
    raw(): WithdrawalBytes;
    toValue(): {
        index: bigint;
        validatorIndex: bigint;
        address: Uint8Array;
        amount: bigint;
    };
    toJSON(): {
        index: `0x${string}`;
        validatorIndex: `0x${string}`;
        address: `0x${string}`;
        amount: `0x${string}`;
    };
}
//# sourceMappingURL=withdrawal.d.ts.map