import { AccessLists } from '../util.js';
import * as Legacy from './legacy.js';
/**
 * The amount of gas paid for the data in this tx
 */
export function getDataFee(tx) {
    return Legacy.getDataFee(tx, BigInt(AccessLists.getDataFeeEIP2930(tx.accessList, tx.common)));
}
//# sourceMappingURL=eip2930.js.map