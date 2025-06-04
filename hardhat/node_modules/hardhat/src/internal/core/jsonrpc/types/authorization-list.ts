import * as t from "io-ts";

import { rpcAddress, rpcQuantity, rpcHash, rpcParity } from "./base-types";

const rpcAuthorizationListTuple = t.type({
  chainId: rpcQuantity,
  address: rpcAddress,
  nonce: rpcQuantity,
  yParity: rpcParity,
  r: rpcHash,
  s: rpcHash,
});

export const rpcAuthorizationList = t.array(rpcAuthorizationListTuple);

export type RpcAuthorizationListTuple = t.TypeOf<
  typeof rpcAuthorizationListTuple
>;

export type RpcAuthorizationList = t.TypeOf<typeof rpcAuthorizationList>;
