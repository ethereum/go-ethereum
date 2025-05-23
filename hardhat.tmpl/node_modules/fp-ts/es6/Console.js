/**
 * @file Adapted from https://github.com/purescript/purescript-console
 */
import { IO } from './IO';
/**
 * @since 1.0.0
 */
export var log = function (s) {
    return new IO(function () { return console.log(s); }); // tslint:disable-line:no-console
};
/**
 * @since 1.0.0
 */
export var warn = function (s) {
    return new IO(function () { return console.warn(s); }); // tslint:disable-line:no-console
};
/**
 * @since 1.0.0
 */
export var error = function (s) {
    return new IO(function () { return console.error(s); }); // tslint:disable-line:no-console
};
/**
 * @since 1.0.0
 */
export var info = function (s) {
    return new IO(function () { return console.info(s); }); // tslint:disable-line:no-console
};
