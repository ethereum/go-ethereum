import { AssertionError } from "chai";
import { isBigNumber, normalizeToBigInt } from "hardhat/common/bigInt";
import util from "util";

export function supportBigNumber(
  Assertion: Chai.AssertionStatic,
  chaiUtils: Chai.ChaiUtils
) {
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

function createLengthOverride(
  method: string
): [string, (...args: any[]) => any, (...args: any[]) => any] {
  return [
    method,
    function (_super: any) {
      return function (this: Chai.AssertionPrototype, value: any) {
        const actual = this._obj;
        if (isBigNumber(value)) {
          const sizeOrLength =
            actual instanceof Map || actual instanceof Set ? "size" : "length";
          const actualLength = normalizeToBigInt(actual[sizeOrLength]);
          const expectedLength = normalizeToBigInt(value);
          this.assert(
            actualLength === expectedLength,
            `expected #{this} to have a ${sizeOrLength} of ${expectedLength.toString()} but got ${actualLength.toString()}`,
            `expected #{this} not to have a ${sizeOrLength} of ${expectedLength.toString()} but got ${actualLength.toString()}`,
            actualLength.toString(),
            expectedLength.toString()
          );
        } else {
          _super.apply(this, arguments);
        }
      };
    },
    function (_super: any) {
      return function (this: any) {
        _super.apply(this, arguments);
      };
    } as any,
  ];
}

type Methods = "eq" | "gt" | "lt" | "gte" | "lte";

function override(
  method: Methods,
  name: string,
  negativeName: string,
  chaiUtils: Chai.ChaiUtils
) {
  return (_super: (...args: any[]) => any) =>
    overwriteBigNumberFunction(method, name, negativeName, _super, chaiUtils);
}

function overwriteBigNumberFunction(
  functionName: Methods,
  readableName: string,
  readableNegativeName: string,
  _super: (...args: any[]) => any,
  chaiUtils: Chai.ChaiUtils
) {
  return function (this: Chai.AssertionStatic, ...args: any[]) {
    const [actualArg, message] = args;
    const expectedFlag = chaiUtils.flag(this, "object");

    if (message !== undefined) {
      chaiUtils.flag(this, "message", message);
    }

    function compare(method: Methods, lhs: bigint, rhs: bigint): boolean {
      if (method === "eq") {
        return lhs === rhs;
      } else if (method === "gt") {
        return lhs > rhs;
      } else if (method === "lt") {
        return lhs < rhs;
      } else if (method === "gte") {
        return lhs >= rhs;
      } else if (method === "lte") {
        return lhs <= rhs;
      } else {
        throw new Error(`Unknown comparison operation ${method as any}`);
      }
    }
    if (Boolean(chaiUtils.flag(this, "doLength")) && isBigNumber(actualArg)) {
      const sizeOrLength =
        expectedFlag instanceof Map || expectedFlag instanceof Set
          ? "size"
          : "length";
      if (expectedFlag[sizeOrLength] === undefined) {
        _super.apply(this, args);
        return;
      }
      const expected = normalizeToBigInt(expectedFlag[sizeOrLength]);
      const actual = normalizeToBigInt(actualArg);
      this.assert(
        compare(functionName, expected, actual),
        `expected #{this} to have a ${sizeOrLength} ${readableName.replace(
          "be ",
          ""
        )} ${actual.toString()} but got ${expected}`,
        `expected #{this} to have a ${sizeOrLength} ${readableNegativeName} ${actual.toString()}`,
        expected,
        actual
      );
    } else if (functionName === "eq" && Boolean(chaiUtils.flag(this, "deep"))) {
      const deepEqual = require("deep-eql");
      // this is close enough to what chai itself does, except we compare
      // numbers after normalizing them
      const comparator = (a: any, b: any): boolean | null => {
        try {
          const normalizedA = normalizeToBigInt(a);
          const normalizedB = normalizeToBigInt(b);
          return normalizedA === normalizedB;
        } catch (e) {
          // use default comparator
          return null;
        }
      };

      // "ssfi" stands for "start stack function indicator", it's a chai concept
      // used to control which frames are included in the stack trace
      // this pattern here was taken from chai's implementation of .deep.equal
      const prevLockSsfi = chaiUtils.flag(this, "lockSsfi");
      chaiUtils.flag(this, "lockSsfi", true);
      this.assert(
        deepEqual(actualArg, expectedFlag, { comparator }),
        `expected ${util.inspect(expectedFlag)} to deeply equal ${util.inspect(
          actualArg
        )}`,
        `expected ${util.inspect(
          expectedFlag
        )} to not deeply equal ${util.inspect(actualArg)}`,
        null
      );
      chaiUtils.flag(this, "lockSsfi", prevLockSsfi);
    } else if (isBigNumber(expectedFlag) || isBigNumber(actualArg)) {
      const expected = normalizeToBigInt(expectedFlag);
      const actual = normalizeToBigInt(actualArg);
      this.assert(
        compare(functionName, expected, actual),
        `expected ${expected} to ${readableName} ${actual}.`,
        `expected ${expected} to ${readableNegativeName} ${actual}.`,
        actual.toString(),
        expected.toString()
      );
    } else {
      _super.apply(this, args);
    }
  };
}

function overrideWithin(chaiUtils: Chai.ChaiUtils) {
  return (_super: (...args: any[]) => any) =>
    overwriteBigNumberWithin(_super, chaiUtils);
}

function overwriteBigNumberWithin(
  _super: (...args: any[]) => any,
  chaiUtils: Chai.ChaiUtils
) {
  return function (this: Chai.AssertionStatic, ...args: any[]) {
    const [startArg, finishArg] = args;
    const expectedFlag = chaiUtils.flag(this, "object");
    if (
      isBigNumber(expectedFlag) ||
      isBigNumber(startArg) ||
      isBigNumber(finishArg)
    ) {
      const expected = normalizeToBigInt(expectedFlag);
      const start = normalizeToBigInt(startArg);
      const finish = normalizeToBigInt(finishArg);
      this.assert(
        start <= expected && expected <= finish,
        `expected ${expected} to be within ${start}..${finish}`,
        `expected ${expected} to not be within ${start}..${finish}`,
        expected,
        [start, finish]
      );
    } else {
      _super.apply(this, args);
    }
  };
}

function overrideCloseTo(chaiUtils: Chai.ChaiUtils) {
  return (_super: (...args: any[]) => any) =>
    overwriteBigNumberCloseTo(_super, chaiUtils);
}

function overwriteBigNumberCloseTo(
  _super: (...args: any[]) => any,
  chaiUtils: Chai.ChaiUtils
) {
  return function (this: Chai.AssertionStatic, ...args: any[]) {
    const [actualArg, deltaArg] = args;
    const expectedFlag = chaiUtils.flag(this, "object");
    if (
      isBigNumber(expectedFlag) ||
      isBigNumber(actualArg) ||
      isBigNumber(deltaArg)
    ) {
      if (deltaArg === undefined) {
        throw new AssertionError(
          "the arguments to closeTo or approximately must be numbers, and a delta is required"
        );
      }
      const expected = normalizeToBigInt(expectedFlag);
      const actual = normalizeToBigInt(actualArg);
      const delta = normalizeToBigInt(deltaArg);
      function abs(i: bigint): bigint {
        return i < 0 ? BigInt(-1) * i : i;
      }
      this.assert(
        abs(expected - actual) <= delta,
        `expected ${expected} to be close to ${actual} +/- ${delta}`,
        `expected ${expected} not to be close to ${actual} +/- ${delta}`,
        expected,
        `A number between ${actual - delta} and ${actual + delta}`
      );
    } else {
      _super.apply(this, args);
    }
  };
}
