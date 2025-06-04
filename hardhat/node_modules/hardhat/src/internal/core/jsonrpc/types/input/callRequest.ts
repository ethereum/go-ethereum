import * as t from "io-ts";

import { optionalOrNullable } from "../../../../util/io-ts";
import { rpcAccessList } from "../access-list";
import {
  rpcAddress,
  rpcData,
  rpcHash,
  rpcQuantity,
  rpcStorageSlot,
  rpcStorageSlotHexString,
} from "../base-types";
import { address } from "../../../config/config-validation";

// Type used by eth_call and eth_estimateGas
export const rpcCallRequest = t.type(
  {
    from: optionalOrNullable(rpcAddress),
    to: optionalOrNullable(rpcAddress),
    gas: optionalOrNullable(rpcQuantity),
    gasPrice: optionalOrNullable(rpcQuantity),
    value: optionalOrNullable(rpcQuantity),
    data: optionalOrNullable(rpcData),
    accessList: optionalOrNullable(rpcAccessList),
    maxFeePerGas: optionalOrNullable(rpcQuantity),
    maxPriorityFeePerGas: optionalOrNullable(rpcQuantity),
    blobs: optionalOrNullable(t.array(rpcData)),
    blobVersionedHashes: optionalOrNullable(t.array(rpcHash)),
  },
  "RpcCallRequest"
);

export type RpcCallRequest = t.TypeOf<typeof rpcCallRequest>;

// Types used by eth_call to configure the state override set
export const stateProperties = t.record(
  rpcStorageSlotHexString,
  rpcStorageSlot
);

export const stateOverrideOptions = t.type(
  {
    balance: optionalOrNullable(rpcQuantity),
    nonce: optionalOrNullable(rpcQuantity),
    code: optionalOrNullable(rpcData),
    state: optionalOrNullable(stateProperties),
    stateDiff: optionalOrNullable(stateProperties),
  },
  "stateOverrideOptions"
);

export const stateOverrideSet = t.record(address, stateOverrideOptions);
export const optionalStateOverrideSet = optionalOrNullable(stateOverrideSet);

export type StateProperties = t.TypeOf<typeof stateProperties>;
export type StateOverrideOptions = t.TypeOf<typeof stateOverrideOptions>;
export type StateOverrideSet = t.TypeOf<typeof stateOverrideSet>;
export type OptionalStateOverrideSet = t.TypeOf<
  typeof optionalStateOverrideSet
>;
