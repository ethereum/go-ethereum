import * as t from "io-ts";
export declare const rpcCompilerInput: t.TypeC<{
    language: t.StringC;
    sources: t.AnyC;
    settings: t.AnyC;
}>;
export type RpcCompilerInput = t.TypeOf<typeof rpcCompilerInput>;
export declare const rpcCompilerOutput: t.TypeC<{
    sources: t.AnyC;
    contracts: t.AnyC;
}>;
export type RpcCompilerOutput = t.TypeOf<typeof rpcCompilerOutput>;
//# sourceMappingURL=solc.d.ts.map