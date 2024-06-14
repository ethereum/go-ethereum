class Padding {
  constructor (padding) {
    this.left = padding.left
    this.right = padding.right
  }
  length () {
    return this.left.length + this.right.length
  }
}

/**
@module padding
*/
module.exports = Padding
