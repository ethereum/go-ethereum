import type EthersT from "ethers";
import type { Contract, Interface, Transaction } from "ethers";
import type { AssertWithSsfi, Ssfi } from "../utils";

import { AssertionError } from "chai";
import util from "util";

import { buildAssert } from "../utils";
import { ASSERTION_ABORTED, EMIT_MATCHER } from "./constants";
import { HardhatChaiMatchersAssertionError } from "./errors";
import {
  assertArgsArraysEqual,
  assertIsNotNull,
  preventAsyncMatcherChaining,
} from "./utils";

type EventFragment = EthersT.EventFragment;
type Provider = EthersT.Provider;

export const EMIT_CALLED = "emitAssertionCalled";

async function waitForPendingTransaction(
  tx: Promise<Transaction> | Transaction | string,
  provider: Provider
) {
  let hash: string | null;
  if (tx instanceof Promise) {
    ({ hash } = await tx);
  } else if (typeof tx === "string") {
    hash = tx;
  } else {
    ({ hash } = tx);
  }
  if (hash === null) {
    throw new Error(`${JSON.stringify(tx)} is not a valid transaction`);
  }
  return provider.getTransactionReceipt(hash);
}

export function supportEmit(
  Assertion: Chai.AssertionStatic,
  chaiUtils: Chai.ChaiUtils
) {
  Assertion.addMethod(
    EMIT_MATCHER,
    function (
      this: any,
      contract: Contract,
      eventName: string,
      ...args: any[]
    ) {
      // capture negated flag before async code executes; see buildAssert's jsdoc
      const negated = this.__flags.negate;
      const tx = this._obj;

      preventAsyncMatcherChaining(this, EMIT_MATCHER, chaiUtils, true);

      const promise = this.then === undefined ? Promise.resolve() : this;

      const onSuccess = (receipt: EthersT.TransactionReceipt) => {
        // abort if the assertion chain was aborted, for example because
        // a `.not` was combined with a `.withArgs`
        if (chaiUtils.flag(this, ASSERTION_ABORTED) === true) {
          return;
        }

        const assert = buildAssert(negated, onSuccess);

        let eventFragment: EventFragment | null = null;
        try {
          eventFragment = contract.interface.getEvent(eventName, []);
        } catch (e: unknown) {
          if (e instanceof TypeError) {
            const errorMessage = e.message.split(" (argument=")[0];
            throw new AssertionError(errorMessage);
          }
        }

        if (eventFragment === null) {
          throw new AssertionError(
            `Event "${eventName}" doesn't exist in the contract`
          );
        }

        const topic = eventFragment.topicHash;
        const contractAddress = contract.target;
        if (typeof contractAddress !== "string") {
          throw new HardhatChaiMatchersAssertionError(
            `The contract target should be a string`
          );
        }

        if (args.length > 0) {
          throw new Error(
            "`.emit` expects only two arguments: the contract and the event name. Arguments should be asserted with the `.withArgs` helper."
          );
        }

        this.logs = receipt.logs
          .filter((log) => log.topics.includes(topic))
          .filter(
            (log) => log.address.toLowerCase() === contractAddress.toLowerCase()
          );

        assert(
          this.logs.length > 0,
          `Expected event "${eventName}" to be emitted, but it wasn't`,
          `Expected event "${eventName}" NOT to be emitted, but it was`
        );
        chaiUtils.flag(this, "eventName", eventName);
        chaiUtils.flag(this, "contract", contract);
      };

      const derivedPromise = promise.then(() => {
        // abort if the assertion chain was aborted, for example because
        // a `.not` was combined with a `.withArgs`
        if (chaiUtils.flag(this, ASSERTION_ABORTED) === true) {
          return;
        }

        if (contract.runner === null || contract.runner.provider === null) {
          throw new HardhatChaiMatchersAssertionError(
            "contract.runner.provider shouldn't be null"
          );
        }

        return waitForPendingTransaction(tx, contract.runner.provider).then(
          (receipt) => {
            assertIsNotNull(receipt, "receipt");
            return onSuccess(receipt);
          }
        );
      });

      chaiUtils.flag(this, EMIT_CALLED, true);

      this.then = derivedPromise.then.bind(derivedPromise);
      this.catch = derivedPromise.catch.bind(derivedPromise);
      this.promise = derivedPromise;
      return this;
    }
  );
}

export async function emitWithArgs(
  context: any,
  Assertion: Chai.AssertionStatic,
  chaiUtils: Chai.ChaiUtils,
  expectedArgs: any[],
  ssfi: Ssfi
) {
  const negated = false; // .withArgs cannot be negated
  const assert = buildAssert(negated, ssfi);

  tryAssertArgsArraysEqual(
    context,
    Assertion,
    chaiUtils,
    expectedArgs,
    context.logs,
    assert,
    ssfi
  );
}

const tryAssertArgsArraysEqual = (
  context: any,
  Assertion: Chai.AssertionStatic,
  chaiUtils: Chai.ChaiUtils,
  expectedArgs: any[],
  logs: any[],
  assert: AssertWithSsfi,
  ssfi: Ssfi
) => {
  const eventName = chaiUtils.flag(context, "eventName");
  if (logs.length === 1) {
    const parsedLog = (
      chaiUtils.flag(context, "contract").interface as Interface
    ).parseLog(logs[0]);
    assertIsNotNull(parsedLog, "parsedLog");

    return assertArgsArraysEqual(
      Assertion,
      expectedArgs,
      parsedLog.args,
      `"${eventName}" event`,
      "event",
      assert,
      ssfi
    );
  }
  for (const index in logs) {
    if (index === undefined) {
      break;
    } else {
      try {
        const parsedLog = (
          chaiUtils.flag(context, "contract").interface as Interface
        ).parseLog(logs[index]);
        assertIsNotNull(parsedLog, "parsedLog");

        assertArgsArraysEqual(
          Assertion,
          expectedArgs,
          parsedLog.args,
          `"${eventName}" event`,
          "event",
          assert,
          ssfi
        );
        return;
      } catch {}
    }
  }

  assert(
    false,
    `The specified arguments (${util.inspect(
      expectedArgs
    )}) were not included in any of the ${
      context.logs.length
    } emitted "${eventName}" events`
  );
};
