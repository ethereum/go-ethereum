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

export function isUint8Array(data: unknown | Uint8Array): data is Uint8Array {
	return (
		data instanceof Uint8Array ||
		(data as { constructor: { name: string } })?.constructor?.name === 'Uint8Array' ||
		(data as { constructor: { name: string } })?.constructor?.name === 'Buffer'
	);
}

export function uint8ArrayConcat(...parts: Uint8Array[]): Uint8Array {
	const length = parts.reduce((prev, part) => {
		const agg = prev + part.length;
		return agg;
	}, 0);
	const result = new Uint8Array(length);
	let offset = 0;
	for (const part of parts) {
		result.set(part, offset);
		offset += part.length;
	}
	return result;
}

/**
 * Returns true if the two passed Uint8Arrays have the same content
 */
export function uint8ArrayEquals(a: Uint8Array, b: Uint8Array): boolean {
	if (a === b) {
		return true;
	}

	if (a.byteLength !== b.byteLength) {
		return false;
	}

	for (let i = 0; i < a.byteLength; i += 1) {
		if (a[i] !== b[i]) {
			return false;
		}
	}

	return true;
}
