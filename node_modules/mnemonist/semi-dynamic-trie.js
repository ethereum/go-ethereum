/* eslint no-constant-condition: 0 */
/**
 * Mnemonist SemiDynamicTrie
 * ==========================
 *
 * Lowlevel Trie working at character level, storing information in typed
 * array and organizing its children in linked lists.
 *
 * This implementation also uses a "fat node" strategy to boost access to some
 * bloated node's children when the number of children rises above a certain
 * threshold.
 */
var Vector = require('./vector.js');

// TODO: rename => ternary search tree

/**
 * Constants.
 */
const MAX_LINKED = 7;

/**
 * SemiDynamicTrie.
 *
 * @constructor
 */
function SemiDynamicTrie() {

  // Properties

  // TODO: make it 16 bits
  this.characters = new Vector.Uint8Vector(256);
  this.nextPointers = new Vector.Int32Vector(256);
  this.childPointers = new Vector.Uint32Vector(256);
  this.maps = new Vector.Uint32Vector(256);
}

/**
 * Method used to clear the structure.
 *
 * @return {undefined}
 */
SemiDynamicTrie.prototype.clear = function() {

  // Properties
};

SemiDynamicTrie.prototype.ensureSibling = function(block, character) {
  var nextCharacter,
      nextBlock,
      newBlock;

  // Do we have a root?
  if (this.characters.length === 0) {

    this.nextPointers.push(0);
    this.childPointers.push(0);
    this.characters.push(character);

    return block;
  }

  // Are we traversing a fat node?
  var fatNode = this.nextPointers.array[block];

  if (fatNode < 0) {
    var mapIndex = -fatNode + character;

    nextBlock = this.maps.array[mapIndex];

    if (nextBlock !== 0)
      return nextBlock;

    newBlock = this.characters.length;

    this.nextPointers.push(0);
    this.childPointers.push(0);
    this.characters.push(character);

    this.maps.set(mapIndex, newBlock);

    return newBlock;
  }

  var listLength = 1,
      startingBlock = block;

  while (true) {
    nextCharacter = this.characters.array[block];

    if (nextCharacter === character)
      return block;

    nextBlock = this.nextPointers.array[block];

    if (nextBlock === 0)
      break;

    listLength++;
    block = nextBlock;
  }

  // If the list is too long, we create a fat node
  if (listLength > MAX_LINKED) {
    block = startingBlock;

    var offset = this.maps.length;

    this.maps.resize(offset + 255);
    this.maps.set(offset + 255, 0);

    while (true) {
      nextBlock = this.nextPointers.array[block];

      if (nextBlock === 0)
        break;

      nextCharacter = this.characters.array[nextBlock];
      this.maps.set(offset + nextCharacter, nextBlock);

      block = nextBlock;
    }

    this.nextPointers.set(startingBlock, -offset);

    newBlock = this.characters.length;

    this.nextPointers.push(0);
    this.childPointers.push(0);
    this.characters.push(character);

    this.maps.set(offset + character, newBlock);

    return newBlock;
  }

  // Else, we append the character to the list
  newBlock = this.characters.length;

  this.nextPointers.push(0);
  this.childPointers.push(0);
  this.nextPointers.set(block, newBlock);
  this.characters.push(character);

  return newBlock;
};

SemiDynamicTrie.prototype.findSibling = function(block, character) {
  var nextCharacter;

  // Do we have a fat node?
  var fatNode = this.nextPointers.array[block];

  if (fatNode < 0) {
    var mapIndex = -fatNode + character;

    var nextBlock = this.maps.array[mapIndex];

    if (nextBlock === 0)
      return -1;

    return nextBlock;
  }

  while (true) {
    nextCharacter = this.characters.array[block];

    if (nextCharacter === character)
      return block;

    block = this.nextPointers.array[block];

    if (block === 0)
      return -1;
  }
};

SemiDynamicTrie.prototype.add = function(key) {
  var keyCharacter,
      childBlock,
      block = 0;

  var i = 0, l = key.length;

  // Going as far as possible
  while (i < l) {
    keyCharacter = key.charCodeAt(i);

    // Ensuring a correct sibling exists
    block = this.ensureSibling(block, keyCharacter);

    i++;

    if (i < l) {

      // Descending
      childBlock = this.childPointers.array[block];

      if (childBlock === 0)
        break;

      block = childBlock;
    }
  }

  // Adding as many blocks as necessary
  while (i < l) {

    childBlock = this.characters.length;
    this.characters.push(key.charCodeAt(i));

    this.childPointers.push(0);
    this.nextPointers.push(0);
    this.childPointers.set(block, childBlock);

    block = childBlock;

    i++;
  }
};

SemiDynamicTrie.prototype.has = function(key) {
  var i, l;

  var block = 0,
      siblingBlock;

  for (i = 0, l = key.length; i < l; i++) {
    siblingBlock = this.findSibling(block, key.charCodeAt(i));

    if (siblingBlock === -1)
      return false;

    // TODO: be sure
    if (i === l - 1)
      return true;

    block = this.childPointers.array[siblingBlock];

    if (block === 0)
      return false;
  }

  // TODO: fix, should have a leaf pointer somehow
  return true;
};

/**
 * Exporting.
 */
module.exports = SemiDynamicTrie;
