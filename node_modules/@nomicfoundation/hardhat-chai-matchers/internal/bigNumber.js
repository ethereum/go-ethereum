"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.supportBigNumber = void 0;
const chai_1 = require("chai");
const bigInt_1 = require("hardhat/common/bigInt");
const util_1 = __importDefault(require("util"));
function supportBigNumber(Assertion, chaiUtils) {
    const equalsFunction = override("eq", "equal", "not equal", chaiUtils);
    Assertion.overwriteMethod("equals", equalsFunction);
    Assertion.overwriteMethod("equal", equalsFunction);
    Assertion.overwriteMethod("eq", equalsFunction);
    const gtFunction = override("gt", "be above", "be at most", chaiUtils);
    Assertion.overwriteMethod("above", gtFunction);
    Assertion.overwriteMethod("gt", gtFunction);
    Assertion.overwriteMethod("greaterThan", gtFunction);
    const ltFunction = override("lt", "be below", "be at least", chaiUtils);
    Assertion.overwriteMethod("below", ltFunction);
    Assertion.overwriteMethod("lt", ltFunction);
    Assertion.overwriteMethod("lessThan", ltFunction);
    const gteFunction = override("gte", "be at least", "be below", chaiUtils);
    Assertion.overwriteMethod("least", gteFunction);
    Assertion.overwriteMethod("gte", gteFunction);
    Assertion.overwriteMethod("greaterThanOrEqual", gteFunction);
    const lteFunction = override("lte", "be at most", "be above", chaiUtils);
    Assertion.overwriteMethod("most", lteFunction);
    Assertion.overwriteMethod("lte", lteFunction);
    Assertion.overwriteMethod("lessThanOrEqual", lteFunction);
    Assertion.overwriteChainableMethod(...createLengthOverride("length"));
    Assertion.overwriteChainableMethod(...createLengthOverride("lengthOf"));
    Assertion.overwriteMethod("within", overrideWithin(chaiUtils));
    Assertion.overwriteMethod("closeTo", overrideCloseTo(chaiUtils));
    Assertion.overwriteMethod("approximately", overrideCloseTo(chaiUtils));
}
exports.supportBigNumber = supportBigNumber;
function createLengthOverride(method) {
    return [
        method,
        function (_super) {
            return function (value) {
                const actual = this._obj;
                if ((0, bigInt_1.isBigNumber)(value)) {
                    const sizeOrLength = actual instanceof Map || actual instanceof Set ? "size" : "length";
                    const actualLength = (0, bigInt_1.normalizeToBigInt)(actual[sizeOrLength]);
                    const expectedLength = (0, bigInt_1.normalizeToBigInt)(value);
                    this.assert(actualLength === expectedLength, `expected #{this} to have a ${sizeOrLength} of ${expectedLength.toString()} but got ${actualLength.toString()}`, `expected #{this} not to have a ${sizeOrLength} of ${expectedLength.toString()} but got ${actualLength.toString()}`, actualLength.toString(), expectedLength.toString());
                }
                else {
                    _super.apply(this, arguments);
                }
            };
        },
        function (_super) {
            return function () {
                _super.apply(this, arguments);
            };
        },
    ];
}
function override(method, name, negativeName, chaiUtils) {
    return (_super) => overwriteBigNumberFunction(method, name, negativeName, _super, chaiUtils);
}
function overwriteBigNumberFunction(functionName, readableName, readableNegativeName, _super, chaiUtils) {
    return function (...args) {
        const [actualArg, message] = args;
        const expectedFlag = chaiUtils.flag(this, "object");
        if (message !== undefined) {
            chaiUtils.flag(this, "message", message);
        }
        function compare(method, lhs, rhs) {
            if (method === "eq") {
                return lhs === rhs;
            }
            else if (method === "gt") {
                return lhs > rhs;
            }
            else if (method === "lt") {
                return lhs < rhs;
            }
            else if (method === "gte") {
                return lhs >= rhs;
            }
            else if (method === "lte") {
                return lhs <= rhs;
            }
            else {
                throw new Error(`Unknown comparison operation ${method}`);
            }
        }
        if (Boolean(chaiUtils.flag(this, "doLength")) && (0, bigInt_1.isBigNumber)(actualArg)) {
            const sizeOrLength = expectedFlag instanceof Map || expectedFlag instanceof Set
                ? "size"
                : "length";
            if (expectedFlag[sizeOrLength] === undefined) {
                _super.apply(this, args);
                return;
            }
            const expected = (0, bigInt_1.normalizeToBigInt)(expectedFlag[sizeOrLength]);
            const actual = (0, bigInt_1.normalizeToBigInt)(actualArg);
            this.assert(compare(functionName, expected, actual), `expected #{this} to have a ${sizeOrLength} ${readableName.replace("be ", "")} ${actual.toString()} but got ${expected}`, `expected #{this} to have a ${sizeOrLength} ${readableNegativeName} ${actual.toString()}`, expected, actual);
        }
        else if (functionName === "eq" && Boolean(chaiUtils.flag(this, "deep"))) {
            const deepEqual = require("deep-eql");
            // this is close enough to what chai itself does, except we compare
            // numbers after normalizing them
            const comparator = (a, b) => {
                try {
                    const normalizedA = (0, bigInt_1.normalizeToBigInt)(a);
                    const normalizedB = (0, bigInt_1.normalizeToBigInt)(b);
                    return normalizedA === normalizedB;
                }
                catch (e) {
                    // use default comparator
                    return null;
                }
            };
            // "ssfi" stands for "start stack function indicator", it's a chai concept
            // used to control which frames are included in the stack trace
            // this pattern here was taken from chai's implementation of .deep.equal
            const prevLockSsfi = chaiUtils.flag(this, "lockSsfi");
            chaiUtils.flag(this, "lockSsfi", true);
            this.assert(deepEqual(actualArg, expectedFlag, { comparator }), `expected ${util_1.default.inspect(expectedFlag)} to deeply equal ${util_1.default.inspect(actualArg)}`, `expected ${util_1.default.inspect(expectedFlag)} to not deeply equal ${util_1.default.inspect(actualArg)}`, null);
            chaiUtils.flag(this, "lockSsfi", prevLockSsfi);
        }
        else if ((0, bigInt_1.isBigNumber)(expectedFlag) || (0, bigInt_1.isBigNumber)(actualArg)) {
            const expected = (0, bigInt_1.normalizeToBigInt)(expectedFlag);
            const actual = (0, bigInt_1.normalizeToBigInt)(actualArg);
            this.assert(compare(functionName, expected, actual), `expected ${expected} to ${readableName} ${actual}.`, `expected ${expected} to ${readableNegativeName} ${actual}.`, actual.toString(), expected.toString());
        }
        else {
            _super.apply(this, args);
        }
    };
}
function overrideWithin(chaiUtils) {
    return (_super) => overwriteBigNumberWithin(_super, chaiUtils);
}
function overwriteBigNumberWithin(_super, chaiUtils) {
    return function (...args) {
        const [startArg, finishArg] = args;
        const expectedFlag = chaiUtils.flag(this, "object");
        if ((0, bigInt_1.isBigNumber)(expectedFlag) ||
            (0, bigInt_1.isBigNumber)(startArg) ||
            (0, bigInt_1.isBigNumber)(finishArg)) {
            const expected = (0, bigInt_1.normalizeToBigInt)(expectedFlag);
            const start = (0, bigInt_1.normalizeToBigInt)(startArg);
            const finish = (0, bigInt_1.normalizeToBigInt)(finishArg);
            this.assert(start <= expected && expected <= finish, `expected ${expected} to be within ${start}..${finish}`, `expected ${expected} to not be within ${start}..${finish}`, expected, [start, finish]);
        }
        else {
            _super.apply(this, args);
        }
    };
}
function overrideCloseTo(chaiUtils) {
    return (_super) => overwriteBigNumberCloseTo(_super, chaiUtils);
}
function overwriteBigNumberCloseTo(_super, chaiUtils) {
    return function (...args) {
        const [actualArg, deltaArg] = args;
        const expectedFlag = chaiUtils.flag(this, "object");
        if ((0, bigInt_1.isBigNumber)(expectedFlag) ||
            (0, bigInt_1.isBigNumber)(actualArg) ||
            (0, bigInt_1.isBigNumber)(deltaArg)) {
            if (deltaArg === undefined) {
                throw new chai_1.AssertionError("the arguments to closeTo or approximately must be numbers, and a delta is required");
            }
            const expected = (0, bigInt_1.normalizeToBigInt)(expectedFlag);
            const actual = (0, bigInt_1.normalizeToBigInt)(actualArg);
            const delta = (0, bigInt_1.normalizeToBigInt)(deltaArg);
            function abs(i) {
                return i < 0 ? BigInt(-1) * i : i;
            }
            this.assert(abs(expected - actual) <= delta, `expected ${expected} to be close to ${actual} +/- ${delta}`, `expected ${expected} not to be close to ${actual} +/- ${delta}`, expected, `A number between ${actual - delta} and ${actual + delta}`);
        }
        else {
            _super.apply(this, args);
        }
    };
}
//# sourceMappingURL=bigNumber.js.map