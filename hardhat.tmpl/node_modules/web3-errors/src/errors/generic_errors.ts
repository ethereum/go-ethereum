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

import {
	ERR_ABI_ENCODING,
	ERR_FORMATTERS,
	ERR_METHOD_NOT_IMPLEMENTED,
	ERR_OPERATION_ABORT,
	ERR_OPERATION_TIMEOUT,
	ERR_PARAM,
	ERR_EXISTING_PLUGIN_NAMESPACE,
	ERR_INVALID_METHOD_PARAMS,
} from '../error_codes.js';
import { BaseWeb3Error } from '../web3_error_base.js';

export class InvalidNumberOfParamsError extends BaseWeb3Error {
	public code = ERR_PARAM;

	public constructor(public got: number, public expected: number, public method: string) {
		super(`Invalid number of parameters for "${method}". Got "${got}" expected "${expected}"!`);
	}

	public toJSON() {
		return {
			...super.toJSON(),
			got: this.got,
			expected: this.expected,
			method: this.method,
		};
	}
}

export class InvalidMethodParamsError extends BaseWeb3Error {
	public code = ERR_INVALID_METHOD_PARAMS;

	public constructor(public hint?: string) {
		super(`Invalid parameters passed. "${typeof hint !== 'undefined' ? hint : ''}"`);
	}

	public toJSON() {
		return {
			...super.toJSON(),
			hint: this.hint,
		};
	}
}

export class FormatterError extends BaseWeb3Error {
	public code = ERR_FORMATTERS;
}

export class MethodNotImplementedError extends BaseWeb3Error {
	public code = ERR_METHOD_NOT_IMPLEMENTED;

	public constructor() {
		super("The method you're trying to call is not implemented.");
	}
}

export class OperationTimeoutError extends BaseWeb3Error {
	public code = ERR_OPERATION_TIMEOUT;
}

export class OperationAbortError extends BaseWeb3Error {
	public code = ERR_OPERATION_ABORT;
}

export class AbiError extends BaseWeb3Error {
	public code = ERR_ABI_ENCODING;
	public readonly props: Record<string, unknown> & { name?: string };

	public constructor(message: string, props?: Record<string, unknown> & { name?: string }) {
		super(message);
		this.props = props ?? {};
	}
}

export class ExistingPluginNamespaceError extends BaseWeb3Error {
	public code = ERR_EXISTING_PLUGIN_NAMESPACE;

	public constructor(pluginNamespace: string) {
		super(`A plugin with the namespace: ${pluginNamespace} has already been registered.`);
	}
}
