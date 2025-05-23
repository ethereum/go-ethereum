import type { BigNumber as EthersBigNumberType } from "ethers-v5";
// eslint-disable-next-line import/no-extraneous-dependencies
import type { default as BigNumberJsType } from "bignumber.js";
// eslint-disable-next-line import/no-extraneous-dependencies
import type { default as BNType } from "bn.js";

import { HardhatError } from "../internal/core/errors";
import { ERRORS } from "../internal/core/errors-list";

export function normalizeToBigInt(
  source:
    | number
    | bigint
    | BNType
    | EthersBigNumberType
    | BigNumberJsType
    | string
): bigint {
  switch (typeof source) {
    case "object":
      if (isBigNumber(source)) {
        return BigInt(source.toString());
      } else {
        throw new HardhatError(ERRORS.GENERAL.INVALID_BIG_NUMBER, {
          message: `Value ${JSON.stringify(
            source
          )} is of type "object" but is not an instanceof one of the known big number object types.`,
        });
      }
    case "number":
      if (!Number.isInteger(source)) {
        throw new HardhatError(ERRORS.GENERAL.INVALID_BIG_NUMBER, {
          message: `${source} is not an integer`,
        });
      }
      if (!Number.isSafeInteger(source)) {
        throw new HardhatError(ERRORS.GENERAL.INVALID_BIG_NUMBER, {
          message: `Integer ${source} is unsafe. Consider using ${source}n instead. For more details, see https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Number/isSafeInteger`,
        });
      }
    // `break;` intentionally omitted. fallthrough desired.
    case "string":
    case "bigint":
      return BigInt(source);
    default:
      const _exhaustiveCheck: never = source;
      throw new HardhatError(ERRORS.GENERAL.INVALID_BIG_NUMBER, {
        message: `Unsupported type ${typeof source}`,
      });
  }
}

export function isBigNumber(source: any): boolean {
  return (
    typeof source === "bigint" ||
    isEthersBigNumber(source) ||
    isBN(source) ||
    isBigNumberJsBigNumber(source)
  );
}

function isBN(n: any) {
  try {
    // eslint-disable-next-line import/no-extraneous-dependencies
    const BN: typeof BNType = require("bn.js");
    return BN.isBN(n);
  } catch (e) {
    return false;
  }
}

function isEthersBigNumber(n: any) {
  try {
    const BigNumber: typeof EthersBigNumberType =
      // eslint-disable-next-line import/no-extraneous-dependencies
      require("ethers").ethers.BigNumber;
    return BigNumber.isBigNumber(n);
  } catch (e) {
    return false;
  }
}

function isBigNumberJsBigNumber(n: any) {
  try {
    // eslint-disable-next-line import/no-extraneous-dependencies
    const BigNumber: typeof BigNumberJsType = require("bignumber.js").BigNumber;
    return BigNumber.isBigNumber(n);
  } catch (e) {
    return false;
  }
}

export function formatNumberType(
  n: string | bigint | BNType | EthersBigNumberType | BigNumberJsType
): string {
  if (typeof n === "object") {
    if (isBN(n)) {
      return "BN";
    } else if (isEthersBigNumber(n)) {
      return "ethers.BigNumber";
    } else if (isBigNumberJsBigNumber(n)) {
      return "bignumber.js";
    }
  }
  return typeof n;
}
