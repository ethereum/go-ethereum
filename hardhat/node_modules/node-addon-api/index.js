var path = require('path');

var versionArray = process.version
  .substr(1)
  .replace(/-.*$/, '')
  .split('.')
  .map(function(item) {
    return +item;
  });

// TODO: Check if the main node semantic version is within multiple ranges,
// or detect presence of built-in N-API by some other mechanism TBD.

// We know which version of Node.js first shipped the incarnation of the API
// available in *this* package. So, if we find that the Node.js version is below
// that, we indicate that the API is missing from Node.js.
var isNodeApiBuiltin = (
  versionArray[0] > 8 ||
  (versionArray[0] == 8 && versionArray[1] >= 6) ||
  (versionArray[0] == 6 && versionArray[1] >= 15) ||
  (versionArray[0] == 6 && versionArray[1] >= 14 && versionArray[2] >= 2));

// The flag is not needed when the Node version is not 8, nor if the API is
// built-in, because we removed the flag at the same time as creating the final
// incarnation of the built-in API.
var needsFlag = (!isNodeApiBuiltin && versionArray[0] == 8);

var include = [__dirname];
var gyp = path.join(__dirname, 'src', 'node_api.gyp');

if (isNodeApiBuiltin) {
  gyp += ':nothing';
} else {
  gyp += ':node-api';
  include.unshift(path.join(__dirname, 'external-napi'));
}

module.exports = {
  include: include.map(function(item) {
    return '"' + item + '"';
  }).join(' '),
  gyp: gyp,
  isNodeApiBuiltin: isNodeApiBuiltin,
  needsFlag: needsFlag
};
