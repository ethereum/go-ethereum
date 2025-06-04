import * as t from "io-ts";

import { optionalOrNullable } from "../../../../util/io-ts";

export const rpcDebugTracingConfig = optionalOrNullable(
  t.type(
    {
      tracer: optionalOrNullable(t.string),
      disableStorage: optionalOrNullable(t.boolean),
      disableMemory: optionalOrNullable(t.boolean),
      disableStack: optionalOrNullable(t.boolean),
    },
    "RpcDebugTracingConfig"
  )
);

export type RpcDebugTracingConfig = t.TypeOf<typeof rpcDebugTracingConfig>;
