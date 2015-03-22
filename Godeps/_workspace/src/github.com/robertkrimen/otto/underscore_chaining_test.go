package otto

import (
	"testing"
)

// map/flatten/reduce
func Test_underscore_chaining_0(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test("map/flatten/reduce", function() {
    var lyrics = [
      "I'm a lumberjack and I'm okay",
      "I sleep all night and I work all day",
      "He's a lumberjack and he's okay",
      "He sleeps all night and he works all day"
    ];
    var counts = _(lyrics).chain()
      .map(function(line) { return line.split(''); })
      .flatten()
      .reduce(function(hash, l) {
        hash[l] = hash[l] || 0;
        hash[l]++;
        return hash;
    }, {}).value();
    ok(counts['a'] == 16 && counts['e'] == 10, 'counted all the letters in the song');
  });
        `)
	})
}

// select/reject/sortBy
func Test_underscore_chaining_1(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test("select/reject/sortBy", function() {
    var numbers = [1,2,3,4,5,6,7,8,9,10];
    numbers = _(numbers).chain().select(function(n) {
      return n % 2 == 0;
    }).reject(function(n) {
      return n % 4 == 0;
    }).sortBy(function(n) {
      return -n;
    }).value();
    equal(numbers.join(', '), "10, 6, 2", "filtered and reversed the numbers");
  });
        `)
	})
}

// select/reject/sortBy in functional style
func Test_underscore_chaining_2(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test("select/reject/sortBy in functional style", function() {
    var numbers = [1,2,3,4,5,6,7,8,9,10];
    numbers = _.chain(numbers).select(function(n) {
      return n % 2 == 0;
    }).reject(function(n) {
      return n % 4 == 0;
    }).sortBy(function(n) {
      return -n;
    }).value();
    equal(numbers.join(', '), "10, 6, 2", "filtered and reversed the numbers");
  });
        `)
	})
}

// reverse/concat/unshift/pop/map
func Test_underscore_chaining_3(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test("reverse/concat/unshift/pop/map", function() {
    var numbers = [1,2,3,4,5];
    numbers = _(numbers).chain()
      .reverse()
      .concat([5, 5, 5])
      .unshift(17)
      .pop()
      .map(function(n){ return n * 2; })
      .value();
    equal(numbers.join(', '), "34, 10, 8, 6, 4, 2, 10, 10", 'can chain together array functions.');
  });
        `)
	})
}
