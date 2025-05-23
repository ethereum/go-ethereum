"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Validator = void 0;
/*
This file is part of web3.js.

web3.js is free software: you can redistribute it and/or modify
it under the terms of the GNU Lesser General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

web3.js is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public License
along with web3.js.  If not, see <http://www.gnu.org/licenses/>.
*/
const web3_errors_1 = require("web3-errors");
const zod_1 = require("zod");
const errors_js_1 = require("./errors.js");
const formats_js_1 = __importDefault(require("./formats.js"));
const convertToZod = (schema) => {
    if ((!(schema === null || schema === void 0 ? void 0 : schema.type) || (schema === null || schema === void 0 ? void 0 : schema.type) === 'object') && (schema === null || schema === void 0 ? void 0 : schema.properties)) {
        const obj = {};
        for (const name of Object.keys(schema.properties)) {
            const zItem = convertToZod(schema.properties[name]);
            if (zItem) {
                obj[name] = zItem;
            }
        }
        if (Array.isArray(schema.required)) {
            return zod_1.z
                .object(obj)
                .partial()
                .required(schema.required.reduce((acc, v) => (Object.assign(Object.assign({}, acc), { [v]: true })), {}));
        }
        return zod_1.z.object(obj).partial();
    }
    if ((schema === null || schema === void 0 ? void 0 : schema.type) === 'array' && (schema === null || schema === void 0 ? void 0 : schema.items)) {
        if (Array.isArray(schema.items) && schema.items.length > 1
            && schema.maxItems !== undefined
            && new Set(schema.items.map((item) => item.$id)).size === schema.items.length) {
            const arr = [];
            for (const item of schema.items) {
                const zItem = convertToZod(item);
                if (zItem) {
                    arr.push(zItem);
                }
            }
            return zod_1.z.tuple(arr);
        }
        const nextSchema = Array.isArray(schema.items) ? schema.items[0] : schema.items;
        let zodArraySchema = zod_1.z.array(convertToZod(nextSchema));
        zodArraySchema = schema.minItems !== undefined ? zodArraySchema.min(schema.minItems) : zodArraySchema;
        zodArraySchema = schema.maxItems !== undefined ? zodArraySchema.max(schema.maxItems) : zodArraySchema;
        return zodArraySchema;
    }
    if (schema.oneOf && Array.isArray(schema.oneOf)) {
        return zod_1.z.union(schema.oneOf.map(oneOfSchema => convertToZod(oneOfSchema)));
    }
    if (schema === null || schema === void 0 ? void 0 : schema.format) {
        if (!formats_js_1.default[schema.format]) {
            throw new web3_errors_1.SchemaFormatError(schema.format);
        }
        return zod_1.z.any().refine(formats_js_1.default[schema.format], (value) => ({
            params: { value, format: schema.format },
        }));
    }
    if ((schema === null || schema === void 0 ? void 0 : schema.type) &&
        (schema === null || schema === void 0 ? void 0 : schema.type) !== 'object' &&
        typeof zod_1.z[String(schema.type)] === 'function') {
        return zod_1.z[String(schema.type)]();
    }
    return zod_1.z.object({ data: zod_1.z.any() }).partial();
};
class Validator {
    // eslint-disable-next-line no-useless-constructor, @typescript-eslint/no-empty-function
    static factory() {
        if (!Validator.validatorInstance) {
            Validator.validatorInstance = new Validator();
        }
        return Validator.validatorInstance;
    }
    validate(schema, data, options) {
        var _a, _b;
        const zod = convertToZod(schema);
        const result = zod.safeParse(data);
        if (!result.success) {
            const errors = this.convertErrors((_b = (_a = result.error) === null || _a === void 0 ? void 0 : _a.issues) !== null && _b !== void 0 ? _b : []);
            if (errors) {
                if (options === null || options === void 0 ? void 0 : options.silent) {
                    return errors;
                }
                throw new errors_js_1.Web3ValidatorError(errors);
            }
        }
        return undefined;
    }
    // eslint-disable-next-line class-methods-use-this
    convertErrors(errors) {
        if (errors && Array.isArray(errors) && errors.length > 0) {
            return errors.map((error) => {
                var _a;
                let message;
                let keyword;
                let params;
                let schemaPath;
                schemaPath = error.path.join('/');
                const field = String(error.path[error.path.length - 1]);
                const instancePath = error.path.join('/');
                if (error.code === zod_1.ZodIssueCode.too_big) {
                    keyword = 'maxItems';
                    schemaPath = `${instancePath}/maxItems`;
                    params = { limit: error.maximum };
                    message = `must NOT have more than ${error.maximum} items`;
                }
                else if (error.code === zod_1.ZodIssueCode.too_small) {
                    keyword = 'minItems';
                    schemaPath = `${instancePath}/minItems`;
                    params = { limit: error.minimum };
                    message = `must NOT have fewer than ${error.minimum} items`;
                }
                else if (error.code === zod_1.ZodIssueCode.custom) {
                    const { value, format } = ((_a = error.params) !== null && _a !== void 0 ? _a : {});
                    if (typeof value === 'undefined') {
                        message = `value at "/${schemaPath}" is required`;
                    }
                    else {
                        message = `value "${
                        // eslint-disable-next-line @typescript-eslint/restrict-template-expressions
                        typeof value === 'object' ? JSON.stringify(value) : value}" at "/${schemaPath}" must pass "${format}" validation`;
                    }
                    params = { value };
                }
                return {
                    keyword: keyword !== null && keyword !== void 0 ? keyword : field,
                    instancePath: instancePath ? `/${instancePath}` : '',
                    schemaPath: schemaPath ? `#${schemaPath}` : '#',
                    // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
                    params: params !== null && params !== void 0 ? params : { value: error.message },
                    message: message !== null && message !== void 0 ? message : error.message,
                };
            });
        }
        return undefined;
    }
}
exports.Validator = Validator;
//# sourceMappingURL=validator.js.map