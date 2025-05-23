<p align="center">
  <a href="https://sentry.io" target="_blank" align="center">
    <img src="https://sentry-brand.storage.googleapis.com/sentry-logo-black.png" width="280">
  </a>
  <br />
</p>

# Official Sentry SDK for NodeJS

[![npm version](https://img.shields.io/npm/v/@sentry/node.svg)](https://www.npmjs.com/package/@sentry/node)
[![npm dm](https://img.shields.io/npm/dm/@sentry/node.svg)](https://www.npmjs.com/package/@sentry/node)
[![npm dt](https://img.shields.io/npm/dt/@sentry/node.svg)](https://www.npmjs.com/package/@sentry/node)
[![typedoc](https://img.shields.io/badge/docs-typedoc-blue.svg)](http://getsentry.github.io/sentry-javascript/)

## Links

- [Official SDK Docs](https://docs.sentry.io/quickstart/)
- [TypeDoc](http://getsentry.github.io/sentry-javascript/)

## Usage

To use this SDK, call `init(options)` as early as possible in the main entry module. This will initialize the SDK and
hook into the environment. Note that you can turn off almost all side effects using the respective options.

```javascript
// ES5 Syntax
const Sentry = require('@sentry/node');
// ES6 Syntax
import * as Sentry from '@sentry/node';

Sentry.init({
  dsn: '__DSN__',
  // ...
});
```

To set context information or send manual events, use the exported functions of `@sentry/node`. Note that these
functions will not perform any action before you have called `init()`:

```javascript
// Set user information, as well as tags and further extras
Sentry.configureScope(scope => {
  scope.setExtra('battery', 0.7);
  scope.setTag('user_mode', 'admin');
  scope.setUser({ id: '4711' });
  // scope.clear();
});

// Add a breadcrumb for future events
Sentry.addBreadcrumb({
  message: 'My Breadcrumb',
  // ...
});

// Capture exceptions, messages or manual events
Sentry.captureMessage('Hello, world!');
Sentry.captureException(new Error('Good bye'));
Sentry.captureEvent({
  message: 'Manual',
  stacktrace: [
    // ...
  ],
});
```
