"use strict";
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
Object.defineProperty(exports, "__esModule", { value: true });
exports.read_emoji_trie = exports.read_zero_terminated_array = exports.read_mapped_map = exports.read_member_array = exports.signed = exports.read_compressed_payload = exports.read_payload = exports.decode_arithmetic = void 0;
// https://github.com/behnammodi/polyfill/blob/master/array.polyfill.js
function flat(array, depth) {
    if (depth == null) {
        depth = 1;
    }
    var result = [];
    var forEach = result.forEach;
    var flatDeep = function (arr, depth) {
        forEach.call(arr, function (val) {
            if (depth > 0 && Array.isArray(val)) {
                flatDeep(val, depth - 1);
            }
            else {
                result.push(val);
            }
        });
    };
    flatDeep(array, depth);
    return result;
}
function fromEntries(array) {
    var result = {};
    for (var i = 0; i < array.length; i++) {
        var value = array[i];
        result[value[0]] = value[1];
    }
    return result;
}
function decode_arithmetic(bytes) {
    var pos = 0;
    function u16() { return (bytes[pos++] << 8) | bytes[pos++]; }
    // decode the frequency table
    var symbol_count = u16();
    var total = 1;
    var acc = [0, 1]; // first symbol has frequency 1
    for (var i = 1; i < symbol_count; i++) {
        acc.push(total += u16());
    }
    // skip the sized-payload that the last 3 symbols index into
    var skip = u16();
    var pos_payload = pos;
    pos += skip;
    var read_width = 0;
    var read_buffer = 0;
    function read_bit() {
        if (read_width == 0) {
            // this will read beyond end of buffer
            // but (undefined|0) => zero pad
            read_buffer = (read_buffer << 8) | bytes[pos++];
            read_width = 8;
        }
        return (read_buffer >> --read_width) & 1;
    }
    var N = 31;
    var FULL = Math.pow(2, N);
    var HALF = FULL >>> 1;
    var QRTR = HALF >> 1;
    var MASK = FULL - 1;
    // fill register
    var register = 0;
    for (var i = 0; i < N; i++)
        register = (register << 1) | read_bit();
    var symbols = [];
    var low = 0;
    var range = FULL; // treat like a float
    while (true) {
        var value = Math.floor((((register - low + 1) * total) - 1) / range);
        var start = 0;
        var end = symbol_count;
        while (end - start > 1) { // binary search
            var mid = (start + end) >>> 1;
            if (value < acc[mid]) {
                end = mid;
            }
            else {
                start = mid;
            }
        }
        if (start == 0)
            break; // first symbol is end mark
        symbols.push(start);
        var a = low + Math.floor(range * acc[start] / total);
        var b = low + Math.floor(range * acc[start + 1] / total) - 1;
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
    var offset = symbol_count - 4;
    return symbols.map(function (x) {
        switch (x - offset) {
            case 3: return offset + 0x10100 + ((bytes[pos_payload++] << 16) | (bytes[pos_payload++] << 8) | bytes[pos_payload++]);
            case 2: return offset + 0x100 + ((bytes[pos_payload++] << 8) | bytes[pos_payload++]);
            case 1: return offset + bytes[pos_payload++];
            default: return x - 1;
        }
    });
}
exports.decode_arithmetic = decode_arithmetic;
// returns an iterator which returns the next symbol
function read_payload(v) {
    var pos = 0;
    return function () { return v[pos++]; };
}
exports.read_payload = read_payload;
function read_compressed_payload(bytes) {
    return read_payload(decode_arithmetic(bytes));
}
exports.read_compressed_payload = read_compressed_payload;
// eg. [0,1,2,3...] => [0,-1,1,-2,...]
function signed(i) {
    return (i & 1) ? (~i >> 1) : (i >> 1);
}
exports.signed = signed;
function read_counts(n, next) {
    var v = Array(n);
    for (var i = 0; i < n; i++)
        v[i] = 1 + next();
    return v;
}
function read_ascending(n, next) {
    var v = Array(n);
    for (var i = 0, x = -1; i < n; i++)
        v[i] = x += 1 + next();
    return v;
}
function read_deltas(n, next) {
    var v = Array(n);
    for (var i = 0, x = 0; i < n; i++)
        v[i] = x += signed(next());
    return v;
}
function read_member_array(next, lookup) {
    var v = read_ascending(next(), next);
    var n = next();
    var vX = read_ascending(n, next);
    var vN = read_counts(n, next);
    for (var i = 0; i < n; i++) {
        for (var j = 0; j < vN[i]; j++) {
            v.push(vX[i] + j);
        }
    }
    return lookup ? v.map(function (x) { return lookup[x]; }) : v;
}
exports.read_member_array = read_member_array;
// returns array of 
// [x, ys] => single replacement rule
// [x, ys, n, dx, dx] => linear map
function read_mapped_map(next) {
    var ret = [];
    while (true) {
        var w = next();
        if (w == 0)
            break;
        ret.push(read_linear_table(w, next));
    }
    while (true) {
        var w = next() - 1;
        if (w < 0)
            break;
        ret.push(read_replacement_table(w, next));
    }
    return fromEntries(flat(ret));
}
exports.read_mapped_map = read_mapped_map;
function read_zero_terminated_array(next) {
    var v = [];
    while (true) {
        var i = next();
        if (i == 0)
            break;
        v.push(i);
    }
    return v;
}
exports.read_zero_terminated_array = read_zero_terminated_array;
function read_transposed(n, w, next) {
    var m = Array(n).fill(undefined).map(function () { return []; });
    for (var i = 0; i < w; i++) {
        read_deltas(n, next).forEach(function (x, j) { return m[j].push(x); });
    }
    return m;
}
function read_linear_table(w, next) {
    var dx = 1 + next();
    var dy = next();
    var vN = read_zero_terminated_array(next);
    var m = read_transposed(vN.length, 1 + w, next);
    return flat(m.map(function (v, i) {
        var x = v[0], ys = v.slice(1);
        //let [x, ...ys] = v;
        //return Array(vN[i]).fill().map((_, j) => {
        return Array(vN[i]).fill(undefined).map(function (_, j) {
            var j_dy = j * dy;
            return [x + j * dx, ys.map(function (y) { return y + j_dy; })];
        });
    }));
}
function read_replacement_table(w, next) {
    var n = 1 + next();
    var m = read_transposed(n, 1 + w, next);
    return m.map(function (v) { return [v[0], v.slice(1)]; });
}
function read_emoji_trie(next) {
    var sorted = read_member_array(next).sort(function (a, b) { return a - b; });
    return read();
    function read() {
        var branches = [];
        while (true) {
            var keys = read_member_array(next, sorted);
            if (keys.length == 0)
                break;
            branches.push({ set: new Set(keys), node: read() });
        }
        branches.sort(function (a, b) { return b.set.size - a.set.size; }); // sort by likelihood
        var temp = next();
        var valid = temp % 3;
        temp = (temp / 3) | 0;
        var fe0f = !!(temp & 1);
        temp >>= 1;
        var save = temp == 1;
        var check = temp == 2;
        return { branches: branches, valid: valid, fe0f: fe0f, save: save, check: check };
    }
}
exports.read_emoji_trie = read_emoji_trie;
//# sourceMappingURL=decoder.js.map