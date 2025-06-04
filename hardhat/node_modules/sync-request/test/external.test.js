var request = require('../');
var FormData = request.FormData;

// Test GET request
test('http://nodejs.org', () => {
  var res = request('GET', 'http://nodejs.org');

  expect(res.statusCode).toBe(200);
  expect(res.url).toBe('https://nodejs.org/en/');
});

test('http://httpbin.org/post', () => {
  var res = JSON.parse(
    request('POST', 'http://httpbin.org/post', {
      body: '<body/>',
    }).getBody('utf8')
  );
  delete res.origin;
  expect(res).toMatchSnapshot();
});

test('http://httpbin.org/post json', () => {
  var res = JSON.parse(
    request('POST', 'http://httpbin.org/post', {
      json: {foo: 'bar'},
    }).getBody('utf8')
  );
  delete res.origin;
  expect(res).toMatchSnapshot();
});

test('http://httpbin.org/post form', () => {
  var fd = new FormData();
  fd.append('foo', 'bar');
  var res = JSON.parse(
    request('POST', 'http://httpbin.org/post', {
      form: fd,
    }).getBody('utf8')
  );
  delete res.headers['Content-Type'];
  delete res.origin;
  expect(res).toMatchSnapshot();
});

test('https://expired.badssl.com', () => {
  var errored = false;
  try {
    // Test unauthorized HTTPS GET request
    var res = request('GET', 'https://expired.badssl.com');
  } catch (ex) {
    return;
  }
  throw new Error('Should have rejected unauthorized https get request');
});
