var f = require('./formatters');
var SolidityType = require('./type');

/**
 * SolidityTypeBytes is a prootype that represents bytes type
 * It matches:
 * bytes
 * bytes[]
 * bytes[4]
 * bytes[][]
 * bytes[3][]
 * bytes[][6][], ...
 * bytes32
 * bytes64[]
 * bytes8[4]
 * bytes256[][]
 * bytes[3][]
 * bytes64[][6][], ...
 */
var SolidityTypeBytes = function () {
    this._inputFormatter = f.formatInputBytes;
    this._outputFormatter = f.formatOutputBytes;
};

SolidityTypeBytes.prototype = new SolidityType({});
SolidityTypeBytes.prototype.constructor = SolidityTypeBytes;

SolidityTypeBytes.prototype.isType = function (name) {
    return !!name.match(/^bytes([0-9]{1,})(\[([0-9]*)\])*$/);
};

SolidityTypeBytes.prototype.staticPartLength = function (name) {
    var matches = name.match(/^bytes([0-9]*)/);
    var size = parseInt(matches[1]);
    return size * this.staticArrayLength(name);
};

module.exports = SolidityTypeBytes;
