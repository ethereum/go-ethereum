"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.clearTokenDescriptionsCache = exports.getBalanceChange = exports.supportChangeTokenBalance = void 0;
const utils_1 = require("../utils");
const utils_2 = require("./calledOnContract/utils");
const account_1 = require("./misc/account");
const constants_1 = require("./constants");
const utils_3 = require("./utils");
function supportChangeTokenBalance(Assertion, chaiUtils) {
    Assertion.addMethod(constants_1.CHANGE_TOKEN_BALANCE_MATCHER, function (token, account, balanceChange) {
        const ethers = require("ethers");
        // capture negated flag before async code executes; see buildAssert's jsdoc
        const negated = this.__flags.negate;
        let subject = this._obj;
        if (typeof subject === "function") {
            subject = subject();
        }
        (0, utils_3.preventAsyncMatcherChaining)(this, constants_1.CHANGE_TOKEN_BALANCE_MATCHER, chaiUtils);
        checkToken(token, constants_1.CHANGE_TOKEN_BALANCE_MATCHER);
        const checkBalanceChange = ([actualChange, address, tokenDescription]) => {
            const assert = (0, utils_1.buildAssert)(negated, checkBalanceChange);
            if (typeof balanceChange === "function") {
                assert(balanceChange(actualChange), `Expected the balance of ${tokenDescription} tokens for "${address}" to satisfy the predicate, but it didn't (token balance change: ${actualChange.toString()} wei)`, `Expected the balance of ${tokenDescription} tokens for "${address}" to NOT satisfy the predicate, but it did (token balance change: ${actualChange.toString()} wei)`);
            }
            else {
                assert(actualChange === ethers.toBigInt(balanceChange), `Expected the balance of ${tokenDescription} tokens for "${address}" to change by ${balanceChange.toString()}, but it changed by ${actualChange.toString()}`, `Expected the balance of ${tokenDescription} tokens for "${address}" NOT to change by ${balanceChange.toString()}, but it did`);
            }
        };
        const derivedPromise = Promise.all([
            getBalanceChange(subject, token, account),
            (0, account_1.getAddressOf)(account),
            getTokenDescription(token),
        ]).then(checkBalanceChange);
        this.then = derivedPromise.then.bind(derivedPromise);
        this.catch = derivedPromise.catch.bind(derivedPromise);
        return this;
    });
    Assertion.addMethod(constants_1.CHANGE_TOKEN_BALANCES_MATCHER, function (token, accounts, balanceChanges) {
        const ethers = require("ethers");
        // capture negated flag before async code executes; see buildAssert's jsdoc
        const negated = this.__flags.negate;
        let subject = this._obj;
        if (typeof subject === "function") {
            subject = subject();
        }
        (0, utils_3.preventAsyncMatcherChaining)(this, constants_1.CHANGE_TOKEN_BALANCES_MATCHER, chaiUtils);
        validateInput(this._obj, token, accounts, balanceChanges);
        const balanceChangesPromise = Promise.all(accounts.map((account) => getBalanceChange(subject, token, account)));
        const addressesPromise = Promise.all(accounts.map(account_1.getAddressOf));
        const checkBalanceChanges = ([actualChanges, addresses, tokenDescription,]) => {
            const assert = (0, utils_1.buildAssert)(negated, checkBalanceChanges);
            if (typeof balanceChanges === "function") {
                assert(balanceChanges(actualChanges), `Expected the balance changes of ${tokenDescription} to satisfy the predicate, but they didn't`, `Expected the balance changes of ${tokenDescription} to NOT satisfy the predicate, but they did`);
            }
            else {
                assert(actualChanges.every((change, ind) => change === ethers.toBigInt(balanceChanges[ind])), `Expected the balances of ${tokenDescription} tokens for ${addresses} to change by ${balanceChanges}, respectively, but they changed by ${actualChanges}`, `Expected the balances of ${tokenDescription} tokens for ${addresses} NOT to change by ${balanceChanges}, respectively, but they did`);
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
    });
}
exports.supportChangeTokenBalance = supportChangeTokenBalance;
function validateInput(obj, token, accounts, balanceChanges) {
    try {
        checkToken(token, constants_1.CHANGE_TOKEN_BALANCES_MATCHER);
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
function checkToken(token, method) {
    if (typeof token !== "object" || token === null || !("interface" in token)) {
        throw new Error(`The first argument of ${method} must be the contract instance of the token`);
    }
    else if (token.interface.getFunction("balanceOf") === null) {
        throw new Error("The given contract instance is not an ERC20 token");
    }
}
async function getBalanceChange(transaction, token, account) {
    const ethers = require("ethers");
    const hre = await Promise.resolve().then(() => __importStar(require("hardhat")));
    const provider = hre.network.provider;
    const txResponse = await transaction;
    const txReceipt = await txResponse.wait();
    (0, utils_3.assertIsNotNull)(txReceipt, "txReceipt");
    const txBlockNumber = txReceipt.blockNumber;
    const block = await provider.send("eth_getBlockByHash", [
        txReceipt.blockHash,
        false,
    ]);
    (0, utils_2.ensure)(block.transactions.length === 1, Error, "Multiple transactions found in block");
    const address = await (0, account_1.getAddressOf)(account);
    const balanceAfter = await token.balanceOf(address, {
        blockTag: txBlockNumber,
    });
    const balanceBefore = await token.balanceOf(address, {
        blockTag: txBlockNumber - 1,
    });
    return ethers.toBigInt(balanceAfter) - balanceBefore;
}
exports.getBalanceChange = getBalanceChange;
let tokenDescriptionsCache = {};
/**
 * Get a description for the given token. Use the symbol of the token if
 * possible; if it doesn't exist, the name is used; if the name doesn't
 * exist, the address of the token is used.
 */
async function getTokenDescription(token) {
    const tokenAddress = await token.getAddress();
    if (tokenDescriptionsCache[tokenAddress] === undefined) {
        let tokenDescription = `<token at ${tokenAddress}>`;
        try {
            tokenDescription = await token.symbol();
        }
        catch (e) {
            try {
                tokenDescription = await token.name();
            }
            catch (e2) { }
        }
        tokenDescriptionsCache[tokenAddress] = tokenDescription;
    }
    return tokenDescriptionsCache[tokenAddress];
}
// only used by tests
function clearTokenDescriptionsCache() {
    tokenDescriptionsCache = {};
}
exports.clearTokenDescriptionsCache = clearTokenDescriptionsCache;
//# sourceMappingURL=changeTokenBalance.js.map