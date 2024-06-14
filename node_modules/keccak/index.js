try {
  module.exports = require('./bindings')
} catch (err) {
  module.exports = require('./js')
}
