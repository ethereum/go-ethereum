import type EthersT from "ethers";

import { tryDereference } from "./typed";

export function supportAddressable(
  Assertion: Chai.AssertionStatic,
  chaiUtils: Chai.ChaiUtils
) {
  const equalsFunction = override("eq", "equal", "not equal", chaiUtils);
  Assertion.overwriteMethod("equals", equalsFunction);
  Assertion.overwriteMethod("equal", equalsFunction);
  Assertion.overwriteMethod("eq", equalsFunction);
}

type Methods = "eq";

function override(
  method: Methods,
  name: string,
  negativeName: string,
  chaiUtils: Chai.ChaiUtils
) {
  return (_super: (...args: any[]) => any) =>
    overwriteAddressableFunction(method, name, negativeName, _super, chaiUtils);
}

// ethers's Addressable have a .getAddress() that returns a Promise<string>. We don't want to deal with async here,
// so we are looking for a sync way of getting the address. If an address was recovered, it is returned as a string,
// otherwise undefined is returned.
function tryGetAddressSync(value: any): string | undefined {
  const { isAddress, isAddressable } = require("ethers") as typeof EthersT;

  value = tryDereference(value, "address");
  if (isAddressable(value)) {
    value = (value as any).address ?? (value as any).target;
  }
  if (isAddress(value)) {
    return value;
  } else {
    return undefined;
  }
}

function overwriteAddressableFunction(
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

    const actual = tryGetAddressSync(actualArg);
    const expected = tryGetAddressSync(expectedFlag);
    if (
      functionName === "eq" &&
      expected !== undefined &&
      actual !== undefined
    ) {
      this.assert(
        expected === actual,
        `expected '${expected}' to ${readableName} '${actual}'.`,
        `expected '${expected}' to ${readableNegativeName} '${actual}'.`,
        actual.toString(),
        expected.toString()
      );
    } else {
      _super.apply(this, args);
    }
  };
}
