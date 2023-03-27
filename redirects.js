const redirects = require('./redirects.json')

const redirectsPermanent = redirects.map((redirect) => {
  return {
    ...redirect,
    "permanent": true
  }
})

module.exports = {
  redirects: redirectsPermanent
};
