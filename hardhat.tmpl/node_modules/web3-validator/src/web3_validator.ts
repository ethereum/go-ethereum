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
import { Web3ValidationErrorObject } from 'web3-types';

import { Validator } from './validator.js';
import { ethAbiToJsonSchema } from './utils.js';
import { Json, ValidationSchemaInput, Web3ValidationOptions } from './types.js';
import { Web3ValidatorError } from './errors.js';

export class Web3Validator {
	private readonly _validator: Validator;
	public constructor() {
		this._validator = Validator.factory();
	}
	public validateJSONSchema(
		schema: object,
		data: object,
		options?: Web3ValidationOptions,
	): Web3ValidationErrorObject[] | undefined {
		return this._validator.validate(schema, data as Json, options);
	}
	public validate(
		schema: ValidationSchemaInput,
		data: ReadonlyArray<unknown>,
		options: Web3ValidationOptions = { silent: false },
	): Web3ValidationErrorObject[] | undefined {
		const jsonSchema = ethAbiToJsonSchema(schema);
		if (
			Array.isArray(jsonSchema.items) &&
			jsonSchema.items?.length === 0 &&
			data.length === 0
		) {
			return undefined;
		}

		if (
			Array.isArray(jsonSchema.items) &&
			jsonSchema.items?.length === 0 &&
			data.length !== 0
		) {
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

		return this._validator.validate(jsonSchema, data as Json, options);
	}
}
