import type EthersT from "ethers";
import type OrdinalT from "ordinal";

import { AssertWithSsfi, Ssfi } from "../utils";
import { PREVIOUS_MATCHER_NAME } from "./constants";
import {
  HardhatChaiMatchersAssertionError,
  HardhatChaiMatchersNonChainableMatcherError,
} from "./errors";

export function assertIsNotNull<T>(
  value: T,
  valueName: string
): asserts value is Exclude<T, null> {
  if (value === null) {
    throw new HardhatChaiMatchersAssertionError(
      `${valueName} should not be null`
    );
  }
}

export function preventAsyncMatcherChaining(
  context: object,
  matcherName: string,
  chaiUtils: Chai.ChaiUtils,
  allowSelfChaining: boolean = false
) {
  const previousMatcherName: string | undefined = chaiUtils.flag(
    context,
    PREVIOUS_MATCHER_NAME
  );

  if (previousMatcherName === undefined) {
    chaiUtils.flag(context, PREVIOUS_MATCHER_NAME, matcherName);
    return;
  }

  if (previousMatcherName === matcherName && allowSelfChaining) {
    return;
  }

  throw new HardhatChaiMatchersNonChainableMatcherError(
    matcherName,
    previousMatcherName
  );
}

export function assertArgsArraysEqual(
  Assertion: Chai.AssertionStatic,
  expectedArgs: any[],
  actualArgs: any[],
  tag: string,
  assertionType: "event" | "error",
  assert: AssertWithSsfi,
  ssfi: Ssfi
) {
  try {
    innerAssertArgsArraysEqual(
      Assertion,
      expectedArgs,
      actualArgs,
      assertionType,
      assert,
      ssfi
    );
  } catch (err: any) {
    err.message = `Error in ${tag}: ${err.message}`;
    throw err;
  }
}

function innerAssertArgsArraysEqual(
  Assertion: Chai.AssertionStatic,
  expectedArgs: any[],
  actualArgs: any[],
  assertionType: "event" | "error",
  assert: AssertWithSsfi,
  ssfi: Ssfi
) {
  assert(
    actualArgs.length === expectedArgs.length,
    `Expected arguments array to have length ${expectedArgs.length}, but it has ${actualArgs.length}`
  );
  for (const [index, expectedArg] of expectedArgs.entries()) {
    try {
      innerAssertArgEqual(
        Assertion,
        expectedArg,
        actualArgs[index],
        assertionType,
        assert,
        ssfi
      );
    } catch (err: any) {
      const ordinal = require("ordinal") as typeof OrdinalT;
      err.message = `Error in the ${ordinal(index + 1)} argument assertion: ${
        err.message
      }`;
      throw err;
    }
  }
}

function innerAssertArgEqual(
  Assertion: Chai.AssertionStatic,
  expectedArg: any,
  actualArg: any,
  assertionType: "event" | "error",
  assert: AssertWithSsfi,
  ssfi: Ssfi
) {
  const ethers = require("ethers") as typeof EthersT;
  if (typeof expectedArg === "function") {
    try {
      if (expectedArg(actualArg) === true) return;
    } catch (e: any) {
      assert(
        false,
        `The predicate threw when called: ${e.message}`
        // no need for a negated message, since we disallow mixing .not. with
        // .withArgs
      );
    }
    assert(
      false,
      `The predicate did not return true`
      // no need for a negated message, since we disallow mixing .not. with
      // .withArgs
    );
  } else if (expectedArg instanceof Uint8Array) {
    new Assertion(actualArg, undefined, ssfi, true).equal(
      ethers.hexlify(expectedArg)
    );
  } else if (
    expectedArg?.length !== undefined &&
    typeof expectedArg !== "string"
  ) {
    innerAssertArgsArraysEqual(
      Assertion,
      expectedArg,
      actualArg,
      assertionType,
      assert,
      ssfi
    );
  } else {
    if (actualArg.hash !== undefined && actualArg._isIndexed === true) {
      if (assertionType !== "event") {
        throw new Error(
          "Should not get an indexed event when the assertion type is not event. Please open an issue about this."
        );
      }

      new Assertion(actualArg.hash, undefined, ssfi, true).to.not.equal(
        expectedArg,
        "The actual value was an indexed and hashed value of the event argument. The expected value provided to the assertion should be the actual event argument (the pre-image of the hash). You provided the hash itself. Please supply the actual event argument (the pre-image of the hash) instead."
      );
      const expectedArgBytes = ethers.isHexString(expectedArg)
        ? ethers.getBytes(expectedArg)
        : ethers.toUtf8Bytes(expectedArg);
      const expectedHash = ethers.keccak256(expectedArgBytes);
      new Assertion(actualArg.hash, undefined, ssfi, true).to.equal(
        expectedHash,
        `The actual value was an indexed and hashed value of the event argument. The expected value provided to the assertion was hashed to produce ${expectedHash}. The actual hash and the expected hash ${actualArg.hash} did not match`
      );
    } else {
      new Assertion(actualArg, undefined, ssfi, true).equal(expectedArg);
    }
  }
}
