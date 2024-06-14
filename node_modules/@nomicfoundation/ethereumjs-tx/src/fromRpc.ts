import { TypeOutput, setLengthLeft, toBytes, toType } from '@nomicfoundation/ethereumjs-util'

import type { TypedTxData } from './types.js'

export const normalizeTxParams = (_txParams: any): TypedTxData => {
  const txParams = Object.assign({}, _txParams)

  txParams.gasLimit = toType(txParams.gasLimit ?? txParams.gas, TypeOutput.BigInt)
  txParams.data = txParams.data === undefined ? txParams.input : txParams.data

  // check and convert gasPrice and value params
  txParams.gasPrice = txParams.gasPrice !== undefined ? BigInt(txParams.gasPrice) : undefined
  txParams.value = txParams.value !== undefined ? BigInt(txParams.value) : undefined

  // strict byte length checking
  txParams.to =
    txParams.to !== null && txParams.to !== undefined
      ? setLengthLeft(toBytes(txParams.to), 20)
      : null

  // Normalize the v/r/s values. If RPC returns '0x0', ensure v/r/s are set to `undefined` in the tx.
  // If this is not done, then the transaction creation will throw, because `v` is `0`.
  // Note: this still means that `isSigned` will return `false`.
  // v/r/s values are `0x0` on networks like Optimism, where the tx is a system tx.
  // For instance: https://optimistic.etherscan.io/tx/0xf4304cb09b3f58a8e5d20fec5f393c96ccffe0269aaf632cb2be7a8a0f0c91cc

  txParams.v = txParams.v === '0x0' ? '0x' : txParams.v
  txParams.r = txParams.r === '0x0' ? '0x' : txParams.r
  txParams.s = txParams.s === '0x0' ? '0x' : txParams.s

  if (txParams.v !== '0x' || txParams.r !== '0x' || txParams.s !== '0x') {
    txParams.v = toType(txParams.v, TypeOutput.BigInt)
  }

  return txParams
}
