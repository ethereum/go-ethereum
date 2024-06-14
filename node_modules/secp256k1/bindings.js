const addon = require('node-gyp-build')(__dirname)
module.exports = require('./lib')(new addon.Secp256k1())
