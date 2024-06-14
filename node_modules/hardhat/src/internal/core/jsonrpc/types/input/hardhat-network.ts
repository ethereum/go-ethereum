import * as t from "io-ts";

import { optional } from "../../../../util/io-ts";
import { rpcUnsignedInteger } from "../base-types";

export const rpcForkConfig = optional(
  t.type(
    {
      jsonRpcUrl: t.string,
      blockNumber: optional(t.number),
      httpHeaders: optional(t.record(t.string, t.string, "httpHeaders")),
    },
    "RpcForkConfig"
  )
);

export type RpcForkConfig = t.TypeOf<typeof rpcForkConfig>;

export const rpcHardhatNetworkConfig = t.type(
  {
    forking: optional(rpcForkConfig),
  },
  "HardhatNetworkConfig"
);

export type RpcHardhatNetworkConfig = t.TypeOf<typeof rpcHardhatNetworkConfig>;

export const optionalRpcHardhatNetworkConfig = optional(
  rpcHardhatNetworkConfig
);

const isNumberPair = (x: unknown): x is [number, number] =>
  Array.isArray(x) &&
  x.length === 2 &&
  Number.isInteger(x[0]) &&
  Number.isInteger(x[1]);

// TODO: This can be simplified
const rpcIntervalMiningRange = new t.Type<[number, number]>(
  "Interval mining range",
  isNumberPair,
  (u, c) =>
    isNumberPair(u) && u[0] >= 0 && u[1] >= u[0]
      ? t.success(u)
      : t.failure(u, c),
  t.identity
);

export const rpcIntervalMining = t.union([
  rpcUnsignedInteger,
  rpcIntervalMiningRange,
]);

export type RpcIntervalMining = t.TypeOf<typeof rpcIntervalMining>;
