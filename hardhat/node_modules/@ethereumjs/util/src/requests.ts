import { RLP } from '@ethereumjs/rlp'
import { concatBytes } from 'ethereum-cryptography/utils'

import {
  bigIntToBytes,
  bigIntToHex,
  bytesToBigInt,
  bytesToHex,
  hexToBigInt,
  hexToBytes,
} from './bytes.js'
import { BIGINT_0 } from './constants.js'

import type { PrefixedHexString } from './types.js'

export type RequestBytes = Uint8Array

export enum CLRequestType {
  Deposit = 0x00,
  Withdrawal = 0x01,
  Consolidation = 0x02,
}

export type DepositRequestV1 = {
  pubkey: PrefixedHexString // DATA 48 bytes
  withdrawalCredentials: PrefixedHexString // DATA 32 bytes
  amount: PrefixedHexString // QUANTITY 8 bytes in gwei
  signature: PrefixedHexString // DATA 96 bytes
  index: PrefixedHexString // QUANTITY 8 bytes
}

export type WithdrawalRequestV1 = {
  sourceAddress: PrefixedHexString // DATA 20 bytes
  validatorPubkey: PrefixedHexString // DATA 48 bytes
  amount: PrefixedHexString // QUANTITY 8 bytes in gwei
}

export type ConsolidationRequestV1 = {
  sourceAddress: PrefixedHexString // DATA 20 bytes
  sourcePubkey: PrefixedHexString // DATA 48 bytes
  targetPubkey: PrefixedHexString // DATA 48 bytes
}

export interface RequestJSON {
  [CLRequestType.Deposit]: DepositRequestV1
  [CLRequestType.Withdrawal]: WithdrawalRequestV1
  [CLRequestType.Consolidation]: ConsolidationRequestV1
}

export type DepositRequestData = {
  pubkey: Uint8Array
  withdrawalCredentials: Uint8Array
  amount: bigint
  signature: Uint8Array
  index: bigint
}

export type WithdrawalRequestData = {
  sourceAddress: Uint8Array
  validatorPubkey: Uint8Array
  amount: bigint
}

export type ConsolidationRequestData = {
  sourceAddress: Uint8Array
  sourcePubkey: Uint8Array
  targetPubkey: Uint8Array
}

export interface RequestData {
  [CLRequestType.Deposit]: DepositRequestData
  [CLRequestType.Withdrawal]: WithdrawalRequestData
  [CLRequestType.Consolidation]: ConsolidationRequestData
}

export type TypedRequestData = RequestData[CLRequestType]

export interface CLRequestInterface<T extends CLRequestType = CLRequestType> {
  readonly type: T
  serialize(): Uint8Array
  toJSON(): RequestJSON[T]
}

export abstract class CLRequest<T extends CLRequestType> implements CLRequestInterface<T> {
  readonly type: T
  abstract serialize(): Uint8Array
  abstract toJSON(): RequestJSON[T]
  constructor(type: T) {
    this.type = type
  }
}

export class DepositRequest extends CLRequest<CLRequestType.Deposit> {
  constructor(
    public readonly pubkey: Uint8Array,
    public readonly withdrawalCredentials: Uint8Array,
    public readonly amount: bigint,
    public readonly signature: Uint8Array,
    public readonly index: bigint
  ) {
    super(CLRequestType.Deposit)
  }

  public static fromRequestData(depositData: DepositRequestData): DepositRequest {
    const { pubkey, withdrawalCredentials, amount, signature, index } = depositData
    return new DepositRequest(pubkey, withdrawalCredentials, amount, signature, index)
  }

  public static fromJSON(jsonData: DepositRequestV1): DepositRequest {
    const { pubkey, withdrawalCredentials, amount, signature, index } = jsonData
    return this.fromRequestData({
      pubkey: hexToBytes(pubkey),
      withdrawalCredentials: hexToBytes(withdrawalCredentials),
      amount: hexToBigInt(amount),
      signature: hexToBytes(signature),
      index: hexToBigInt(index),
    })
  }

  serialize() {
    const indexBytes = this.index === BIGINT_0 ? new Uint8Array() : bigIntToBytes(this.index)

    const amountBytes = this.amount === BIGINT_0 ? new Uint8Array() : bigIntToBytes(this.amount)

    return concatBytes(
      Uint8Array.from([this.type]),
      RLP.encode([this.pubkey, this.withdrawalCredentials, amountBytes, this.signature, indexBytes])
    )
  }

  toJSON(): DepositRequestV1 {
    return {
      pubkey: bytesToHex(this.pubkey),
      withdrawalCredentials: bytesToHex(this.withdrawalCredentials),
      amount: bigIntToHex(this.amount),
      signature: bytesToHex(this.signature),
      index: bigIntToHex(this.index),
    }
  }

  public static deserialize(bytes: Uint8Array): DepositRequest {
    const [pubkey, withdrawalCredentials, amount, signature, index] = RLP.decode(
      bytes.slice(1)
    ) as [Uint8Array, Uint8Array, Uint8Array, Uint8Array, Uint8Array]
    return this.fromRequestData({
      pubkey,
      withdrawalCredentials,
      amount: bytesToBigInt(amount),
      signature,
      index: bytesToBigInt(index),
    })
  }
}

export class WithdrawalRequest extends CLRequest<CLRequestType.Withdrawal> {
  constructor(
    public readonly sourceAddress: Uint8Array,
    public readonly validatorPubkey: Uint8Array,
    public readonly amount: bigint
  ) {
    super(CLRequestType.Withdrawal)
  }

  public static fromRequestData(withdrawalData: WithdrawalRequestData): WithdrawalRequest {
    const { sourceAddress, validatorPubkey, amount } = withdrawalData
    return new WithdrawalRequest(sourceAddress, validatorPubkey, amount)
  }

  public static fromJSON(jsonData: WithdrawalRequestV1): WithdrawalRequest {
    const { sourceAddress, validatorPubkey, amount } = jsonData
    return this.fromRequestData({
      sourceAddress: hexToBytes(sourceAddress),
      validatorPubkey: hexToBytes(validatorPubkey),
      amount: hexToBigInt(amount),
    })
  }

  serialize() {
    const amountBytes = this.amount === BIGINT_0 ? new Uint8Array() : bigIntToBytes(this.amount)

    return concatBytes(
      Uint8Array.from([this.type]),
      RLP.encode([this.sourceAddress, this.validatorPubkey, amountBytes])
    )
  }

  toJSON(): WithdrawalRequestV1 {
    return {
      sourceAddress: bytesToHex(this.sourceAddress),
      validatorPubkey: bytesToHex(this.validatorPubkey),
      amount: bigIntToHex(this.amount),
    }
  }

  public static deserialize(bytes: Uint8Array): WithdrawalRequest {
    const [sourceAddress, validatorPubkey, amount] = RLP.decode(bytes.slice(1)) as [
      Uint8Array,
      Uint8Array,
      Uint8Array
    ]
    return this.fromRequestData({
      sourceAddress,
      validatorPubkey,
      amount: bytesToBigInt(amount),
    })
  }
}

export class ConsolidationRequest extends CLRequest<CLRequestType.Consolidation> {
  constructor(
    public readonly sourceAddress: Uint8Array,
    public readonly sourcePubkey: Uint8Array,
    public readonly targetPubkey: Uint8Array
  ) {
    super(CLRequestType.Consolidation)
  }

  public static fromRequestData(consolidationData: ConsolidationRequestData): ConsolidationRequest {
    const { sourceAddress, sourcePubkey, targetPubkey } = consolidationData
    return new ConsolidationRequest(sourceAddress, sourcePubkey, targetPubkey)
  }

  public static fromJSON(jsonData: ConsolidationRequestV1): ConsolidationRequest {
    const { sourceAddress, sourcePubkey, targetPubkey } = jsonData
    return this.fromRequestData({
      sourceAddress: hexToBytes(sourceAddress),
      sourcePubkey: hexToBytes(sourcePubkey),
      targetPubkey: hexToBytes(targetPubkey),
    })
  }

  serialize() {
    return concatBytes(
      Uint8Array.from([this.type]),
      RLP.encode([this.sourceAddress, this.sourcePubkey, this.targetPubkey])
    )
  }

  toJSON(): ConsolidationRequestV1 {
    return {
      sourceAddress: bytesToHex(this.sourceAddress),
      sourcePubkey: bytesToHex(this.sourcePubkey),
      targetPubkey: bytesToHex(this.targetPubkey),
    }
  }

  public static deserialize(bytes: Uint8Array): ConsolidationRequest {
    const [sourceAddress, sourcePubkey, targetPubkey] = RLP.decode(bytes.slice(1)) as [
      Uint8Array,
      Uint8Array,
      Uint8Array
    ]
    return this.fromRequestData({
      sourceAddress,
      sourcePubkey,
      targetPubkey,
    })
  }
}

export class CLRequestFactory {
  public static fromSerializedRequest(bytes: Uint8Array): CLRequest<CLRequestType> {
    switch (bytes[0]) {
      case CLRequestType.Deposit:
        return DepositRequest.deserialize(bytes)
      case CLRequestType.Withdrawal:
        return WithdrawalRequest.deserialize(bytes)
      case CLRequestType.Consolidation:
        return ConsolidationRequest.deserialize(bytes)
      default:
        throw Error(`Invalid request type=${bytes[0]}`)
    }
  }
}
