import { keccak_256 } from '@noble/hashes/sha3';
import { bytesToHex, concatBytes, hexToBytes } from '@noble/hashes/utils';
import * as P from 'micro-packed';
import { add0x, ethHex, omit, strip0x, zip, } from "../utils.js";
/*
There is NO network code in the file. However, a user can pass
NetProvider instance to createContract, and the method would do
network requests with the api.

There is some really crazy stuff going on here with Typescript types.
*/
function EPad(p) {
    return P.padLeft(32, p, P.ZeroPad);
}
// Main difference between regular array: length stored outside and offsets calculated without length
function ethArray(inner) {
    return P.wrap({
        size: undefined,
        encodeStream: (w, value) => {
            U256BE_LEN.encodeStream(w, value.length);
            w.bytes(P.array(value.length, inner).encode(value));
        },
        decodeStream: (r) => P.array(U256BE_LEN.decodeStream(r), inner).decodeStream(r.offsetReader(r.pos)),
    });
}
const PTR = EPad(P.U32BE);
const ARRAY_RE = /(.+)(\[(\d+)?\])$/; // TODO: is this correct?
// Because u32 in eth is not real u32, just U256BE with limits...
const ethInt = (bits, signed = false) => {
    if (!Number.isSafeInteger(bits) || bits <= 0 || bits % 8 !== 0 || bits > 256)
        throw new Error('ethInt: invalid numeric type');
    const _bits = BigInt(bits);
    const inner = P.bigint(32, false, signed);
    return P.validate(P.wrap({
        size: inner.size,
        encodeStream: (w, value) => inner.encodeStream(w, value),
        decodeStream: (r) => inner.decodeStream(r),
    }), (value) => {
        // TODO: validate useful for narrowing types, need to add support in types?
        if (typeof value === 'number')
            value = BigInt(value);
        P.utils.checkBounds(value, _bits, !!signed);
        return value;
    });
};
// Ugly hack, because tuple of pointers considered "dynamic" without any reason.
function isDyn(args) {
    let res = false;
    if (Array.isArray(args)) {
        for (let arg of args)
            if (arg.size === undefined)
                res = true;
    }
    else {
        for (let arg in args)
            if (args[arg].size === undefined)
                res = true;
    }
    return res;
}
// Re-use ptr for len. u32 should be enough.
const U256BE_LEN = PTR;
// NOTE: we need as const if we want to access string as values inside types :(
export function mapComponent(c) {
    // Arrays (should be first one, since recursive)
    let m;
    if ((m = ARRAY_RE.exec(c.type))) {
        const inner = mapComponent({ ...c, type: m[1] });
        if (inner.size === 0)
            throw new Error('mapComponent: arrays of zero-size elements disabled (possible DoS attack)');
        // Static array
        if (m[3] !== undefined) {
            const m3 = Number.parseInt(m[3]);
            if (!Number.isSafeInteger(m3))
                throw new Error(`mapComponent: wrong array size=${m[3]}`);
            let out = P.array(m3, inner);
            // Static array of dynamic values should be behind pointer too, again without reason.
            if (inner.size === undefined)
                out = P.pointer(PTR, out);
            return out;
        }
        else {
            // Dynamic array
            return P.pointer(PTR, ethArray(inner));
        }
    }
    if (c.type === 'tuple') {
        const components = c.components;
        let hasNames = true;
        const args = [];
        for (let comp of components) {
            if (!comp.name)
                hasNames = false;
            args.push(mapComponent(comp));
        }
        let out;
        // If there is names for all fields -- return struct, otherwise tuple
        if (hasNames) {
            const struct = {};
            for (const arg of components) {
                if (struct[arg.name])
                    throw new Error(`mapType: same field name=${arg.name}`);
                struct[arg.name] = mapComponent(arg);
            }
            out = P.struct(struct);
        }
        else
            out = P.tuple(args);
        // If tuple has dynamic elements it becomes dynamic too, without reason.
        if (isDyn(args))
            out = P.pointer(PTR, out);
        return out;
    }
    if (c.type === 'string')
        return P.pointer(PTR, P.padRight(32, P.string(U256BE_LEN), P.ZeroPad));
    if (c.type === 'bytes')
        return P.pointer(PTR, P.padRight(32, P.bytes(U256BE_LEN), P.ZeroPad));
    if (c.type === 'address')
        return EPad(P.hex(20, { isLE: false, with0x: true }));
    if (c.type === 'bool')
        return EPad(P.bool);
    if ((m = /^(u?)int([0-9]+)?$/.exec(c.type)))
        return ethInt(m[2] ? +m[2] : 256, m[1] !== 'u');
    if ((m = /^bytes([0-9]{1,2})$/.exec(c.type))) {
        const parsed = +m[1];
        if (!parsed || parsed > 32)
            throw new Error('wrong bytes<N> type');
        return P.padRight(32, P.bytes(parsed), P.ZeroPad);
    }
    throw new Error(`mapComponent: unknown component=${c}`);
}
// Because args and output are not tuple
// TODO: try merge with mapComponent
export function mapArgs(args) {
    // More ergonomic input/output
    if (args.length === 1)
        return mapComponent(args[0]);
    let hasNames = true;
    for (const arg of args)
        if (!arg.name)
            hasNames = false;
    if (hasNames) {
        const out = {};
        for (const arg of args) {
            const name = arg.name;
            if (out[name])
                throw new Error(`mapArgs: same field name=${name}`);
            out[name] = mapComponent(arg);
        }
        return P.struct(out);
    }
    else
        return P.tuple(args.map(mapComponent));
}
function fnSignature(o) {
    if (!o.type)
        throw new Error('ABI.fnSignature wrong argument');
    if (o.type === 'function' || o.type === 'event')
        return `${o.name || 'function'}(${(o.inputs || []).map((i) => fnSignature(i)).join(',')})`;
    if (o.type.startsWith('tuple')) {
        if (!o.components || !o.components.length)
            throw new Error('ABI.fnSignature wrong tuple');
        return `(${o.components.map((i) => fnSignature(i)).join(',')})${o.type.slice(5)}`;
    }
    return o.type;
}
// Function signature hash
export function evSigHash(o) {
    return bytesToHex(keccak_256(fnSignature(o)));
}
export function fnSigHash(o) {
    return evSigHash(o).slice(0, 8);
}
export function createContract(abi, net, contract) {
    // Find non-uniq function names so we can handle overloads
    let nameCnt = {};
    for (let fn of abi) {
        if (fn.type !== 'function')
            continue;
        const name = fn.name || 'function';
        if (!nameCnt[name])
            nameCnt[name] = 1;
        else
            nameCnt[name]++;
    }
    const res = {};
    for (let fn of abi) {
        if (fn.type !== 'function')
            continue;
        let name = fn.name || 'function';
        if (nameCnt[name] > 1)
            name = fnSignature(fn);
        const sh = fnSigHash(fn);
        const inputs = fn.inputs && fn.inputs.length ? mapArgs(fn.inputs) : undefined;
        const outputs = fn.outputs ? mapArgs(fn.outputs) : undefined;
        const decodeOutput = (b) => outputs && outputs.decode(b);
        const encodeInput = (v) => concatBytes(hexToBytes(sh), inputs ? inputs.encode(v) : new Uint8Array());
        res[name] = { decodeOutput, encodeInput };
        // .call and .estimateGas call network, when net is available
        if (!net)
            continue;
        res[name].call = async (args, overrides = {}) => {
            if (!contract && !overrides.to)
                throw new Error('No contract address');
            const data = add0x(bytesToHex(encodeInput(args)));
            const callArgs = Object.assign({ to: contract, data }, overrides);
            return decodeOutput(hexToBytes(strip0x(await net.ethCall(callArgs))));
        };
        res[name].estimateGas = async (args, overrides = {}) => {
            if (!contract && !overrides.to)
                throw new Error('No contract address');
            const data = add0x(bytesToHex(encodeInput(args)));
            const callArgs = Object.assign({ to: contract, data }, overrides);
            return await net.estimateGas(callArgs);
        };
    }
    return res;
}
export function deployContract(abi, bytecodeHex, ...args) {
    const bytecode = ethHex.decode(bytecodeHex);
    let consCall;
    for (let fn of abi) {
        if (fn.type !== 'constructor')
            continue;
        const inputs = fn.inputs && fn.inputs.length ? mapArgs(fn.inputs) : undefined;
        if (inputs === undefined && args !== undefined && args.length)
            throw new Error('arguments to constructor without any');
        consCall = inputs ? inputs.encode(args[0]) : new Uint8Array();
    }
    if (!consCall)
        throw new Error('constructor not found');
    return ethHex.encode(concatBytes(bytecode, consCall));
}
// TODO: try to simplify further
export function events(abi) {
    let res = {};
    for (let elm of abi) {
        // Only named events supported
        if (elm.type !== 'event' || !elm.name)
            continue;
        const inputs = elm.inputs || [];
        let hasNames = true;
        for (let i of inputs)
            if (!i.name)
                hasNames = false;
        const plainInp = inputs.filter((i) => !i.indexed);
        const indexedInp = inputs.filter((i) => i.indexed);
        const indexed = indexedInp.map((i) => !['string', 'bytes', 'tuple'].includes(i.type) && !ARRAY_RE.exec(i.type)
            ? mapArgs([i])
            : null);
        const parser = mapArgs(hasNames ? plainInp : plainInp.map((i) => omit(i, 'name')));
        const sigHash = evSigHash(elm);
        res[elm.name] = {
            decode(topics, _data) {
                const data = hexToBytes(strip0x(_data));
                if (!elm.anonymous) {
                    if (!topics[0])
                        throw new Error('No signature on non-anonymous event');
                    if (strip0x(topics[0]).toLowerCase() !== sigHash)
                        throw new Error('Wrong signature');
                    topics = topics.slice(1);
                }
                if (topics.length !== indexed.length)
                    throw new Error('Wrong topics length');
                let parsed = parser ? parser.decode(data) : hasNames ? {} : [];
                const indexedParsed = indexed.map((p, i) => p ? p.decode(hexToBytes(strip0x(topics[i]))) : topics[i]);
                if (plainInp.length === 1)
                    parsed = hasNames ? { [plainInp[0].name]: parsed } : [parsed];
                if (hasNames) {
                    let res = { ...parsed };
                    for (let [a, p] of zip(indexedInp, indexedParsed))
                        res[a.name] = p;
                    return res;
                }
                else
                    return inputs.map((i) => (!i.indexed ? parsed : indexedParsed).shift());
            },
            topics(values) {
                let res = [];
                if (!elm.anonymous)
                    res.push(add0x(sigHash));
                // We require all keys to be set, even if they are null, to be sure nothing is accidentaly missed
                if ((hasNames ? Object.keys(values) : values).length !== inputs.length)
                    throw new Error('Wrong topics args');
                for (let i = 0, ii = 0; i < inputs.length && ii < indexed.length; i++) {
                    const [input, packer] = [inputs[i], indexed[ii]];
                    if (!input.indexed)
                        continue;
                    const value = values[Array.isArray(values) ? i : inputs[i].name];
                    if (value === null) {
                        res.push(null);
                        continue;
                    }
                    let topic;
                    if (packer)
                        topic = bytesToHex(packer.encode(value));
                    else if (['string', 'bytes'].includes(input.type))
                        topic = bytesToHex(keccak_256(value));
                    else {
                        let m, parts;
                        if ((m = ARRAY_RE.exec(input.type)))
                            parts = value.map((j) => mapComponent({ type: m[1] }).encode(j));
                        else if (input.type === 'tuple' && input.components)
                            parts = input.components.map((j) => mapArgs([j]).encode(value[j.name]));
                        else
                            throw new Error('Unknown unsized type');
                        topic = bytesToHex(keccak_256(concatBytes(...parts)));
                    }
                    res.push(add0x(topic));
                    ii++;
                }
                return res;
            },
        };
    }
    return res;
}
export class Decoder {
    constructor() {
        this.contracts = {};
        this.sighashes = {};
        this.evContracts = {};
        this.evSighashes = {};
    }
    add(contract, abi) {
        const ev = events(abi);
        contract = strip0x(contract).toLowerCase();
        if (!this.contracts[contract])
            this.contracts[contract] = {};
        if (!this.evContracts[contract])
            this.evContracts[contract] = {};
        for (let fn of abi) {
            if (fn.type === 'function') {
                const selector = fnSigHash(fn);
                const value = {
                    name: fn.name || 'function',
                    signature: fnSignature(fn),
                    packer: fn.inputs && fn.inputs.length ? mapArgs(fn.inputs) : undefined,
                    hint: fn.hint,
                    hook: fn.hook,
                };
                this.contracts[contract][selector] = value;
                if (!this.sighashes[selector])
                    this.sighashes[selector] = [];
                this.sighashes[selector].push(value);
            }
            else if (fn.type === 'event') {
                if (fn.anonymous || !fn.name)
                    continue;
                const selector = evSigHash(fn);
                const value = {
                    name: fn.name,
                    signature: fnSignature(fn),
                    decoder: ev[fn.name]?.decode,
                    hint: fn.hint,
                };
                this.evContracts[contract][selector] = value;
                if (!this.evSighashes[selector])
                    this.evSighashes[selector] = [];
                this.evSighashes[selector].push(value);
            }
        }
    }
    method(contract, data) {
        contract = strip0x(contract).toLowerCase();
        const sh = bytesToHex(data.slice(0, 4));
        if (!this.contracts[contract] || !this.contracts[contract][sh])
            return;
        const { name } = this.contracts[contract][sh];
        return name;
    }
    // Returns: exact match, possible options of matches (array) or undefined.
    // Note that empty value possible if there is no arguments in call.
    decode(contract, _data, opt) {
        contract = strip0x(contract).toLowerCase();
        const sh = bytesToHex(_data.slice(0, 4));
        const data = _data.slice(4);
        if (this.contracts[contract] && this.contracts[contract][sh]) {
            let { name, signature, packer, hint, hook } = this.contracts[contract][sh];
            const value = packer ? packer.decode(data) : undefined;
            let res = { name, signature, value };
            // NOTE: hint && hook fn is used only on exact match of contract!
            if (hook)
                res = hook(this, contract, res, opt);
            try {
                if (hint)
                    res.hint = hint(value, Object.assign({ contract: add0x(contract) }, opt));
            }
            catch (e) { }
            return res;
        }
        if (!this.sighashes[sh] || !this.sighashes[sh].length)
            return;
        let res = [];
        for (let { name, signature, packer } of this.sighashes[sh]) {
            try {
                res.push({ name, signature, value: packer ? packer.decode(data) : undefined });
            }
            catch (err) { }
        }
        if (res.length)
            return res;
        return;
    }
    decodeEvent(contract, topics, data, opt) {
        contract = strip0x(contract).toLowerCase();
        if (!topics.length)
            return;
        const sh = strip0x(topics[0]);
        const event = this.evContracts[contract];
        if (event && event[sh]) {
            let { name, signature, decoder, hint } = event[sh];
            const value = decoder(topics, data);
            let res = { name, signature, value };
            try {
                if (hint)
                    res.hint = hint(value, Object.assign({ contract: add0x(contract) }, opt));
            }
            catch (e) { }
            return res;
        }
        if (!this.evSighashes[sh] || !this.evSighashes[sh].length)
            return;
        let res = [];
        for (let { name, signature, decoder } of this.evSighashes[sh]) {
            try {
                res.push({ name, signature, value: decoder(topics, data) });
            }
            catch (err) { }
        }
        if (res.length)
            return res;
        return;
    }
}
//# sourceMappingURL=decoder.js.map