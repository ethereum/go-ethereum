"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.defineProperties = void 0;
var assert = require('assert');
var ethjsUtil = require('ethjs-util');
var rlp = require("rlp");
var bytes_1 = require("./bytes");
/**
 * Defines properties on a `Object`. It make the assumption that underlying data is binary.
 * @param self the `Object` to define properties on
 * @param fields an array fields to define. Fields can contain:
 * * `name` - the name of the properties
 * * `length` - the number of bytes the field can have
 * * `allowLess` - if the field can be less than the length
 * * `allowEmpty`
 * @param data data to be validated against the definitions
 * @deprecated
 */
exports.defineProperties = function (self, fields, data) {
    self.raw = [];
    self._fields = [];
    // attach the `toJSON`
    self.toJSON = function (label) {
        if (label === void 0) { label = false; }
        if (label) {
            var obj_1 = {};
            self._fields.forEach(function (field) {
                obj_1[field] = "0x" + self[field].toString('hex');
            });
            return obj_1;
        }
        return bytes_1.baToJSON(self.raw);
    };
    self.serialize = function serialize() {
        return rlp.encode(self.raw);
    };
    fields.forEach(function (field, i) {
        self._fields.push(field.name);
        function getter() {
            return self.raw[i];
        }
        function setter(v) {
            v = bytes_1.toBuffer(v);
            if (v.toString('hex') === '00' && !field.allowZero) {
                v = Buffer.allocUnsafe(0);
            }
            if (field.allowLess && field.length) {
                v = bytes_1.stripZeros(v);
                assert(field.length >= v.length, "The field " + field.name + " must not have more " + field.length + " bytes");
            }
            else if (!(field.allowZero && v.length === 0) && field.length) {
                assert(field.length === v.length, "The field " + field.name + " must have byte length of " + field.length);
            }
            self.raw[i] = v;
        }
        Object.defineProperty(self, field.name, {
            enumerable: true,
            configurable: true,
            get: getter,
            set: setter,
        });
        if (field.default) {
            self[field.name] = field.default;
        }
        // attach alias
        if (field.alias) {
            Object.defineProperty(self, field.alias, {
                enumerable: false,
                configurable: true,
                set: setter,
                get: getter,
            });
        }
    });
    // if the constuctor is passed data
    if (data) {
        if (typeof data === 'string') {
            data = Buffer.from(ethjsUtil.stripHexPrefix(data), 'hex');
        }
        if (Buffer.isBuffer(data)) {
            data = rlp.decode(data);
        }
        if (Array.isArray(data)) {
            if (data.length > self._fields.length) {
                throw new Error('wrong number of fields in data');
            }
            // make sure all the items are buffers
            data.forEach(function (d, i) {
                self[self._fields[i]] = bytes_1.toBuffer(d);
            });
        }
        else if (typeof data === 'object') {
            var keys_1 = Object.keys(data);
            fields.forEach(function (field) {
                if (keys_1.indexOf(field.name) !== -1)
                    self[field.name] = data[field.name];
                if (keys_1.indexOf(field.alias) !== -1)
                    self[field.alias] = data[field.alias];
            });
        }
        else {
            throw new Error('invalid data');
        }
    }
};
//# sourceMappingURL=object.js.map