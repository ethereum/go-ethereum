"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.supportAddressable = void 0;
const typed_1 = require("./typed");
function supportAddressable(Assertion, chaiUtils) {
    const equalsFunction = override("eq", "equal", "not equal", chaiUtils);
    Assertion.overwriteMethod("equals", equalsFunction);
    Assertion.overwriteMethod("equal", equalsFunction);
    Assertion.overwriteMethod("eq", equalsFunction);
}
exports.supportAddressable = supportAddressable;
function override(method, name, negativeName, chaiUtils) {
    return (_super) => overwriteAddressableFunction(method, name, negativeName, _super, chaiUtils);
}
// ethers's Addressable have a .getAddress() that returns a Promise<string>. We don't want to deal with async here,
// so we are looking for a sync way of getting the address. If an address was recovered, it is returned as a string,
// otherwise undefined is returned.
function tryGetAddressSync(value) {
    const { isAddress, isAddressable } = require("ethers");
    value = (0, typed_1.tryDereference)(value, "address");
    if (isAddressable(value)) {
        value = value.address ?? value.target;
    }
    if (isAddress(value)) {
        return value;
    }
    else {
        return undefined;
    }
}
function overwriteAddressableFunction(functionName, readableName, readableNegativeName, _super, chaiUtils) {
    return function (...args) {
        const [actualArg, message] = args;
        const expectedFlag = chaiUtils.flag(this, "object");
        if (message !== undefined) {
            chaiUtils.flag(this, "message", message);
        }
        const actual = tryGetAddressSync(actualArg);
        const expected = tryGetAddressSync(expectedFlag);
        if (functionName === "eq" &&
            expected !== undefined &&
            actual !== undefined) {
            this.assert(expected === actual, `expected '${expected}' to ${readableName} '${actual}'.`, `expected '${expected}' to ${readableNegativeName} '${actual}'.`, actual.toString(), expected.toString());
        }
        else {
            _super.apply(this, args);
        }
    };
}
//# sourceMappingURL=addressable.js.map