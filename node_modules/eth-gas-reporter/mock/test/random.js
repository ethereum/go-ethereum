const randomstring = require("randomstring");

function getRandomInt(max) {
  return Math.floor(Math.random() * Math.floor(max));
}

function random() {
  return randomstring.generate(getRandomInt(50));
}

module.exports = random;
