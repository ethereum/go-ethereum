/// <reference types="node" />
import * as t from "io-ts";
export declare const rpcLogAddress: t.UnionC<[t.Type<Buffer, Buffer, unknown>, t.ArrayC<t.Type<Buffer, Buffer, unknown>>]>;
export type RpcLogAddress = t.TypeOf<typeof rpcLogAddress>;
export declare const optionalRpcLogAddress: t.Type<Buffer | Buffer[] | undefined, Buffer | Buffer[] | undefined, unknown>;
export type OptionalRpcLogAddress = t.TypeOf<typeof optionalRpcLogAddress>;
//# sourceMappingURL=logAddress.d.ts.map