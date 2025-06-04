/**
 * Mnemonist Set
 * ==============
 *
 * Useful function related to sets such as union, intersection and so on...
 */

// TODO: optimize versions for less variadicities

/**
 * Variadic function computing the intersection of multiple sets.
 *
 * @param  {...Set} sets - Sets to intersect.
 * @return {Set}         - The intesection.
 */
exports.intersection = function() {
  if (arguments.length < 2)
    throw new Error('mnemonist/Set.intersection: needs at least two arguments.');

  var I = new Set();

  // First we need to find the smallest set
  var smallestSize = Infinity,
      smallestSet = null;

  var s, i, l = arguments.length;

  for (i = 0; i < l; i++) {
    s = arguments[i];

    // If one of the set has no items, we can stop right there
    if (s.size === 0)
      return I;

    if (s.size < smallestSize) {
      smallestSize = s.size;
      smallestSet = s;
    }
  }

  // Now we need to intersect this set with the others
  var iterator = smallestSet.values(),
      step,
      item,
      add,
      set;

  // TODO: we can optimize by iterating each next time over the current intersection
  // but this probably means more RAM to consume since we'll create n-1 sets rather than
  // only the one.
  while ((step = iterator.next(), !step.done)) {
    item = step.value;
    add = true;

    for (i = 0; i < l; i++) {
      set = arguments[i];

      if (set === smallestSet)
        continue;

      if (!set.has(item)) {
        add = false;
        break;
      }
    }

    if (add)
      I.add(item);
  }

  return I;
};

/**
 * Variadic function computing the union of multiple sets.
 *
 * @param  {...Set} sets - Sets to unite.
 * @return {Set}         - The union.
 */
exports.union = function() {
  if (arguments.length < 2)
    throw new Error('mnemonist/Set.union: needs at least two arguments.');

  var U = new Set();

  var i, l = arguments.length;

  var iterator,
      step;

  for (i = 0; i < l; i++) {
    iterator = arguments[i].values();

    while ((step = iterator.next(), !step.done))
      U.add(step.value);
  }

  return U;
};

/**
 * Function computing the difference between two sets.
 *
 * @param  {Set} A - First set.
 * @param  {Set} B - Second set.
 * @return {Set}   - The difference.
 */
exports.difference = function(A, B) {

  // If first set is empty
  if (!A.size)
    return new Set();

  if (!B.size)
    return new Set(A);

  var D = new Set();

  var iterator = A.values(),
      step;

  while ((step = iterator.next(), !step.done)) {
    if (!B.has(step.value))
      D.add(step.value);
  }

  return D;
};

/**
 * Function computing the symmetric difference between two sets.
 *
 * @param  {Set} A - First set.
 * @param  {Set} B - Second set.
 * @return {Set}   - The symmetric difference.
 */
exports.symmetricDifference = function(A, B) {
  var S = new Set();

  var iterator = A.values(),
      step;

  while ((step = iterator.next(), !step.done)) {
    if (!B.has(step.value))
      S.add(step.value);
  }

  iterator = B.values();

  while ((step = iterator.next(), !step.done)) {
    if (!A.has(step.value))
      S.add(step.value);
  }

  return S;
};

/**
 * Function returning whether A is a subset of B.
 *
 * @param  {Set} A - First set.
 * @param  {Set} B - Second set.
 * @return {boolean}
 */
exports.isSubset = function(A, B) {
  var iterator = A.values(),
      step;

  // Shortcuts
  if (A === B)
    return true;

  if (A.size > B.size)
    return false;

  while ((step = iterator.next(), !step.done)) {
    if (!B.has(step.value))
      return false;
  }

  return true;
};

/**
 * Function returning whether A is a superset of B.
 *
 * @param  {Set} A - First set.
 * @param  {Set} B - Second set.
 * @return {boolean}
 */
exports.isSuperset = function(A, B) {
  return exports.isSubset(B, A);
};

/**
 * Function adding the items of set B to the set A.
 *
 * @param  {Set} A - First set.
 * @param  {Set} B - Second set.
 */
exports.add = function(A, B) {
  var iterator = B.values(),
      step;

  while ((step = iterator.next(), !step.done))
    A.add(step.value);

  return;
};

/**
 * Function subtracting the items of set B from the set A.
 *
 * @param  {Set} A - First set.
 * @param  {Set} B - Second set.
 */
exports.subtract = function(A, B) {
  var iterator = B.values(),
      step;

  while ((step = iterator.next(), !step.done))
    A.delete(step.value);

  return;
};

/**
 * Function intersecting the items of A & B.
 *
 * @param  {Set} A - First set.
 * @param  {Set} B - Second set.
 */
exports.intersect = function(A, B) {
  var iterator = A.values(),
      step;

  while ((step = iterator.next(), !step.done)) {
    if (!B.has(step.value))
      A.delete(step.value);
  }

  return;
};

/**
 * Function disjuncting the items of A & B.
 *
 * @param  {Set} A - First set.
 * @param  {Set} B - Second set.
 */
exports.disjunct = function(A, B) {
  var iterator = A.values(),
      step;

  var toRemove = [];

  while ((step = iterator.next(), !step.done)) {
    if (B.has(step.value))
      toRemove.push(step.value);
  }

  iterator = B.values();

  while ((step = iterator.next(), !step.done)) {
    if (!A.has(step.value))
      A.add(step.value);
  }

  for (var i = 0, l = toRemove.length; i < l; i++)
    A.delete(toRemove[i]);

  return;
};

/**
 * Function returning the size of the intersection of A & B.
 *
 * @param  {Set} A - First set.
 * @param  {Set} B - Second set.
 * @return {number}
 */
exports.intersectionSize = function(A, B) {
  var tmp;

  // We need to know the smallest set
  if (A.size > B.size) {
    tmp = A;
    A = B;
    B = tmp;
  }

  if (A.size === 0)
    return 0;

  if (A === B)
    return A.size;

  var iterator = A.values(),
      step;

  var I = 0;

  while ((step = iterator.next(), !step.done)) {
    if (B.has(step.value))
      I++;
  }

  return I;
};

/**
 * Function returning the size of the union of A & B.
 *
 * @param  {Set} A - First set.
 * @param  {Set} B - Second set.
 * @return {number}
 */
exports.unionSize = function(A, B) {
  var I = exports.intersectionSize(A, B);

  return A.size + B.size - I;
};

/**
 * Function returning the Jaccard similarity between A & B.
 *
 * @param  {Set} A - First set.
 * @param  {Set} B - Second set.
 * @return {number}
 */
exports.jaccard = function(A, B) {
  var I = exports.intersectionSize(A, B);

  if (I === 0)
    return 0;

  var U = A.size + B.size - I;

  return I / U;
};

/**
 * Function returning the overlap coefficient between A & B.
 *
 * @param  {Set} A - First set.
 * @param  {Set} B - Second set.
 * @return {number}
 */
exports.overlap = function(A, B) {
  var I = exports.intersectionSize(A, B);

  if (I === 0)
    return 0;

  return I / Math.min(A.size, B.size);
};
