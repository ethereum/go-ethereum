
/**
 * Module dependencies.
 */

var global = (function() { return this; })(); // jshint ignore:line

/**
 * XMLHttpRequest constructor.
 */

var XMLHttpRequest = window.XMLHttpRequest; // jshint ignore:line

/**
 * Module exports.
 */

module.exports.XMLHttpRequest = XMLHttpRequest ? xhr : null;

/**
 * XMLHttpRequest constructor.
 *
 * @param {Object) opts (optional)
 * @api public
 */

function xhr(obj) {
  var instance;

  instance = new XMLHttpRequest(obj);

  return instance;
}

if (XMLHttpRequest) xhr.prototype = XMLHttpRequest.prototype;
