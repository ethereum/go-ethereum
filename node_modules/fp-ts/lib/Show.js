"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
/**
 * @since 1.17.0
 */
exports.showString = {
    show: function (a) { return JSON.stringify(a); }
};
/**
 * @since 1.17.0
 */
exports.showNumber = {
    show: function (a) { return JSON.stringify(a); }
};
/**
 * @since 1.17.0
 */
exports.showBoolean = {
    show: function (a) { return JSON.stringify(a); }
};
/**
 * @since 1.17.0
 */
exports.getStructShow = function (shows) {
    return {
        show: function (s) {
            return "{ " + Object.keys(shows)
                .map(function (k) { return k + ": " + shows[k].show(s[k]); })
                .join(', ') + " }";
        }
    };
};
/**
 * @since 1.17.0
 */
exports.getTupleShow = function () {
    var shows = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        shows[_i] = arguments[_i];
    }
    return {
        show: function (t) { return "[" + t.map(function (a, i) { return shows[i].show(a); }).join(', ') + "]"; }
    };
};
