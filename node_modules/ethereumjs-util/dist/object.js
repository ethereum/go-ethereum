"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.defineProperties = void 0;
const assert_1 = __importDefault(require("assert"));
const internal_1 = require("./internal");
const externals_1 = require("./externals");
const bytes_1 = require("./bytes");
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
const defineProperties = function (self, fields, data) {
    self.raw = [];
    self._fields = [];
    // attach the `toJSON`
    self.toJSON = function (label = false) {
        if (label) {
            const obj = {};
            self._fields.forEach((field) => {
                obj[field] = `0x${self[field].toString('hex')}`;
            });
            return obj;
        }
        return (0, bytes_1.baToJSON)(self.raw);
    };
    self.serialize = function serialize() {
        return externals_1.rlp.encode(self.raw);
    };
    fields.forEach((field, i) => {
        self._fields.push(field.name);
        function getter() {
            return self.raw[i];
        }
        function setter(v) {
            v = (0, bytes_1.toBuffer)(v);
            if (v.toString('hex') === '00' && !field.allowZero) {
                v = Buffer.allocUnsafe(0);
            }
            if (field.allowLess && field.length) {
                v = (0, bytes_1.unpadBuffer)(v);
                (0, assert_1.default)(field.length >= v.length, `The field ${field.name} must not have more ${field.length} bytes`);
            }
            else if (!(field.allowZero && v.length === 0) && field.length) {
                (0, assert_1.default)(field.length === v.length, `The field ${field.name} must have byte length of ${field.length}`);
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
            data = Buffer.from((0, internal_1.stripHexPrefix)(data), 'hex');
        }
        if (Buffer.isBuffer(data)) {
            data = externals_1.rlp.decode(data);
        }
        if (Array.isArray(data)) {
            if (data.length > self._fields.length) {
                throw new Error('wrong number of fields in data');
            }
            // make sure all the items are buffers
            data.forEach((d, i) => {
                self[self._fields[i]] = (0, bytes_1.toBuffer)(d);
            });
        }
        else if (typeof data === 'object') {
            const keys = Object.keys(data);
            fields.forEach((field) => {
                if (keys.indexOf(field.name) !== -1)
                    self[field.name] = data[field.name];
                if (keys.indexOf(field.alias) !== -1)
                    self[field.alias] = data[field.alias];
            });
        }
        else {
            throw new Error('invalid data');
        }
    }
};
exports.defineProperties = defineProperties;
//# sourceMappingURL=object.js.map