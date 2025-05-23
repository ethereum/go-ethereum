#!/usr/bin/env coffee

fs         = require('fs')
path       = require('path')
uglify     = require('uglify-js')
browserify = require('browserify')

BANNER = '''
/**
 * @fileoverview Text diff library ported from Python's difflib module. 
 *     https://github.com/qiao/difflib.js
 */

'''

build = (dest) ->
  browserified = browserify.bundle(__dirname + '/../lib/difflib.js')
  namespaced   = 'var difflib = (function() {' + browserified + 'return require("/difflib");})();'
  uglified     = uglify(namespaced)
  bannered     = BANNER + uglified
  fs.writeFileSync(dest, bannered)

build(__dirname + '/../dist/difflib-browser.js')
