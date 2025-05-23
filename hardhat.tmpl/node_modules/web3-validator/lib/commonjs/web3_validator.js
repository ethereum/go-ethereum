"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Web3Validator = void 0;
const validator_js_1 = require("./validator.js");
const utils_js_1 = require("./utils.js");
const errors_js_1 = require("./errors.js");
class Web3Validator {
    constructor() {
        this._validator = validator_js_1.Validator.factory();
    }
    validateJSONSchema(schema, data, options) {
        return this._validator.validate(schema, data, options);
    }
    validate(schema, data, options = { silent: false }) {
        var _a, _b;
        const jsonSchema = (0, utils_js_1.ethAbiToJsonSchema)(schema);
        if (Array.isArray(jsonSchema.items) &&
            ((_a = jsonSchema.items) === null || _a === void 0 ? void 0 : _a.length) === 0 &&
            data.length === 0) {
            return undefined;
        }
        if (Array.isArray(jsonSchema.items) &&
            ((_b = jsonSchema.items) === null || _b === void 0 ? void 0 : _b.length) === 0 &&
            data.length !== 0) {
            throw new errors_js_1.Web3ValidatorError([
                {
                    instancePath: '/0',
                    schemaPath: '/',
                    keyword: 'required',
                    message: 'empty schema against data can not be validated',
                    params: data,
                },
            ]);
        }
        return this._validator.validate(jsonSchema, data, options);
    }
}
exports.Web3Validator = Web3Validator;
//# sourceMappingURL=web3_validator.js.map