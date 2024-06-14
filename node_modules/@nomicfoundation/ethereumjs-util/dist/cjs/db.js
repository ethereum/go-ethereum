"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.ValueEncoding = exports.KeyEncoding = void 0;
var KeyEncoding;
(function (KeyEncoding) {
    KeyEncoding["String"] = "string";
    KeyEncoding["Bytes"] = "view";
    KeyEncoding["Number"] = "number";
})(KeyEncoding = exports.KeyEncoding || (exports.KeyEncoding = {}));
var ValueEncoding;
(function (ValueEncoding) {
    ValueEncoding["String"] = "string";
    ValueEncoding["Bytes"] = "view";
    ValueEncoding["JSON"] = "json";
})(ValueEncoding = exports.ValueEncoding || (exports.ValueEncoding = {}));
//# sourceMappingURL=db.js.map