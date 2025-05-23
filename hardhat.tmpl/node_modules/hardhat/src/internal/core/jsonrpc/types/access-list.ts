import * as t from "io-ts";

import { nullable } from "../../../util/io-ts";

import { rpcData } from "./base-types";

const rpcAccessListTuple = t.type({
  address: rpcData,
  storageKeys: nullable(t.array(rpcData)),
});

export const rpcAccessList = t.array(rpcAccessListTuple);

export type RpcAccessListTuple = t.TypeOf<typeof rpcAccessListTuple>;

export type RpcAccessList = t.TypeOf<typeof rpcAccessList>;
