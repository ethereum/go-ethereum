var ON_DEATH = require('./lib/death')({debug: true})

//to kill this, call `kill -9 pid`

process.stdin.resume()

ON_DEATH(function(err) {
  if (!err)
    console.log('Caught foo')
  else
    console.error('We got an uncaught exception! ' + err)
})

setTimeout(function() {
  throw new Error('stupid exception')
}, 5000)

