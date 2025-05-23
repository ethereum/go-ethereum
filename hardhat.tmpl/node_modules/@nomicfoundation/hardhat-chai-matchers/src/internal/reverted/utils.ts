import type EthersT from "ethers";

import { AssertionError } from "chai";

import { HardhatChaiMatchersDecodingError } from "../errors";
import { panicErrorCodeToReason } from "./panic";

// method id of 'Error(string)'
const ERROR_STRING_PREFIX = "0x08c379a0";

// method id of 'Panic(uint256)'
const PANIC_CODE_PREFIX = "0x4e487b71";

/**
 * Try to obtain the return data of a transaction from the given value.
 *
 * If the value is an error but it doesn't have data, we assume it's not related
 * to a reverted transaction and we re-throw it.
 */
export function getReturnDataFromError(error: any): string {
  if (!(error instanceof Error)) {
    throw new AssertionError("Expected an Error object");
  }

  // cast to any again so we don't have to cast it every time we access
  // some property that doesn't exist on Error
  error = error as any;

  const errorData = error.data ?? error.error?.data;

  if (errorData === undefined) {
    throw error;
  }

  const returnData = typeof errorData === "string" ? errorData : errorData.data;

  if (returnData === undefined || typeof returnData !== "string") {
    throw error;
  }

  return returnData;
}

type DecodedReturnData =
  | {
      kind: "Error";
      reason: string;
    }
  | {
      kind: "Empty";
    }
  | {
      kind: "Panic";
      code: bigint;
      description: string;
    }
  | {
      kind: "Custom";
      id: string;
      data: string;
    };

export function decodeReturnData(returnData: string): DecodedReturnData {
  const { AbiCoder } = require("ethers") as typeof EthersT;
  const abi = new AbiCoder();

  if (returnData === "0x") {
    return { kind: "Empty" };
  } else if (returnData.startsWith(ERROR_STRING_PREFIX)) {
    const encodedReason = returnData.slice(ERROR_STRING_PREFIX.length);
    let reason: string;
    try {
      reason = abi.decode(["string"], `0x${encodedReason}`)[0];
    } catch (e: any) {
      throw new HardhatChaiMatchersDecodingError(encodedReason, "string", e);
    }

    return {
      kind: "Error",
      reason,
    };
  } else if (returnData.startsWith(PANIC_CODE_PREFIX)) {
    const encodedReason = returnData.slice(PANIC_CODE_PREFIX.length);
    let code: bigint;
    try {
      code = abi.decode(["uint256"], `0x${encodedReason}`)[0];
    } catch (e: any) {
      throw new HardhatChaiMatchersDecodingError(encodedReason, "uint256", e);
    }

    const description = panicErrorCodeToReason(code) ?? "unknown panic code";

    return {
      kind: "Panic",
      code,
      description,
    };
  }

  return {
    kind: "Custom",
    id: returnData.slice(0, 10),
    data: `0x${returnData.slice(10)}`,
  };
}

/**
 * Takes an ethers result object and converts it into a (potentially nested) array.
 *
 * For example, given this error:
 *
 *   struct Point(uint x, uint y)
 *   error MyError(string, Point)
 *
 *   revert MyError("foo", Point(1, 2))
 *
 * The resulting array will be: ["foo", [1n, 2n]]
 */
export function resultToArray(result: EthersT.Result): any[] {
  return result
    .toArray()
    .map((x) =>
      typeof x === "object" && x !== null && "toArray" in x
        ? resultToArray(x)
        : x
    );
}

export function parseBytes32String(v: string): string {
  const ethers = require("ethers") as typeof EthersT;
  return ethers.decodeBytes32String(v);
}
