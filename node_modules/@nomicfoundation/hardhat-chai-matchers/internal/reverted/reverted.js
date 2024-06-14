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
exports.supportReverted = void 0;
const utils_1 = require("../../utils");
const constants_1 = require("../constants");
const utils_2 = require("../utils");
const utils_3 = require("./utils");
function supportReverted(Assertion, chaiUtils) {
    Assertion.addProperty(constants_1.REVERTED_MATCHER, function () {
        // capture negated flag before async code executes; see buildAssert's jsdoc
        const negated = this.__flags.negate;
        const subject = this._obj;
        (0, utils_2.preventAsyncMatcherChaining)(this, constants_1.REVERTED_MATCHER, chaiUtils);
        // Check if the received value can be linked to a transaction, and then
        // get the receipt of that transaction and check its status.
        //
        // If the value doesn't correspond to a transaction, then the `reverted`
        // assertion is false.
        const onSuccess = async (value) => {
            const assert = (0, utils_1.buildAssert)(negated, onSuccess);
            if (isTransactionResponse(value) || typeof value === "string") {
                const hash = typeof value === "string" ? value : value.hash;
                if (!isValidTransactionHash(hash)) {
                    throw new TypeError(`Expected a valid transaction hash, but got '${hash}'`);
                }
                const receipt = await getTransactionReceipt(hash);
                (0, utils_2.assertIsNotNull)(receipt, "receipt");
                assert(receipt.status === 0, "Expected transaction to be reverted", "Expected transaction NOT to be reverted");
            }
            else if (isTransactionReceipt(value)) {
                const receipt = value;
                assert(receipt.status === 0, "Expected transaction to be reverted", "Expected transaction NOT to be reverted");
            }
            else {
                // If the subject of the assertion is not connected to a transaction
                // (hash, receipt, etc.), then the assertion fails.
                // Since we use `false` here, this means that `.not.to.be.reverted`
                // assertions will pass instead of always throwing a validation error.
                // This allows users to do things like:
                //   `expect(c.callStatic.f()).to.not.be.reverted`
                assert(false, "Expected transaction to be reverted");
            }
        };
        const onError = (error) => {
            const { toBeHex } = require("ethers");
            const assert = (0, utils_1.buildAssert)(negated, onError);
            const returnData = (0, utils_3.getReturnDataFromError)(error);
            const decodedReturnData = (0, utils_3.decodeReturnData)(returnData);
            if (decodedReturnData.kind === "Empty" ||
                decodedReturnData.kind === "Custom") {
                // in the negated case, if we can't decode the reason, we just indicate
                // that the transaction didn't revert
                assert(true, undefined, `Expected transaction NOT to be reverted`);
            }
            else if (decodedReturnData.kind === "Error") {
                assert(true, undefined, `Expected transaction NOT to be reverted, but it reverted with reason '${decodedReturnData.reason}'`);
            }
            else if (decodedReturnData.kind === "Panic") {
                assert(true, undefined, `Expected transaction NOT to be reverted, but it reverted with panic code ${toBeHex(decodedReturnData.code)} (${decodedReturnData.description})`);
            }
            else {
                const _exhaustiveCheck = decodedReturnData;
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
exports.supportReverted = supportReverted;
async function getTransactionReceipt(hash) {
    const hre = await Promise.resolve().then(() => __importStar(require("hardhat")));
    return hre.ethers.provider.getTransactionReceipt(hash);
}
function isTransactionResponse(x) {
    if (typeof x === "object" && x !== null) {
        return "hash" in x;
    }
    return false;
}
function isTransactionReceipt(x) {
    if (typeof x === "object" && x !== null && "status" in x) {
        const status = x.status;
        // this means we only support ethers's receipts for now; adding support for
        // raw receipts, where the status is an hexadecimal string, should be easy
        // and we can do it if there's demand for that
        return typeof status === "number";
    }
    return false;
}
function isValidTransactionHash(x) {
    return /0x[0-9a-fA-F]{64}/.test(x);
}
//# sourceMappingURL=reverted.js.map