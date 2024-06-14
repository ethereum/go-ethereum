import * as t from "io-ts";

export const rpcCompilerInput = t.type(
  {
    language: t.string,
    sources: t.any,
    settings: t.any,
  },
  "RpcCompilerInput"
);

export type RpcCompilerInput = t.TypeOf<typeof rpcCompilerInput>;

export const rpcCompilerOutput = t.type(
  {
    sources: t.any,
    contracts: t.any,
  },
  "RpcCompilerOutput"
);

export type RpcCompilerOutput = t.TypeOf<typeof rpcCompilerOutput>;
