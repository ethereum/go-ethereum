import { fetchFromProvider, getProvider } from '@nomicfoundation/ethereumjs-util'

import { FeeMarketEIP1559Transaction } from './eip1559Transaction.js'
import { AccessListEIP2930Transaction } from './eip2930Transaction.js'
import { BlobEIP4844Transaction } from './eip4844Transaction.js'
import { normalizeTxParams } from './fromRpc.js'
import { LegacyTransaction } from './legacyTransaction.js'
import {
  TransactionType,
  isAccessListEIP2930TxData,
  isBlobEIP4844TxData,
  isFeeMarketEIP1559TxData,
  isLegacyTxData,
} from './types.js'

import type { Transaction, TxData, TxOptions, TypedTxData } from './types.js'
import type { EthersProvider } from '@nomicfoundation/ethereumjs-util'

export class TransactionFactory {
  // It is not possible to instantiate a TransactionFactory object.
  private constructor() {}

  /**
   * Create a transaction from a `txData` object
   *
   * @param txData - The transaction data. The `type` field will determine which transaction type is returned (if undefined, creates a legacy transaction)
   * @param txOptions - Options to pass on to the constructor of the transaction
   */
  public static fromTxData<T extends TransactionType>(
    txData: TypedTxData,
    txOptions: TxOptions = {}
  ): Transaction[T] {
    if (!('type' in txData) || txData.type === undefined) {
      // Assume legacy transaction
      return LegacyTransaction.fromTxData(txData, txOptions) as Transaction[T]
    } else {
      if (isLegacyTxData(txData)) {
        return LegacyTransaction.fromTxData(txData, txOptions) as Transaction[T]
      } else if (isAccessListEIP2930TxData(txData)) {
        return AccessListEIP2930Transaction.fromTxData(txData, txOptions) as Transaction[T]
      } else if (isFeeMarketEIP1559TxData(txData)) {
        return FeeMarketEIP1559Transaction.fromTxData(txData, txOptions) as Transaction[T]
      } else if (isBlobEIP4844TxData(txData)) {
        return BlobEIP4844Transaction.fromTxData(txData, txOptions) as Transaction[T]
      } else {
        throw new Error(`Tx instantiation with type ${(txData as TypedTxData)?.type} not supported`)
      }
    }
  }

  /**
   * This method tries to decode serialized data.
   *
   * @param data - The data Uint8Array
   * @param txOptions - The transaction options
   */
  public static fromSerializedData<T extends TransactionType>(
    data: Uint8Array,
    txOptions: TxOptions = {}
  ): Transaction[T] {
    if (data[0] <= 0x7f) {
      // Determine the type.
      switch (data[0]) {
        case TransactionType.AccessListEIP2930:
          return AccessListEIP2930Transaction.fromSerializedTx(data, txOptions) as Transaction[T]
        case TransactionType.FeeMarketEIP1559:
          return FeeMarketEIP1559Transaction.fromSerializedTx(data, txOptions) as Transaction[T]
        case TransactionType.BlobEIP4844:
          return BlobEIP4844Transaction.fromSerializedTx(data, txOptions) as Transaction[T]
        default:
          throw new Error(`TypedTransaction with ID ${data[0]} unknown`)
      }
    } else {
      return LegacyTransaction.fromSerializedTx(data, txOptions) as Transaction[T]
    }
  }

  /**
   * When decoding a BlockBody, in the transactions field, a field is either:
   * A Uint8Array (a TypedTransaction - encoded as TransactionType || rlp(TransactionPayload))
   * A Uint8Array[] (Legacy Transaction)
   * This method returns the right transaction.
   *
   * @param data - A Uint8Array or Uint8Array[]
   * @param txOptions - The transaction options
   */
  public static fromBlockBodyData(data: Uint8Array | Uint8Array[], txOptions: TxOptions = {}) {
    if (data instanceof Uint8Array) {
      return this.fromSerializedData(data, txOptions)
    } else if (Array.isArray(data)) {
      // It is a legacy transaction
      return LegacyTransaction.fromValuesArray(data, txOptions)
    } else {
      throw new Error('Cannot decode transaction: unknown type input')
    }
  }

  /**
   *  Method to retrieve a transaction from the provider
   * @param provider - a url string for a JSON-RPC provider or an Ethers JsonRPCProvider object
   * @param txHash - Transaction hash
   * @param txOptions - The transaction options
   * @returns the transaction specified by `txHash`
   */
  public static async fromJsonRpcProvider(
    provider: string | EthersProvider,
    txHash: string,
    txOptions?: TxOptions
  ) {
    const prov = getProvider(provider)
    const txData = await fetchFromProvider(prov, {
      method: 'eth_getTransactionByHash',
      params: [txHash],
    })
    if (txData === null) {
      throw new Error('No data returned from provider')
    }
    return TransactionFactory.fromRPC(txData, txOptions)
  }

  /**
   * Method to decode data retrieved from RPC, such as `eth_getTransactionByHash`
   * Note that this normalizes some of the parameters
   * @param txData The RPC-encoded data
   * @param txOptions The transaction options
   * @returns
   */
  public static async fromRPC<T extends TransactionType>(
    txData: TxData[T],
    txOptions: TxOptions = {}
  ): Promise<Transaction[T]> {
    return TransactionFactory.fromTxData(normalizeTxParams(txData), txOptions)
  }
}
