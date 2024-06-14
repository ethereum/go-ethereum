"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.InvalidStatusCodeError = exports.InvalidCertError = void 0;
const DEFAULT_OPT = Object.freeze({
    redirect: true,
    expectStatusCode: 200,
    headers: {},
    full: false,
    keepAlive: true,
    cors: false,
    referrer: false,
    sslAllowSelfSigned: false,
    _redirectCount: 0,
});
class InvalidCertError extends Error {
    constructor(msg, fingerprint256) {
        super(msg);
        this.fingerprint256 = fingerprint256;
    }
}
exports.InvalidCertError = InvalidCertError;
class InvalidStatusCodeError extends Error {
    constructor(statusCode) {
        super(`Request Failed. Status Code: ${statusCode}`);
        this.statusCode = statusCode;
    }
}
exports.InvalidStatusCodeError = InvalidStatusCodeError;
function detectType(b, type) {
    if (!type || type === 'text' || type === 'json') {
        try {
            let text = new TextDecoder('utf8', { fatal: true }).decode(b);
            if (type === 'text')
                return text;
            try {
                return JSON.parse(text);
            }
            catch (err) {
                if (type === 'json')
                    throw err;
                return text;
            }
        }
        catch (err) {
            if (type === 'text' || type === 'json')
                throw err;
        }
    }
    return b;
}
let agents = {};
function fetchNode(url, _options) {
    let options = { ...DEFAULT_OPT, ..._options };
    const http = require('http');
    const https = require('https');
    const zlib = require('zlib');
    const { promisify } = require('util');
    const { resolve: urlResolve } = require('url');
    const isSecure = !!/^https/.test(url);
    let opts = {
        method: options.method || 'GET',
        headers: { 'Accept-Encoding': 'gzip, deflate, br' },
    };
    const compactFP = (s) => s.replace(/:| /g, '').toLowerCase();
    if (options.keepAlive) {
        const agentOpt = {
            keepAlive: true,
            keepAliveMsecs: 30 * 1000,
            maxFreeSockets: 1024,
            maxCachedSessions: 1024,
        };
        const agentKey = [
            isSecure,
            isSecure && options.sslPinnedCertificates?.map((i) => compactFP(i)).sort(),
        ].join();
        opts.agent =
            agents[agentKey] || (agents[agentKey] = new (isSecure ? https : http).Agent(agentOpt));
    }
    if (options.type === 'json')
        opts.headers['Content-Type'] = 'application/json';
    if (options.data) {
        if (!options.method)
            opts.method = 'POST';
        opts.body = options.type === 'json' ? JSON.stringify(options.data) : options.data;
    }
    opts.headers = { ...opts.headers, ...options.headers };
    if (options.sslAllowSelfSigned)
        opts.rejectUnauthorized = false;
    const handleRes = async (res) => {
        const status = res.statusCode;
        if (options.redirect && 300 <= status && status < 400 && res.headers['location']) {
            if (options._redirectCount == 10)
                throw new Error('Request failed. Too much redirects.');
            options._redirectCount += 1;
            return await fetchNode(urlResolve(url, res.headers['location']), options);
        }
        if (options.expectStatusCode && status !== options.expectStatusCode) {
            res.resume();
            throw new InvalidStatusCodeError(status);
        }
        let buf = [];
        for await (const chunk of res)
            buf.push(chunk);
        let bytes = Buffer.concat(buf);
        const encoding = res.headers['content-encoding'];
        if (encoding === 'br')
            bytes = await promisify(zlib.brotliDecompress)(bytes);
        if (encoding === 'gzip' || encoding === 'deflate')
            bytes = await promisify(zlib.unzip)(bytes);
        const body = detectType(bytes, options.type);
        if (options.full)
            return { headers: res.headers, status, body };
        return body;
    };
    return new Promise((resolve, reject) => {
        const handleError = async (err) => {
            if (err && err.code === 'DEPTH_ZERO_SELF_SIGNED_CERT') {
                try {
                    await fetchNode(url, { ...options, sslAllowSelfSigned: true, sslPinnedCertificates: [] });
                }
                catch (e) {
                    if (e && e.fingerprint256) {
                        err = new InvalidCertError(`Self-signed SSL certificate: ${e.fingerprint256}`, e.fingerprint256);
                    }
                }
            }
            reject(err);
        };
        const req = (isSecure ? https : http).request(url, opts, (res) => {
            res.on('error', handleError);
            (async () => {
                try {
                    resolve(await handleRes(res));
                }
                catch (error) {
                    reject(error);
                }
            })();
        });
        req.on('error', handleError);
        const pinned = options.sslPinnedCertificates?.map((i) => compactFP(i));
        const mfetchSecureConnect = (socket) => {
            const fp256 = compactFP(socket.getPeerCertificate()?.fingerprint256 || '');
            if (!fp256 && socket.isSessionReused())
                return;
            if (pinned.includes(fp256))
                return;
            req.emit('error', new InvalidCertError(`Invalid SSL certificate: ${fp256} Expected: ${pinned}`, fp256));
            return req.abort();
        };
        if (options.sslPinnedCertificates) {
            req.on('socket', (socket) => {
                const hasListeners = socket
                    .listeners('secureConnect')
                    .map((i) => (i.name || '').replace('bound ', ''))
                    .includes('mfetchSecureConnect');
                if (hasListeners)
                    return;
                socket.on('secureConnect', mfetchSecureConnect.bind(null, socket));
            });
        }
        if (options.keepAlive)
            req.setNoDelay(true);
        if (opts.body)
            req.write(opts.body);
        req.end();
    });
}
const SAFE_HEADERS = new Set(['Accept', 'Accept-Language', 'Content-Language', 'Content-Type'].map((i) => i.toLowerCase()));
const FORBIDDEN_HEADERS = new Set(['Accept-Charset', 'Accept-Encoding', 'Access-Control-Request-Headers', 'Access-Control-Request-Method',
    'Connection', 'Content-Length', 'Cookie', 'Cookie2', 'Date', 'DNT', 'Expect', 'Host', 'Keep-Alive', 'Origin', 'Referer', 'TE', 'Trailer',
    'Transfer-Encoding', 'Upgrade', 'Via'].map((i) => i.toLowerCase()));
async function fetchBrowser(url, _options) {
    let options = { ...DEFAULT_OPT, ..._options };
    const headers = new Headers();
    if (options.type === 'json')
        headers.set('Content-Type', 'application/json');
    let parsed = new URL(url);
    if (parsed.username) {
        const auth = btoa(`${parsed.username}:${parsed.password}`);
        headers.set('Authorization', `Basic ${auth}`);
        parsed.username = '';
        parsed.password = '';
    }
    url = '' + parsed;
    for (let k in options.headers) {
        const name = k.toLowerCase();
        if (SAFE_HEADERS.has(name) || (options.cors && !FORBIDDEN_HEADERS.has(name)))
            headers.set(k, options.headers[k]);
    }
    let opts = { headers, redirect: options.redirect ? 'follow' : 'manual' };
    if (!options.referrer)
        opts.referrerPolicy = 'no-referrer';
    if (options.cors)
        opts.mode = 'cors';
    if (options.data) {
        if (!options.method)
            opts.method = 'POST';
        opts.body = options.type === 'json' ? JSON.stringify(options.data) : options.data;
    }
    const res = await fetch(url, opts);
    if (options.expectStatusCode && res.status !== options.expectStatusCode)
        throw new InvalidStatusCodeError(res.status);
    const body = detectType(new Uint8Array(await res.arrayBuffer()), options.type);
    if (options.full)
        return { headers: Object.fromEntries(res.headers.entries()), status: res.status, body };
    return body;
}
const IS_NODE = !!(typeof process == 'object' &&
    process.versions &&
    process.versions.node &&
    process.versions.v8);
function fetchUrl(url, options) {
    const fn = IS_NODE ? fetchNode : fetchBrowser;
    return fn(url, options);
}
exports.default = fetchUrl;
