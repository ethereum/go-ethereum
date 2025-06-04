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
exports.TupleCoder = void 0;
var abstract_coder_1 = require("./abstract-coder");
var array_1 = require("./array");
var TupleCoder = /** @class */ (function (_super) {
    __extends(TupleCoder, _super);
    function TupleCoder(coders, localName) {
        var _this = this;
        var dynamic = false;
        var types = [];
        coders.forEach(function (coder) {
            if (coder.dynamic) {
                dynamic = true;
            }
            types.push(coder.type);
        });
        var type = ("tuple(" + types.join(",") + ")");
        _this = _super.call(this, "tuple", type, localName, dynamic) || this;
        _this.coders = coders;
        return _this;
    }
    TupleCoder.prototype.defaultValue = function () {
        var values = [];
        this.coders.forEach(function (coder) {
            values.push(coder.defaultValue());
        });
        // We only output named properties for uniquely named coders
        var uniqueNames = this.coders.reduce(function (accum, coder) {
            var name = coder.localName;
            if (name) {
                if (!accum[name]) {
                    accum[name] = 0;
                }
                accum[name]++;
            }
            return accum;
        }, {});
        // Add named values
        this.coders.forEach(function (coder, index) {
            var name = coder.localName;
            if (!name || uniqueNames[name] !== 1) {
                return;
            }
            if (name === "length") {
                name = "_length";
            }
            if (values[name] != null) {
                return;
            }
            values[name] = values[index];
        });
        return Object.freeze(values);
    };
    TupleCoder.prototype.encode = function (writer, value) {
        return (0, array_1.pack)(writer, this.coders, value);
    };
    TupleCoder.prototype.decode = function (reader) {
        return reader.coerce(this.name, (0, array_1.unpack)(reader, this.coders));
    };
    return TupleCoder;
}(abstract_coder_1.Coder));
exports.TupleCoder = TupleCoder;
//# sourceMappingURL=tuple.js.map