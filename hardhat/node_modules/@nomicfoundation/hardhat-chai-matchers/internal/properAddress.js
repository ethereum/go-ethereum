"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.supportProperAddress = void 0;
function supportProperAddress(Assertion) {
    Assertion.addProperty("properAddress", function () {
        const subject = this._obj;
        this.assert(/^0x[0-9a-fA-F]{40}$/.test(subject), `Expected "${subject}" to be a proper address`, `Expected "${subject}" NOT to be a proper address`, "proper address (eg.: 0x1234567890123456789012345678901234567890)", subject);
    });
}
exports.supportProperAddress = supportProperAddress;
//# sourceMappingURL=properAddress.js.map