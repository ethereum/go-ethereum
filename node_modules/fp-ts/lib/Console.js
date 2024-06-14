"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
/**
 * @file Adapted from https://github.com/purescript/purescript-console
 */
var IO_1 = require("./IO");
/**
 * @since 1.0.0
 */
exports.log = function (s) {
    return new IO_1.IO(function () { return console.log(s); }); // tslint:disable-line:no-console
};
/**
 * @since 1.0.0
 */
exports.warn = function (s) {
    return new IO_1.IO(function () { return console.warn(s); }); // tslint:disable-line:no-console
};
/**
 * @since 1.0.0
 */
exports.error = function (s) {
    return new IO_1.IO(function () { return console.error(s); }); // tslint:disable-line:no-console
};
/**
 * @since 1.0.0
 */
exports.info = function (s) {
    return new IO_1.IO(function () { return console.info(s); }); // tslint:disable-line:no-console
};
