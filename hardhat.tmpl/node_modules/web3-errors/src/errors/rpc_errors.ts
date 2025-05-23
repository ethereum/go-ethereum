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

import { JsonRpcResponseWithError, JsonRpcId, JsonRpcError } from 'web3-types';
import { BaseWeb3Error } from '../web3_error_base.js';
import {
	ERR_RPC_INTERNAL_ERROR,
	ERR_RPC_INVALID_INPUT,
	ERR_RPC_INVALID_JSON,
	ERR_RPC_INVALID_METHOD,
	ERR_RPC_INVALID_PARAMS,
	ERR_RPC_INVALID_REQUEST,
	ERR_RPC_LIMIT_EXCEEDED,
	ERR_RPC_MISSING_RESOURCE,
	ERR_RPC_NOT_SUPPORTED,
	ERR_RPC_TRANSACTION_REJECTED,
	ERR_RPC_UNAVAILABLE_RESOURCE,
	ERR_RPC_UNSUPPORTED_METHOD,
} from '../error_codes.js';
import { RpcErrorMessages, genericRpcErrorMessageTemplate } from './rpc_error_messages.js';

export class RpcError extends BaseWeb3Error {
	public code: number;
	public id: JsonRpcId;
	public jsonrpc: string;
	public jsonRpcError: JsonRpcError;
	public constructor(rpcError: JsonRpcResponseWithError, message?: string) {
		super(
			message ??
				genericRpcErrorMessageTemplate.replace('*code*', rpcError.error.code.toString()),
		);
		this.code = rpcError.error.code;
		this.id = rpcError.id;
		this.jsonrpc = rpcError.jsonrpc;
		this.jsonRpcError = rpcError.error;
	}

	public toJSON() {
		return { ...super.toJSON(), error: this.jsonRpcError, id: this.id, jsonRpc: this.jsonrpc };
	}
}

export class EIP1193ProviderRpcError extends BaseWeb3Error {
	public code: number;
	public data?: unknown;

	public constructor(code: number, data?: unknown) {
		if (!code) {
			// this case should ideally not happen
			super();
		} else if (RpcErrorMessages[code]?.message) {
			super(RpcErrorMessages[code].message);
		} else {
			// Retrieve the status code object for the given code from the table, by searching through the appropriate range
			const statusCodeRange = Object.keys(RpcErrorMessages).find(
				statusCode =>
					typeof statusCode === 'string' &&
					code >= parseInt(statusCode.split('-')[0], 10) &&
					code <= parseInt(statusCode.split('-')[1], 10),
			);
			super(
				RpcErrorMessages[statusCodeRange ?? '']?.message ??
					genericRpcErrorMessageTemplate.replace('*code*', code?.toString() ?? '""'),
			);
		}
		this.code = code;
		this.data = data;
	}
}

export class ParseError extends RpcError {
	public code = ERR_RPC_INVALID_JSON;
	public constructor(rpcError: JsonRpcResponseWithError) {
		super(rpcError, RpcErrorMessages[ERR_RPC_INVALID_JSON].message);
	}
}

export class InvalidRequestError extends RpcError {
	public code = ERR_RPC_INVALID_REQUEST;
	public constructor(rpcError: JsonRpcResponseWithError) {
		super(rpcError, RpcErrorMessages[ERR_RPC_INVALID_REQUEST].message);
	}
}

export class MethodNotFoundError extends RpcError {
	public code = ERR_RPC_INVALID_METHOD;
	public constructor(rpcError: JsonRpcResponseWithError) {
		super(rpcError, RpcErrorMessages[ERR_RPC_INVALID_METHOD].message);
	}
}

export class InvalidParamsError extends RpcError {
	public code = ERR_RPC_INVALID_PARAMS;
	public constructor(rpcError: JsonRpcResponseWithError) {
		super(rpcError, RpcErrorMessages[ERR_RPC_INVALID_PARAMS].message);
	}
}

export class InternalError extends RpcError {
	public code = ERR_RPC_INTERNAL_ERROR;
	public constructor(rpcError: JsonRpcResponseWithError) {
		super(rpcError, RpcErrorMessages[ERR_RPC_INTERNAL_ERROR].message);
	}
}

export class InvalidInputError extends RpcError {
	public code = ERR_RPC_INVALID_INPUT;
	public constructor(rpcError: JsonRpcResponseWithError) {
		super(rpcError, RpcErrorMessages[ERR_RPC_INVALID_INPUT].message);
	}
}

export class MethodNotSupported extends RpcError {
	public code = ERR_RPC_UNSUPPORTED_METHOD;
	public constructor(rpcError: JsonRpcResponseWithError) {
		super(rpcError, RpcErrorMessages[ERR_RPC_UNSUPPORTED_METHOD].message);
	}
}

export class ResourceUnavailableError extends RpcError {
	public code = ERR_RPC_UNAVAILABLE_RESOURCE;
	public constructor(rpcError: JsonRpcResponseWithError) {
		super(rpcError, RpcErrorMessages[ERR_RPC_UNAVAILABLE_RESOURCE].message);
	}
}

export class ResourcesNotFoundError extends RpcError {
	public code = ERR_RPC_MISSING_RESOURCE;
	public constructor(rpcError: JsonRpcResponseWithError) {
		super(rpcError, RpcErrorMessages[ERR_RPC_MISSING_RESOURCE].message);
	}
}

export class VersionNotSupportedError extends RpcError {
	public code = ERR_RPC_NOT_SUPPORTED;
	public constructor(rpcError: JsonRpcResponseWithError) {
		super(rpcError, RpcErrorMessages[ERR_RPC_NOT_SUPPORTED].message);
	}
}

export class TransactionRejectedError extends RpcError {
	public code = ERR_RPC_TRANSACTION_REJECTED;
	public constructor(rpcError: JsonRpcResponseWithError) {
		super(rpcError, RpcErrorMessages[ERR_RPC_TRANSACTION_REJECTED].message);
	}
}

export class LimitExceededError extends RpcError {
	public code = ERR_RPC_LIMIT_EXCEEDED;
	public constructor(rpcError: JsonRpcResponseWithError) {
		super(rpcError, RpcErrorMessages[ERR_RPC_LIMIT_EXCEEDED].message);
	}
}

export const rpcErrorsMap = new Map<number, { error: typeof RpcError }>();
rpcErrorsMap.set(ERR_RPC_INVALID_JSON, { error: ParseError });
rpcErrorsMap.set(ERR_RPC_INVALID_REQUEST, {
	error: InvalidRequestError,
});
rpcErrorsMap.set(ERR_RPC_INVALID_METHOD, {
	error: MethodNotFoundError,
});
rpcErrorsMap.set(ERR_RPC_INVALID_PARAMS, { error: InvalidParamsError });
rpcErrorsMap.set(ERR_RPC_INTERNAL_ERROR, { error: InternalError });
rpcErrorsMap.set(ERR_RPC_INVALID_INPUT, { error: InvalidInputError });
rpcErrorsMap.set(ERR_RPC_UNSUPPORTED_METHOD, {
	error: MethodNotSupported,
});
rpcErrorsMap.set(ERR_RPC_UNAVAILABLE_RESOURCE, {
	error: ResourceUnavailableError,
});
rpcErrorsMap.set(ERR_RPC_TRANSACTION_REJECTED, {
	error: TransactionRejectedError,
});
rpcErrorsMap.set(ERR_RPC_MISSING_RESOURCE, {
	error: ResourcesNotFoundError,
});
rpcErrorsMap.set(ERR_RPC_NOT_SUPPORTED, {
	error: VersionNotSupportedError,
});
rpcErrorsMap.set(ERR_RPC_LIMIT_EXCEEDED, { error: LimitExceededError });
