const test = require('tape')
const os = require('os')
const concat = require('concat-stream')
const ndj = require('../')

test('.parse', function(t) {
  const parser = ndj.parse()
  parser.on('data', function(obj) {
    t.equal(obj.hello, 'world')
    t.end()
  })

  parser.write('{"hello": "world"}\n')
})

test('.parse twice', function(t) {
  const parser = ndj.parse()
  parser.once('data', function(obj) {
    t.equal(obj.hello, 'world')
    parser.once('data', function(obj) {
      t.equal(obj.hola, 'mundo')
      t.end()
    })
  })

  parser.write('{"hello": "world"}\n{"hola": "mundo"}\n')
})

test('.parse - strict:true error', function (t) {
  const parser = ndj.parse({strict: true})
  try {
    parser.write('{"no":"json"\n')
  } catch(e) {
    t.pass('error thrown')
    t.end()
  }
})

test('.parse - strict:true error event', function (t) {
  const parser = ndj.parse({strict: true})
  parser.on('error', function (err) {
    t.pass('error event called')
    t.end()
  })
  try {
    parser.write('{"no":"json"\n')
  } catch(e) {
    t.fail('should not throw')
  }
})

test('.parse - strict:false error', function (t) {
  const parser = ndj.parse({strict: false})
  parser.once('data', function (data) {
    t.ok(data.json, 'parse second one')
    t.end()
  })
  try {
    parser.write('{"json":false\n{"json":true}\n')
  } catch(e) {
    t.fail('should not call an error')
  }
})

test('.stringify', function(t) {
  const serializer = ndj.stringify()
  serializer.pipe(concat(function(data) {
    t.equal(data, '{"hello":"world"}' + os.EOL)
    t.end()
  }))
  serializer.write({hello: 'world'})
  serializer.end()
})

test('.stringify circular', function(t) {
  const serializer = ndj.stringify()
  serializer.pipe(concat(function(data) {
    t.equal(data, '{"obj":"[Circular ~]"}' + os.EOL)
    t.end()
  }))
  const obj = {}
  obj.obj = obj
  serializer.write(obj)
  serializer.end()
})
