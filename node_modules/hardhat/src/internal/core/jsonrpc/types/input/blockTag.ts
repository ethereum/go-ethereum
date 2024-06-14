import * as t from "io-ts";

import { optionalOrNullable } from "../../../../util/io-ts";
import { rpcData, rpcQuantity } from "../base-types";

export const rpcNewBlockTagObjectWithNumber = t.type({
  blockNumber: rpcQuantity,
});

export const rpcNewBlockTagObjectWithHash = t.type({
  blockHash: rpcData,
  requireCanonical: optionalOrNullable(t.boolean),
});

export const rpcBlockTagName = t.keyof({
  earliest: null,
  latest: null,
  pending: null,
  safe: null,
  finalized: null,
});

// This is the new kind of block tag as defined by https://github.com/ethereum/EIPs/blob/master/EIPS/eip-1898.md
export const rpcNewBlockTag = t.union([
  rpcQuantity,
  rpcNewBlockTagObjectWithNumber,
  rpcNewBlockTagObjectWithHash,
  rpcBlockTagName,
]);

export type RpcNewBlockTag = t.TypeOf<typeof rpcNewBlockTag>;

export const optionalRpcNewBlockTag = optionalOrNullable(rpcNewBlockTag);

export type OptionalRpcNewBlockTag = t.TypeOf<typeof optionalRpcNewBlockTag>;

// This is the old kind of block tag which is described in the ethereum wiki
export const rpcOldBlockTag = t.union([rpcQuantity, rpcBlockTagName]);

export type RpcOldBlockTag = t.TypeOf<typeof rpcOldBlockTag>;

export const optionalRpcOldBlockTag = optionalOrNullable(rpcOldBlockTag);

export type OptionalRpcOldBlockTag = t.TypeOf<typeof optionalRpcOldBlockTag>;
