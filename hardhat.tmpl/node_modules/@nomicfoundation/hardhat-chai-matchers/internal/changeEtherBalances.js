"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getBalanceChanges = exports.supportChangeEtherBalances = void 0;
const utils_1 = require("../utils");
const account_1 = require("./misc/account");
const balance_1 = require("./misc/balance");
const constants_1 = require("./constants");
const utils_2 = require("./utils");
function supportChangeEtherBalances(Assertion, chaiUtils) {
    Assertion.addMethod(constants_1.CHANGE_ETHER_BALANCES_MATCHER, function (accounts, balanceChanges, options) {
        const { toBigInt } = require("ethers");
        const ordinal = require("ordinal");
        // capture negated flag before async code executes; see buildAssert's jsdoc
        const negated = this.__flags.negate;
        let subject = this._obj;
        if (typeof subject === "function") {
            subject = subject();
        }
        (0, utils_2.preventAsyncMatcherChaining)(this, constants_1.CHANGE_ETHER_BALANCES_MATCHER, chaiUtils);
        validateInput(this._obj, accounts, balanceChanges);
        const checkBalanceChanges = ([actualChanges, accountAddresses]) => {
            const assert = (0, utils_1.buildAssert)(negated, checkBalanceChanges);
            if (typeof balanceChanges === "function") {
                assert(balanceChanges(actualChanges), "Expected the balance changes of the accounts to satisfy the predicate, but they didn't", "Expected the balance changes of the accounts to NOT satisfy the predicate, but they did");
            }
            else {
                assert(actualChanges.every((change, ind) => change === toBigInt(balanceChanges[ind])), () => {
                    const lines = [];
                    actualChanges.forEach((change, i) => {
                        if (change !== toBigInt(balanceChanges[i])) {
                            lines.push(`Expected the ether balance of ${accountAddresses[i]} (the ${ordinal(i + 1)} address in the list) to change by ${balanceChanges[i].toString()} wei, but it changed by ${change.toString()} wei`);
                        }
                    });
                    return lines.join("\n");
                }, () => {
                    const lines = [];
                    actualChanges.forEach((change, i) => {
                        if (change === toBigInt(balanceChanges[i])) {
                            lines.push(`Expected the ether balance of ${accountAddresses[i]} (the ${ordinal(i + 1)} address in the list) NOT to change by ${balanceChanges[i].toString()} wei, but it did`);
                        }
                    });
                    return lines.join("\n");
                });
            }
        };
        const derivedPromise = Promise.all([
            getBalanceChanges(subject, accounts, options),
            (0, balance_1.getAddresses)(accounts),
        ]).then(checkBalanceChanges);
        this.then = derivedPromise.then.bind(derivedPromise);
        this.catch = derivedPromise.catch.bind(derivedPromise);
        this.promise = derivedPromise;
        return this;
    });
}
exports.supportChangeEtherBalances = supportChangeEtherBalances;
function validateInput(obj, accounts, balanceChanges) {
    try {
        if (Array.isArray(balanceChanges) &&
            accounts.length !== balanceChanges.length) {
            throw new Error(`The number of accounts (${accounts.length}) is different than the number of expected balance changes (${balanceChanges.length})`);
        }
    }
    catch (e) {
        // if the input validation fails, we discard the subject since it could
        // potentially be a rejected promise
        Promise.resolve(obj).catch(() => { });
        throw e;
    }
}
async function getBalanceChanges(transaction, accounts, options) {
    const txResponse = await transaction;
    const txReceipt = await txResponse.wait();
    (0, utils_2.assertIsNotNull)(txReceipt, "txReceipt");
    const txBlockNumber = txReceipt.blockNumber;
    const balancesAfter = await (0, balance_1.getBalances)(accounts, txBlockNumber);
    const balancesBefore = await (0, balance_1.getBalances)(accounts, txBlockNumber - 1);
    const txFees = await getTxFees(accounts, txResponse, options);
    return balancesAfter.map((balance, ind) => balance + txFees[ind] - balancesBefore[ind]);
}
exports.getBalanceChanges = getBalanceChanges;
async function getTxFees(accounts, txResponse, options) {
    return Promise.all(accounts.map(async (account) => {
        if (options?.includeFee !== true &&
            (await (0, account_1.getAddressOf)(account)) === txResponse.from) {
            const txReceipt = await txResponse.wait();
            (0, utils_2.assertIsNotNull)(txReceipt, "txReceipt");
            const gasPrice = txReceipt.gasPrice ?? txResponse.gasPrice;
            const gasUsed = txReceipt.gasUsed;
            const txFee = gasPrice * gasUsed;
            return txFee;
        }
        return 0n;
    }));
}
//# sourceMappingURL=changeEtherBalances.js.map