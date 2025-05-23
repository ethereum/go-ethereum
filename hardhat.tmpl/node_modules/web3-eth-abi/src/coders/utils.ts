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

import { AbiParameter as ExternalAbiParameter, parseAbiParameter } from 'abitype';
import { AbiError } from 'web3-errors';
import { AbiInput, AbiParameter, AbiStruct } from 'web3-types';
import { isNullish } from 'web3-utils';
import {
	isSimplifiedStructFormat,
	mapStructNameAndType,
	mapStructToCoderFormat,
} from '../utils.js';

export const WORD_SIZE = 32;

export function alloc(size = 0): Uint8Array {
	if (globalThis.Buffer?.alloc !== undefined) {
		const buf = globalThis.Buffer.alloc(size);
		return new Uint8Array(buf.buffer, buf.byteOffset, buf.byteLength);
	}

	return new Uint8Array(size);
}

/**
 * Where possible returns a Uint8Array of the requested size that references
 * uninitialized memory. Only use if you are certain you will immediately
 * overwrite every value in the returned `Uint8Array`.
 */
export function allocUnsafe(size = 0): Uint8Array {
	if (globalThis.Buffer?.allocUnsafe !== undefined) {
		const buf = globalThis.Buffer.allocUnsafe(size);
		return new Uint8Array(buf.buffer, buf.byteOffset, buf.byteLength);
	}

	return new Uint8Array(size);
}

export function convertExternalAbiParameter(abiParam: ExternalAbiParameter): AbiParameter {
	return {
		...abiParam,
		name: abiParam.name ?? '',
		components: (abiParam as { components: readonly AbiParameter[] }).components?.map(c =>
			convertExternalAbiParameter(c),
		),
	};
}

export function isAbiParameter(param: unknown): param is AbiParameter {
	return (
		!isNullish(param) &&
		typeof param === 'object' &&
		!isNullish((param as { type: unknown }).type) &&
		typeof (param as { type: unknown }).type === 'string'
	);
}

export function toAbiParams(abi: ReadonlyArray<AbiInput>): ReadonlyArray<AbiParameter> {
	return abi.map(input => {
		if (isAbiParameter(input)) {
			return input;
		}
		if (typeof input === 'string') {
			return convertExternalAbiParameter(parseAbiParameter(input.replace(/tuple/, '')));
		}

		if (isSimplifiedStructFormat(input)) {
			const structName = Object.keys(input)[0];
			const structInfo = mapStructNameAndType(structName);
			structInfo.name = structInfo.name ?? '';
			return {
				...structInfo,
				components: mapStructToCoderFormat(
					input[structName as keyof typeof input] as unknown as AbiStruct,
				),
			};
		}
		throw new AbiError('Invalid abi');
	});
}

export function extractArrayType(param: AbiParameter): { size: number; param: AbiParameter } {
	const arrayParenthesisStart = param.type.lastIndexOf('[');
	const arrayParamType = param.type.substring(0, arrayParenthesisStart);
	const sizeString = param.type.substring(arrayParenthesisStart);
	let size = -1;
	if (sizeString !== '[]') {
		size = Number(sizeString.slice(1, -1));
		// eslint-disable-next-line no-restricted-globals
		if (isNaN(size)) {
			throw new AbiError('Invalid fixed array size', { size: sizeString });
		}
	}
	return {
		param: { type: arrayParamType, name: '', components: param.components },
		size,
	};
}

/**
 * Param is dynamic if it's dynamic base type or if some of his children (components, array items)
 * is of dynamic type
 * @param param
 */
export function isDynamic(param: AbiParameter): boolean {
	if (param.type === 'string' || param.type === 'bytes' || param.type.endsWith('[]')) return true;
	if (param.type === 'tuple') {
		return param.components?.some(isDynamic) ?? false;
	}
	if (param.type.endsWith(']')) {
		return isDynamic(extractArrayType(param).param);
	}
	return false;
}
