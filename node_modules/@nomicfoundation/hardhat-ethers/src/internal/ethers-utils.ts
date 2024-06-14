// these helpers functions were copied verbatim from ethers

import type {
  TransactionRequest,
  PreparedTransactionRequest,
  BlockParams,
  TransactionResponseParams,
  TransactionReceiptParams,
  LogParams,
  JsonRpcTransactionRequest,
} from "ethers";

import {
  accessListify,
  assert,
  assertArgument,
  getAddress,
  getBigInt,
  getCreateAddress,
  getNumber,
  hexlify,
  isHexString,
  Signature,
  toQuantity,
} from "ethers";
import { HardhatEthersError } from "./errors";

export type FormatFunc = (value: any) => any;

export function copyRequest(
  req: TransactionRequest
): PreparedTransactionRequest {
  const result: any = {};

  // These could be addresses, ENS names or Addressables
  if (req.to !== null && req.to !== undefined) {
    result.to = req.to;
  }
  if (req.from !== null && req.from !== undefined) {
    result.from = req.from;
  }

  if (req.data !== null && req.data !== undefined) {
    result.data = hexlify(req.data);
  }

  const bigIntKeys =
    "chainId,gasLimit,gasPrice,maxFeePerGas,maxPriorityFeePerGas,value".split(
      /,/
    );
  for (const key of bigIntKeys) {
    if (
      !(key in req) ||
      (req as any)[key] === null ||
      (req as any)[key] === undefined
    ) {
      continue;
    }
    result[key] = getBigInt((req as any)[key], `request.${key}`);
  }

  const numberKeys = "type,nonce".split(/,/);
  for (const key of numberKeys) {
    if (
      !(key in req) ||
      (req as any)[key] === null ||
      (req as any)[key] === undefined
    ) {
      continue;
    }
    result[key] = getNumber((req as any)[key], `request.${key}`);
  }

  if (req.accessList !== null && req.accessList !== undefined) {
    result.accessList = accessListify(req.accessList);
  }

  if ("blockTag" in req) {
    result.blockTag = req.blockTag;
  }

  if ("enableCcipRead" in req) {
    result.enableCcipReadEnabled = Boolean(req.enableCcipRead);
  }

  if ("customData" in req) {
    result.customData = req.customData;
  }

  return result;
}

export async function resolveProperties<T>(value: {
  [P in keyof T]: T[P] | Promise<T[P]>;
}): Promise<T> {
  const keys = Object.keys(value);
  const results = await Promise.all(
    keys.map((k) => Promise.resolve(value[k as keyof T]))
  );
  return results.reduce((accum: any, v, index) => {
    accum[keys[index]] = v;
    return accum;
  }, {} as { [P in keyof T]: T[P] });
}

export function formatBlock(value: any): BlockParams {
  const result = _formatBlock(value);
  result.transactions = value.transactions.map(
    (tx: string | TransactionResponseParams) => {
      if (typeof tx === "string") {
        return tx;
      }
      return formatTransactionResponse(tx);
    }
  );
  return result;
}

const _formatBlock = object({
  hash: allowNull(formatHash),
  parentHash: formatHash,
  number: getNumber,

  timestamp: getNumber,
  nonce: allowNull(formatData),
  difficulty: getBigInt,

  gasLimit: getBigInt,
  gasUsed: getBigInt,

  miner: allowNull(getAddress),
  extraData: formatData,

  baseFeePerGas: allowNull(getBigInt),
});

function object(
  format: Record<string, FormatFunc>,
  altNames?: Record<string, string[]>
): FormatFunc {
  return (value: any) => {
    const result: any = {};
    // eslint-disable-next-line guard-for-in
    for (const key in format) {
      let srcKey = key;
      if (altNames !== undefined && key in altNames && !(srcKey in value)) {
        for (const altKey of altNames[key]) {
          if (altKey in value) {
            srcKey = altKey;
            break;
          }
        }
      }

      try {
        const nv = format[key](value[srcKey]);
        if (nv !== undefined) {
          result[key] = nv;
        }
      } catch (error) {
        const message = error instanceof Error ? error.message : "not-an-error";
        assert(
          false,
          `invalid value for value.${key} (${message})`,
          "BAD_DATA",
          { value }
        );
      }
    }
    return result;
  };
}

function allowNull(format: FormatFunc, nullValue?: any): FormatFunc {
  return function (value: any) {
    // eslint-disable-next-line eqeqeq
    if (value === null || value === undefined) {
      return nullValue;
    }
    return format(value);
  };
}

function formatHash(value: any): string {
  assertArgument(isHexString(value, 32), "invalid hash", "value", value);
  return value;
}

function formatData(value: string): string {
  assertArgument(isHexString(value, true), "invalid data", "value", value);
  return value;
}

export function formatTransactionResponse(
  value: any
): TransactionResponseParams {
  // Some clients (TestRPC) do strange things like return 0x0 for the
  // 0 address; correct this to be a real address
  // eslint-disable-next-line @typescript-eslint/strict-boolean-expressions
  if (value.to && getBigInt(value.to) === 0n) {
    value.to = "0x0000000000000000000000000000000000000000";
  }

  const result = object(
    {
      hash: formatHash,

      type: (v: any) => {
        // eslint-disable-next-line eqeqeq
        if (v === "0x" || v == null) {
          return 0;
        }
        return getNumber(v);
      },
      accessList: allowNull(accessListify, null),

      blockHash: allowNull(formatHash, null),
      blockNumber: allowNull(getNumber, null),
      transactionIndex: allowNull(getNumber, null),

      from: getAddress,

      // either (gasPrice) or (maxPriorityFeePerGas + maxFeePerGas) must be set
      gasPrice: allowNull(getBigInt),
      maxPriorityFeePerGas: allowNull(getBigInt),
      maxFeePerGas: allowNull(getBigInt),

      gasLimit: getBigInt,
      to: allowNull(getAddress, null),
      value: getBigInt,
      nonce: getNumber,
      data: formatData,

      creates: allowNull(getAddress, null),

      chainId: allowNull(getBigInt, null),
    },
    {
      data: ["input"],
      gasLimit: ["gas"],
    }
  )(value);

  // If to and creates are empty, populate the creates from the value
  // eslint-disable-next-line eqeqeq
  if (result.to == null && result.creates == null) {
    result.creates = getCreateAddress(result);
  }

  // @TODO: Check fee data

  // Add an access list to supported transaction types
  // eslint-disable-next-line eqeqeq
  if ((value.type === 1 || value.type === 2) && value.accessList == null) {
    result.accessList = [];
  }

  // Compute the signature
  // eslint-disable-next-line @typescript-eslint/strict-boolean-expressions
  if (value.signature) {
    result.signature = Signature.from(value.signature);
  } else {
    result.signature = Signature.from(value);
  }

  // Some backends omit ChainId on legacy transactions, but we can compute it
  // eslint-disable-next-line eqeqeq
  if (result.chainId == null) {
    const chainId = result.signature.legacyChainId;
    // eslint-disable-next-line eqeqeq
    if (chainId != null) {
      result.chainId = chainId;
    }
  }

  // 0x0000... should actually be null
  // eslint-disable-next-line @typescript-eslint/strict-boolean-expressions
  if (result.blockHash && getBigInt(result.blockHash) === 0n) {
    result.blockHash = null;
  }

  return result;
}

function arrayOf(format: FormatFunc): FormatFunc {
  return (array: any) => {
    if (!Array.isArray(array)) {
      throw new HardhatEthersError("not an array");
    }
    return array.map((i) => format(i));
  };
}

const _formatReceiptLog = object(
  {
    transactionIndex: getNumber,
    blockNumber: getNumber,
    transactionHash: formatHash,
    address: getAddress,
    topics: arrayOf(formatHash),
    data: formatData,
    index: getNumber,
    blockHash: formatHash,
  },
  {
    index: ["logIndex"],
  }
);

const _formatTransactionReceipt = object(
  {
    to: allowNull(getAddress, null),
    from: allowNull(getAddress, null),
    contractAddress: allowNull(getAddress, null),
    // should be allowNull(hash), but broken-EIP-658 support is handled in receipt
    index: getNumber,
    root: allowNull(hexlify),
    gasUsed: getBigInt,
    logsBloom: allowNull(formatData),
    blockHash: formatHash,
    hash: formatHash,
    logs: arrayOf(formatReceiptLog),
    blockNumber: getNumber,
    cumulativeGasUsed: getBigInt,
    effectiveGasPrice: allowNull(getBigInt),
    status: allowNull(getNumber),
    type: allowNull(getNumber, 0),
  },
  {
    effectiveGasPrice: ["gasPrice"],
    hash: ["transactionHash"],
    index: ["transactionIndex"],
  }
);

export function formatTransactionReceipt(value: any): TransactionReceiptParams {
  return _formatTransactionReceipt(value);
}

export function formatReceiptLog(value: any): LogParams {
  return _formatReceiptLog(value);
}

function formatBoolean(value: any): boolean {
  switch (value) {
    case true:
    case "true":
      return true;
    case false:
    case "false":
      return false;
  }
  assertArgument(
    false,
    `invalid boolean; ${JSON.stringify(value)}`,
    "value",
    value
  );
}

const _formatLog = object(
  {
    address: getAddress,
    blockHash: formatHash,
    blockNumber: getNumber,
    data: formatData,
    index: getNumber,
    removed: formatBoolean,
    topics: arrayOf(formatHash),
    transactionHash: formatHash,
    transactionIndex: getNumber,
  },
  {
    index: ["logIndex"],
  }
);

export function formatLog(value: any): LogParams {
  return _formatLog(value);
}

export function getRpcTransaction(
  tx: TransactionRequest
): JsonRpcTransactionRequest {
  const result: JsonRpcTransactionRequest = {};

  // JSON-RPC now requires numeric values to be "quantity" values
  [
    "chainId",
    "gasLimit",
    "gasPrice",
    "type",
    "maxFeePerGas",
    "maxPriorityFeePerGas",
    "nonce",
    "value",
  ].forEach((key) => {
    if ((tx as any)[key] === null || (tx as any)[key] === undefined) {
      return;
    }
    let dstKey = key;
    if (key === "gasLimit") {
      dstKey = "gas";
    }
    (result as any)[dstKey] = toQuantity(
      getBigInt((tx as any)[key], `tx.${key}`)
    );
  });

  // Make sure addresses and data are lowercase
  ["from", "to", "data"].forEach((key) => {
    if ((tx as any)[key] === null || (tx as any)[key] === undefined) {
      return;
    }
    (result as any)[key] = hexlify((tx as any)[key]);
  });

  // Normalize the access list object
  if (tx.accessList !== null && tx.accessList !== undefined) {
    result.accessList = accessListify(tx.accessList);
  }

  return result;
}
