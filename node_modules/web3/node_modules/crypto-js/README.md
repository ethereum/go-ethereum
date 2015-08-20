# crypto-js

JavaScript library of crypto standards.

## Node.js (Install)

Requirements:

- Node.js
- npm (Node.js package manager)

```bash
npm install crypto-js
```

### Usage

Modular include:

```javascript
var AES = require("crypto-js/aes");
var SHA256 = require("crypto-js/sha256");
...
console.log(SHA256("Message"));
```

Including all libraries, for access to extra methods:

```javascript
var CryptoJS = require("crypto-js");
console.log(CryptoJS.HmacSHA1("Message", "Key"));
```

## Client (browser)

Requirements:

- Node.js
- Bower (package manager for frontend)

```bash
bower install crypto-js
```

### Usage

Modular include:

```javascript
require.config({
    packages: [
        {
            name: 'crypto-js',
            location: 'path-to/bower_components/crypto-js',
            main: 'index'
        }
    ]
});

require(["crypto-js/aes", "crypto-js/sha256"], function (AES, SHA256) {
    console.log(SHA256("Message"));
});
```

Including all libraries, for access to extra methods:

```javascript
// Above-mentioned will work or use this simple form
require.config({
    paths: {
        'require-js': 'path-to/bower_components/crypto-js/crypto-js'
    }
});

require(["crypto-js"], function (CryptoJS) {
    console.log(CryptoJS.HmacSHA1("Message", "Key"));
});
```

### Usage without RequireJS

```html
<script type="text/javascript" src="path-to/bower_components/crypto-js/crypto-js.js"></script>
<script type="text/javascript">
    var encrypted = CryptoJS.AES(...);
    var encrypted = CryptoJS.SHA256(...);
</script>
```

## API

See: https://code.google.com/p/crypto-js

### List of modules


- ```crypto-js/core```
- ```crypto-js/x64-core```
- ```crypto-js/lib-typedarrays```

---

- ```crypto-js/md5```
- ```crypto-js/sha1```
- ```crypto-js/sha256```
- ```crypto-js/sha224```
- ```crypto-js/sha512```
- ```crypto-js/sha384```
- ```crypto-js/sha3```
- ```crypto-js/ripemd160```

---

- ```crypto-js/hmac-md5```
- ```crypto-js/hmac-sha1```
- ```crypto-js/hmac-sha256```
- ```crypto-js/hmac-sha224```
- ```crypto-js/hmac-sha512```
- ```crypto-js/hmac-sha384```
- ```crypto-js/hmac-sha3```
- ```crypto-js/hmac-ripemd160```

---

- ```crypto-js/pbkdf2```

---

- ```crypto-js/aes```
- ```crypto-js/tripledes```
- ```crypto-js/rc4```
- ```crypto-js/rabbit```
- ```crypto-js/rabbit-legacy```
- ```crypto-js/evpkdf```

---

- ```crypto-js/format-openssl```
- ```crypto-js/format-hex```

---

- ```crypto-js/enc-latin1```
- ```crypto-js/enc-utf8```
- ```crypto-js/enc-hex```
- ```crypto-js/enc-utf16```
- ```crypto-js/enc-base64```

---

- ```crypto-js/mode-cfb```
- ```crypto-js/mode-ctr```
- ```crypto-js/mode-ctr-gladman```
- ```crypto-js/mode-ofb```
- ```crypto-js/mode-ecb```

---

- ```crypto-js/pad-pkcs7```
- ```crypto-js/pad-ansix923```
- ```crypto-js/pad-iso10126```
- ```crypto-js/pad-iso97971```
- ```crypto-js/pad-zeropadding```
- ```crypto-js/pad-nopadding```

## License

[The MIT License (MIT)](http://opensource.org/licenses/MIT)

Copyright (c) 2009-2013 Jeff Mott  
Copyright (c) 2013-2015 Evan Vosberg

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.