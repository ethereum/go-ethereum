"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.RLP = void 0;
const utils_1 = require("@noble/curves/abstract/utils");
const P = require("micro-packed");
const utils_ts_1 = require("./utils.js");
// length: first 3 bit !== 111 ? 6 bit length : 3bit lenlen
const RLPLength = P.wrap({
    encodeStream(w, value) {
        if (value < 56)
            return w.bits(value, 6);
        w.bits(0b111, 3);
        const length = P.U32BE.encode(value);
        let pos = 0;
        for (; pos < length.length; pos++)
            if (length[pos] !== 0)
                break;
        w.bits(4 - pos - 1, 3);
        w.bytes(length.slice(pos));
    },
    decodeStream(r) {
        const start = r.bits(3);
        if (start !== 0b111)
            return (start << 3) | r.bits(3);
        const len = r.bytes(r.bits(3) + 1);
        for (let i = 0; i < len.length; i++) {
            if (len[i])
                break;
            throw new Error('Wrong length encoding with leading zeros');
        }
        const res = P.int(len.length).decode(len);
        if (res <= 55)
            throw new Error('RLPLength: less than 55, but used multi-byte flag');
        return res;
    },
});
const rlpInner = P.tag(P.map(P.bits(1), { byte: 0, complex: 1 }), {
    byte: P.bits(7),
    complex: P.tag(P.map(P.bits(1), { string: 0, list: 1 }), {
        string: P.bytes(RLPLength),
        list: P.prefix(RLPLength, P.array(null, P.lazy(() => rlpInner))),
    }),
});
const phex = P.hex(null);
const pstr = P.string(null);
const empty = Uint8Array.from([]);
/**
 * RLP parser.
 * Real type of rlp is `Item = Uint8Array | Item[]`.
 * Strings/number encoded to Uint8Array, but not decoded back: type information is lost.
 */
exports.RLP = P.apply(rlpInner, {
    encode(from) {
        if (from.TAG === 'byte')
            return new Uint8Array([from.data]);
        if (from.TAG !== 'complex')
            throw new Error('RLP.encode: unexpected type');
        const complex = from.data;
        if (complex.TAG === 'string') {
            if (complex.data.length === 1 && complex.data[0] < 128)
                throw new Error('RLP.encode: wrong string length encoding, should use single byte mode');
            return complex.data;
        }
        if (complex.TAG === 'list')
            return complex.data.map((i) => this.encode(i));
        throw new Error('RLP.encode: unknown TAG');
    },
    decode(data) {
        if (data == null)
            return this.decode(empty);
        switch (typeof data) {
            case 'object':
                if ((0, utils_ts_1.isBytes)(data)) {
                    if (data.length === 1) {
                        const head = data[0];
                        if (head < 128)
                            return { TAG: 'byte', data: head };
                    }
                    return { TAG: 'complex', data: { TAG: 'string', data: data } };
                }
                if (Array.isArray(data))
                    return { TAG: 'complex', data: { TAG: 'list', data: data.map((i) => this.decode(i)) } };
                throw new Error('RLP.encode: unknown type');
            case 'number':
                if (data < 0)
                    throw new Error('RLP.encode: invalid integer as argument, must be unsigned');
                if (data === 0)
                    return this.decode(empty);
                return this.decode((0, utils_1.numberToVarBytesBE)(data));
            case 'bigint':
                if (data < BigInt(0))
                    throw new Error('RLP.encode: invalid integer as argument, must be unsigned');
                return this.decode((0, utils_1.numberToVarBytesBE)(data));
            case 'string':
                return this.decode(data.startsWith('0x') ? phex.encode(data) : pstr.encode(data));
            default:
                throw new Error('RLP.encode: unknown type');
        }
    },
});
//# sourceMappingURL=rlp.js.map