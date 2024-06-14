const nativeAddon = require('node-gyp-build')(__dirname)
if (typeof nativeAddon !== 'function') {
  // Some new runtimes (bun) don't support N-API
  // but the build step incorrectly succeeds.
  // The value should be a function, but in bun it returns
  // an empty object {} so we use typeof to check that
  // it is a function and throw otherwise.
  // This throw will cause "keccak" import to fallback to JS.
  throw new Error('Native add-on failed to load')
}
module.exports = require('./lib/api')(nativeAddon)
