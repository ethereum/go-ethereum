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
exports.ArrayCoder = exports.unpack = exports.pack = void 0;
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("../_version");
var logger = new logger_1.Logger(_version_1.version);
var abstract_coder_1 = require("./abstract-coder");
var anonymous_1 = require("./anonymous");
function pack(writer, coders, values) {
    var arrayValues = null;
    if (Array.isArray(values)) {
        arrayValues = values;
    }
    else if (values && typeof (values) === "object") {
        var unique_1 = {};
        arrayValues = coders.map(function (coder) {
            var name = coder.localName;
            if (!name) {
                logger.throwError("cannot encode object for signature with missing names", logger_1.Logger.errors.INVALID_ARGUMENT, {
                    argument: "values",
                    coder: coder,
                    value: values
                });
            }
            if (unique_1[name]) {
                logger.throwError("cannot encode object for signature with duplicate names", logger_1.Logger.errors.INVALID_ARGUMENT, {
                    argument: "values",
                    coder: coder,
                    value: values
                });
            }
            unique_1[name] = true;
            return values[name];
        });
    }
    else {
        logger.throwArgumentError("invalid tuple value", "tuple", values);
    }
    if (coders.length !== arrayValues.length) {
        logger.throwArgumentError("types/value length mismatch", "tuple", values);
    }
    var staticWriter = new abstract_coder_1.Writer(writer.wordSize);
    var dynamicWriter = new abstract_coder_1.Writer(writer.wordSize);
    var updateFuncs = [];
    coders.forEach(function (coder, index) {
        var value = arrayValues[index];
        if (coder.dynamic) {
            // Get current dynamic offset (for the future pointer)
            var dynamicOffset_1 = dynamicWriter.length;
            // Encode the dynamic value into the dynamicWriter
            coder.encode(dynamicWriter, value);
            // Prepare to populate the correct offset once we are done
            var updateFunc_1 = staticWriter.writeUpdatableValue();
            updateFuncs.push(function (baseOffset) {
                updateFunc_1(baseOffset + dynamicOffset_1);
            });
        }
        else {
            coder.encode(staticWriter, value);
        }
    });
    // Backfill all the dynamic offsets, now that we know the static length
    updateFuncs.forEach(function (func) { func(staticWriter.length); });
    var length = writer.appendWriter(staticWriter);
    length += writer.appendWriter(dynamicWriter);
    return length;
}
exports.pack = pack;
function unpack(reader, coders) {
    var values = [];
    // A reader anchored to this base
    var baseReader = reader.subReader(0);
    coders.forEach(function (coder) {
        var value = null;
        if (coder.dynamic) {
            var offset = reader.readValue();
            var offsetReader = baseReader.subReader(offset.toNumber());
            try {
                value = coder.decode(offsetReader);
            }
            catch (error) {
                // Cannot recover from this
                if (error.code === logger_1.Logger.errors.BUFFER_OVERRUN) {
                    throw error;
                }
                value = error;
                value.baseType = coder.name;
                value.name = coder.localName;
                value.type = coder.type;
            }
        }
        else {
            try {
                value = coder.decode(reader);
            }
            catch (error) {
                // Cannot recover from this
                if (error.code === logger_1.Logger.errors.BUFFER_OVERRUN) {
                    throw error;
                }
                value = error;
                value.baseType = coder.name;
                value.name = coder.localName;
                value.type = coder.type;
            }
        }
        if (value != undefined) {
            values.push(value);
        }
    });
    // We only output named properties for uniquely named coders
    var uniqueNames = coders.reduce(function (accum, coder) {
        var name = coder.localName;
        if (name) {
            if (!accum[name]) {
                accum[name] = 0;
            }
            accum[name]++;
        }
        return accum;
    }, {});
    // Add any named parameters (i.e. tuples)
    coders.forEach(function (coder, index) {
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
        var value = values[index];
        if (value instanceof Error) {
            Object.defineProperty(values, name, {
                enumerable: true,
                get: function () { throw value; }
            });
        }
        else {
            values[name] = value;
        }
    });
    var _loop_1 = function (i) {
        var value = values[i];
        if (value instanceof Error) {
            Object.defineProperty(values, i, {
                enumerable: true,
                get: function () { throw value; }
            });
        }
    };
    for (var i = 0; i < values.length; i++) {
        _loop_1(i);
    }
    return Object.freeze(values);
}
exports.unpack = unpack;
var ArrayCoder = /** @class */ (function (_super) {
    __extends(ArrayCoder, _super);
    function ArrayCoder(coder, length, localName) {
        var _this = this;
        var type = (coder.type + "[" + (length >= 0 ? length : "") + "]");
        var dynamic = (length === -1 || coder.dynamic);
        _this = _super.call(this, "array", type, localName, dynamic) || this;
        _this.coder = coder;
        _this.length = length;
        return _this;
    }
    ArrayCoder.prototype.defaultValue = function () {
        // Verifies the child coder is valid (even if the array is dynamic or 0-length)
        var defaultChild = this.coder.defaultValue();
        var result = [];
        for (var i = 0; i < this.length; i++) {
            result.push(defaultChild);
        }
        return result;
    };
    ArrayCoder.prototype.encode = function (writer, value) {
        if (!Array.isArray(value)) {
            this._throwError("expected array value", value);
        }
        var count = this.length;
        if (count === -1) {
            count = value.length;
            writer.writeValue(value.length);
        }
        logger.checkArgumentCount(value.length, count, "coder array" + (this.localName ? (" " + this.localName) : ""));
        var coders = [];
        for (var i = 0; i < value.length; i++) {
            coders.push(this.coder);
        }
        return pack(writer, coders, value);
    };
    ArrayCoder.prototype.decode = function (reader) {
        var count = this.length;
        if (count === -1) {
            count = reader.readValue().toNumber();
            // Check that there is *roughly* enough data to ensure
            // stray random data is not being read as a length. Each
            // slot requires at least 32 bytes for their value (or 32
            // bytes as a link to the data). This could use a much
            // tighter bound, but we are erroring on the side of safety.
            if (count * 32 > reader._data.length) {
                logger.throwError("insufficient data length", logger_1.Logger.errors.BUFFER_OVERRUN, {
                    length: reader._data.length,
                    count: count
                });
            }
        }
        var coders = [];
        for (var i = 0; i < count; i++) {
            coders.push(new anonymous_1.AnonymousCoder(this.coder));
        }
        return reader.coerce(this.name, unpack(reader, coders));
    };
    return ArrayCoder;
}(abstract_coder_1.Coder));
exports.ArrayCoder = ArrayCoder;
//# sourceMappingURL=array.js.map