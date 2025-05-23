import type EthersT from "ethers";

import { buildAssert } from "../../utils";
import { REVERTED_MATCHER } from "../constants";
import { assertIsNotNull, preventAsyncMatcherChaining } from "../utils";
import {
  decodeReturnData,
  getReturnDataFromError,
  parseBytes32String,
} from "./utils";

export function supportReverted(
  Assertion: Chai.AssertionStatic,
  chaiUtils: Chai.ChaiUtils
) {
  Assertion.addProperty(REVERTED_MATCHER, function (this: any) {
    // capture negated flag before async code executes; see buildAssert's jsdoc
    const negated = this.__flags.negate;

    const subject: unknown = this._obj;

    preventAsyncMatcherChaining(this, REVERTED_MATCHER, chaiUtils);

    // Check if the received value can be linked to a transaction, and then
    // get the receipt of that transaction and check its status.
    //
    // If the value doesn't correspond to a transaction, then the `reverted`
    // assertion is false.
    const onSuccess = async (value: unknown) => {
      const assert = buildAssert(negated, onSuccess);

      if (isTransactionResponse(value) || typeof value === "string") {
        const hash = typeof value === "string" ? value : value.hash;

        if (!isValidTransactionHash(hash)) {
          throw new TypeError(
            `Expected a valid transaction hash, but got '${hash}'`
          );
        }

        const receipt = await getTransactionReceipt(hash);

        if (receipt === null) {
          // If the receipt is null, maybe the string is a bytes32 string
          if (isBytes32String(hash)) {
            assert(false, "Expected transaction to be reverted");
            return;
          }
        }

        assertIsNotNull(receipt, "receipt");
        assert(
          receipt.status === 0,
          "Expected transaction to be reverted",
          "Expected transaction NOT to be reverted"
        );
      } else if (isTransactionReceipt(value)) {
        const receipt = value;

        assert(
          receipt.status === 0,
          "Expected transaction to be reverted",
          "Expected transaction NOT to be reverted"
        );
      } else {
        // If the subject of the assertion is not connected to a transaction
        // (hash, receipt, etc.), then the assertion fails.
        // Since we use `false` here, this means that `.not.to.be.reverted`
        // assertions will pass instead of always throwing a validation error.
        // This allows users to do things like:
        //   `expect(c.callStatic.f()).to.not.be.reverted`
        assert(false, "Expected transaction to be reverted");
      }
    };

    const onError = (error: any) => {
      const { toBeHex } = require("ethers") as typeof EthersT;
      const assert = buildAssert(negated, onError);
      const returnData = getReturnDataFromError(error);
      const decodedReturnData = decodeReturnData(returnData);

      if (
        decodedReturnData.kind === "Empty" ||
        decodedReturnData.kind === "Custom"
      ) {
        // in the negated case, if we can't decode the reason, we just indicate
        // that the transaction didn't revert
        assert(true, undefined, `Expected transaction NOT to be reverted`);
      } else if (decodedReturnData.kind === "Error") {
        assert(
          true,
          undefined,
          `Expected transaction NOT to be reverted, but it reverted with reason '${decodedReturnData.reason}'`
        );
      } else if (decodedReturnData.kind === "Panic") {
        assert(
          true,
          undefined,
          `Expected transaction NOT to be reverted, but it reverted with panic code ${toBeHex(
            decodedReturnData.code
          )} (${decodedReturnData.description})`
        );
      } else {
        const _exhaustiveCheck: never = decodedReturnData;
      }
    };

    // we use `Promise.resolve(subject)` so we can process both values and
    // promises of values in the same way
    const derivedPromise = Promise.resolve(subject).then(onSuccess, onError);

    this.then = derivedPromise.then.bind(derivedPromise);
    this.catch = derivedPromise.catch.bind(derivedPromise);

    return this;
  });
}

async function getTransactionReceipt(hash: string) {
  const hre = await import("hardhat");

  return hre.ethers.provider.getTransactionReceipt(hash);
}

function isTransactionResponse(x: unknown): x is { hash: string } {
  if (typeof x === "object" && x !== null) {
    return "hash" in x;
  }

  return false;
}

function isTransactionReceipt(x: unknown): x is { status: number } {
  if (typeof x === "object" && x !== null && "status" in x) {
    const status = (x as any).status;

    // this means we only support ethers's receipts for now; adding support for
    // raw receipts, where the status is an hexadecimal string, should be easy
    // and we can do it if there's demand for that
    return typeof status === "number";
  }

  return false;
}

function isValidTransactionHash(x: string): boolean {
  return /0x[0-9a-fA-F]{64}/.test(x);
}

function isBytes32String(v: string): boolean {
  try {
    parseBytes32String(v);
    return true;
  } catch {
    return false;
  }
}
