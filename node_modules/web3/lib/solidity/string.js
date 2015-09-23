var f = require('./formatters');
var SolidityType = require('./type');

var SolidityTypeString = function () {
    this._inputFormatter = f.formatInputString;
    this._outputFormatter = f.formatOutputString;
};

SolidityTypeString.prototype = new SolidityType({});
SolidityTypeString.prototype.constructor = SolidityTypeString;

SolidityTypeString.prototype.isType = function (name) {
    return !!name.match(/^string(\[([0-9]*)\])*$/);
};

SolidityTypeString.prototype.staticPartLength = function (name) {
    return 32 * this.staticArrayLength(name);
};

SolidityTypeString.prototype.isDynamicType = function () {
    return true;
};

module.exports = SolidityTypeString;

