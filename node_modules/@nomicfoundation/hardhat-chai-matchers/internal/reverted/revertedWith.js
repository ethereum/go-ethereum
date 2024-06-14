"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.supportRevertedWith = void 0;
const utils_1 = require("../../utils");
const constants_1 = require("../constants");
const utils_2 = require("../utils");
const utils_3 = require("./utils");
function supportRevertedWith(Assertion, chaiUtils) {
    Assertion.addMethod(constants_1.REVERTED_WITH_MATCHER, function (expectedReason) {
        // capture negated flag before async code executes; see buildAssert's jsdoc
        const negated = this.__flags.negate;
        // validate expected reason
        if (!(expectedReason instanceof RegExp) &&
            typeof expectedReason !== "string") {
            // if the input validation fails, we discard the subject since it could
            // potentially be a rejected promise
            Promise.resolve(this._obj).catch(() => { });
            throw new TypeError("Expected the revert reason to be a string or a regular expression");
        }
        const expectedReasonString = expectedReason instanceof RegExp
            ? expectedReason.source
            : expectedReason;
        (0, utils_2.preventAsyncMatcherChaining)(this, constants_1.REVERTED_WITH_MATCHER, chaiUtils);
        const onSuccess = () => {
            const assert = (0, utils_1.buildAssert)(negated, onSuccess);
            assert(false, `Expected transaction to be reverted with reason '${expectedReasonString}', but it didn't revert`);
        };
        const onError = (error) => {
            const { toBeHex } = require("ethers");
            const assert = (0, utils_1.buildAssert)(negated, onError);
            const returnData = (0, utils_3.getReturnDataFromError)(error);
            const decodedReturnData = (0, utils_3.decodeReturnData)(returnData);
            if (decodedReturnData.kind === "Empty") {
                assert(false, `Expected transaction to be reverted with reason '${expectedReasonString}', but it reverted without a reason`);
            }
            else if (decodedReturnData.kind === "Error") {
                const matchesExpectedReason = expectedReason instanceof RegExp
                    ? expectedReason.test(decodedReturnData.reason)
                    : decodedReturnData.reason === expectedReasonString;
                assert(matchesExpectedReason, `Expected transaction to be reverted with reason '${expectedReasonString}', but it reverted with reason '${decodedReturnData.reason}'`, `Expected transaction NOT to be reverted with reason '${expectedReasonString}', but it was`);
            }
            else if (decodedReturnData.kind === "Panic") {
                assert(false, `Expected transaction to be reverted with reason '${expectedReasonString}', but it reverted with panic code ${toBeHex(decodedReturnData.code)} (${decodedReturnData.description})`);
            }
            else if (decodedReturnData.kind === "Custom") {
                assert(false, `Expected transaction to be reverted with reason '${expectedReasonString}', but it reverted with a custom error`);
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
exports.supportRevertedWith = supportRevertedWith;
//# sourceMappingURL=revertedWith.js.map