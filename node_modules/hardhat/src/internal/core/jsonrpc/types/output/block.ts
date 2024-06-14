import * as t from "io-ts";

import { nullable, optional } from "../../../../util/io-ts";
import { rpcAddress, rpcData, rpcHash, rpcQuantity } from "../base-types";

import { rpcTransaction } from "./transaction";

const rpcWithdrawalItem = t.type(
  {
    index: rpcQuantity,
    validatorIndex: rpcQuantity,
    address: rpcAddress,
    amount: rpcQuantity,
  },
  "RpcBlockWithdrawalItem"
);

const baseBlockResponse = {
  number: nullable(rpcQuantity),
  hash: nullable(rpcHash),
  parentHash: rpcHash,
  nonce: optional(rpcData),
  sha3Uncles: rpcHash,
  logsBloom: rpcData,
  transactionsRoot: rpcHash,
  stateRoot: rpcHash,
  receiptsRoot: rpcHash,
  miner: rpcAddress,
  difficulty: rpcQuantity,
  totalDifficulty: rpcQuantity,
  extraData: rpcData,
  size: rpcQuantity,
  gasLimit: rpcQuantity,
  gasUsed: rpcQuantity,
  timestamp: rpcQuantity,
  uncles: t.array(rpcHash, "HASH Array"),
  mixHash: optional(rpcHash),
  baseFeePerGas: optional(rpcQuantity),
  withdrawals: optional(t.array(rpcWithdrawalItem)),
  withdrawalsRoot: optional(rpcHash),
  parentBeaconBlockRoot: optional(rpcHash),
  blobGasUsed: optional(rpcQuantity),
  excessBlobGas: optional(rpcQuantity),
};

export type RpcBlock = t.TypeOf<typeof rpcBlock>;

export const rpcBlock = t.type(
  {
    ...baseBlockResponse,
    transactions: t.array(rpcHash, "HASH Array"),
  },
  "RpcBlock"
);

export type RpcBlockWithTransactions = t.TypeOf<
  typeof rpcBlockWithTransactions
>;

export const rpcBlockWithTransactions = t.type(
  {
    ...baseBlockResponse,
    transactions: t.array(rpcTransaction, "RpcTransaction Array"),
  },
  "RpcBlockWithTransactions"
);
