"use strict";
exports.__esModule = true;
var FormData = /** @class */ (function () {
    function FormData() {
        this._entries = [];
    }
    FormData.prototype.append = function (key, value, fileName) {
        this._entries.push({ key: key, value: value, fileName: fileName });
    };
    return FormData;
}());
exports.FormData = FormData;
function getFormDataEntries(fd) {
    return fd._entries;
}
exports.getFormDataEntries = getFormDataEntries;
