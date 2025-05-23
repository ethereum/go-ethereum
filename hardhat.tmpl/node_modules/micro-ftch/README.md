# micro-ftch

Wraps nodejs built-in modules and browser fetch into one function

## Usage

Can be used in browser and in node.js.

> npm install micro-ftch

```js
const fetch = require('micro-ftch');
fetch('https://google.com').then(...)
```

## Options

The list of options that can be supplied as second argument to fetch(url, opts):

```typescript
export type FETCH_OPT = {
  method?: string;
  type?: 'text' | 'json' | 'bytes'; // Response encoding (auto-detect if empty)
  redirect: boolean; // Follow redirects
  expectStatusCode?: number | false; // Expect this status code
  headers: Record<string, string>;
  data?: object; // POST/PUT/DELETE request data
  full: boolean; // Return full request {headers, status, body}
  keepAlive: boolean; // Enable keep-alive (node only)
  cors: boolean; // Allow CORS safe-listed headers (browser-only)
  referrer: boolean; // Send referrer (browser-only)
  sslAllowSelfSigned: boolean; // Allow self-signed ssl certs (node only)
  sslPinnedCertificates?: string[]; // Verify fingerprint of certificate (node only)
  _redirectCount: number;
};
```

## License

MIT License (c) 2020, Paul Miller (https://paulmillr.com)