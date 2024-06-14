Base64 Coder
============

function decode(textData: string): Uint8Array
---------------------------------------------

Decodes a base64 encoded string into the binary data.

```javascript
import * as base64 from "@ethersproject/base64";

let encodedData = "...";
let data = base64.decode(encodedData);
console.log(data);
// { Uint8Array: [] }
```

function encode(data: Arrayish): string
---------------------------------------

Decodes a base64 encoded string into the binary data.

```javascript
import * as base64 from "@ethersproject/base64";

let data = [ ];
let encodedData = base64.encode(data);
console.log(encodedData);
// "..."
```

License
=======

MIT License
