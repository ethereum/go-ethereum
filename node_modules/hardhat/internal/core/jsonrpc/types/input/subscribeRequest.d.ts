import * as t from "io-ts";
import { RpcFilterRequest } from "./filterRequest";
export interface RpcSubscribe {
    request: RpcFilterRequest;
}
export type RpcSubscribeRequest = t.TypeOf<typeof rpcSubscribeRequest>;
export declare const rpcSubscribeRequest: t.KeyofC<{
    newHeads: null;
    newPendingTransactions: null;
    logs: null;
}>;
//# sourceMappingURL=subscribeRequest.d.ts.map