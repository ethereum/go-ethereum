import { Address } from './address.js'
import { bigIntToHex, bytesToHex, toBytes } from './bytes.js'
import { BIGINT_0 } from './constants.js'
import { TypeOutput, toType } from './types.js'

import type { AddressLike, BigIntLike, PrefixedHexString } from './types.js'

/**
 * Flexible input data type for EIP-4895 withdrawal data with amount in Gwei to
 * match CL representation and for eventual ssz withdrawalsRoot
 */
export type WithdrawalData = {
  index: BigIntLike
  validatorIndex: BigIntLike
  address: AddressLike
  amount: BigIntLike
}

/**
 * JSON RPC interface for EIP-4895 withdrawal data with amount in Gwei to
 * match CL representation and for eventual ssz withdrawalsRoot
 */
export interface JsonRpcWithdrawal {
  index: PrefixedHexString // QUANTITY - bigint 8 bytes
  validatorIndex: PrefixedHexString // QUANTITY - bigint 8 bytes
  address: PrefixedHexString // DATA, 20 Bytes  address to withdraw to
  amount: PrefixedHexString // QUANTITY - bigint amount in Gwei 8 bytes
}

export type WithdrawalBytes = [Uint8Array, Uint8Array, Uint8Array, Uint8Array]

/**
 * Representation of EIP-4895 withdrawal data
 */
export class Withdrawal {
  /**
   * This constructor assigns and validates the values.
   * Use the static factory methods to assist in creating a Withdrawal object from varying data types.
   * Its amount is in Gwei to match CL representation and for eventual ssz withdrawalsRoot
   */
  constructor(
    public readonly index: bigint,
    public readonly validatorIndex: bigint,
    public readonly address: Address,
    /**
     * withdrawal amount in Gwei to match the CL repesentation and eventually ssz withdrawalsRoot
     */
    public readonly amount: bigint
  ) {}

  public static fromWithdrawalData(withdrawalData: WithdrawalData) {
    const {
      index: indexData,
      validatorIndex: validatorIndexData,
      address: addressData,
      amount: amountData,
    } = withdrawalData
    const index = toType(indexData, TypeOutput.BigInt)
    const validatorIndex = toType(validatorIndexData, TypeOutput.BigInt)
    const address = addressData instanceof Address ? addressData : new Address(toBytes(addressData))
    const amount = toType(amountData, TypeOutput.BigInt)

    return new Withdrawal(index, validatorIndex, address, amount)
  }

  public static fromValuesArray(withdrawalArray: WithdrawalBytes) {
    if (withdrawalArray.length !== 4) {
      throw Error(`Invalid withdrawalArray length expected=4 actual=${withdrawalArray.length}`)
    }
    const [index, validatorIndex, address, amount] = withdrawalArray
    return Withdrawal.fromWithdrawalData({ index, validatorIndex, address, amount })
  }

  /**
   * Convert a withdrawal to a buffer array
   * @param withdrawal the withdrawal to convert
   * @returns buffer array of the withdrawal
   */
  public static toBytesArray(withdrawal: Withdrawal | WithdrawalData): WithdrawalBytes {
    const { index, validatorIndex, address, amount } = withdrawal
    const indexBytes =
      toType(index, TypeOutput.BigInt) === BIGINT_0
        ? new Uint8Array()
        : toType(index, TypeOutput.Uint8Array)
    const validatorIndexBytes =
      toType(validatorIndex, TypeOutput.BigInt) === BIGINT_0
        ? new Uint8Array()
        : toType(validatorIndex, TypeOutput.Uint8Array)
    const addressBytes =
      address instanceof Address ? (<Address>address).bytes : toType(address, TypeOutput.Uint8Array)

    const amountBytes =
      toType(amount, TypeOutput.BigInt) === BIGINT_0
        ? new Uint8Array()
        : toType(amount, TypeOutput.Uint8Array)

    return [indexBytes, validatorIndexBytes, addressBytes, amountBytes]
  }

  raw() {
    return Withdrawal.toBytesArray(this)
  }

  toValue() {
    return {
      index: this.index,
      validatorIndex: this.validatorIndex,
      address: this.address.bytes,
      amount: this.amount,
    }
  }

  toJSON() {
    return {
      index: bigIntToHex(this.index),
      validatorIndex: bigIntToHex(this.validatorIndex),
      address: bytesToHex(this.address.bytes),
      amount: bigIntToHex(this.amount),
    }
  }
}
