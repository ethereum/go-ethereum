"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.hardhatChaiMatchers = void 0;
const addressable_1 = require("./addressable");
const bigNumber_1 = require("./bigNumber");
const emit_1 = require("./emit");
const hexEqual_1 = require("./hexEqual");
const properAddress_1 = require("./properAddress");
const properHex_1 = require("./properHex");
const properPrivateKey_1 = require("./properPrivateKey");
const changeEtherBalance_1 = require("./changeEtherBalance");
const changeEtherBalances_1 = require("./changeEtherBalances");
const changeTokenBalance_1 = require("./changeTokenBalance");
const reverted_1 = require("./reverted/reverted");
const revertedWith_1 = require("./reverted/revertedWith");
const revertedWithCustomError_1 = require("./reverted/revertedWithCustomError");
const revertedWithPanic_1 = require("./reverted/revertedWithPanic");
const revertedWithoutReason_1 = require("./reverted/revertedWithoutReason");
const withArgs_1 = require("./withArgs");
function hardhatChaiMatchers(chai, chaiUtils) {
    (0, addressable_1.supportAddressable)(chai.Assertion, chaiUtils);
    (0, bigNumber_1.supportBigNumber)(chai.Assertion, chaiUtils);
    (0, emit_1.supportEmit)(chai.Assertion, chaiUtils);
    (0, hexEqual_1.supportHexEqual)(chai.Assertion);
    (0, properAddress_1.supportProperAddress)(chai.Assertion);
    (0, properHex_1.supportProperHex)(chai.Assertion);
    (0, properPrivateKey_1.supportProperPrivateKey)(chai.Assertion);
    (0, changeEtherBalance_1.supportChangeEtherBalance)(chai.Assertion, chaiUtils);
    (0, changeEtherBalances_1.supportChangeEtherBalances)(chai.Assertion, chaiUtils);
    (0, changeTokenBalance_1.supportChangeTokenBalance)(chai.Assertion, chaiUtils);
    (0, reverted_1.supportReverted)(chai.Assertion, chaiUtils);
    (0, revertedWith_1.supportRevertedWith)(chai.Assertion, chaiUtils);
    (0, revertedWithCustomError_1.supportRevertedWithCustomError)(chai.Assertion, chaiUtils);
    (0, revertedWithPanic_1.supportRevertedWithPanic)(chai.Assertion, chaiUtils);
    (0, revertedWithoutReason_1.supportRevertedWithoutReason)(chai.Assertion, chaiUtils);
    (0, withArgs_1.supportWithArgs)(chai.Assertion, chaiUtils);
}
exports.hardhatChaiMatchers = hardhatChaiMatchers;
//# sourceMappingURL=hardhatChaiMatchers.js.map