/* eslint no-loop-func: 0 */
/**
 * Mnemonist SymSpell
 * ===================
 *
 * JavaScript implementation of the Symmetric Delete Spelling dictionary to
 * efficiently index & query expression based on edit distance.
 * Note that the current implementation target the v3.0 of the algorithm.
 *
 * [Reference]:
 * http://blog.faroo.com/2012/06/07/improved-edit-distance-based-spelling-correction/
 * https://github.com/wolfgarbe/symspell
 *
 * [Author]:
 * Wolf Garbe
 */
var forEach = require('obliterator/foreach');

/**
 * Constants.
 */
var DEFAULT_MAX_DISTANCE = 2,
    DEFAULT_VERBOSITY = 2;

var VERBOSITY = new Set([
  // Returns only the top suggestion
  0,
  // Returns suggestions with the smallest edit distance
  1,
  // Returns every suggestion (no early termination)
  2
]);

var VERBOSITY_EXPLANATIONS = {
  0: 'Returns only the top suggestion',
  1: 'Returns suggestions with the smallest edit distance',
  2: 'Returns every suggestion (no early termination)'
};

/**
 * Functions.
 */

/**
 * Function creating a dictionary item.
 *
 * @param  {number} [value] - An optional suggestion.
 * @return {object}         - The created item.
 */
function createDictionaryItem(value) {
  var suggestions = new Set();

  if (typeof value === 'number')
    suggestions.add(value);

  return {
    suggestions,
    count: 0
  };
}

/**
 * Function creating a suggestion item.
 *
 * @return {object} - The created item.
 */
function createSuggestionItem(term, distance, count) {
  return {
    term: term || '',
    distance: distance || 0,
    count: count || 0
  };
}

/**
 * Simplified edit function.
 *
 * @param {string} word      - Target word.
 * @param {number} distance  - Distance.
 * @param {number} max       - Max distance.
 * @param {Set}    [deletes] - Set mutated to store deletes.
 */
function edits(word, distance, max, deletes) {
  deletes = deletes || new Set();
  distance++;

  var deletedItem,
      l = word.length,
      i;

  if (l > 1) {
    for (i = 0; i < l; i++) {
      deletedItem = word.substring(0, i) + word.substring(i + 1);

      if (!deletes.has(deletedItem)) {
        deletes.add(deletedItem);

        if (distance < max)
          edits(deletedItem, distance, max, deletes);
      }
    }
  }

  return deletes;
}

/**
 * Function used to conditionally add suggestions.
 *
 * @param {array}  words       - Words list.
 * @param {number} verbosity   - Verbosity level.
 * @param {object} item        - The target item.
 * @param {string} suggestion  - The target suggestion.
 * @param {number} int         - Integer key of the word.
 * @param {object} deletedItem - Considered deleted item.
 * @param {SymSpell}
 */
function addLowestDistance(words, verbosity, item, suggestion, int, deletedItem) {
  var first = item.suggestions.values().next().value;

  if (verbosity < 2 &&
      item.suggestions.size > 0 &&
      words[first].length - deletedItem.length > suggestion.length - deletedItem.length) {
    item.suggestions = new Set();
    item.count = 0;
  }

  if (verbosity === 2 ||
      !item.suggestions.size ||
      words[first].length - deletedItem.length >= suggestion.length - deletedItem.length) {
    item.suggestions.add(int);
  }
}

/**
 * Custom Damerau-Levenshtein used by the algorithm.
 *
 * @param  {string} source - First string.
 * @param  {string} target - Second string.
 * @return {number}        - The distance.
 */
function damerauLevenshtein(source, target) {
  var m = source.length,
      n = target.length,
      H = [[]],
      INF = m + n,
      sd = new Map(),
      i,
      l,
      j;

  H[0][0] = INF;

  for (i = 0; i <= m; i++) {
    if (!H[i + 1])
      H[i + 1] = [];
    H[i + 1][1] = i;
    H[i + 1][0] = INF;
  }

  for (j = 0; j <= n; j++) {
    H[1][j + 1] = j;
    H[0][j + 1] = INF;
  }

  var st = source + target,
      letter;

  for (i = 0, l = st.length; i < l; i++) {
    letter = st[i];

    if (!sd.has(letter))
      sd.set(letter, 0);
  }

  // Iterating
  for (i = 1; i <= m; i++) {
    var DB = 0;

    for (j = 1; j <= n; j++) {
      var i1 = sd.get(target[j - 1]),
          j1 = DB;

      if (source[i - 1] === target[j - 1]) {
        H[i + 1][j + 1] = H[i][j];
        DB = j;
      }
      else {
        H[i + 1][j + 1] = Math.min(
          H[i][j],
          H[i + 1][j],
          H[i][j + 1]
        ) + 1;
      }

      H[i + 1][j + 1] = Math.min(
        H[i + 1][j + 1],
        H[i1][j1] + (i - i1 - 1) + 1 + (j - j1 - 1)
      );
    }

    sd.set(source[i - 1], i);
  }

  return H[m + 1][n + 1];
}

/**
 * Lookup function.
 *
 * @param  {object} dictionary  - A SymSpell dictionary.
 * @param  {array}  words       - Unique words list.
 * @param  {number} verbosity   - Verbosity level.
 * @param  {number} maxDistance - Maximum distance.
 * @param  {number} maxLength   - Maximum word length in the dictionary.
 * @param  {string} input       - Input string.
 * @return {array}              - The list of suggestions.
 */
function lookup(dictionary, words, verbosity, maxDistance, maxLength, input) {
  var length = input.length;

  if (length - maxDistance > maxLength)
    return [];

  var candidates = [input],
      candidateSet = new Set(),
      suggestionSet = new Set();

  var suggestions = [],
      candidate,
      item;

  // Exhausting every candidates
  while (candidates.length > 0) {
    candidate = candidates.shift();

    // Early termination
    if (
      verbosity < 2 &&
      suggestions.length > 0 &&
      length - candidate.length > suggestions[0].distance
    )
      break;

    item = dictionary[candidate];

    if (item !== undefined) {
      if (typeof item === 'number')
        item = createDictionaryItem(item);

      if (item.count > 0 && !suggestionSet.has(candidate)) {
        suggestionSet.add(candidate);

        var suggestItem = createSuggestionItem(
          candidate,
          length - candidate.length,
          item.count
        );

        suggestions.push(suggestItem);

        // Another early termination
        if (verbosity < 2 && length - candidate.length === 0)
          break;
      }

      // Iterating over the item's suggestions
      item.suggestions.forEach(index => {
        var suggestion = words[index];

        // Do we already have this suggestion?
        if (suggestionSet.has(suggestion))
          return;

        suggestionSet.add(suggestion);

        // Computing distance between candidate & suggestion
        var distance = 0;

        if (input !== suggestion) {
          if (suggestion.length === candidate.length) {
            distance = length - candidate.length;
          }
          else if (length === candidate.length) {
            distance = suggestion.length - candidate.length;
          }
          else {
            var ii = 0,
                jj = 0;

            var l = suggestion.length;

            while (
              ii < l &&
              ii < length &&
              suggestion[ii] === input[ii]
            ) {
              ii++;
            }

            while (
              jj < l - ii &&
              jj < length &&
              suggestion[l - jj - 1] === input[length - jj - 1]
            ) {
              jj++;
            }

            if (ii > 0 || jj > 0) {
              distance = damerauLevenshtein(
                suggestion.substr(ii, l - ii - jj),
                input.substr(ii, length - ii - jj)
              );
            }
            else {
              distance = damerauLevenshtein(suggestion, input);
            }
          }
        }

        // Removing suggestions of higher distance
        if (verbosity < 2 &&
            suggestions.length > 0 &&
            suggestions[0].distance > distance) {
          suggestions = [];
        }

        if (verbosity < 2 &&
            suggestions.length > 0 &&
            distance > suggestions[0].distance) {
          return;
        }

        if (distance <= maxDistance) {
          var target = dictionary[suggestion];

          if (target !== undefined) {
            suggestions.push(createSuggestionItem(
              suggestion,
              distance,
              target.count
            ));
          }
        }
      });
    }

    // Adding edits
    if (length - candidate.length < maxDistance) {

      if (verbosity < 2 &&
          suggestions.length > 0 &&
          length - candidate.length >= suggestions[0].distance)
        continue;

      for (var i = 0, l = candidate.length; i < l; i++) {
        var deletedItem = (
          candidate.substring(0, i) +
          candidate.substring(i + 1)
        );

        if (!candidateSet.has(deletedItem)) {
          candidateSet.add(deletedItem);
          candidates.push(deletedItem);
        }
      }
    }
  }

  if (verbosity === 0)
    return suggestions.slice(0, 1);

  return suggestions;
}

/**
 * SymSpell.
 *
 * @constructor
 */
function SymSpell(options) {
  options = options || {};

  this.clear();

  // Properties
  this.maxDistance = typeof options.maxDistance === 'number' ?
    options.maxDistance :
    DEFAULT_MAX_DISTANCE;
  this.verbosity = typeof options.verbosity === 'number' ?
    options.verbosity :
    DEFAULT_VERBOSITY;

  // Sanity checks
  if (typeof this.maxDistance !== 'number' || this.maxDistance <= 0)
    throw Error('mnemonist/SymSpell.constructor: invalid `maxDistance` option. Should be a integer greater than 0.');

  if (!VERBOSITY.has(this.verbosity))
    throw Error('mnemonist/SymSpell.constructor: invalid `verbosity` option. Should be either 0, 1 or 2.');
}

/**
 * Method used to clear the structure.
 *
 * @return {undefined}
 */
SymSpell.prototype.clear = function() {

  // Properties
  this.size = 0;
  this.dictionary = Object.create(null);
  this.maxLength = 0;
  this.words = [];
};

/**
 * Method used to add a word to the index.
 *
 * @param {string} word - Word to add.
 * @param {SymSpell}
 */
SymSpell.prototype.add = function(word) {
  var item = this.dictionary[word];

  if (item !== undefined) {
    if (typeof item === 'number') {
      item = createDictionaryItem(item);
      this.dictionary[word] = item;
    }

    item.count++;
  }

  else {
    item = createDictionaryItem();
    item.count++;

    this.dictionary[word] = item;

    if (word.length > this.maxLength)
      this.maxLength = word.length;
  }

  if (item.count === 1) {
    var number = this.words.length;
    this.words.push(word);

    var deletes = edits(word, 0, this.maxDistance);

    deletes.forEach(deletedItem => {
      var target = this.dictionary[deletedItem];

      if (target !== undefined) {
        if (typeof target === 'number') {
          target = createDictionaryItem(target);

          this.dictionary[deletedItem] = target;
        }

        if (!target.suggestions.has(number)) {
          addLowestDistance(
            this.words,
            this.verbosity,
            target,
            word,
            number,
            deletedItem
          );
        }
      }
      else {
        this.dictionary[deletedItem] = number;
      }
    });
  }

  this.size++;

  return this;
};

/**
 * Method used to search the index.
 *
 * @param  {string} input - Input query.
 * @return {array}        - The found suggestions.
 */
SymSpell.prototype.search = function(input) {
  return lookup(
    this.dictionary,
    this.words,
    this.verbosity,
    this.maxDistance,
    this.maxLength,
    input
  );
};

/**
 * Convenience known methods.
 */
SymSpell.prototype.inspect = function() {
  var array = [];

  array.size = this.size;
  array.maxDistance = this.maxDistance;
  array.verbosity = this.verbosity;
  array.behavior = VERBOSITY_EXPLANATIONS[this.verbosity];

  for (var k in this.dictionary) {
    if (typeof this.dictionary[k] === 'object' && this.dictionary[k].count)
      array.push([k, this.dictionary[k].count]);
  }

  // Trick so that node displays the name of the constructor
  Object.defineProperty(array, 'constructor', {
    value: SymSpell,
    enumerable: false
  });

  return array;
};

if (typeof Symbol !== 'undefined')
  SymSpell.prototype[Symbol.for('nodejs.util.inspect.custom')] = SymSpell.prototype.inspect;

/**
 * Static @.from function taking an arbitrary iterable & converting it into
 * a structure.
 *
 * @param  {Iterable} iterable - Target iterable.
 * @return {SymSpell}
 */
SymSpell.from = function(iterable, options) {
  var index = new SymSpell(options);

  forEach(iterable, function(value) {
    index.add(value);
  });

  return index;
};

/**
 * Exporting.
 */
module.exports = SymSpell;
