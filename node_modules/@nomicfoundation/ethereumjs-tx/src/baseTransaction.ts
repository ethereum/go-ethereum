import { Chain, Common } from '@nomicfoundation/ethereumjs-common'
import {
  Address,
  BIGINT_0,
  MAX_INTEGER,
  MAX_UINT64,
  bigIntToHex,
  bytesToBigInt,
  bytesToHex,
  ecsign,
  publicToAddress,
  toBytes,
  unpadBytes,
} from '@nomicfoundation/ethereumjs-util'

import { Capability, TransactionType } from './types.js'
import { checkMaxInitCodeSize } from './util.js'

import type {
  JsonTx,
  Transaction,
  TransactionCache,
  TransactionInterface,
  TxData,
  TxOptions,
  TxValuesArray,
} from './types.js'
import type { BigIntLike } from '@nomicfoundation/ethereumjs-util'

/**
 * This base class will likely be subject to further
 * refactoring along the introduction of additional tx types
 * on the Ethereum network.
 *
 * It is therefore not recommended to use directly.
 */
export abstract class BaseTransaction<T extends TransactionType>
  implements TransactionInterface<T>
{
  protected readonly _type: TransactionType

  public readonly nonce: bigint
  public readonly gasLimit: bigint
  public readonly to?: Address
  public readonly value: bigint
  public readonly data: Uint8Array

  public readonly v?: bigint
  public readonly r?: bigint
  public readonly s?: bigint

  public readonly common!: Common

  public cache: TransactionCache = {
    hash: undefined,
    dataFee: undefined,
    senderPubKey: undefined,
    senderAddress: undefined,
  }

  protected readonly txOptions: TxOptions

  /**
   * List of tx type defining EIPs,
   * e.g. 1559 (fee market) and 2930 (access lists)
   * for FeeMarketEIP1559Transaction objects
   */
  protected activeCapabilities: number[] = []

  /**
   * The default chain the tx falls back to if no Common
   * is provided and if the chain can't be derived from
   * a passed in chainId (only EIP-2718 typed txs) or
   * EIP-155 signature (legacy txs).
   *
   * @hidden
   */
  protected DEFAULT_CHAIN = Chain.Mainnet

  constructor(txData: TxData[T], opts: TxOptions) {
    const { nonce, gasLimit, to, value, data, v, r, s, type } = txData
    this._type = Number(bytesToBigInt(toBytes(type)))

    this.txOptions = opts

    const toB = toBytes(to === '' ? '0x' : to)
    const vB = toBytes(v === '' ? '0x' : v)
    const rB = toBytes(r === '' ? '0x' : r)
    const sB = toBytes(s === '' ? '0x' : s)

    this.nonce = bytesToBigInt(toBytes(nonce === '' ? '0x' : nonce))
    this.gasLimit = bytesToBigInt(toBytes(gasLimit === '' ? '0x' : gasLimit))
    this.to = toB.length > 0 ? new Address(toB) : undefined
    this.value = bytesToBigInt(toBytes(value === '' ? '0x' : value))
    this.data = toBytes(data === '' ? '0x' : data)

    this.v = vB.length > 0 ? bytesToBigInt(vB) : undefined
    this.r = rB.length > 0 ? bytesToBigInt(rB) : undefined
    this.s = sB.length > 0 ? bytesToBigInt(sB) : undefined

    this._validateCannotExceedMaxInteger({ value: this.value, r: this.r, s: this.s })

    // geth limits gasLimit to 2^64-1
    this._validateCannotExceedMaxInteger({ gasLimit: this.gasLimit }, 64)

    // EIP-2681 limits nonce to 2^64-1 (cannot equal 2^64-1)
    this._validateCannotExceedMaxInteger({ nonce: this.nonce }, 64, true)

    const createContract = this.to === undefined || this.to === null
    const allowUnlimitedInitCodeSize = opts.allowUnlimitedInitCodeSize ?? false
    const common = opts.common ?? this._getCommon()
    if (createContract && common.isActivatedEIP(3860) && allowUnlimitedInitCodeSize === false) {
      checkMaxInitCodeSize(common, this.data.length)
    }
  }

  /**
   * Returns the transaction type.
   *
   * Note: legacy txs will return tx type `0`.
   */
  get type() {
    return this._type
  }

  /**
   * Checks if a tx type defining capability is active
   * on a tx, for example the EIP-1559 fee market mechanism
   * or the EIP-2930 access list feature.
   *
   * Note that this is different from the tx type itself,
   * so EIP-2930 access lists can very well be active
   * on an EIP-1559 tx for example.
   *
   * This method can be useful for feature checks if the
   * tx type is unknown (e.g. when instantiated with
   * the tx factory).
   *
   * See `Capabilities` in the `types` module for a reference
   * on all supported capabilities.
   */
  supports(capability: Capability) {
    return this.activeCapabilities.includes(capability)
  }

  /**
   * Validates the transaction signature and minimum gas requirements.
   * @returns {string[]} an array of error strings
   */
  getValidationErrors(): string[] {
    const errors = []

    if (this.isSigned() && !this.verifySignature()) {
      errors.push('Invalid Signature')
    }

    if (this.getBaseFee() > this.gasLimit) {
      errors.push(`gasLimit is too low. given ${this.gasLimit}, need at least ${this.getBaseFee()}`)
    }

    return errors
  }

  /**
   * Validates the transaction signature and minimum gas requirements.
   * @returns {boolean} true if the transaction is valid, false otherwise
   */
  isValid(): boolean {
    const errors = this.getValidationErrors()

    return errors.length === 0
  }

  /**
   * The minimum amount of gas the tx must have (DataFee + TxFee + Creation Fee)
   */
  getBaseFee(): bigint {
    const txFee = this.common.param('gasPrices', 'tx')
    let fee = this.getDataFee()
    if (txFee) fee += txFee
    if (this.common.gteHardfork('homestead') && this.toCreationAddress()) {
      const txCreationFee = this.common.param('gasPrices', 'txCreation')
      if (txCreationFee) fee += txCreationFee
    }
    return fee
  }

  /**
   * The amount of gas paid for the data in this tx
   */
  getDataFee(): bigint {
    const txDataZero = this.common.param('gasPrices', 'txDataZero')
    const txDataNonZero = this.common.param('gasPrices', 'txDataNonZero')

    let cost = BIGINT_0
    for (let i = 0; i < this.data.length; i++) {
      this.data[i] === 0 ? (cost += txDataZero) : (cost += txDataNonZero)
    }

    if ((this.to === undefined || this.to === null) && this.common.isActivatedEIP(3860)) {
      const dataLength = BigInt(Math.ceil(this.data.length / 32))
      const initCodeCost = this.common.param('gasPrices', 'initCodeWordCost') * dataLength
      cost += initCodeCost
    }

    return cost
  }

  /**
   * The up front amount that an account must have for this transaction to be valid
   */
  abstract getUpfrontCost(): bigint

  /**
   * If the tx's `to` is to the creation address
   */
  toCreationAddress(): boolean {
    return this.to === undefined || this.to.bytes.length === 0
  }

  /**
   * Returns a Uint8Array Array of the raw Bytes of this transaction, in order.
   *
   * Use {@link BaseTransaction.serialize} to add a transaction to a block
   * with {@link Block.fromValuesArray}.
   *
   * For an unsigned tx this method uses the empty Bytes values for the
   * signature parameters `v`, `r` and `s` for encoding. For an EIP-155 compliant
   * representation for external signing use {@link BaseTransaction.getMessageToSign}.
   */
  abstract raw(): TxValuesArray[T]

  /**
   * Returns the encoding of the transaction.
   */
  abstract serialize(): Uint8Array

  // Returns the raw unsigned tx, which is used to sign the transaction.
  abstract getMessageToSign(): Uint8Array | Uint8Array[]

  // Returns the hashed unsigned tx, which is used to sign the transaction.
  abstract getHashedMessageToSign(): Uint8Array

  abstract hash(): Uint8Array

  abstract getMessageToVerifySignature(): Uint8Array

  public isSigned(): boolean {
    const { v, r, s } = this
    if (v === undefined || r === undefined || s === undefined) {
      return false
    } else {
      return true
    }
  }

  /**
   * Determines if the signature is valid
   */
  verifySignature(): boolean {
    try {
      // Main signature verification is done in `getSenderPublicKey()`
      const publicKey = this.getSenderPublicKey()
      return unpadBytes(publicKey).length !== 0
    } catch (e: any) {
      return false
    }
  }

  /**
   * Returns the sender's address
   */
  getSenderAddress(): Address {
    if (this.cache.senderAddress === undefined) {
      this.cache.senderAddress = new Address(publicToAddress(this.getSenderPublicKey()))
    }
    return this.cache.senderAddress
  }

  /**
   * Returns the public key of the sender
   */
  abstract _getSenderPublicKey(): Uint8Array

  getSenderPublicKey(): Uint8Array {
    if (this.cache.senderPubKey === undefined) {
      this.cache.senderPubKey = this._getSenderPublicKey()
    }
    return this.cache.senderPubKey
  }

  /**
   * Signs a transaction.
   *
   * Note that the signed tx is returned as a new object,
   * use as follows:
   * ```javascript
   * const signedTx = tx.sign(privateKey)
   * ```
   */
  sign(privateKey: Uint8Array): Transaction[T] {
    if (privateKey.length !== 32) {
      const msg = this._errorMsg('Private key must be 32 bytes in length.')
      throw new Error(msg)
    }

    // Hack for the constellation that we have got a legacy tx after spuriousDragon with a non-EIP155 conforming signature
    // and want to recreate a signature (where EIP155 should be applied)
    // Leaving this hack lets the legacy.spec.ts -> sign(), verifySignature() test fail
    // 2021-06-23
    let hackApplied = false
    if (
      this.type === TransactionType.Legacy &&
      this.common.gteHardfork('spuriousDragon') &&
      !this.supports(Capability.EIP155ReplayProtection)
    ) {
      this.activeCapabilities.push(Capability.EIP155ReplayProtection)
      hackApplied = true
    }

    const msgHash = this.getHashedMessageToSign()
    const ecSignFunction = this.common.customCrypto?.ecsign ?? ecsign
    const { v, r, s } = ecSignFunction(msgHash, privateKey)
    const tx = this._processSignature(v, r, s)

    // Hack part 2
    if (hackApplied) {
      const index = this.activeCapabilities.indexOf(Capability.EIP155ReplayProtection)
      if (index > -1) {
        this.activeCapabilities.splice(index, 1)
      }
    }

    return tx
  }

  /**
   * Returns an object with the JSON representation of the transaction
   */
  toJSON(): JsonTx {
    return {
      type: bigIntToHex(BigInt(this.type)),
      nonce: bigIntToHex(this.nonce),
      gasLimit: bigIntToHex(this.gasLimit),
      to: this.to !== undefined ? this.to.toString() : undefined,
      value: bigIntToHex(this.value),
      data: bytesToHex(this.data),
      v: this.v !== undefined ? bigIntToHex(this.v) : undefined,
      r: this.r !== undefined ? bigIntToHex(this.r) : undefined,
      s: this.s !== undefined ? bigIntToHex(this.s) : undefined,
    }
  }

  // Accept the v,r,s values from the `sign` method, and convert this into a T
  protected abstract _processSignature(v: bigint, r: Uint8Array, s: Uint8Array): Transaction[T]

  /**
   * Does chain ID checks on common and returns a common
   * to be used on instantiation
   * @hidden
   *
   * @param common - {@link Common} instance from tx options
   * @param chainId - Chain ID from tx options (typed txs) or signature (legacy tx)
   */
  protected _getCommon(common?: Common, chainId?: BigIntLike) {
    // Chain ID provided
    if (chainId !== undefined) {
      const chainIdBigInt = bytesToBigInt(toBytes(chainId))
      if (common) {
        if (common.chainId() !== chainIdBigInt) {
          const msg = this._errorMsg('The chain ID does not match the chain ID of Common')
          throw new Error(msg)
        }
        // Common provided, chain ID does match
        // -> Return provided Common
        return common.copy()
      } else {
        if (Common.isSupportedChainId(chainIdBigInt)) {
          // No Common, chain ID supported by Common
          // -> Instantiate Common with chain ID
          return new Common({ chain: chainIdBigInt })
        } else {
          // No Common, chain ID not supported by Common
          // -> Instantiate custom Common derived from DEFAULT_CHAIN
          return Common.custom(
            {
              name: 'custom-chain',
              networkId: chainIdBigInt,
              chainId: chainIdBigInt,
            },
            { baseChain: this.DEFAULT_CHAIN }
          )
        }
      }
    } else {
      // No chain ID provided
      // -> return Common provided or create new default Common
      return common?.copy() ?? new Common({ chain: this.DEFAULT_CHAIN })
    }
  }

  /**
   * Validates that an object with BigInt values cannot exceed the specified bit limit.
   * @param values Object containing string keys and BigInt values
   * @param bits Number of bits to check (64 or 256)
   * @param cannotEqual Pass true if the number also cannot equal one less the maximum value
   */
  protected _validateCannotExceedMaxInteger(
    values: { [key: string]: bigint | undefined },
    bits = 256,
    cannotEqual = false
  ) {
    for (const [key, value] of Object.entries(values)) {
      switch (bits) {
        case 64:
          if (cannotEqual) {
            if (value !== undefined && value >= MAX_UINT64) {
              const msg = this._errorMsg(
                `${key} cannot equal or exceed MAX_UINT64 (2^64-1), given ${value}`
              )
              throw new Error(msg)
            }
          } else {
            if (value !== undefined && value > MAX_UINT64) {
              const msg = this._errorMsg(`${key} cannot exceed MAX_UINT64 (2^64-1), given ${value}`)
              throw new Error(msg)
            }
          }
          break
        case 256:
          if (cannotEqual) {
            if (value !== undefined && value >= MAX_INTEGER) {
              const msg = this._errorMsg(
                `${key} cannot equal or exceed MAX_INTEGER (2^256-1), given ${value}`
              )
              throw new Error(msg)
            }
          } else {
            if (value !== undefined && value > MAX_INTEGER) {
              const msg = this._errorMsg(
                `${key} cannot exceed MAX_INTEGER (2^256-1), given ${value}`
              )
              throw new Error(msg)
            }
          }
          break
        default: {
          const msg = this._errorMsg('unimplemented bits value')
          throw new Error(msg)
        }
      }
    }
  }

  protected static _validateNotArray(values: { [key: string]: any }) {
    const txDataKeys = [
      'nonce',
      'gasPrice',
      'gasLimit',
      'to',
      'value',
      'data',
      'v',
      'r',
      's',
      'type',
      'baseFee',
      'maxFeePerGas',
      'chainId',
    ]
    for (const [key, value] of Object.entries(values)) {
      if (txDataKeys.includes(key)) {
        if (Array.isArray(value)) {
          throw new Error(`${key} cannot be an array`)
        }
      }
    }
  }

  /**
   * Return a compact error string representation of the object
   */
  public abstract errorStr(): string

  /**
   * Internal helper function to create an annotated error message
   *
   * @param msg Base error message
   * @hidden
   */
  protected abstract _errorMsg(msg: string): string

  /**
   * Returns the shared error postfix part for _error() method
   * tx type implementations.
   */
  protected _getSharedErrorPostfix() {
    let hash = ''
    try {
      hash = this.isSigned() ? bytesToHex(this.hash()) : 'not available (unsigned)'
    } catch (e: any) {
      hash = 'error'
    }
    let isSigned = ''
    try {
      isSigned = this.isSigned().toString()
    } catch (e: any) {
      hash = 'error'
    }
    let hf = ''
    try {
      hf = this.common.hardfork()
    } catch (e: any) {
      hf = 'error'
    }

    let postfix = `tx type=${this.type} hash=${hash} nonce=${this.nonce} value=${this.value} `
    postfix += `signed=${isSigned} hf=${hf}`

    return postfix
  }
}
