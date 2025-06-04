var test = require('tape');
var parseCacheControl = require('./index');

test('parseCacheControl', function (t) {
  var header = parseCacheControl('must-revalidate, max-age=3600');
  t.ok(header);
  t.equal(header['must-revalidate'], true);
  t.equal(header['max-age'], 3600);

  header = parseCacheControl('must-revalidate, max-age="3600"');
  t.ok(header);
  t.equal(header['must-revalidate'], true);
  t.equal(header['max-age'], 3600);

  header = parseCacheControl('must-revalidate, b =3600');
  t.notOk(header);

  header = parseCacheControl('must-revalidate, max-age=a3600');
  t.notOk(header);

  header = parseCacheControl(123);
  t.notOk(header);

  header = parseCacheControl(null);
  t.notOk(header);

  header = parseCacheControl(undefined);
  t.notOk(header);

  t.end();
});
