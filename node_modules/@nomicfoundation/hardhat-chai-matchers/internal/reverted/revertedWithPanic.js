"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.supportRevertedWithPanic = void 0;
const common_1 = require("hardhat/common");
const utils_1 = require("../../utils");
const constants_1 = require("../constants");
const utils_2 = require("../utils");
const panic_1 = require("./panic");
const utils_3 = require("./utils");
function supportRevertedWithPanic(Assertion, chaiUtils) {
    Assertion.addMethod(constants_1.REVERTED_WITH_PANIC_MATCHER, function (expectedCodeArg) {
        const ethers = require("ethers");
        // capture negated flag before async code executes; see buildAssert's jsdoc
        const negated = this.__flags.negate;
        let expectedCode;
        try {
            if (expectedCodeArg !== undefined) {
                expectedCode = (0, common_1.normalizeToBigInt)(expectedCodeArg);
            }
        }
        catch {
            // if the input validation fails, we discard the subject since it could
            // potentially be a rejected promise
            Promise.resolve(this._obj).catch(() => { });
            throw new TypeError(`Expected the given panic code to be a number-like value, but got '${expectedCodeArg}'`);
        }
        const code = expectedCode;
        let description;
        let formattedPanicCode;
        if (code === undefined) {
            formattedPanicCode = "some panic code";
        }
        else {
            const codeBN = ethers.toBigInt(code);
            description = (0, panic_1.panicErrorCodeToReason)(codeBN) ?? "unknown panic code";
            formattedPanicCode = `panic code ${ethers.toBeHex(codeBN)} (${description})`;
        }
        (0, utils_2.preventAsyncMatcherChaining)(this, constants_1.REVERTED_WITH_PANIC_MATCHER, chaiUtils);
        const onSuccess = () => {
            const assert = (0, utils_1.buildAssert)(negated, onSuccess);
            assert(false, `Expected transaction to be reverted with ${formattedPanicCode}, but it didn't revert`);
        };
        const onError = (error) => {
            const assert = (0, utils_1.buildAssert)(negated, onError);
            const returnData = (0, utils_3.getReturnDataFromError)(error);
            const decodedReturnData = (0, utils_3.decodeReturnData)(returnData);
            if (decodedReturnData.kind === "Empty") {
                assert(false, `Expected transaction to be reverted with ${formattedPanicCode}, but it reverted without a reason`);
            }
            else if (decodedReturnData.kind === "Error") {
                assert(false, `Expected transaction to be reverted with ${formattedPanicCode}, but it reverted with reason '${decodedReturnData.reason}'`);
            }
            else if (decodedReturnData.kind === "Panic") {
                if (code !== undefined) {
                    assert(decodedReturnData.code === code, `Expected transaction to be reverted with ${formattedPanicCode}, but it reverted with panic code ${ethers.toBeHex(decodedReturnData.code)} (${decodedReturnData.description})`, `Expected transaction NOT to be reverted with ${formattedPanicCode}, but it was`);
                }
                else {
                    assert(true, undefined, `Expected transaction NOT to be reverted with ${formattedPanicCode}, but it reverted with panic code ${ethers.toBeHex(decodedReturnData.code)} (${decodedReturnData.description})`);
                }
            }
            else if (decodedReturnData.kind === "Custom") {
                assert(false, `Expected transaction to be reverted with ${formattedPanicCode}, but it reverted with a custom error`);
            }
            else {
                const _exhaustiveCheck = decodedReturnData;
            }
        };
        const derivedPromise = Promise.resolve(this._obj).then(onSuccess, onError);
        this.then = derivedPromise.then.bind(derivedPromise);
        this.catch = derivedPromise.catch.bind(derivedPromise);
        return this;
    });
}
exports.supportRevertedWithPanic = supportRevertedWithPanic;
//# sourceMappingURL=revertedWithPanic.js.map