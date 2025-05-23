import { Validator } from './validator.js';
import { ethAbiToJsonSchema } from './utils.js';
import { Web3ValidatorError } from './errors.js';
export class Web3Validator {
    constructor() {
        this._validator = Validator.factory();
    }
    validateJSONSchema(schema, data, options) {
        return this._validator.validate(schema, data, options);
    }
    validate(schema, data, options = { silent: false }) {
        var _a, _b;
        const jsonSchema = ethAbiToJsonSchema(schema);
        if (Array.isArray(jsonSchema.items) &&
            ((_a = jsonSchema.items) === null || _a === void 0 ? void 0 : _a.length) === 0 &&
            data.length === 0) {
            return undefined;
        }
        if (Array.isArray(jsonSchema.items) &&
            ((_b = jsonSchema.items) === null || _b === void 0 ? void 0 : _b.length) === 0 &&
            data.length !== 0) {
            throw new Web3ValidatorError([
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
//# sourceMappingURL=web3_validator.js.map