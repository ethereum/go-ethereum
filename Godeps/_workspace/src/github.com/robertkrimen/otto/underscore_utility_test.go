package otto

import (
	"testing"
)

// #750 - Return _ instance.
func Test_underscore_utility_0(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test("#750 - Return _ instance.", 2, function() {
    var instance = _([]);
    ok(_(instance) === instance);
    ok(new _(instance) === instance);
  });
        `)
	})
}

// identity
func Test_underscore_utility_1(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test("identity", function() {
    var moe = {name : 'moe'};
    equal(_.identity(moe), moe, 'moe is the same as his identity');
  });
        `)
	})
}

// random
func Test_underscore_utility_2(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test("random", function() {
    var array = _.range(1000);
    var min = Math.pow(2, 31);
    var max = Math.pow(2, 62);

    ok(_.every(array, function() {
      return _.random(min, max) >= min;
    }), "should produce a random number greater than or equal to the minimum number");

    ok(_.some(array, function() {
      return _.random(Number.MAX_VALUE) > 0;
    }), "should produce a random number when passed <Number.MAX_VALUE>");
  });
        `)
	})
}

// uniqueId
func Test_underscore_utility_3(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test("uniqueId", function() {
    var ids = [], i = 0;
    while(i++ < 100) ids.push(_.uniqueId());
    equal(_.uniq(ids).length, ids.length, 'can generate a globally-unique stream of ids');
  });
        `)
	})
}

// times
func Test_underscore_utility_4(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test("times", function() {
    var vals = [];
    _.times(3, function (i) { vals.push(i); });
    ok(_.isEqual(vals, [0,1,2]), "is 0 indexed");
    //
    vals = [];
    _(3).times(function(i) { vals.push(i); });
    ok(_.isEqual(vals, [0,1,2]), "works as a wrapper");
    // collects return values
    ok(_.isEqual([0, 1, 2], _.times(3, function(i) { return i; })), "collects return values");
  });
        `)
	})
}

// mixin
func Test_underscore_utility_5(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test("mixin", function() {
    _.mixin({
      myReverse: function(string) {
        return string.split('').reverse().join('');
      }
    });
    equal(_.myReverse('panacea'), 'aecanap', 'mixed in a function to _');
    equal(_('champ').myReverse(), 'pmahc', 'mixed in a function to the OOP wrapper');
  });
        `)
	})
}

// _.escape
func Test_underscore_utility_6(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test("_.escape", function() {
    equal(_.escape("Curly & Moe"), "Curly &amp; Moe");
    equal(_.escape("Curly &amp; Moe"), "Curly &amp;amp; Moe");
    equal(_.escape(null), '');
  });
        `)
	})
}

// _.unescape
func Test_underscore_utility_7(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test("_.unescape", function() {
    var string = "Curly & Moe";
    equal(_.unescape("Curly &amp; Moe"), string);
    equal(_.unescape("Curly &amp;amp; Moe"), "Curly &amp; Moe");
    equal(_.unescape(null), '');
    equal(_.unescape(_.escape(string)), string);
  });
        `)
	})
}

// template
func Test_underscore_utility_8(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test("template", function() {
    var basicTemplate = _.template("<%= thing %> is gettin' on my noives!");
    var result = basicTemplate({thing : 'This'});
    equal(result, "This is gettin' on my noives!", 'can do basic attribute interpolation');

    var sansSemicolonTemplate = _.template("A <% this %> B");
    equal(sansSemicolonTemplate(), "A  B");

    var backslashTemplate = _.template("<%= thing %> is \\ridanculous");
    equal(backslashTemplate({thing: 'This'}), "This is \\ridanculous");

    var escapeTemplate = _.template('<%= a ? "checked=\\"checked\\"" : "" %>');
    equal(escapeTemplate({a: true}), 'checked="checked"', 'can handle slash escapes in interpolations.');

    var fancyTemplate = _.template("<ul><% \
      for (var key in people) { \
    %><li><%= people[key] %></li><% } %></ul>");
    result = fancyTemplate({people : {moe : "Moe", larry : "Larry", curly : "Curly"}});
    equal(result, "<ul><li>Moe</li><li>Larry</li><li>Curly</li></ul>", 'can run arbitrary javascript in templates');

    var escapedCharsInJavascriptTemplate = _.template("<ul><% _.each(numbers.split('\\n'), function(item) { %><li><%= item %></li><% }) %></ul>");
    result = escapedCharsInJavascriptTemplate({numbers: "one\ntwo\nthree\nfour"});
    equal(result, "<ul><li>one</li><li>two</li><li>three</li><li>four</li></ul>", 'Can use escaped characters (e.g. \\n) in Javascript');

    var namespaceCollisionTemplate = _.template("<%= pageCount %> <%= thumbnails[pageCount] %> <% _.each(thumbnails, function(p) { %><div class=\"thumbnail\" rel=\"<%= p %>\"></div><% }); %>");
    result = namespaceCollisionTemplate({
      pageCount: 3,
      thumbnails: {
        1: "p1-thumbnail.gif",
        2: "p2-thumbnail.gif",
        3: "p3-thumbnail.gif"
      }
    });
    equal(result, "3 p3-thumbnail.gif <div class=\"thumbnail\" rel=\"p1-thumbnail.gif\"></div><div class=\"thumbnail\" rel=\"p2-thumbnail.gif\"></div><div class=\"thumbnail\" rel=\"p3-thumbnail.gif\"></div>");

    var noInterpolateTemplate = _.template("<div><p>Just some text. Hey, I know this is silly but it aids consistency.</p></div>");
    result = noInterpolateTemplate();
    equal(result, "<div><p>Just some text. Hey, I know this is silly but it aids consistency.</p></div>");

    var quoteTemplate = _.template("It's its, not it's");
    equal(quoteTemplate({}), "It's its, not it's");

    var quoteInStatementAndBody = _.template("<%\
      if(foo == 'bar'){ \
    %>Statement quotes and 'quotes'.<% } %>");
    equal(quoteInStatementAndBody({foo: "bar"}), "Statement quotes and 'quotes'.");

    var withNewlinesAndTabs = _.template('This\n\t\tis: <%= x %>.\n\tok.\nend.');
    equal(withNewlinesAndTabs({x: 'that'}), 'This\n\t\tis: that.\n\tok.\nend.');

    var template = _.template("<i><%- value %></i>");
    var result = template({value: "<script>"});
    equal(result, '<i>&lt;script&gt;</i>');

    var stooge = {
      name: "Moe",
      template: _.template("I'm <%= this.name %>")
    };
    equal(stooge.template(), "I'm Moe");

    // TEST: ReferenceError: $ is not defined
    if (false) {
        if (!$.browser.msie) {
        var fromHTML = _.template($('#template').html());
        equal(fromHTML({data : 12345}).replace(/\s/g, ''), '<li>24690</li>');
        }
    }

    _.templateSettings = {
      evaluate    : /\{\{([\s\S]+?)\}\}/g,
      interpolate : /\{\{=([\s\S]+?)\}\}/g
    };

    var custom = _.template("<ul>{{ for (var key in people) { }}<li>{{= people[key] }}</li>{{ } }}</ul>");
    result = custom({people : {moe : "Moe", larry : "Larry", curly : "Curly"}});
    equal(result, "<ul><li>Moe</li><li>Larry</li><li>Curly</li></ul>", 'can run arbitrary javascript in templates');

    var customQuote = _.template("It's its, not it's");
    equal(customQuote({}), "It's its, not it's");

    var quoteInStatementAndBody = _.template("{{ if(foo == 'bar'){ }}Statement quotes and 'quotes'.{{ } }}");
    equal(quoteInStatementAndBody({foo: "bar"}), "Statement quotes and 'quotes'.");

    _.templateSettings = {
      evaluate    : /<\?([\s\S]+?)\?>/g,
      interpolate : /<\?=([\s\S]+?)\?>/g
    };

    var customWithSpecialChars = _.template("<ul><? for (var key in people) { ?><li><?= people[key] ?></li><? } ?></ul>");
    result = customWithSpecialChars({people : {moe : "Moe", larry : "Larry", curly : "Curly"}});
    equal(result, "<ul><li>Moe</li><li>Larry</li><li>Curly</li></ul>", 'can run arbitrary javascript in templates');

    var customWithSpecialCharsQuote = _.template("It's its, not it's");
    equal(customWithSpecialCharsQuote({}), "It's its, not it's");

    var quoteInStatementAndBody = _.template("<? if(foo == 'bar'){ ?>Statement quotes and 'quotes'.<? } ?>");
    equal(quoteInStatementAndBody({foo: "bar"}), "Statement quotes and 'quotes'.");

    _.templateSettings = {
      interpolate : /\{\{(.+?)\}\}/g
    };

    var mustache = _.template("Hello {{planet}}!");
    equal(mustache({planet : "World"}), "Hello World!", "can mimic mustache.js");

    var templateWithNull = _.template("a null undefined {{planet}}");
    equal(templateWithNull({planet : "world"}), "a null undefined world", "can handle missing escape and evaluate settings");
  });
        `)
	})
}

// _.template provides the generated function source, when a SyntaxError occurs
func Test_underscore_utility_9(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test('_.template provides the generated function source, when a SyntaxError occurs', function() {
    try {
      _.template('<b><%= if x %></b>');
    } catch (ex) {
      var source = ex.source;
    }
    ok(/__p/.test(source));
  });
        `)
	})
}

// _.template handles \\u2028 & \\u2029
func Test_underscore_utility_10(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test('_.template handles \\u2028 & \\u2029', function() {
    var tmpl = _.template('<p>\u2028<%= "\\u2028\\u2029" %>\u2029</p>');
    strictEqual(tmpl(), '<p>\u2028\u2028\u2029\u2029</p>');
  });
        `)
	})
}

// result calls functions and returns primitives
func Test_underscore_utility_11(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test('result calls functions and returns primitives', function() {
    var obj = {w: '', x: 'x', y: function(){ return this.x; }};
    strictEqual(_.result(obj, 'w'), '');
    strictEqual(_.result(obj, 'x'), 'x');
    strictEqual(_.result(obj, 'y'), 'x');
    strictEqual(_.result(obj, 'z'), undefined);
    strictEqual(_.result(null, 'x'), null);
  });
        `)
	})
}

// _.templateSettings.variable
func Test_underscore_utility_12(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test('_.templateSettings.variable', function() {
    var s = '<%=data.x%>';
    var data = {x: 'x'};
    strictEqual(_.template(s, data, {variable: 'data'}), 'x');
    _.templateSettings.variable = 'data';
    strictEqual(_.template(s)(data), 'x');
  });
        `)
	})
}

// #547 - _.templateSettings is unchanged by custom settings.
func Test_underscore_utility_13(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test('#547 - _.templateSettings is unchanged by custom settings.', function() {
    ok(!_.templateSettings.variable);
    _.template('', {}, {variable: 'x'});
    ok(!_.templateSettings.variable);
  });
        `)
	})
}

// #556 - undefined template variables.
func Test_underscore_utility_14(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test('#556 - undefined template variables.', function() {
    var template = _.template('<%=x%>');
    strictEqual(template({x: null}), '');
    strictEqual(template({x: undefined}), '');

    var templateEscaped = _.template('<%-x%>');
    strictEqual(templateEscaped({x: null}), '');
    strictEqual(templateEscaped({x: undefined}), '');

    var templateWithProperty = _.template('<%=x.foo%>');
    strictEqual(templateWithProperty({x: {} }), '');
    strictEqual(templateWithProperty({x: {} }), '');

    var templateWithPropertyEscaped = _.template('<%-x.foo%>');
    strictEqual(templateWithPropertyEscaped({x: {} }), '');
    strictEqual(templateWithPropertyEscaped({x: {} }), '');
  });
        `)
	})
}

// interpolate evaluates code only once.
func Test_underscore_utility_15(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test('interpolate evaluates code only once.', 2, function() {
    var count = 0;
    var template = _.template('<%= f() %>');
    template({f: function(){ ok(!(count++)); }});

    var countEscaped = 0;
    var templateEscaped = _.template('<%- f() %>');
    templateEscaped({f: function(){ ok(!(countEscaped++)); }});
  });
        `)
	})
}

// #746 - _.template settings are not modified.
func Test_underscore_utility_16(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test('#746 - _.template settings are not modified.', 1, function() {
    var settings = {};
    _.template('', null, settings);
    deepEqual(settings, {});
  });
        `)
	})
}

// #779 - delimeters are applied to unescaped text.
func Test_underscore_utility_17(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test('#779 - delimeters are applied to unescaped text.', 1, function() {
    var template = _.template('<<\nx\n>>', null, {evaluate: /<<(.*?)>>/g});
    strictEqual(template(), '<<\nx\n>>');
  });
        `)
	})
}
