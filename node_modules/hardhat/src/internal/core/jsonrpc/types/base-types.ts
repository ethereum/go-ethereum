import {
  bytesToHex as bufferToHex,
  isValidAddress,
  toBytes,
} from "@nomicfoundation/ethereumjs-util";
import * as t from "io-ts";

import * as BigIntUtils from "../../../util/bigint";
import { assertHardhatInvariant, HardhatError } from "../../errors";
import { ERRORS } from "../../errors-list";

const ADDRESS_LENGTH_BYTES = 20;
const HASH_LENGTH_BYTES = 32;

export const rpcQuantity = new t.Type<bigint>(
  "QUANTITY",
  BigIntUtils.isBigInt,
  (u, c) => (isRpcQuantityString(u) ? t.success(BigInt(u)) : t.failure(u, c)),
  t.identity
);

export const rpcData = new t.Type<Buffer>(
  "DATA",
  Buffer.isBuffer,
  (u, c) =>
    isRpcDataString(u) ? t.success(Buffer.from(toBytes(u))) : t.failure(u, c),
  t.identity
);

export const rpcHash = new t.Type<Buffer>(
  "HASH",
  (v): v is Buffer => Buffer.isBuffer(v) && v.length === HASH_LENGTH_BYTES,
  (u, c) =>
    isRpcHashString(u) ? t.success(Buffer.from(toBytes(u))) : t.failure(u, c),
  t.identity
);

export const rpcStorageSlot = new t.Type<bigint>(
  "Storage slot",
  BigIntUtils.isBigInt,
  validateStorageSlot,
  t.identity
);

// This type is necessary because objects' keys need to be either strings or numbers to be properly handled by the 'io-ts' module.
// If they are not defined as strings or numbers, the type definition will result in an empty object without the required properties.
// For example, instead of displaying { ke1: value1 }, it will display {}
export const rpcStorageSlotHexString = new t.Type<string>(
  "Storage slot hex string",
  (x): x is string => typeof x === "string",
  (u, c) =>
    validateRpcStorageSlotHexString(u) ? t.success(u) : t.failure(u, c),
  t.identity
);

function validateStorageSlot(u: unknown, c: t.Context): t.Validation<bigint> {
  if (typeof u !== "string") {
    return t.failure(
      u,
      c,
      `Storage slot argument must be a string, got '${u as any}'`
    );
  }

  if (u === "") {
    return t.failure(u, c, "Storage slot argument cannot be an empty string");
  }

  if (u.startsWith("0x")) {
    if (u.length > 66) {
      return t.failure(
        u,
        c,
        `Storage slot argument must have a length of at most 66 ("0x" + 32 bytes), but '${u}' has a length of ${u.length}`
      );
    }
  } else {
    if (u.length > 64) {
      return t.failure(
        u,
        c,
        `Storage slot argument must have a length of at most 64 (32 bytes), but '${u}' has a length of ${u.length}`
      );
    }
  }

  if (u.match(/^(0x)?([0-9a-fA-F]){0,64}$/) === null) {
    return t.failure(
      u,
      c,
      `Storage slot argument must be a valid hexadecimal, got '${u}'`
    );
  }

  return t.success(u === "0x" ? 0n : BigInt(u.startsWith("0x") ? u : `0x${u}`));
}

export const rpcAddress = new t.Type<Buffer>(
  "ADDRESS",
  (v): v is Buffer => Buffer.isBuffer(v) && v.length === ADDRESS_LENGTH_BYTES,
  (u, c) =>
    isRpcAddressString(u)
      ? t.success(Buffer.from(toBytes(u)))
      : t.failure(u, c),
  t.identity
);

export const rpcUnsignedInteger = new t.Type<number>(
  "Unsigned integer",
  isInteger,
  (u, c) => (isInteger(u) && u >= 0 ? t.success(u) : t.failure(u, c)),
  t.identity
);

export const rpcQuantityAsNumber = new t.Type<bigint>(
  "Integer",
  BigIntUtils.isBigInt,
  (u, c) => (isInteger(u) ? t.success(BigInt(u)) : t.failure(u, c)),
  t.identity
);

export const rpcFloat = new t.Type<number>(
  "Float number",
  isNumber,
  (u, c) => (typeof u === "number" ? t.success(u) : t.failure(u, c)),
  t.identity
);

// Conversion functions

/**
 * Transforms a QUANTITY into a number. It should only be used if you are 100% sure that the value
 * fits in a number.
 */
export function rpcQuantityToNumber(quantity: string): number {
  return Number(rpcQuantityToBigInt(quantity));
}

export function rpcQuantityToBigInt(quantity: string): bigint {
  // We validate it in case a value gets here through a cast or any
  if (!isRpcQuantityString(quantity)) {
    throw new HardhatError(ERRORS.NETWORK.INVALID_RPC_QUANTITY_VALUE, {
      value: quantity,
    });
  }

  return BigInt(quantity);
}

export function numberToRpcQuantity(n: number | bigint): string {
  assertHardhatInvariant(
    typeof n === "number" || typeof n === "bigint",
    "Expected number"
  );

  return `0x${n.toString(16)}`;
}

export function numberToRpcStorageSlot(n: number | bigint): string {
  assertHardhatInvariant(
    typeof n === "number" || typeof n === "bigint",
    "Expected number"
  );

  return `0x${BigIntUtils.toEvmWord(n)}`;
}

/**
 * Transforms a DATA into a number. It should only be used if you are 100% sure that the data
 * represents a value fits in a number.
 */
export function rpcDataToNumber(data: string): number {
  return Number(rpcDataToBigInt(data));
}

export function rpcDataToBigInt(data: string): bigint {
  return data === "0x" ? 0n : BigInt(data);
}

export function bufferToRpcData(
  buffer: Uint8Array,
  padToBytes: number = 0
): string {
  let s = bufferToHex(buffer);
  if (padToBytes > 0 && s.length < padToBytes * 2 + 2) {
    s = `0x${"0".repeat(padToBytes * 2 + 2 - s.length)}${s.slice(2)}`;
  }
  return s;
}

export function rpcDataToBuffer(data: string): Buffer {
  // We validate it in case a value gets here through a cast or any
  if (!isRpcDataString(data)) {
    throw new HardhatError(ERRORS.NETWORK.INVALID_RPC_DATA_VALUE, {
      value: data,
    });
  }

  return Buffer.from(toBytes(data));
}

// Type guards

function validateRpcStorageSlotHexString(u: unknown): u is string {
  return typeof u === "string" && /^0x([0-9a-fA-F]){64}$/.test(u);
}

function isRpcQuantityString(u: unknown): u is string {
  return (
    typeof u === "string" &&
    u.match(/^0x(?:0|(?:[1-9a-fA-F][0-9a-fA-F]*))$/) !== null
  );
}

function isRpcDataString(u: unknown): u is string {
  return typeof u === "string" && u.match(/^0x(?:[0-9a-fA-F]{2})*$/) !== null;
}

function isRpcHashString(u: unknown): u is string {
  return typeof u === "string" && u.length === 66 && isRpcDataString(u);
}

function isRpcAddressString(u: unknown): u is string {
  return typeof u === "string" && isValidAddress(u);
}

function isInteger(num: unknown): num is number {
  return Number.isInteger(num);
}

function isNumber(num: unknown): num is number {
  return typeof num === "number";
}
