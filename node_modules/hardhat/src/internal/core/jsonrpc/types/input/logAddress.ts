import * as t from "io-ts";

import { optionalOrNullable } from "../../../../util/io-ts";
import { rpcAddress } from "../base-types";

export const rpcLogAddress = t.union([rpcAddress, t.array(rpcAddress)]);

export type RpcLogAddress = t.TypeOf<typeof rpcLogAddress>;

export const optionalRpcLogAddress = optionalOrNullable(rpcLogAddress);

export type OptionalRpcLogAddress = t.TypeOf<typeof optionalRpcLogAddress>;
