/**
 * MIT License
 *
 * Copyright (c) 2021 Andrew Raffensperger
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 *
 * This is a near carbon-copy of the original source (link below) with the
 * TypeScript typings added and a few tweaks to make it ES3-compatible.
 *
 * See: https://github.com/adraffy/ens-normalize.js
 */

export type Numbers = Uint8Array | Array<number>;
export type NextFunc = (...args: Array<any>) => number;

// https://github.com/behnammodi/polyfill/blob/master/array.polyfill.js
function flat(array: Array<any>, depth?: number): Array<any> {
    if (depth == null) { depth = 1; }
    const result: Array<any> = [];

    const forEach = result.forEach;

    const flatDeep = function (arr: Array<any>, depth: number) {
        forEach.call(arr, function (val: any) {
            if (depth > 0 && Array.isArray(val)) {
                flatDeep(val, depth - 1);
            } else {
               result.push(val);
            }
        });
    };

    flatDeep(array, depth);
    return result;
}

function fromEntries<T extends string | number | symbol = string | number | symbol, U = any>(array: Array<[T, U]>): Record<T, U> {
    const result: Record<T, U> = <Record<T, U>>{ };
    for (let i = 0; i < array.length; i++) {
        const value = array[i];
        result[value[0]] = value[1];
    }
    return result;
}

export function decode_arithmetic(bytes: Numbers): Array<number> {
	let pos = 0;
	function u16() { return (bytes[pos++] << 8) | bytes[pos++]; }
	
	// decode the frequency table
	let symbol_count = u16();
	let total = 1;
	let acc = [0, 1]; // first symbol has frequency 1
	for (let i = 1; i < symbol_count; i++) {
		acc.push(total += u16());
	}

	// skip the sized-payload that the last 3 symbols index into
	let skip = u16();
	let pos_payload = pos;
	pos += skip;

	let read_width = 0;
	let read_buffer = 0; 
	function read_bit() {
		if (read_width == 0) {
			// this will read beyond end of buffer
			// but (undefined|0) => zero pad
			read_buffer = (read_buffer << 8) | bytes[pos++];
			read_width = 8;
		}
		return (read_buffer >> --read_width) & 1;
	}

	const N = 31;
	const FULL = 2**N;
	const HALF = FULL >>> 1;
	const QRTR = HALF >> 1;
	const MASK = FULL - 1;

	// fill register
	let register = 0;
	for (let i = 0; i < N; i++) register = (register << 1) | read_bit();

	let symbols = [];
	let low = 0;
	let range = FULL; // treat like a float
	while (true) {
		let value = Math.floor((((register - low + 1) * total) - 1) / range);
		let start = 0;
		let end = symbol_count;
		while (end - start > 1) { // binary search
			let mid = (start + end) >>> 1;
			if (value < acc[mid]) {
				end = mid;
			} else {
				start = mid;
			}
		}
		if (start == 0) break; // first symbol is end mark
		symbols.push(start);
		let a = low + Math.floor(range * acc[start]   / total);
		let b = low + Math.floor(range * acc[start+1] / total) - 1
		while (((a ^ b) & HALF) == 0) {
			register = (register << 1) & MASK | read_bit();
			a = (a << 1) & MASK;
			b = (b << 1) & MASK | 1;
		}
		while (a & ~b & QRTR) {
			register = (register & HALF) | ((register << 1) & (MASK >>> 1)) | read_bit();
			a = (a << 1) ^ HALF;
			b = ((b ^ HALF) << 1) | HALF | 1;
		}
		low = a;
		range = 1 + b - a;
	}
	let offset = symbol_count - 4;
	return symbols.map(x => { // index into payload
		switch (x - offset) {
			case 3: return offset + 0x10100 + ((bytes[pos_payload++] << 16) | (bytes[pos_payload++] << 8) | bytes[pos_payload++]);
			case 2: return offset + 0x100 + ((bytes[pos_payload++] << 8) | bytes[pos_payload++]);
			case 1: return offset + bytes[pos_payload++];
			default: return x - 1;
		}
	});
}	


// returns an iterator which returns the next symbol
export function read_payload(v: Numbers): NextFunc {
	let pos = 0;
	return () => v[pos++];
}
export function read_compressed_payload(bytes: Numbers): NextFunc {
	return read_payload(decode_arithmetic(bytes));
}

// eg. [0,1,2,3...] => [0,-1,1,-2,...]
export function signed(i: number): number { 
	return (i & 1) ? (~i >> 1) : (i >> 1);
}

function read_counts(n: number, next: NextFunc): Array<number> {
	let v = Array(n);
	for (let i = 0; i < n; i++) v[i] = 1 + next();
	return v;
}

function read_ascending(n: number, next: NextFunc): Array<number> {
	let v = Array(n);
	for (let i = 0, x = -1; i < n; i++) v[i] = x += 1 + next();
	return v;
}

function read_deltas(n: number, next: NextFunc): Array<number> {
	let v = Array(n);
	for (let i = 0, x = 0; i < n; i++) v[i] = x += signed(next());
	return v;
}

export function read_member_array(next: NextFunc, lookup?: Record<number, number>) {
    let v = read_ascending(next(), next);
    let n = next();
    let vX = read_ascending(n, next);
    let vN = read_counts(n, next);
    for (let i = 0; i < n; i++) {
        for (let j = 0; j < vN[i]; j++) {
            v.push(vX[i] + j);
        }
    }
    return lookup ? v.map(x => lookup[x]) : v;
}

// returns array of 
// [x, ys] => single replacement rule
// [x, ys, n, dx, dx] => linear map
export function read_mapped_map(next: NextFunc): Record<number, Array<number>> {
	let ret = [];
	while (true) {
		let w = next();
		if (w == 0) break;
		ret.push(read_linear_table(w, next));
	}
	while (true) {
		let w = next() - 1;
		if (w < 0) break;
		ret.push(read_replacement_table(w, next));
	}
	return fromEntries<number, Array<number>>(flat(ret));
}

export function read_zero_terminated_array(next: NextFunc): Array<number> {
	let v = [];
	while (true) {
		let i = next();
		if (i == 0) break;
		v.push(i);
	}
	return v;
}

function read_transposed(n: number, w: number, next: NextFunc): Array<Array<number>> {
    let m = Array(n).fill(undefined).map(() => []);
    for (let i = 0; i < w; i++) {
        read_deltas(n, next).forEach((x, j) => m[j].push(x));
    }
    return m;
}


function read_linear_table(w: number, next: NextFunc): Array<Array<number | Array<number>>> {
	let dx = 1 + next();
	let dy = next();
	let vN = read_zero_terminated_array(next);
	let m = read_transposed(vN.length, 1+w, next);
	return flat(m.map((v, i) => {
	  const x = v[0], ys = v.slice(1);
		//let [x, ...ys] = v;
		//return Array(vN[i]).fill().map((_, j) => {
		return Array(vN[i]).fill(undefined).map((_, j) => {
			let j_dy = j * dy;
			return [x + j * dx, ys.map(y => y + j_dy)];
		});
	}));
}

function read_replacement_table(w: number, next: NextFunc): Array<[ number, Array<number> ]> {
	let n = 1 + next();
	let m = read_transposed(n, 1+w, next);
	return m.map(v => [v[0], v.slice(1)]);
}

export type Branch = {
    set: Set<number>;
    node: Node;
};

export type Node = {
    branches: Array<Branch>;
    valid: number;
    fe0f: boolean;
    save: boolean;
    check: boolean;
};

export function read_emoji_trie(next: NextFunc): Node {
	let sorted = read_member_array(next).sort((a, b) => a - b);
	return read();
	function read(): Node {
		let branches = [];
		while (true) {
			let keys = read_member_array(next, sorted);
			if (keys.length == 0) break;
			branches.push({set: new Set(keys), node: read()});
		}
    branches.sort((a, b) => b.set.size - a.set.size); // sort by likelihood
 		let temp = next();
 		let valid = temp % 3;
 		temp = (temp / 3)|0;
 		let fe0f = !!(temp & 1);
 		temp >>= 1;
 		let save = temp == 1;
 		let check = temp == 2;
 		return {branches, valid, fe0f, save, check};
	}
}
