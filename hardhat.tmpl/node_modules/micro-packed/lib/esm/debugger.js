import { base64, hex } from '@scure/base';
import * as P from "./index.js";
const Path = P._TEST.Path; // Internal, debug-only
const UNKNOWN = '(???)';
const codes = { esc: 27, nl: 10 };
const esc = String.fromCharCode(codes.esc);
const nl = String.fromCharCode(codes.nl);
const bold = esc + '[1m';
const gray = esc + '[90m';
const reset = esc + '[0m';
const red = esc + '[31m';
const green = esc + '[32m';
const yellow = esc + '[33m';
class DebugReader extends P._TEST._Reader {
    constructor() {
        super(...arguments);
        this.debugLst = [];
    }
    get lastElm() {
        if (this.debugLst.length)
            return this.debugLst[this.debugLst.length - 1];
        return { start: 0, end: 0, path: '' };
    }
    pushObj(obj, objFn) {
        return Path.pushObj(this.stack, obj, (cb) => {
            objFn((field, fieldFn) => {
                cb(field, () => {
                    {
                        const last = this.lastElm;
                        if (last.end === undefined)
                            last.end = this.pos;
                        else if (last.end !== this.pos) {
                            this.debugLst.push({
                                path: `${Path.path(this.stack)}/${UNKNOWN}`,
                                start: last.end,
                                end: this.pos,
                            });
                        }
                        this.cur = { path: `${Path.path(this.stack)}/${field}`, start: this.pos };
                    }
                    fieldFn();
                    {
                        // happens if pop after pop (exit from nested structure)
                        if (!this.cur) {
                            const last = this.lastElm;
                            if (last.end === undefined)
                                last.end = this.pos;
                            else if (last.end !== this.pos) {
                                this.debugLst.push({
                                    start: last.end,
                                    end: this.pos,
                                    path: last.path + `/${UNKNOWN}`,
                                });
                            }
                        }
                        else {
                            this.cur.end = this.pos;
                            const last = this.stack[this.stack.length - 1];
                            const lastItem = last.obj;
                            const lastField = last.field;
                            if (lastItem && lastField !== undefined)
                                this.cur.value = lastItem[lastField];
                            this.debugLst.push(this.cur);
                            this.cur = undefined;
                        }
                    }
                });
            });
        });
    }
    finishDebug() {
        const end = this.data.length;
        if (this.cur)
            this.debugLst.push({ end, ...this.cur });
        const last = this.lastElm;
        if (!last || last.end !== end)
            this.debugLst.push({ start: this.pos, end, path: UNKNOWN });
    }
}
function toBytes(data) {
    if (P.utils.isBytes(data))
        return data;
    if (typeof data !== 'string')
        throw new Error('PD: data should be string or Uint8Array');
    try {
        return base64.decode(data);
    }
    catch (e) { }
    try {
        return hex.decode(data);
    }
    catch (e) { }
    throw new Error(`PD: data has unknown string format: ${data}`);
}
function mapData(lst, data) {
    let end = 0;
    const res = [];
    for (const elm of lst) {
        if (elm.start !== end)
            throw new Error(`PD: elm start=${elm.start} after prev elm end=${end}`);
        if (elm.end === undefined)
            throw new Error(`PD: elm.end is undefined=${elm}`);
        res.push({ path: elm.path, data: data.slice(elm.start, elm.end), value: elm.value });
        end = elm.end;
    }
    if (end !== data.length)
        throw new Error('PD: not all data mapped');
    return res;
}
function chrWidth(s) {
    /*
    It is almost impossible to find out real characters width in terminal since it depends on terminal itself, current unicode version and moon's phase.
    So, we just stripping ANSI, tabs and unicode supplimental characters. Emoji support requires big tables (and have no guarantee to work), so we ignore it for now.
    Also, no support for full width unicode characters for now.
    */
    return s
        .replace(/[\u001B\u009B][[\]()#;?]*(?:(?:(?:[a-zA-Z\d]*(?:;[-a-zA-Z\d\/#&.:=?%@~_]*)*)?\u0007)|(?:(?:\d{1,4}(?:;\d{0,4})*)?[\dA-PR-TZcf-ntqry=><~]))/g, '')
        .replace('\t', '  ')
        .replace(/[\uD800-\uDBFF][\uDC00-\uDFFF]/g, ' ').length;
}
function wrap(s, padding = 0) {
    // @ts-ignore
    const limit = process.stdout.columns - 3 - padding;
    if (chrWidth(s) <= limit)
        return s;
    while (chrWidth(s) > limit)
        s = s.slice(0, -1);
    return `${s}${reset}...`;
}
export function table(data) {
    let res = [];
    const str = (v) => (v === undefined ? '' : '' + v);
    const pad = (s, width) => `${s}${''.padEnd(Math.max(0, width - chrWidth(s)), ' ')}`;
    let widths = {};
    for (let elm of data) {
        for (let k in elm) {
            widths[k] = Math.max(widths[k] || 0, chrWidth(str(k)), str(elm[k])
                .split(nl)
                .reduce((a, b) => Math.max(a, chrWidth(b)), 0));
        }
    }
    const columns = Object.keys(widths);
    if (!data.length || !columns.length)
        throw new Error('No data');
    const padding = ` ${reset}${gray}│${reset} `;
    res.push(wrap(` ${columns.map((c) => `${bold}${pad(c, widths[c])}`).join(padding)}${reset}`, 3));
    for (let idx = 0; idx < data.length; idx++) {
        const elm = data[idx];
        const row = columns.map((i) => str(elm[i]).split(nl));
        let message = [...Array(Math.max(...row.map((i) => i.length))).keys()]
            .map((line) => row.map((c, i) => pad(str(c[line]), widths[columns[i]])))
            .map((line, _) => wrap(` ${line.join(padding)} `, 1))
            .join(nl);
        res.push(message);
    }
    for (let i = 0; i < res.length; i++) {
        const border = columns
            .map((c) => ''.padEnd(widths[c], '─'))
            .join(`─${i === res.length - 1 ? '┴' : '┼'}─`);
        res[i] += wrap(`${nl}${reset}${gray}─${border}─${reset}`);
    }
    // @ts-ignore
    console.log(res.join(nl));
}
function fmtData(data, perLine = 8) {
    const res = [];
    for (let i = 0; i < data.length; i += perLine) {
        res.push(hex.encode(data.slice(i, i + perLine)));
    }
    return res.map((i) => `${bold}${i}${reset}`).join(nl);
}
function fmtValue(value) {
    if (P.utils.isBytes(value))
        return `b(${green}${hex.encode(value)}${reset} len=${value.length})`;
    if (typeof value === 'string')
        return `s(${green}"${value}"${reset} len=${value.length})`;
    if (typeof value === 'number' || typeof value === 'bigint')
        return `n(${value})`;
    // console.log('fmt', value);
    // if (Object.prototype.toString.call(value) === '[object Object]') return inspect(value);
    return '' + value;
}
export function decode(coder, data, forcePrint = false) {
    data = toBytes(data);
    const r = new DebugReader(data);
    let res, e;
    try {
        res = coder.decodeStream(r);
        r.finish();
    }
    catch (_e) {
        e = _e;
    }
    r.finishDebug();
    if (e || forcePrint) {
        // @ts-ignore
        console.log('==== DECODED BEFORE ERROR ====');
        table(mapData(r.debugLst, data).map((elm) => ({
            Data: fmtData(elm.data),
            Len: elm.data.length,
            Path: `${green}${elm.path}${reset}`,
            Value: fmtValue(elm.value),
        })));
        // @ts-ignore
        console.log('==== /DECODED BEFORE ERROR ====');
    }
    if (e)
        throw e;
    return res;
}
function getMap(coder, data) {
    data = toBytes(data);
    const r = new DebugReader(data);
    coder.decodeStream(r);
    r.finish();
    r.finishDebug();
    return mapData(r.debugLst, data);
}
function diffData(a, e) {
    const len = Math.max(a.length, e.length);
    let outA = '', outE = '';
    const charHex = (n) => n.toString(16).padStart(2, '0');
    for (let i = 0; i < len; i++) {
        const [aI, eI] = [a[i], e[i]];
        if (i && !(i % 8)) {
            if (aI !== undefined)
                outA += nl;
            if (eI !== undefined)
                outE += nl;
        }
        if (aI !== undefined)
            outA += aI === eI ? charHex(aI) : `${yellow}${charHex(aI)}${reset}`;
        if (eI !== undefined)
            outE += aI === eI ? charHex(eI) : `${yellow}${charHex(eI)}${reset}`;
    }
    return [outA, outE];
}
function diffPath(a, e) {
    if (a === e)
        return a;
    return `A: ${red}${a}${reset}${nl}E: ${green}${e}${reset}`;
}
function diffLength(a, e) {
    const [aLen, eLen] = [a.length, e.length];
    if (aLen === eLen)
        return aLen;
    return `A: ${red}${aLen}${reset}${nl}E: ${green}${eLen}${reset}`;
}
function diffValue(a, e) {
    const [aV, eV] = [a, e].map(fmtValue);
    if (aV === eV)
        return aV;
    return `A: ${red}${aV}${reset}${nl}E: ${green}${eV}${reset}`;
}
export function diff(coder, actual, expected, skipSame = true) {
    // @ts-ignore
    console.log('==== DIFF ====');
    const [_actual, _expected] = [actual, expected].map((i) => getMap(coder, i));
    const len = Math.max(_actual.length, _expected.length);
    const data = [];
    const DEF = { data: P.EMPTY, path: '' };
    for (let i = 0; i < len; i++) {
        const [a, e] = [_actual[i] || DEF, _expected[i] || DEF];
        if (P.utils.equalBytes(a.data, e.data) && skipSame)
            continue;
        const [adata, edata] = diffData(a.data, e.data);
        data.push({
            'Data (A)': adata,
            'Data (E)': edata,
            Len: diffLength(a.data, e.data),
            Path: diffPath(a.path, e.path),
            Value: diffValue(a.value, e.value),
        });
    }
    table(data);
    // @ts-ignore
    console.log('==== /DIFF ====');
}
/**
 * Wraps a CoderType with debug logging for encoding and decoding operations.
 * @param inner - Inner CoderType to wrap.
 * @returns Inner wrapped in debug prints via console.log.
 * @example
 * const debugInt = P.debug(P.U32LE); // Will print info to console on encoding/decoding
 */
export function debug(inner) {
    if (!P.utils.isCoder(inner))
        throw new Error(`debug: invalid inner value ${inner}`);
    const log = (name, rw, value) => {
        // @ts-ignore
        console.log(`DEBUG/${name}(${Path.path(rw.stack)}):`, { type: typeof value, value });
        return value;
    };
    return P.wrap({
        size: inner.size,
        encodeStream: (w, value) => inner.encodeStream(w, log('encode', w, value)),
        decodeStream: (r) => log('decode', r, inner.decodeStream(r)),
    });
}
//# sourceMappingURL=debugger.js.map