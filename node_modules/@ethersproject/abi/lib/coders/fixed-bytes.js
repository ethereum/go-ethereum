"use strict";
var __extends = (this && this.__extends) || (function () {
    var extendStatics = function (d, b) {
        extendStatics = Object.setPrototypeOf ||
            ({ __proto__: [] } instanceof Array && function (d, b) { d.__proto__ = b; }) ||
            function (d, b) { for (var p in b) if (Object.prototype.hasOwnProperty.call(b, p)) d[p] = b[p]; };
        return extendStatics(d, b);
    };
    return function (d, b) {
        if (typeof b !== "function" && b !== null)
            throw new TypeError("Class extends value " + String(b) + " is not a constructor or null");
        extendStatics(d, b);
        function __() { this.constructor = d; }
        d.prototype = b === null ? Object.create(b) : (__.prototype = b.prototype, new __());
    };
})();
Object.defineProperty(exports, "__esModule", { value: true });
exports.FixedBytesCoder = void 0;
var bytes_1 = require("@ethersproject/bytes");
var abstract_coder_1 = require("./abstract-coder");
// @TODO: Merge this with bytes
var FixedBytesCoder = /** @class */ (function (_super) {
    __extends(FixedBytesCoder, _super);
    function FixedBytesCoder(size, localName) {
        var _this = this;
        var name = "bytes" + String(size);
        _this = _super.call(this, name, name, localName, false) || this;
        _this.size = size;
        return _this;
    }
    FixedBytesCoder.prototype.defaultValue = function () {
        return ("0x0000000000000000000000000000000000000000000000000000000000000000").substring(0, 2 + this.size * 2);
    };
    FixedBytesCoder.prototype.encode = function (writer, value) {
        var data = (0, bytes_1.arrayify)(value);
        if (data.length !== this.size) {
            this._throwError("incorrect data length", value);
        }
        return writer.writeBytes(data);
    };
    FixedBytesCoder.prototype.decode = function (reader) {
        return reader.coerce(this.name, (0, bytes_1.hexlify)(reader.readBytes(this.size)));
    };
    return FixedBytesCoder;
}(abstract_coder_1.Coder));
exports.FixedBytesCoder = FixedBytesCoder;
//# sourceMappingURL=fixed-bytes.js.map