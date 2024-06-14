import { assertHardhatInvariant } from "../core/errors";

export function min(x: bigint, y: bigint): bigint {
  return x < y ? x : y;
}

export function max(x: bigint, y: bigint): bigint {
  return x > y ? x : y;
}

export function isBigInt(x: unknown): x is bigint {
  return typeof x === "bigint";
}

export function divUp(x: bigint, y: bigint): bigint {
  let result = x / y;

  if (x % y !== 0n) {
    result = result + 1n;
  }

  return result;
}

export function cmp(a: bigint, b: bigint): number {
  return a < b ? -1 : a > b ? 1 : 0;
}

/**
 * Converts the number to a hexadecimal string with a length of 32
 * bytes. This hex string is NOT 0x-prefixed.
 */
export function toEvmWord(x: bigint | number): string {
  return x.toString(16).padStart(64, "0");
}

function bufferToBigInt(x: Uint8Array): bigint {
  const hex = `0x${Buffer.from(x).toString("hex")}`;
  return hex === "0x" ? 0n : BigInt(hex);
}

export function fromBigIntLike(
  x: string | number | bigint | Uint8Array | undefined
): bigint | undefined {
  if (x === undefined || typeof x === "bigint") {
    return x;
  }
  if (typeof x === "number" || typeof x === "string") {
    return BigInt(x);
  }
  if (x instanceof Uint8Array) {
    return bufferToBigInt(x);
  }

  const exhaustiveCheck: never = x;
  return exhaustiveCheck;
}

export function toHex(x: number | bigint): string {
  assertHardhatInvariant(
    x >= 0,
    `toHex can only be used with non-negative numbers, but received ${x}`
  );

  return `0x${x.toString(16)}`;
}
