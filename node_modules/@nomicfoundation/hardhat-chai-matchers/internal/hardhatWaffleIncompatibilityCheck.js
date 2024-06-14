"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.hardhatWaffleIncompatibilityCheck = void 0;
function hardhatWaffleIncompatibilityCheck() {
    if (global.__HARDHAT_WAFFLE_IS_LOADED === true) {
        throw new Error(`You are using both @nomicfoundation/hardhat-chai-matchers and @nomiclabs/hardhat-waffle. They don't work correctly together, so please make sure you only use one.

We recommend you migrate to @nomicfoundation/hardhat-chai-matchers. Learn how to do it here: https://hardhat.org/migrate-from-waffle`);
    }
    global.__HARDHAT_CHAI_MATCHERS_IS_LOADED = true;
}
exports.hardhatWaffleIncompatibilityCheck = hardhatWaffleIncompatibilityCheck;
//# sourceMappingURL=hardhatWaffleIncompatibilityCheck.js.map