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
	JSONRPC_ERR_CHAIN_DISCONNECTED,
	JSONRPC_ERR_DISCONNECTED,
	JSONRPC_ERR_REJECTED_REQUEST,
	JSONRPC_ERR_UNAUTHORIZED,
	JSONRPC_ERR_UNSUPPORTED_METHOD,
} from '../error_codes.js';

/**
 * A template string for a generic Rpc Error. The `*code*` will be replaced with the code number.
 * Note: consider in next version that a spelling mistake could be corrected for `occured` and the value could be:
 * 	`An Rpc error has occurred with a code of *code*`
 */
export const genericRpcErrorMessageTemplate = 'An Rpc error has occured with a code of *code*';

/* eslint-disable @typescript-eslint/naming-convention */
export const RpcErrorMessages: {
	[key: number | string]: { name?: string; message: string; description?: string };
} = {
	//  EIP-1474 & JSON RPC 2.0
	// https://github.com/ethereum/EIPs/blob/master/EIPS/eip-1474.md
	[ERR_RPC_INVALID_JSON]: {
		message: 'Parse error',
		description: 'Invalid JSON',
	},
	[ERR_RPC_INVALID_REQUEST]: {
		message: 'Invalid request',
		description: 'JSON is not a valid request object	',
	},
	[ERR_RPC_INVALID_METHOD]: {
		message: 'Method not found',
		description: 'Method does not exist	',
	},
	[ERR_RPC_INVALID_PARAMS]: {
		message: 'Invalid params',
		description: 'Invalid method parameters',
	},
	[ERR_RPC_INTERNAL_ERROR]: {
		message: 'Internal error',
		description: 'Internal JSON-RPC error',
	},

	[ERR_RPC_INVALID_INPUT]: {
		message: 'Invalid input',
		description: 'Missing or invalid parameters',
	},
	[ERR_RPC_MISSING_RESOURCE]: {
		message: 'Resource not found',
		description: 'Requested resource not found',
	},
	[ERR_RPC_UNAVAILABLE_RESOURCE]: {
		message: 'Resource unavailable',
		description: 'Requested resource not available',
	},
	[ERR_RPC_TRANSACTION_REJECTED]: {
		message: 'Transaction rejected',
		description: 'Transaction creation failed',
	},
	[ERR_RPC_UNSUPPORTED_METHOD]: {
		message: 'Method not supported',
		description: 'Method is not implemented',
	},
	[ERR_RPC_LIMIT_EXCEEDED]: {
		message: 'Limit exceeded',
		description: 'Request exceeds defined limit',
	},
	[ERR_RPC_NOT_SUPPORTED]: {
		message: 'JSON-RPC version not supported',
		description: 'Version of JSON-RPC protocol is not supported',
	},

	// EIP-1193
	// https://github.com/ethereum/EIPs/blob/master/EIPS/eip-1193.md#provider-errors
	[JSONRPC_ERR_REJECTED_REQUEST]: {
		name: 'User Rejected Request',
		message: 'The user rejected the request.',
	},
	[JSONRPC_ERR_UNAUTHORIZED]: {
		name: 'Unauthorized',
		message: 'The requested method and/or account has not been authorized by the user.',
	},
	[JSONRPC_ERR_UNSUPPORTED_METHOD]: {
		name: 'Unsupported Method',
		message: 'The Provider does not support the requested method.',
	},
	[JSONRPC_ERR_DISCONNECTED]: {
		name: 'Disconnected',
		message: 'The Provider is disconnected from all chains.',
	},
	[JSONRPC_ERR_CHAIN_DISCONNECTED]: {
		name: 'Chain Disconnected',
		message: 'The Provider is not connected to the requested chain.',
	},

	// EIP-1193 - CloseEvent
	// https://developer.mozilla.org/en-US/docs/Web/API/CloseEvent/code
	'0-999': {
		name: '',
		message: 'Not used.',
	},
	1000: {
		name: 'Normal Closure',
		message: 'The connection successfully completed the purpose for which it was created.',
	},
	1001: {
		name: 'Going Away',
		message:
			'The endpoint is going away, either because of a server failure or because the browser is navigating away from the page that opened the connection.',
	},
	1002: {
		name: 'Protocol error',
		message: 'The endpoint is terminating the connection due to a protocol error.',
	},
	1003: {
		name: 'Unsupported Data',
		message:
			'The connection is being terminated because the endpoint received data of a type it cannot accept. (For example, a text-only endpoint received binary data.)',
	},
	1004: {
		name: 'Reserved',
		message: 'Reserved. A meaning might be defined in the future.',
	},
	1005: {
		name: 'No Status Rcvd',
		message:
			'Reserved. Indicates that no status code was provided even though one was expected.',
	},
	1006: {
		name: 'Abnormal Closure',
		message:
			'Reserved. Indicates that a connection was closed abnormally (that is, with no close frame being sent) when a status code is expected.',
	},
	1007: {
		name: 'Invalid frame payload data',
		message:
			'The endpoint is terminating the connection because a message was received that contained inconsistent data (e.g., non-UTF-8 data within a text message).',
	},
	1008: {
		name: 'Policy Violation',
		message:
			'The endpoint is terminating the connection because it received a message that violates its policy. This is a generic status code, used when codes 1003 and 1009 are not suitable.',
	},
	1009: {
		name: 'Message Too Big',
		message:
			'The endpoint is terminating the connection because a data frame was received that is too large.',
	},
	1010: {
		name: 'Mandatory Ext.',
		message:
			"The client is terminating the connection because it expected the server to negotiate one or more extension, but the server didn't.",
	},
	1011: {
		name: 'Internal Error',
		message:
			'The server is terminating the connection because it encountered an unexpected condition that prevented it from fulfilling the request.',
	},
	1012: {
		name: 'Service Restart',
		message: 'The server is terminating the connection because it is restarting.',
	},
	1013: {
		name: 'Try Again Later',
		message:
			'The server is terminating the connection due to a temporary condition, e.g. it is overloaded and is casting off some of its clients.',
	},
	1014: {
		name: 'Bad Gateway',
		message:
			'The server was acting as a gateway or proxy and received an invalid response from the upstream server. This is similar to 502 HTTP Status Code.',
	},
	1015: {
		name: 'TLS handshake',
		message:
			"Reserved. Indicates that the connection was closed due to a failure to perform a TLS handshake (e.g., the server certificate can't be verified).",
	},
	'1016-2999': {
		name: '',
		message:
			'For definition by future revisions of the WebSocket Protocol specification, and for definition by extension specifications.',
	},
	'3000-3999': {
		name: '',
		message:
			'For use by libraries, frameworks, and applications. These status codes are registered directly with IANA. The interpretation of these codes is undefined by the WebSocket protocol.',
	},
	'4000-4999': {
		name: '',
		message:
			"For private use, and thus can't be registered. Such codes can be used by prior agreements between WebSocket applications. The interpretation of these codes is undefined by the WebSocket protocol.",
	},
};
