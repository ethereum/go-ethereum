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
import { SchemaFormatError } from 'web3-errors';
import { Web3ValidationErrorObject } from 'web3-types';

import { z, ZodType, ZodIssue, ZodIssueCode, ZodTypeAny } from 'zod';

import { RawCreateParams } from 'zod/lib/types';
import { Web3ValidatorError } from './errors.js';
import { Json, JsonSchema } from './types.js';
import formats from './formats.js';

const convertToZod = (schema: JsonSchema): ZodType => {
	if ((!schema?.type || schema?.type === 'object') && schema?.properties) {
		const obj: { [key: string]: ZodType } = {};
		for (const name of Object.keys(schema.properties)) {
			const zItem = convertToZod(schema.properties[name]);
			if (zItem) {
				obj[name] = zItem;
			}
		}

		if (Array.isArray(schema.required)) {
			return z
				.object(obj)
				.partial()
				.required(schema.required.reduce((acc, v: string) => ({ ...acc, [v]: true }), {}));
		}
		return z.object(obj).partial();
	}

	if (schema?.type === 'array' && schema?.items) {
		if (Array.isArray(schema.items) && schema.items.length > 1
		    && schema.maxItems !== undefined
		    && new Set(schema.items.map((item: JsonSchema) => item.$id)).size === schema.items.length) {
			const arr: Partial<[ZodTypeAny, ...ZodTypeAny[]]> = [];
			for (const item of schema.items) {
				const zItem = convertToZod(item);
				if (zItem) {
					arr.push(zItem);
				}
			}
			return z.tuple(arr as [ZodTypeAny, ...ZodTypeAny[]]);
		}
		const nextSchema = Array.isArray(schema.items) ? schema.items[0] : schema.items;
        let zodArraySchema = z.array(convertToZod(nextSchema));

        zodArraySchema = schema.minItems !== undefined ? zodArraySchema.min(schema.minItems) : zodArraySchema;
        zodArraySchema = schema.maxItems !== undefined ? zodArraySchema.max(schema.maxItems) : zodArraySchema;
		return zodArraySchema;
	}

	if (schema.oneOf && Array.isArray(schema.oneOf)) {
		return z.union(
			schema.oneOf.map(oneOfSchema => convertToZod(oneOfSchema)) as [
				ZodTypeAny,
				ZodTypeAny,
				...ZodTypeAny[],
			],
		);
	}

	if (schema?.format) {
		if (!formats[schema.format]) {
			throw new SchemaFormatError(schema.format);
		}

		return z.any().refine(formats[schema.format], (value: unknown) => ({
			params: { value, format: schema.format },
		}));
	}

	if (
		schema?.type &&
		schema?.type !== 'object' &&
		typeof (z as unknown as { [key: string]: (params?: RawCreateParams) => ZodType })[
			String(schema.type)
		] === 'function'
	) {
		return (z as unknown as { [key: string]: (params?: RawCreateParams) => ZodType })[
			String(schema.type)
		]();
	}
	return z.object({ data: z.any() }).partial();
};

export class Validator {
	// eslint-disable-next-line no-use-before-define
	private static validatorInstance?: Validator;

	// eslint-disable-next-line no-useless-constructor, @typescript-eslint/no-empty-function
	public static factory(): Validator {
		if (!Validator.validatorInstance) {
			Validator.validatorInstance = new Validator();
		}
		return Validator.validatorInstance;
	}

	public validate(schema: JsonSchema, data: Json, options?: { silent?: boolean }) {
		const zod = convertToZod(schema);
		const result = zod.safeParse(data);
		if (!result.success) {
			const errors = this.convertErrors(result.error?.issues ?? []);
			if (errors) {
				if (options?.silent) {
					return errors;
				}
				throw new Web3ValidatorError(errors);
			}
		}
		return undefined;
	}
	// eslint-disable-next-line class-methods-use-this
	private convertErrors(errors: ZodIssue[] | undefined): Web3ValidationErrorObject[] | undefined {
		if (errors && Array.isArray(errors) && errors.length > 0) {
			return errors.map((error: ZodIssue) => {
				let message;
				let keyword;
				let params;
				let schemaPath;

				schemaPath = error.path.join('/');

				const field = String(error.path[error.path.length - 1]);
				const instancePath = error.path.join('/');
				if (error.code === ZodIssueCode.too_big) {
					keyword = 'maxItems';
					schemaPath = `${instancePath}/maxItems`;
					params = { limit: error.maximum };
					message = `must NOT have more than ${error.maximum} items`;
				} else if (error.code === ZodIssueCode.too_small) {
					keyword = 'minItems';
					schemaPath = `${instancePath}/minItems`;
					params = { limit: error.minimum };
					message = `must NOT have fewer than ${error.minimum} items`;
				} else if (error.code === ZodIssueCode.custom) {
					const { value, format } = (error.params ?? {}) as {
						value: unknown;
						format: string;
					};

					if (typeof value === 'undefined') {
						message = `value at "/${schemaPath}" is required`;
					} else {
						message = `value "${
							// eslint-disable-next-line @typescript-eslint/restrict-template-expressions
							typeof value === 'object' ? JSON.stringify(value) : value
						}" at "/${schemaPath}" must pass "${format}" validation`;
					}

					params = { value };
				}

				return {
					keyword: keyword ?? field,
					instancePath: instancePath ? `/${instancePath}` : '',
					schemaPath: schemaPath ? `#${schemaPath}` : '#',
					// eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
					params: params ?? { value: error.message },
					message: message ?? error.message,
				} as Web3ValidationErrorObject;
			});
		}
		return undefined;
	}
}
