import type {
  Addressable,
  BigNumberish,
  TransactionResponse,
  default as EthersT,
} from "ethers";
import type { BalanceChangeOptions } from "./misc/balance";

import { buildAssert } from "../utils";
import { ensure } from "./calledOnContract/utils";
import { getAddressOf } from "./misc/account";
import { CHANGE_ETHER_BALANCE_MATCHER } from "./constants";
import { assertIsNotNull, preventAsyncMatcherChaining } from "./utils";

export function supportChangeEtherBalance(
  Assertion: Chai.AssertionStatic,
  chaiUtils: Chai.ChaiUtils
) {
  Assertion.addMethod(
    CHANGE_ETHER_BALANCE_MATCHER,
    function (
      this: any,
      account: Addressable | string,
      balanceChange: BigNumberish | ((change: bigint) => boolean),
      options?: BalanceChangeOptions
    ) {
      const { toBigInt } = require("ethers") as typeof EthersT;
      // capture negated flag before async code executes; see buildAssert's jsdoc
      const negated = this.__flags.negate;
      const subject = this._obj;

      preventAsyncMatcherChaining(
        this,
        CHANGE_ETHER_BALANCE_MATCHER,
        chaiUtils
      );

      const checkBalanceChange = ([actualChange, address]: [
        bigint,
        string
      ]) => {
        const assert = buildAssert(negated, checkBalanceChange);

        if (typeof balanceChange === "function") {
          assert(
            balanceChange(actualChange),
            `Expected the ether balance change of "${address}" to satisfy the predicate, but it didn't (balance change: ${actualChange.toString()} wei)`,
            `Expected the ether balance change of "${address}" to NOT satisfy the predicate, but it did (balance change: ${actualChange.toString()} wei)`
          );
        } else {
          const expectedChange = toBigInt(balanceChange);
          assert(
            actualChange === expectedChange,
            `Expected the ether balance of "${address}" to change by ${balanceChange.toString()} wei, but it changed by ${actualChange.toString()} wei`,
            `Expected the ether balance of "${address}" NOT to change by ${balanceChange.toString()} wei, but it did`
          );
        }
      };

      const derivedPromise = Promise.all([
        getBalanceChange(subject, account, options),
        getAddressOf(account),
      ]).then(checkBalanceChange);
      this.then = derivedPromise.then.bind(derivedPromise);
      this.catch = derivedPromise.catch.bind(derivedPromise);
      this.promise = derivedPromise;
      return this;
    }
  );
}

export async function getBalanceChange(
  transaction:
    | TransactionResponse
    | Promise<TransactionResponse>
    | (() => Promise<TransactionResponse> | TransactionResponse),
  account: Addressable | string,
  options?: BalanceChangeOptions
): Promise<bigint> {
  const hre = await import("hardhat");
  const provider = hre.network.provider;

  let txResponse: TransactionResponse;

  if (typeof transaction === "function") {
    txResponse = await transaction();
  } else {
    txResponse = await transaction;
  }

  const txReceipt = await txResponse.wait();
  assertIsNotNull(txReceipt, "txReceipt");
  const txBlockNumber = txReceipt.blockNumber;

  const block = await provider.send("eth_getBlockByHash", [
    txReceipt.blockHash,
    false,
  ]);

  ensure(
    block.transactions.length === 1,
    Error,
    "Multiple transactions found in block"
  );

  const address = await getAddressOf(account);

  const balanceAfterHex = await provider.send("eth_getBalance", [
    address,
    `0x${txBlockNumber.toString(16)}`,
  ]);

  const balanceBeforeHex = await provider.send("eth_getBalance", [
    address,
    `0x${(txBlockNumber - 1).toString(16)}`,
  ]);

  const balanceAfter = BigInt(balanceAfterHex);
  const balanceBefore = BigInt(balanceBeforeHex);

  if (options?.includeFee !== true && address === txResponse.from) {
    const gasPrice = txReceipt.gasPrice;
    const gasUsed = txReceipt.gasUsed;
    const txFee = gasPrice * gasUsed;

    return balanceAfter + txFee - balanceBefore;
  } else {
    return balanceAfter - balanceBefore;
  }
}
