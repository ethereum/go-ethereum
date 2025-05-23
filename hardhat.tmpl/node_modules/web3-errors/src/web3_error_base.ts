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

/* eslint-disable max-classes-per-file */

import { Web3Error } from 'web3-types';
import { ERR_MULTIPLE_ERRORS } from './error_codes.js';

/**
 * Base class for Web3 errors.
 */
export abstract class BaseWeb3Error extends Error implements Web3Error {
	public readonly name: string;
	public abstract readonly code: number;
	public stack: string | undefined;

	public cause: Error | undefined;

	/**
	 * @deprecated Use the `cause` property instead.
	 */
	public get innerError(): Error | Error[] | undefined {
		// eslint-disable-next-line no-use-before-define
		if (this.cause instanceof MultipleErrors) {
			return this.cause.errors;
		}
		return this.cause;
	}
	/**
	 * @deprecated Use the `cause` property instead.
	 */
	public set innerError(cause: Error | Error[] | undefined) {
		if (Array.isArray(cause)) {
			// eslint-disable-next-line no-use-before-define
			this.cause = new MultipleErrors(cause);
		} else {
			this.cause = cause;
		}
	}

	public constructor(msg?: string, cause?: Error | Error[]) {
		super(msg);

		if (Array.isArray(cause)) {
			// eslint-disable-next-line no-use-before-define
			this.cause = new MultipleErrors(cause);
		} else {
			this.cause = cause;
		}

		this.name = this.constructor.name;

		if (typeof Error.captureStackTrace === 'function') {
			Error.captureStackTrace(new.target.constructor);
		} else {
			this.stack = new Error().stack;
		}
	}

	public static convertToString(value: unknown, unquotValue = false) {
		// Using "null" value intentionally for validation
		// eslint-disable-next-line no-null/no-null
		if (value === null || value === undefined) return 'undefined';

		const result = JSON.stringify(
			value,
			(_, v) => (typeof v === 'bigint' ? v.toString() : v) as unknown,
		);

		return unquotValue && ['bigint', 'string'].includes(typeof value)
			? result.replace(/['\\"]+/g, '')
			: result;
	}

	public toJSON() {
		return {
			name: this.name,
			code: this.code,
			message: this.message,
			cause: this.cause,
			// deprecated
			innerError: this.cause,
		};
	}
}

export class MultipleErrors extends BaseWeb3Error {
	public code = ERR_MULTIPLE_ERRORS;
	public errors: Error[];

	public constructor(errors: Error[]) {
		super(`Multiple errors occurred: [${errors.map(e => e.message).join('], [')}]`);
		this.errors = errors;
	}
}

export abstract class InvalidValueError extends BaseWeb3Error {
	public readonly name: string;

	public constructor(value: unknown, msg: string) {
		super(
			`Invalid value given "${BaseWeb3Error.convertToString(value, true)}". Error: ${msg}.`,
		);
		this.name = this.constructor.name;
	}
}
