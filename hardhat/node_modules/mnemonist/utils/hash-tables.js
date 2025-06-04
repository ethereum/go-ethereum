/* eslint no-constant-condition: 0 */
/**
 * Mnemonist Hashtable Helpers
 * ============================
 *
 * Miscellaneous helpers helper function dealing with hashtables.
 */
function jenkinsInt32(a) {

  a = (a + 0x7ed55d16) + (a << 12);
  a = (a ^ 0xc761c23c) ^ (a >> 19);
  a = (a + 0x165667b1) + (a << 5);
  a = (a + 0xd3a2646c) ^ (a << 9);
  a = (a + 0xfd7046c5) + (a << 3);
  a = (a ^ 0xb55a4f09) ^ (a >> 16);

  return a;
}

function linearProbingGet(hash, keys, values, key) {
  var n = keys.length,
      j = hash(key) & (n - 1),
      i = j;

  var c;

  while (true) {
    c = keys[i];

    if (c === key)
      return values[i];

    else if (c === 0)
      return;

    // Handling wrapping around
    i += 1;
    i %= n;

    // Full turn
    if (i === j)
      return;
  }
}

function linearProbingHas(hash, keys, key) {
  var n = keys.length,
      j = hash(key) & (n - 1),
      i = j;

  var c;

  while (true) {
    c = keys[i];

    if (c === key)
      return true;

    else if (c === 0)
      return false;

    // Handling wrapping around
    i += 1;
    i %= n;

    // Full turn
    if (i === j)
      return false;
  }
}

function linearProbingSet(hash, keys, values, key, value) {
  var n = keys.length,
      j = hash(key) & (n - 1),
      i = j;

  var c;

  while (true) {
    c = keys[i];

    if (c === 0 || c === key)
      break;

    // Handling wrapping around
    i += 1;
    i %= n;

    // Full turn
    if (i === j)
      throw new Error('mnemonist/utils/hash-tables.linearProbingSet: table is full.');
  }

  keys[i] = key;
  values[i] = value;
}

module.exports = {
  hashes: {
    jenkinsInt32: jenkinsInt32
  },
  linearProbing: {
    get: linearProbingGet,
    has: linearProbingHas,
    set: linearProbingSet
  }
};
