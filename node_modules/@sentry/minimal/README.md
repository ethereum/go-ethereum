<p align="center">
  <a href="https://sentry.io" target="_blank" align="center">
    <img src="https://sentry-brand.storage.googleapis.com/sentry-logo-black.png" width="280">
  </a>
  <br />
</p>

# Sentry JavaScript SDK Minimal

[![npm version](https://img.shields.io/npm/v/@sentry/minimal.svg)](https://www.npmjs.com/package/@sentry/minimal)
[![npm dm](https://img.shields.io/npm/dm/@sentry/minimal.svg)](https://www.npmjs.com/package/@sentry/minimal)
[![npm dt](https://img.shields.io/npm/dt/@sentry/minimal.svg)](https://www.npmjs.com/package/@sentry/minimal)
[![typedoc](https://img.shields.io/badge/docs-typedoc-blue.svg)](http://getsentry.github.io/sentry-javascript/)

## Links

- [Official SDK Docs](https://docs.sentry.io/quickstart/)
- [TypeDoc](http://getsentry.github.io/sentry-javascript/)

## General

A minimal Sentry SDK that uses a configured client when embedded into an application. It allows library authors add
support for a Sentry SDK without having to bundle the entire SDK or being dependent on a specific platform. If the user
is using Sentry in their application and your library uses `@sentry/minimal`, the user receives all
breadcrumbs/messages/events you added to your libraries codebase.

## Usage

To use the minimal, you do not have to initialize an SDK. This should be handled by the user of your library. Instead,
directly use the exported functions of `@sentry/minimal` to add breadcrumbs or capture events:

```javascript
import * as Sentry from '@sentry/minimal';

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

Note that while strictly possible, it is discouraged to interfere with the event context. If for some reason your
library needs to inject context information, beware that this might override the user's context values:

```javascript
// Set user information, as well as tags and further extras
Sentry.configureScope(scope => {
  scope.setExtra('battery', 0.7);
  scope.setTag('user_mode', 'admin');
  scope.setUser({ id: '4711' });
  // scope.clear();
});
```
