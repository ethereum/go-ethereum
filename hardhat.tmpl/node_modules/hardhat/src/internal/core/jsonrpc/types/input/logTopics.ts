import * as t from "io-ts";

import { optionalOrNullable } from "../../../../util/io-ts";
import { rpcHash } from "../base-types";

export const rpcLogTopics = t.array(
  t.union([t.null, rpcHash, t.array(t.union([t.null, rpcHash]))])
);

export type RpcLogTopics = t.TypeOf<typeof rpcLogTopics>;

export const optionalRpcLogTopics = optionalOrNullable(rpcLogTopics);

export type OptionalRpcLogTopics = t.TypeOf<typeof optionalRpcLogTopics>;
