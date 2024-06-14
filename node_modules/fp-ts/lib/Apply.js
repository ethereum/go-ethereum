"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var function_1 = require("./function");
function applyFirst(F) {
    return function (fa, fb) { return F.ap(F.map(fa, function_1.constant), fb); };
}
exports.applyFirst = applyFirst;
function applySecond(F) {
    return function (fa, fb) { return F.ap(F.map(fa, function () { return function (b) { return b; }; }), fb); };
}
exports.applySecond = applySecond;
// tslint:disable-next-line: deprecation
function liftA2(F) {
    return function (f) { return function (fa) { return function (fb) { return F.ap(F.map(fa, f), fb); }; }; };
}
exports.liftA2 = liftA2;
function liftA3(F
// tslint:disable-next-line: deprecation
) {
    return function (f) { return function (fa) { return function (fb) { return function (fc) { return F.ap(F.ap(F.map(fa, f), fb), fc); }; }; }; };
}
exports.liftA3 = liftA3;
function liftA4(F
// tslint:disable-next-line: deprecation
) {
    return function (f) { return function (fa) { return function (fb) { return function (fc) { return function (fd) { return F.ap(F.ap(F.ap(F.map(fa, f), fb), fc), fd); }; }; }; }; };
}
exports.liftA4 = liftA4;
function getSemigroup(F, S) {
    var f = function (a) { return function (b) { return S.concat(a, b); }; };
    return function () { return ({
        concat: function (x, y) { return F.ap(F.map(x, f), y); }
    }); };
}
exports.getSemigroup = getSemigroup;
// tslint:disable-next-line: deprecation
var tupleConstructors = {};
function sequenceT(F) {
    return function () {
        var args = [];
        for (var _i = 0; _i < arguments.length; _i++) {
            args[_i] = arguments[_i];
        }
        var len = args.length;
        var f = tupleConstructors[len];
        if (!Boolean(f)) {
            // tslint:disable-next-line: deprecation
            f = tupleConstructors[len] = function_1.curried(function () {
                var args = [];
                for (var _i = 0; _i < arguments.length; _i++) {
                    args[_i] = arguments[_i];
                }
                return args;
            }, len - 1, []);
        }
        var r = F.map(args[0], f);
        for (var i = 1; i < len; i++) {
            r = F.ap(r, args[i]);
        }
        return r;
    };
}
exports.sequenceT = sequenceT;
function sequenceS(F) {
    return function (r) {
        var keys = Object.keys(r);
        var fst = keys[0];
        var others = keys.slice(1);
        var fr = F.map(r[fst], function (a) {
            var _a;
            return (_a = {}, _a[fst] = a, _a);
        });
        var _loop_1 = function (key) {
            fr = F.ap(F.map(fr, function (r) { return function (a) {
                r[key] = a;
                return r;
            }; }), r[key]);
        };
        for (var _i = 0, others_1 = others; _i < others_1.length; _i++) {
            var key = others_1[_i];
            _loop_1(key);
        }
        return fr;
    };
}
exports.sequenceS = sequenceS;
