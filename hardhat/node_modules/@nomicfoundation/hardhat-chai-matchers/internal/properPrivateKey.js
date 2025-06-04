"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.supportProperPrivateKey = void 0;
function supportProperPrivateKey(Assertion) {
    Assertion.addProperty("properPrivateKey", function () {
        const subject = this._obj;
        this.assert(/^0x[0-9a-fA-F]{64}$/.test(subject), `Expected "${subject}" to be a proper private key`, `Expected "${subject}" NOT to be a proper private key`, "proper private key (eg.: 0x1010101010101010101010101010101010101010101010101010101010101010)", subject);
    });
}
exports.supportProperPrivateKey = supportProperPrivateKey;
//# sourceMappingURL=properPrivateKey.js.map