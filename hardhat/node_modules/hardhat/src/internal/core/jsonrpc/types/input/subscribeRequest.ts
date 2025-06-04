import * as t from "io-ts";

import { RpcFilterRequest } from "./filterRequest";

// TODO: This types are incorrect. See: https://geth.ethereum.org/docs/rpc/pubsub
// Actually, the logs filter is not of the same type as eth_newFilter

export interface RpcSubscribe {
  request: RpcFilterRequest;
}

export type RpcSubscribeRequest = t.TypeOf<typeof rpcSubscribeRequest>;

export const rpcSubscribeRequest = t.keyof(
  {
    newHeads: null,
    newPendingTransactions: null,
    logs: null,
  },
  "RpcSubscribe"
);
