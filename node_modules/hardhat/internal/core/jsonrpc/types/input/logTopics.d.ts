/// <reference types="node" />
import * as t from "io-ts";
export declare const rpcLogTopics: t.ArrayC<t.UnionC<[t.NullC, t.Type<Buffer, Buffer, unknown>, t.ArrayC<t.UnionC<[t.NullC, t.Type<Buffer, Buffer, unknown>]>>]>>;
export type RpcLogTopics = t.TypeOf<typeof rpcLogTopics>;
export declare const optionalRpcLogTopics: t.Type<(Buffer | (Buffer | null)[] | null)[] | undefined, (Buffer | (Buffer | null)[] | null)[] | undefined, unknown>;
export type OptionalRpcLogTopics = t.TypeOf<typeof optionalRpcLogTopics>;
//# sourceMappingURL=logTopics.d.ts.map