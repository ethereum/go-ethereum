/// <reference types="node" />
import { Address } from './address';
import type { AddressLike, BigIntLike } from './types';
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
    index: string;
    validatorIndex: string;
    address: string;
    amount: string;
}
export declare type WithdrawalBuffer = [Buffer, Buffer, Buffer, Buffer];
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
    static fromValuesArray(withdrawalArray: WithdrawalBuffer): Withdrawal;
    /**
     * Convert a withdrawal to a buffer array
     * @param withdrawal the withdrawal to convert
     * @returns buffer array of the withdrawal
     */
    static toBufferArray(withdrawal: Withdrawal | WithdrawalData): WithdrawalBuffer;
    raw(): WithdrawalBuffer;
    toValue(): {
        index: bigint;
        validatorIndex: bigint;
        address: Buffer;
        amount: bigint;
    };
    toJSON(): {
        index: string;
        validatorIndex: string;
        address: string;
        amount: string;
    };
}
//# sourceMappingURL=withdrawal.d.ts.map