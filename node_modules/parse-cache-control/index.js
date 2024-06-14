module.exports = function parseCacheControl(field) {

  if (typeof field !== 'string') {
    return null;
  }

  /*
    Cache-Control   = 1#cache-directive
    cache-directive = token [ "=" ( token / quoted-string ) ]
    token           = [^\x00-\x20\(\)<>@\,;\:\\"\/\[\]\?\=\{\}\x7F]+
    quoted-string   = "(?:[^"\\]|\\.)*"
  */

  //                             1: directive                                        =   2: token                                              3: quoted-string
  var regex = /(?:^|(?:\s*\,\s*))([^\x00-\x20\(\)<>@\,;\:\\"\/\[\]\?\=\{\}\x7F]+)(?:\=(?:([^\x00-\x20\(\)<>@\,;\:\\"\/\[\]\?\=\{\}\x7F]+)|(?:\"((?:[^"\\]|\\.)*)\")))?/g;

  var header = {};
  var err = field.replace(regex, function($0, $1, $2, $3) {
    var value = $2 || $3;
    header[$1] = value ? value.toLowerCase() : true;
    return '';
  });

  if (header['max-age']) {
    try {
      var maxAge = parseInt(header['max-age'], 10);
      if (isNaN(maxAge)) {
        return null;
      }

      header['max-age'] = maxAge;
    }
    catch (err) { }
  }

  return (err ? null : header);
};
