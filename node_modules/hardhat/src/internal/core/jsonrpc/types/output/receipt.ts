import * as t from "io-ts";

import { nullable, optional } from "../../../../util/io-ts";
import {
  rpcAddress,
  rpcData,
  rpcHash,
  rpcQuantity,
  rpcQuantityAsNumber,
} from "../base-types";

import { rpcLog } from "./log";

export type RpcTransactionReceipt = t.TypeOf<typeof rpcTransactionReceipt>;
export const rpcTransactionReceipt = t.type(
  {
    transactionHash: rpcHash,
    transactionIndex: rpcQuantity,
    blockHash: rpcHash,
    blockNumber: rpcQuantity,
    from: rpcAddress,
    to: nullable(rpcAddress),
    cumulativeGasUsed: rpcQuantity,
    gasUsed: rpcQuantity,
    contractAddress: nullable(rpcAddress),
    logs: t.array(rpcLog, "RpcLog Array"),
    logsBloom: rpcData,
    // This should be just optional, but Alchemy returns null
    //
    // It shouldn't accept a number, but that's what Erigon returns.
    // See: https://github.com/ledgerwatch/erigon/issues/2288
    status: optional(nullable(t.union([rpcQuantity, rpcQuantityAsNumber]))),
    root: optional(rpcData),
    type: optional(rpcQuantity),
    effectiveGasPrice: optional(rpcQuantity),
  },
  "RpcTransactionReceipt"
);
