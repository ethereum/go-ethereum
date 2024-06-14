import * as t from "io-ts";

import { nullable, optional } from "../../../../util/io-ts";
import { rpcAccessList } from "../access-list";
import { rpcAddress, rpcData, rpcHash, rpcQuantity } from "../base-types";

export type RpcTransaction = t.TypeOf<typeof rpcTransaction>;
export const rpcTransaction = t.type(
  {
    blockHash: nullable(rpcHash),
    blockNumber: nullable(rpcQuantity),
    from: rpcAddress,
    gas: rpcQuantity,
    gasPrice: rpcQuantity, // NOTE: Its meaning was changed by EIP-1559
    hash: rpcHash,
    input: rpcData,
    nonce: rpcQuantity,
    // This is also optional because Alchemy doesn't return to for deployment txs
    to: optional(nullable(rpcAddress)),
    transactionIndex: nullable(rpcQuantity),
    value: rpcQuantity,
    v: rpcQuantity,
    r: rpcQuantity,
    s: rpcQuantity,

    // EIP-2929/2930 properties
    type: optional(rpcQuantity),
    chainId: optional(nullable(rpcQuantity)),
    accessList: optional(rpcAccessList),

    // EIP-1559 properties
    maxFeePerGas: optional(rpcQuantity),
    maxPriorityFeePerGas: optional(rpcQuantity),
  },
  "RpcTransaction"
);
