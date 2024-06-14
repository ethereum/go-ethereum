// Test which will run in nodejs
// $ node test.js
// (Might work with other CommonJS-compatible environments)
const assert = require('assert');
const LRUMap = require('./lru').LRUMap;
const asserteq = assert.equal;
const tests = {

['set and get']() {
  let c = new LRUMap(4);
  asserteq(c.size, 0);
  asserteq(c.limit, 4);
  asserteq(c.oldest, undefined);
  asserteq(c.newest, undefined);

  c.set('adam',   29)
   .set('john',   26)
   .set('angela', 24)
   .set('bob',    48);
  asserteq(c.toString(), 'adam:29 < john:26 < angela:24 < bob:48');
  asserteq(c.size, 4);

  asserteq(c.get('adam'), 29);
  asserteq(c.get('john'), 26);
  asserteq(c.get('angela'), 24);
  asserteq(c.get('bob'), 48);
  asserteq(c.toString(), 'adam:29 < john:26 < angela:24 < bob:48');

  asserteq(c.get('angela'), 24);
  asserteq(c.toString(), 'adam:29 < john:26 < bob:48 < angela:24');

  c.set('ygwie', 81);
  asserteq(c.toString(), 'john:26 < bob:48 < angela:24 < ygwie:81');
  asserteq(c.size, 4);
  asserteq(c.get('adam'), undefined);

  c.set('john', 11);
  asserteq(c.toString(), 'bob:48 < angela:24 < ygwie:81 < john:11');
  asserteq(c.get('john'), 11);

  let expectedKeys = ['bob', 'angela', 'ygwie', 'john'];
  c.forEach(function(v, k) {
    //sys.sets(k+': '+v);
    asserteq(k, expectedKeys.shift());
  })

  // removing one item decrements size by one
  let currentSize = c.size;
  assert(c.delete('john') !== undefined);
  asserteq(currentSize - 1, c.size);
},

['construct with iterator']() {
  let verifyEntries = function(c) {
    asserteq(c.size, 4);
    asserteq(c.limit, 4);
    asserteq(c.oldest.key, 'adam');
    asserteq(c.newest.key, 'bob');
    asserteq(c.get('adam'), 29);
    asserteq(c.get('john'), 26);
    asserteq(c.get('angela'), 24);
    asserteq(c.get('bob'), 48);
  };

  // with explicit limit
  verifyEntries(new LRUMap(4, [
    ['adam',   29],
    ['john',   26],
    ['angela', 24],
    ['bob',    48],
  ]));

  // with inferred limit
  verifyEntries(new LRUMap([
    ['adam',   29],
    ['john',   26],
    ['angela', 24],
    ['bob',    48],
  ]));
},

assign() {
  let c = new LRUMap([
    ['adam',   29],
    ['john',   26],
    ['angela', 24],
    ['bob',    48],
  ]);

  let newEntries = [
    ['mimi',    1],
    ['patrick', 2],
    ['jane',    3],
    ['fred',    4],
  ];
  c.assign(newEntries);
  asserteq(c.size, 4);
  asserteq(c.limit, 4);
  asserteq(c.oldest.key, newEntries[0][0]);
  asserteq(c.newest.key, newEntries[newEntries.length-1][0]);
  let i = 0;
  c.forEach(function(v, k) {
    asserteq(k, newEntries[i][0]);
    asserteq(v, newEntries[i][1]);
    i++;
  });

  // assigning too many items should throw an exception
  assert.throws(() => {
    c.assign([
      ['adam',   29],
      ['john',   26],
      ['angela', 24],
      ['bob',    48],
      ['ken',    30],
    ]);
  }, /overflow/);

  // assigning less than limit should not affect limit but adjust size
  c.assign([
    ['adam',   29],
    ['john',   26],
    ['angela', 24],
  ]);
  asserteq(c.size, 3);
  asserteq(c.limit, 4);
},

delete() {
  let c = new LRUMap([
    ['adam',   29],
    ['john',   26],
    ['angela', 24],
    ['bob',    48],
  ]);
  c.delete('adam');
  asserteq(c.size, 3);
  c.delete('angela');
  asserteq(c.size, 2);
  c.delete('bob');
  asserteq(c.size, 1);
  c.delete('john');
  asserteq(c.size, 0);
  asserteq(c.oldest, undefined);
  asserteq(c.newest, undefined);
},

clear() {
  let c = new LRUMap(4);
  c.set('adam', 29);
  c.set('john', 26);
  asserteq(c.size, 2);
  c.clear();
  asserteq(c.size, 0);
  asserteq(c.oldest, undefined);
  asserteq(c.newest, undefined);
},

shift() {
  let c2 = new LRUMap(4);
  asserteq(c2.size, 0);
  c2.set('a', 1)
  c2.set('b', 2)
  c2.set('c', 3)
  asserteq(c2.size, 3);

  let e = c2.shift();
  asserteq(e[0], 'a');
  asserteq(e[1], 1);
  
  e = c2.shift();
  asserteq(e[0], 'b');
  asserteq(e[1], 2);
  
  e = c2.shift();
  asserteq(e[0], 'c');
  asserteq(e[1], 3);

  // c2 should be empty
  c2.forEach(function () { assert(false); });
  asserteq(c2.size, 0);
},

set() {
  // Note: v0.1 allows putting same key multiple times. v0.2 does not.
  c = new LRUMap(4);
  c.set('a', 1);
  c.set('a', 2);
  c.set('a', 3);
  c.set('a', 4);
  asserteq(c.size, 1);
  asserteq(c.newest, c.oldest);
  assert.deepEqual(c.newest, {key:'a', value:4 });

  c.set('a', 5);
  asserteq(c.size, 1);
  asserteq(c.newest, c.oldest);
  assert.deepEqual(c.newest, {key:'a', value:5 });

  c.set('b', 6);
  asserteq(c.size, 2);
  assert(c.newest !== c.oldest);

  assert.deepEqual(c.newest, { key:'b', value:6 });
  assert.deepEqual(c.oldest, { key:'a', value:5 });

  c.shift();
  asserteq(c.size, 1);
  c.shift();
  asserteq(c.size, 0);
  c.forEach(function(){ assert(false) });
},


['entry iterator']() {
  let c = new LRUMap(4, [
    ['adam',   29],
    ['john',   26],
    ['angela', 24],
    ['bob',    48],
  ]);

  let verifyEntries = function(iterable) {
    asserteq(typeof iterable[Symbol.iterator], 'function');
    let it = iterable[Symbol.iterator]();
    assert.deepEqual(it.next().value, ['adam',   29]);
    assert.deepEqual(it.next().value, ['john',   26]);
    assert.deepEqual(it.next().value, ['angela', 24]);
    assert.deepEqual(it.next().value, ['bob',    48]);
    assert(it.next().done);
  };

  verifyEntries(c);
  verifyEntries(c.entries());
},


['key iterator']() {
  let c = new LRUMap(4, [
    ['adam',   29],
    ['john',   26],
    ['angela', 24],
    ['bob',    48],
  ]);
  let kit = c.keys();
  asserteq(kit.next().value, 'adam');
  asserteq(kit.next().value, 'john');
  asserteq(kit.next().value, 'angela');
  asserteq(kit.next().value, 'bob');
  assert(kit.next().done);
},


['value iterator']() {
  let c = new LRUMap(4, [
    ['adam',   29],
    ['john',   26],
    ['angela', 24],
    ['bob',    48],
  ]);
  let kit = c.values();
  asserteq(kit.next().value, 29);
  asserteq(kit.next().value, 26);
  asserteq(kit.next().value, 24);
  asserteq(kit.next().value, 48);
  assert(kit.next().done);
},


toJSON() {
  let c = new LRUMap(4, [
    ['adam',   29],
    ['john',   26],
    ['angela', 24],
    ['bob',    48],
  ]);
  let json = c.toJSON();
  assert(json.length == 4);
  assert.deepEqual(json, [
    {key:'adam', value:29},
    {key:'john', value:26},
    {key:'angela', value:24},
    {key:'bob', value:48},
  ]);
},


}; // tests


function fmttime(t) {
  return (Math.round((t)*10)/10)+'ms';
}

function die(err) {
  console.error('\n' + (err.stack || err));
  process.exit(1);
}

function runNextTest(tests, testNames, allDoneCallback) {
  let testName = testNames[0];
  if (!testName) {
    return allDoneCallback();
  }
  process.stdout.write(testName+' ... ');
  let t1 = Date.now();
  let next = function() {
    t1 = Date.now() - t1;
    if (t1 > 10) {
      process.stdout.write('ok ('+fmttime(t1)+')\n');
    } else {
      process.stdout.write('ok\n');
    }
    runNextTest(tests, testNames.slice(1), allDoneCallback);
  };
  try {
    let p = tests[testName]();
    if (p && p instanceof Promise) {
      p.then(next).catch(die);
    } else {
      next();
    }
  } catch (err) {
    die(err);
  }
}

let t = Date.now();
runNextTest(tests, Object.keys(tests), function() {
  t = Date.now() - t;
  let timestr = '';
  if (t > 10) {
    timestr = '(' + fmttime(t) + ')';
  }
  console.log(`${Object.keys(tests).length} tests passed ${timestr}`);
});
