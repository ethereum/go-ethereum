const ansiEscapeSequence = /\u001b.*?m/g

/**
 * @module ansi
 */
exports.remove = remove
exports.has = has

function remove (input) {
  return input.replace(ansiEscapeSequence, '')
}

function has (input) {
  return ansiEscapeSequence.test(input)
}
