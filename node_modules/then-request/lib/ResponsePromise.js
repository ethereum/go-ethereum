"use strict";
exports.__esModule = true;
var Promise = require("promise");
function getBody(encoding) {
    if (!encoding) {
        return this.then(getBodyBinary);
    }
    if (encoding === 'utf8') {
        return this.then(getBodyUTF8);
    }
    return this.then(getBodyWithEncoding(encoding));
}
function getBodyWithEncoding(encoding) {
    return function (res) { return res.getBody(encoding); };
}
function getBodyBinary(res) {
    return res.getBody();
}
function getBodyUTF8(res) {
    return res.getBody('utf8');
}
function toResponsePromise(result) {
    result.getBody = getBody;
    return result;
}
exports["default"] = toResponsePromise;
exports.ResponsePromise = undefined;
