assert = require('assert')
RDB    = require('./build/Release/rdb').DBWrapper
exec   = require('child_process').exec
util   = require('util')

DB_NAME = '/tmp/rocksdbtest-' + process.getuid()

a = RDB()
assert.equal(a.open(DB_NAME, ['b']), false)

exec(
    util.format(
        "node -e \"RDB = require('./build/Release/rdb').DBWrapper; \
        a = RDB('%s'); a.createColumnFamily('b')\"",
        DB_NAME
    ).exitCode, null
)


exec(
    util.format(
        "node -e \"RDB = require('./build/Release/rdb').DBWrapper; \
        a = RDB('%s', ['b'])\"",
        DB_NAME
    ).exitCode, null
)

exec('rm -rf ' + DB_NAME)

a = RDB()
assert.equal(a.open(DB_NAME, ['a']), false)
assert(a.open(DB_NAME), true)
assert(a.createColumnFamily('temp'))

b = RDB()
assert.equal(b.open(DB_NAME), false)

exec('rm -rf ' + DB_NAME)

DB_NAME += 'b'

a = RDB()
assert(a.open(DB_NAME))
assert.equal(a.constructor.name, 'DBWrapper')
assert.equal(a.createColumnFamily(), false)
assert.equal(a.createColumnFamily(1), false)
assert.equal(a.createColumnFamily(['']), false)
assert(a.createColumnFamily('b'))
assert.equal(a.createColumnFamily('b'), false)

// Get and Put
assert.equal(a.get(1), null)
assert.equal(a.get(['a']), null)
assert.equal(a.get('a', 1), null)
assert.equal(a.get(1, 'a'), null)
assert.equal(a.get(1, 1), null)

assert.equal(a.put(1), false)
assert.equal(a.put(['a']), false)
assert.equal(a.put('a', 1), false)
assert.equal(a.put(1, 'a'), false)
assert.equal(a.put(1, 1), false)
assert.equal(a.put('a', 'a', 1), false)
assert.equal(a.put('a', 1, 'a'), false)
assert.equal(a.put(1, 'a', 'a'), false)
assert.equal(a.put('a', 1, 1), false)
assert.equal(a.put(1, 'a', 1), false)
assert.equal(a.put(1, 1, 'a'), false)
assert.equal(a.put(1, 1, 1), false)


assert.equal(a.get(), null)
assert.equal(a.get('a'), null)
assert.equal(a.get('a', 'c'), null)
assert.equal(a.put(), false)
assert.equal(a.put('a'), false)
assert.equal(a.get('a', 'b', 'c'), null)

assert(a.put('a', 'axe'))
assert(a.put('a', 'first'))
assert.equal(a.get('a'), 'first')
assert.equal(a.get('a', 'b'), null)
assert.equal(a.get('a', 'c'), null)

assert(a.put('a', 'apple', 'b'))
assert.equal(a.get('a', 'b'), 'apple')
assert.equal(a.get('a'), 'first')
assert(a.put('b', 'butter', 'b'), 'butter')
assert(a.put('b', 'banana', 'b'))
assert.equal(a.get('b', 'b'), 'banana')
assert.equal(a.get('b'), null)
assert.equal(a.get('b', 'c'), null)

// Delete
assert.equal(a.delete(1), false)
assert.equal(a.delete('a', 1), false)
assert.equal(a.delete(1, 'a'), false)
assert.equal(a.delete(1, 1), false)

assert.equal(a.delete('b'), true)
assert(a.delete('a'))
assert.equal(a.get('a'), null)
assert.equal(a.get('a', 'b'), 'apple')
assert.equal(a.delete('c', 'c'), false)
assert.equal(a.delete('c', 'b'), true)
assert(a.delete('b', 'b'))
assert.equal(a.get('b', 'b'), null)

// Dump
console.log("MARKER 1")
assert(a.dump())
console.log("Should be no output between 'MARKER 1' and here\n")
console.log('Next line should be "a" => "apple"')
assert(a.dump('b'))

console.log("\nMARKER 2")
assert.equal(a.dump('c'), false)
console.log("Should be no output between 'MARKER 2' and here\n")

// WriteBatch


// Clean up test database
exec('rm -rf ' + DB_NAME)
