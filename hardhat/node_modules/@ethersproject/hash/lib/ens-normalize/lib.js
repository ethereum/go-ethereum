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
exports.ens_normalize = exports.ens_normalize_post_check = void 0;
var strings_1 = require("@ethersproject/strings");
var include_js_1 = require("./include.js");
var r = (0, include_js_1.getData)();
var decoder_js_1 = require("./decoder.js");
// @TODO: This should be lazily loaded
var VALID = new Set((0, decoder_js_1.read_member_array)(r));
var IGNORED = new Set((0, decoder_js_1.read_member_array)(r));
var MAPPED = (0, decoder_js_1.read_mapped_map)(r);
var EMOJI_ROOT = (0, decoder_js_1.read_emoji_trie)(r);
//const NFC_CHECK = new Set(read_member_array(r, Array.from(VALID.values()).sort((a, b) => a - b)));
//const STOP = 0x2E;
var HYPHEN = 0x2D;
var UNDERSCORE = 0x5F;
function explode_cp(name) {
    return (0, strings_1.toUtf8CodePoints)(name);
}
function filter_fe0f(cps) {
    return cps.filter(function (cp) { return cp != 0xFE0F; });
}
function ens_normalize_post_check(name) {
    for (var _i = 0, _a = name.split('.'); _i < _a.length; _i++) {
        var label = _a[_i];
        var cps = explode_cp(label);
        try {
            for (var i = cps.lastIndexOf(UNDERSCORE) - 1; i >= 0; i--) {
                if (cps[i] !== UNDERSCORE) {
                    throw new Error("underscore only allowed at start");
                }
            }
            if (cps.length >= 4 && cps.every(function (cp) { return cp < 0x80; }) && cps[2] === HYPHEN && cps[3] === HYPHEN) {
                throw new Error("invalid label extension");
            }
        }
        catch (err) {
            throw new Error("Invalid label \"" + label + "\": " + err.message);
        }
    }
    return name;
}
exports.ens_normalize_post_check = ens_normalize_post_check;
function ens_normalize(name) {
    return ens_normalize_post_check(normalize(name, filter_fe0f));
}
exports.ens_normalize = ens_normalize;
function normalize(name, emoji_filter) {
    var input = explode_cp(name).reverse(); // flip for pop
    var output = [];
    while (input.length) {
        var emoji = consume_emoji_reversed(input);
        if (emoji) {
            output.push.apply(output, emoji_filter(emoji));
            continue;
        }
        var cp = input.pop();
        if (VALID.has(cp)) {
            output.push(cp);
            continue;
        }
        if (IGNORED.has(cp)) {
            continue;
        }
        var cps = MAPPED[cp];
        if (cps) {
            output.push.apply(output, cps);
            continue;
        }
        throw new Error("Disallowed codepoint: 0x" + cp.toString(16).toUpperCase());
    }
    return ens_normalize_post_check(nfc(String.fromCodePoint.apply(String, output)));
}
function nfc(s) {
    return s.normalize('NFC');
}
function consume_emoji_reversed(cps, eaten) {
    var _a;
    var node = EMOJI_ROOT;
    var emoji;
    var saved;
    var stack = [];
    var pos = cps.length;
    if (eaten)
        eaten.length = 0; // clear input buffer (if needed)
    var _loop_1 = function () {
        var cp = cps[--pos];
        node = (_a = node.branches.find(function (x) { return x.set.has(cp); })) === null || _a === void 0 ? void 0 : _a.node;
        if (!node)
            return "break";
        if (node.save) { // remember
            saved = cp;
        }
        else if (node.check) { // check exclusion
            if (cp === saved)
                return "break";
        }
        stack.push(cp);
        if (node.fe0f) {
            stack.push(0xFE0F);
            if (pos > 0 && cps[pos - 1] == 0xFE0F)
                pos--; // consume optional FE0F
        }
        if (node.valid) { // this is a valid emoji (so far)
            emoji = stack.slice(); // copy stack
            if (node.valid == 2)
                emoji.splice(1, 1); // delete FE0F at position 1 (RGI ZWJ don't follow spec!)
            if (eaten)
                eaten.push.apply(eaten, cps.slice(pos).reverse()); // copy input (if needed)
            cps.length = pos; // truncate
        }
    };
    while (pos) {
        var state_1 = _loop_1();
        if (state_1 === "break")
            break;
    }
    return emoji;
}
//# sourceMappingURL=lib.js.map