import { AccessLists } from '../util.js'

import * as Legacy from './legacy.js'

import type { EIP2930CompatibleTx } from '../types.js'

/**
 * The amount of gas paid for the data in this tx
 */
export function getDataFee(tx: EIP2930CompatibleTx): bigint {
  return Legacy.getDataFee(tx, BigInt(AccessLists.getDataFeeEIP2930(tx.accessList, tx.common)))
}
