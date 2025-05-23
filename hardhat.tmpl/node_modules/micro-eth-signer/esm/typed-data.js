import { keccak_256 } from '@noble/hashes/sha3';
import { concatBytes, hexToBytes, utf8ToBytes } from '@noble/hashes/utils';
import { mapComponent } from "./abi/decoder.js";
import { addr } from "./address.js";
import { add0x, astr, ethHex, initSig, isObject, sign, strip0x, verify } from "./utils.js";
// 0x19 <1 byte version> <version specific data> <data to sign>.
// VERSIONS:
// - 0x19 <0x00> <intended validator address> <data to sign>
// - 0x19 <0x01> domainSeparator hashStruct(message)
// - 0x19 <0x45 (E)> <thereum Signed Message:\n" + len(message)> <data to sign>
function getSigner(version, msgFn) {
    if (version < 0 || version >= 256 || !Number.isSafeInteger(version))
        throw new Error('Wrong version byte');
    //     bytes32 hash = keccak256(abi.encodePacked(byte(0x19), byte(0), address(this), msg.value, nonce, payload));
    const getHash = (message) => keccak_256(concatBytes(new Uint8Array([0x19, version]), msgFn(message)));
    // TODO: 'v' can contain non-undefined chainId, but not sure if it is used. If used, we need to check it with EIP-712 domain
    return {
        _getHash: (message) => ethHex.encode(getHash(message)),
        sign(message, privateKey, extraEntropy = true) {
            const hash = getHash(message);
            if (typeof privateKey === 'string')
                privateKey = ethHex.decode(privateKey);
            const sig = sign(hash, privateKey, extraEntropy);
            const end = sig.recovery === 0 ? '1b' : '1c';
            return add0x(sig.toCompactHex() + end);
        },
        recoverPublicKey(signature, message) {
            astr(signature);
            const hash = getHash(message);
            signature = strip0x(signature);
            if (signature.length !== 65 * 2)
                throw new Error('invalid signature length');
            const sigh = signature.slice(0, -2);
            const end = signature.slice(-2);
            if (!['1b', '1c'].includes(end))
                throw new Error('invalid recovery bit');
            const sig = initSig(hexToBytes(sigh), end === '1b' ? 0 : 1);
            const pub = sig.recoverPublicKey(hash).toRawBytes(false);
            if (!verify(sig, hash, pub))
                throw new Error('invalid signature');
            return addr.fromPublicKey(pub);
        },
        verify(signature, message, address) {
            const recAddr = this.recoverPublicKey(signature, message);
            const low = recAddr.toLowerCase();
            const upp = recAddr.toUpperCase();
            if (address === low || address === upp)
                return true; // non-checksummed
            return recAddr === address; // checksummed
        },
    };
}
// EIP-191/EIP-7749: 0x19 <0x00> <intended validator address> <data to sign>
// export const intendedValidator = getSigner(
//   0x00,
//   ({ message, validator }: { message: Uint8Array; validator: string }) => {
//     const { data } = addr.parse(validator);
//     return concatBytes(hexToBytes(data), message);
//   }
// );
// EIP-191: 0x19 <0x45 (E)> <thereum Signed Message:\n" + len(message)> <data to sign>
export const personal = getSigner(0x45, (msg) => {
    if (typeof msg === 'string')
        msg = utf8ToBytes(msg);
    return concatBytes(utf8ToBytes(`thereum Signed Message:\n${msg.length}`), msg);
});
// TODO: merge with abi somehow?
function parseType(s) {
    let m = s.match(/^([^\[]+)(?:.*\[(.*?)\])?$/);
    if (!m)
        throw new Error(`parseType: wrong type: ${s}`);
    const base = m[1];
    const isArray = m[2] !== undefined;
    // TODO: check for safe integer
    const arrayLen = m[2] !== undefined && m[2] !== '' ? Number(m[2]) : undefined;
    if (arrayLen !== undefined && (!Number.isSafeInteger(arrayLen) || arrayLen.toString() !== m[2]))
        throw new Error(`parseType: wrong array length: ${s}`);
    let type = 'struct';
    if (['string', 'bytes'].includes(base))
        type = 'dynamic';
    else if (['bool', 'address'].includes(base))
        type = 'atomic';
    else if ((m = /^(u?)int([0-9]+)?$/.exec(base))) {
        const bits = m[2] ? +m[2] : 256;
        if (!Number.isSafeInteger(bits) || bits <= 0 || bits % 8 !== 0 || bits > 256)
            throw new Error('parseType: invalid numeric type');
        type = 'atomic';
    }
    else if ((m = /^bytes([0-9]{1,2})$/.exec(base))) {
        const bytes = +m[1];
        if (!bytes || bytes > 32)
            throw new Error(`parseType: wrong bytes<N=${bytes}> type`);
        type = 'atomic';
    }
    const item = s.replace(/\[[^\]]*\]$/, '');
    return { base, item, type, arrayLen, isArray };
}
// traverse dependency graph, find all transitive dependencies. Also, basic sanity check
function getDependencies(types) {
    if (typeof types !== 'object' || types === null)
        throw new Error('wrong types object');
    // Collect non-basic dependencies & sanity
    const res = {};
    for (const [name, fields] of Object.entries(types)) {
        const cur = new Set(); // type may appear multiple times in struct
        for (const { type } of fields) {
            const p = parseType(type);
            if (p.type !== 'struct')
                continue; // skip basic fields
            if (p.base === name)
                continue; // self reference
            if (!types[p.base])
                throw new Error(`getDependencies: wrong struct type name=${type}`);
            cur.add(p.base);
        }
        res[name] = cur;
    }
    // This should be more efficient with toposort + cycle detection, but I've already spent too much time here
    // and for most cases there won't be a lot of types here anyway.
    for (let changed = true; changed;) {
        changed = false;
        for (const [name, curDeps] of Object.entries(res)) {
            // Map here, because curDeps will change
            const trDeps = Array.from(curDeps).map((i) => res[i]);
            for (const d of trDeps) {
                for (const td of d) {
                    if (td === name || curDeps.has(td))
                        continue;
                    curDeps.add(td);
                    changed = true;
                }
            }
        }
    }
    return res;
}
function getTypes(types) {
    const deps = getDependencies(types);
    const names = {};
    // Build names
    for (const type in types)
        names[type] = `${type}(${types[type].map(({ name, type }) => `${type} ${name}`).join(',')})`;
    const fullNames = {};
    for (const [name, curDeps] of Object.entries(deps)) {
        const n = [name].concat(Array.from(curDeps).sort());
        fullNames[name] = n.map((i) => names[i]).join('');
    }
    const hashes = Object.fromEntries(Object.entries(fullNames).map(([k, v]) => [k, keccak_256(v)]));
    // fields
    const fields = {};
    for (const type in types) {
        const res = new Set();
        for (const { name } of types[type]) {
            if (res.has(name))
                throw new Error(`field ${name} included multiple times in type ${type}`);
            res.add(name);
        }
        fields[type] = res;
    }
    return { names, fullNames, hashes, fields };
}
// This re-uses domain per multiple requests, which is based on assumption that domain is static for different requests with
// different types. Please raise issue if you have different use case.
export function encoder(types, domain) {
    if (!isObject(domain))
        throw Error(`wrong domain=${domain}`);
    if (!isObject(types))
        throw Error(`wrong types=${types}`);
    const info = getTypes(types);
    const encodeField = (type, data, withHash = true) => {
        const p = parseType(type);
        if (p.isArray) {
            if (!Array.isArray(data))
                throw new Error(`expected array, got: ${data}`);
            if (p.arrayLen !== undefined && data.length !== p.arrayLen)
                throw new Error(`wrong array length: expected ${p.arrayLen}, got ${data}`);
            return keccak_256(concatBytes(...data.map((i) => encodeField(p.item, i))));
        }
        if (p.type === 'struct') {
            const def = types[type];
            if (!def)
                throw new Error(`wrong type: ${type}`);
            const fieldNames = info.fields[type];
            if (!isObject(data))
                throw new Error(`encoding non-object as custom type ${type}`);
            for (const k in data)
                if (!fieldNames.has(k))
                    throw new Error(`unexpected field ${k} in ${type}`);
            // TODO: use correct concatBytes (need to export from P?). This will easily crash with stackoverflow if too much fields.
            const fields = [];
            for (const { name, type } of def) {
                // This is not mentioned in spec, but used in eth-sig-util
                // Since there is no 'optional' fields inside eip712, it makes impossible to encode circular structure without arrays,
                // but seems like other project use this.
                // NOTE: this is V4 only stuff. If you need V3 behavior, please open issue.
                if (types[type] && data[name] === undefined) {
                    fields.push(new Uint8Array(32));
                    continue;
                }
                fields.push(encodeField(type, data[name]));
            }
            const res = concatBytes(info.hashes[p.base], ...fields);
            return withHash ? keccak_256(res) : res;
        }
        if (type === 'string' || type === 'bytes') {
            if (type === 'bytes' && typeof data === 'string')
                data = ethHex.decode(data);
            return keccak_256(data); // hashed as is!
        }
        // Type conversion is neccessary here, because we can get data from JSON (no Uint8Arrays/bigints).
        if (type.startsWith('bytes') && typeof data === 'string')
            data = ethHex.decode(data);
        if ((type.startsWith('int') || type.startsWith('uint')) && typeof data === 'string')
            data = BigInt(data);
        return mapComponent({ type }).encode(data);
    };
    const encodeData = (type, data) => {
        astr(type);
        if (!types[type])
            throw new Error(`Unknown type: ${type}`);
        if (!isObject(data))
            throw new Error('wrong data object');
        return encodeField(type, data, false);
    };
    const structHash = (type, data) => keccak_256(encodeData(type, data));
    const domainHash = structHash('EIP712Domain', domain);
    // NOTE: we cannot use Msg here, since its already parametrized and everything will break.
    const signer = getSigner(0x01, (msg) => {
        if (typeof msg.primaryType !== 'string')
            throw Error(`wrong primaryType=${msg.primaryType}`);
        if (!isObject(msg.message))
            throw Error(`wrong message=${msg.message}`);
        if (msg.primaryType === 'EIP712Domain')
            return domainHash;
        return concatBytes(domainHash, structHash(msg.primaryType, msg.message));
    });
    return {
        encodeData: (type, message) => ethHex.encode(encodeData(type, message)),
        structHash: (type, message) => ethHex.encode(structHash(type, message)),
        // Signer
        _getHash: (primaryType, message) => signer._getHash({ primaryType, message }),
        sign: (primaryType, message, privateKey, extraEntropy) => signer.sign({ primaryType, message }, privateKey, extraEntropy),
        verify: (primaryType, signature, message, address) => signer.verify(signature, { primaryType, message }, address),
        recoverPublicKey: (primaryType, signature, message) => signer.recoverPublicKey(signature, { primaryType, message }),
    };
}
export const EIP712Domain = [
    { name: 'name', type: 'string' }, // the user readable name of signing domain, i.e. the name of the DApp or the protocol.
    { name: 'version', type: 'string' }, // the current major version of the signing domain. Signatures from different versions are not compatible.
    { name: 'chainId', type: 'uint256' }, // the EIP-155 chain id. The user-agent should refuse signing if it does not match the currently active chain.
    { name: 'verifyingContract', type: 'address' }, // the address of the contract that will verify the signature. The user-agent may do contract specific phishing prevention.
    { name: 'salt', type: 'bytes32' }, // an disambiguating salt for the protocol. This can be used as a domain separator of last resort.
];
const domainTypes = { EIP712Domain: EIP712Domain };
// Filter unused domain fields from type
export function getDomainType(domain) {
    return EIP712Domain.filter(({ name }) => domain[name] !== undefined);
}
const getTypedTypes = (typed) => ({
    EIP712Domain: getDomainType(typed.domain),
    ...typed.types,
});
function validateTyped(t) {
    if (!isObject(t.message))
        throw new Error('wrong message');
    if (!isObject(t.domain))
        throw new Error('wrong domain');
    if (!isObject(t.types))
        throw new Error('wrong types');
    if (typeof t.primaryType !== 'string' || !t.types[t.primaryType])
        throw new Error('wrong primaryType');
}
export function encodeData(typed) {
    validateTyped(typed);
    return encoder(getTypedTypes(typed), typed.domain).encodeData(typed.primaryType, typed.message);
}
export function sigHash(typed) {
    validateTyped(typed);
    return encoder(getTypedTypes(typed), typed.domain)._getHash(typed.primaryType, typed.message);
}
export function signTyped(typed, privateKey, extraEntropy) {
    validateTyped(typed);
    return encoder(getTypedTypes(typed), typed.domain).sign(typed.primaryType, typed.message, privateKey, extraEntropy);
}
export function verifyTyped(signature, typed, address) {
    validateTyped(typed);
    return encoder(getTypedTypes(typed), typed.domain).verify(typed.primaryType, signature, typed.message, address);
}
export function recoverPublicKeyTyped(signature, typed) {
    return encoder(getTypedTypes(typed), typed.domain).recoverPublicKey(typed.primaryType, signature, typed.message);
}
// Internal methods for test purposes only
export const _TEST = /* @__PURE__ */ { parseType, getDependencies, getTypes, encoder };
//# sourceMappingURL=typed-data.js.map