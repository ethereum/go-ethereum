"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.nameprep = exports.parseBytes32String = exports.formatBytes32String = exports.UnicodeNormalizationForm = exports.Utf8ErrorReason = exports.Utf8ErrorFuncs = exports.toUtf8String = exports.toUtf8CodePoints = exports.toUtf8Bytes = exports._toEscapedUtf8String = void 0;
var bytes32_1 = require("./bytes32");
Object.defineProperty(exports, "formatBytes32String", { enumerable: true, get: function () { return bytes32_1.formatBytes32String; } });
Object.defineProperty(exports, "parseBytes32String", { enumerable: true, get: function () { return bytes32_1.parseBytes32String; } });
var idna_1 = require("./idna");
Object.defineProperty(exports, "nameprep", { enumerable: true, get: function () { return idna_1.nameprep; } });
var utf8_1 = require("./utf8");
Object.defineProperty(exports, "_toEscapedUtf8String", { enumerable: true, get: function () { return utf8_1._toEscapedUtf8String; } });
Object.defineProperty(exports, "toUtf8Bytes", { enumerable: true, get: function () { return utf8_1.toUtf8Bytes; } });
Object.defineProperty(exports, "toUtf8CodePoints", { enumerable: true, get: function () { return utf8_1.toUtf8CodePoints; } });
Object.defineProperty(exports, "toUtf8String", { enumerable: true, get: function () { return utf8_1.toUtf8String; } });
Object.defineProperty(exports, "UnicodeNormalizationForm", { enumerable: true, get: function () { return utf8_1.UnicodeNormalizationForm; } });
Object.defineProperty(exports, "Utf8ErrorFuncs", { enumerable: true, get: function () { return utf8_1.Utf8ErrorFuncs; } });
Object.defineProperty(exports, "Utf8ErrorReason", { enumerable: true, get: function () { return utf8_1.Utf8ErrorReason; } });
//# sourceMappingURL=index.js.map