# ndjson

Streaming [newline delimited json](https://en.wikipedia.org/wiki/Line_Delimited_JSON) parser + serializer. Available as a JS API and a CLI.

[![NPM](https://nodei.co/npm/ndjson.png)](https://nodei.co/npm/ndjson/)

## Usage

```
const ndjson = require('ndjson')
```

#### ndjson.parse([opts])

Returns a transform stream that accepts newline delimited json buffers and emits objects of parsed data.

Example file:

```
{"foo": "bar"}
{"hello": "world"}
```

Parsing it:

```js
fs.createReadStream('data.txt')
  .pipe(ndjson.parse())
  .on('data', function(obj) {
    // obj is a javascript object
  })
```


##### Options

- `strict` can be set to false to discard non-valid JSON messages
- All other options are passed through to the stream class.

#### ndjson.stringify([opts])

Returns a transform stream that accepts JSON objects and emits newline delimited json buffers.

example usage:

```js
var serialize = ndjson.serialize()
serialize.on('data', function(line) {
  // line is a line of stringified JSON with a newline delimiter at the end
})
serialize.write({"foo": "bar"})
serialize.end()
```

##### Options

Options are passed through to the stream class.

### LICENSE

BSD-3-Clause
