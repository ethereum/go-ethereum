import type EthersT from "ethers";
import type {
  Addressable,
  BaseContract,
  BaseContractMethod,
  BigNumberish,
  ContractTransactionResponse,
} from "ethers";

import { buildAssert } from "../utils";
import { ensure } from "./calledOnContract/utils";
import { getAddressOf } from "./misc/account";
import {
  CHANGE_TOKEN_BALANCES_MATCHER,
  CHANGE_TOKEN_BALANCE_MATCHER,
} from "./constants";
import { assertIsNotNull, preventAsyncMatcherChaining } from "./utils";

type TransactionResponse = EthersT.TransactionResponse;

export type Token = BaseContract & {
  balanceOf: BaseContractMethod<[string], bigint, bigint>;
  name: BaseContractMethod<[], string, string>;
  transfer: BaseContractMethod<
    [string, BigNumberish],
    boolean,
    ContractTransactionResponse
  >;
  symbol: BaseContractMethod<[], string, string>;
};

export function supportChangeTokenBalance(
  Assertion: Chai.AssertionStatic,
  chaiUtils: Chai.ChaiUtils
) {
  Assertion.addMethod(
    CHANGE_TOKEN_BALANCE_MATCHER,
    function (
      this: any,
      token: Token,
      account: Addressable | string,
      balanceChange: EthersT.BigNumberish | ((change: bigint) => boolean)
    ) {
      const ethers = require("ethers") as typeof EthersT;

      // capture negated flag before async code executes; see buildAssert's jsdoc
      const negated = this.__flags.negate;

      let subject = this._obj;
      if (typeof subject === "function") {
        subject = subject();
      }

      preventAsyncMatcherChaining(
        this,
        CHANGE_TOKEN_BALANCE_MATCHER,
        chaiUtils
      );

      checkToken(token, CHANGE_TOKEN_BALANCE_MATCHER);

      const checkBalanceChange = ([actualChange, address, tokenDescription]: [
        bigint,
        string,
        string
      ]) => {
        const assert = buildAssert(negated, checkBalanceChange);

        if (typeof balanceChange === "function") {
          assert(
            balanceChange(actualChange),
            `Expected the balance of ${tokenDescription} tokens for "${address}" to satisfy the predicate, but it didn't (token balance change: ${actualChange.toString()} wei)`,
            `Expected the balance of ${tokenDescription} tokens for "${address}" to NOT satisfy the predicate, but it did (token balance change: ${actualChange.toString()} wei)`
          );
        } else {
          assert(
            actualChange === ethers.toBigInt(balanceChange),
            `Expected the balance of ${tokenDescription} tokens for "${address}" to change by ${balanceChange.toString()}, but it changed by ${actualChange.toString()}`,
            `Expected the balance of ${tokenDescription} tokens for "${address}" NOT to change by ${balanceChange.toString()}, but it did`
          );
        }
      };

      const derivedPromise = Promise.all([
        getBalanceChange(subject, token, account),
        getAddressOf(account),
        getTokenDescription(token),
      ]).then(checkBalanceChange);

      this.then = derivedPromise.then.bind(derivedPromise);
      this.catch = derivedPromise.catch.bind(derivedPromise);

      return this;
    }
  );

  Assertion.addMethod(
    CHANGE_TOKEN_BALANCES_MATCHER,
    function (
      this: any,
      token: Token,
      accounts: Array<Addressable | string>,
      balanceChanges: EthersT.BigNumberish[] | ((changes: bigint[]) => boolean)
    ) {
      const ethers = require("ethers") as typeof EthersT;

      // capture negated flag before async code executes; see buildAssert's jsdoc
      const negated = this.__flags.negate;

      let subject = this._obj;
      if (typeof subject === "function") {
        subject = subject();
      }

      preventAsyncMatcherChaining(
        this,
        CHANGE_TOKEN_BALANCES_MATCHER,
        chaiUtils
      );

      validateInput(this._obj, token, accounts, balanceChanges);

      const balanceChangesPromise = Promise.all(
        accounts.map((account) => getBalanceChange(subject, token, account))
      );
      const addressesPromise = Promise.all(accounts.map(getAddressOf));

      const checkBalanceChanges = ([
        actualChanges,
        addresses,
        tokenDescription,
      ]: [bigint[], string[], string]) => {
        const assert = buildAssert(negated, checkBalanceChanges);

        if (typeof balanceChanges === "function") {
          assert(
            balanceChanges(actualChanges),
            `Expected the balance changes of ${tokenDescription} to satisfy the predicate, but they didn't`,
            `Expected the balance changes of ${tokenDescription} to NOT satisfy the predicate, but they did`
          );
        } else {
          assert(
            actualChanges.every(
              (change, ind) => change === ethers.toBigInt(balanceChanges[ind])
            ),
            `Expected the balances of ${tokenDescription} tokens for ${
              addresses as any
            } to change by ${
              balanceChanges as any
            }, respectively, but they changed by ${actualChanges as any}`,
            `Expected the balances of ${tokenDescription} tokens for ${
              addresses as any
            } NOT to change by ${
              balanceChanges as any
            }, respectively, but they did`
          );
        }
      };

      const derivedPromise = Promise.all([
        balanceChangesPromise,
        addressesPromise,
        getTokenDescription(token),
      ]).then(checkBalanceChanges);

      this.then = derivedPromise.then.bind(derivedPromise);
      this.catch = derivedPromise.catch.bind(derivedPromise);

      return this;
    }
  );
}

function validateInput(
  obj: any,
  token: Token,
  accounts: Array<Addressable | string>,
  balanceChanges: EthersT.BigNumberish[] | ((changes: bigint[]) => boolean)
) {
  try {
    checkToken(token, CHANGE_TOKEN_BALANCES_MATCHER);

    if (
      Array.isArray(balanceChanges) &&
      accounts.length !== balanceChanges.length
    ) {
      throw new Error(
        `The number of accounts (${accounts.length}) is different than the number of expected balance changes (${balanceChanges.length})`
      );
    }
  } catch (e) {
    // if the input validation fails, we discard the subject since it could
    // potentially be a rejected promise
    Promise.resolve(obj).catch(() => {});
    throw e;
  }
}

function checkToken(token: unknown, method: string) {
  if (typeof token !== "object" || token === null || !("interface" in token)) {
    throw new Error(
      `The first argument of ${method} must be the contract instance of the token`
    );
  } else if ((token as any).interface.getFunction("balanceOf") === null) {
    throw new Error("The given contract instance is not an ERC20 token");
  }
}

export async function getBalanceChange(
  transaction: TransactionResponse | Promise<TransactionResponse>,
  token: Token,
  account: Addressable | string
) {
  const ethers = require("ethers") as typeof EthersT;
  const hre = await import("hardhat");
  const provider = hre.network.provider;

  const txResponse = await transaction;

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

  const balanceAfter = await token.balanceOf(address, {
    blockTag: txBlockNumber,
  });

  const balanceBefore = await token.balanceOf(address, {
    blockTag: txBlockNumber - 1,
  });

  return ethers.toBigInt(balanceAfter) - balanceBefore;
}

let tokenDescriptionsCache: Record<string, string> = {};
/**
 * Get a description for the given token. Use the symbol of the token if
 * possible; if it doesn't exist, the name is used; if the name doesn't
 * exist, the address of the token is used.
 */
async function getTokenDescription(token: Token): Promise<string> {
  const tokenAddress = await token.getAddress();
  if (tokenDescriptionsCache[tokenAddress] === undefined) {
    let tokenDescription = `<token at ${tokenAddress}>`;
    try {
      tokenDescription = await token.symbol();
    } catch (e) {
      try {
        tokenDescription = await token.name();
      } catch (e2) {}
    }

    tokenDescriptionsCache[tokenAddress] = tokenDescription;
  }

  return tokenDescriptionsCache[tokenAddress];
}

// only used by tests
export function clearTokenDescriptionsCache() {
  tokenDescriptionsCache = {};
}
