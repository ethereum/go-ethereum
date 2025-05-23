import * as t from "io-ts";
declare const rpcAuthorizationListTuple: t.TypeC<{
    chainId: t.Type<bigint, bigint, unknown>;
    address: t.Type<Buffer, Buffer, unknown>;
    nonce: t.Type<bigint, bigint, unknown>;
    yParity: t.Type<Buffer, Buffer, unknown>;
    r: t.Type<Buffer, Buffer, unknown>;
    s: t.Type<Buffer, Buffer, unknown>;
}>;
export declare const rpcAuthorizationList: t.ArrayC<t.TypeC<{
    chainId: t.Type<bigint, bigint, unknown>;
    address: t.Type<Buffer, Buffer, unknown>;
    nonce: t.Type<bigint, bigint, unknown>;
    yParity: t.Type<Buffer, Buffer, unknown>;
    r: t.Type<Buffer, Buffer, unknown>;
    s: t.Type<Buffer, Buffer, unknown>;
}>>;
export type RpcAuthorizationListTuple = t.TypeOf<typeof rpcAuthorizationListTuple>;
export type RpcAuthorizationList = t.TypeOf<typeof rpcAuthorizationList>;
export {};
//# sourceMappingURL=authorization-list.d.ts.map