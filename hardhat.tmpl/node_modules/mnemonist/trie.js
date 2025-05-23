/**
 * Mnemonist Trie
 * ===============
 *
 * JavaScript Trie implementation based upon plain objects. As such this
 * structure is more a convenience building upon the trie's advantages than
 * a real performant alternative to already existing structures.
 *
 * Note that the Trie is based upon the TrieMap since the underlying machine
 * is the very same. The Trie just does not let you set values and only
 * considers the existence of the given prefixes.
 */
var forEach = require('obliterator/foreach'),
    TrieMap = require('./trie-map.js');

/**
 * Constants.
 */
var SENTINEL = String.fromCharCode(0);

/**
 * Trie.
 *
 * @constructor
 */
function Trie(Token) {
  this.mode = Token === Array ? 'array' : 'string';
  this.clear();
}

// Re-using TrieMap's prototype
for (var methodName in TrieMap.prototype)
  Trie.prototype[methodName] = TrieMap.prototype[methodName];

// Dropping irrelevant methods
delete Trie.prototype.set;
delete Trie.prototype.get;
delete Trie.prototype.values;
delete Trie.prototype.entries;

/**
 * Method used to add the given prefix to the trie.
 *
 * @param  {string|array} prefix - Prefix to follow.
 * @return {TrieMap}
 */
Trie.prototype.add = function(prefix) {
  var node = this.root,
      token;

  for (var i = 0, l = prefix.length; i < l; i++) {
    token = prefix[i];

    node = node[token] || (node[token] = {});
  }

  // Do we need to increase size?
  if (!(SENTINEL in node))
    this.size++;

  node[SENTINEL] = true;

  return this;
};

/**
 * Method used to retrieve every item in the trie with the given prefix.
 *
 * @param  {string|array} prefix - Prefix to query.
 * @return {array}
 */
Trie.prototype.find = function(prefix) {
  var isString = typeof prefix === 'string';

  var node = this.root,
      matches = [],
      token,
      i,
      l;

  for (i = 0, l = prefix.length; i < l; i++) {
    token = prefix[i];
    node = node[token];

    if (typeof node === 'undefined')
      return matches;
  }

  // Performing DFS from prefix
  var nodeStack = [node],
      prefixStack = [prefix],
      k;

  while (nodeStack.length) {
    prefix = prefixStack.pop();
    node = nodeStack.pop();

    for (k in node) {
      if (k === SENTINEL) {
        matches.push(prefix);
        continue;
      }

      nodeStack.push(node[k]);
      prefixStack.push(isString ? prefix + k : prefix.concat(k));
    }
  }

  return matches;
};

/**
 * Attaching the #.keys method to Symbol.iterator if possible.
 */
if (typeof Symbol !== 'undefined')
  Trie.prototype[Symbol.iterator] = Trie.prototype.keys;

/**
 * Convenience known methods.
 */
Trie.prototype.inspect = function() {
  var proxy = new Set();

  var iterator = this.keys(),
      step;

  while ((step = iterator.next(), !step.done))
    proxy.add(step.value);

  // Trick so that node displays the name of the constructor
  Object.defineProperty(proxy, 'constructor', {
    value: Trie,
    enumerable: false
  });

  return proxy;
};

if (typeof Symbol !== 'undefined')
  Trie.prototype[Symbol.for('nodejs.util.inspect.custom')] = Trie.prototype.inspect;

Trie.prototype.toJSON = function() {
  return this.root;
};

/**
 * Static @.from function taking an arbitrary iterable & converting it into
 * a trie.
 *
 * @param  {Iterable} iterable   - Target iterable.
 * @return {Trie}
 */
Trie.from = function(iterable) {
  var trie = new Trie();

  forEach(iterable, function(value) {
    trie.add(value);
  });

  return trie;
};

/**
 * Exporting.
 */
Trie.SENTINEL = SENTINEL;
module.exports = Trie;
