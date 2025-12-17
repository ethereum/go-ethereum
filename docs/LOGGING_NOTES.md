# Logging Notes

This document provides a short overview of logging-related options
for geth.

## Verbosity levels

The `--verbosity` flag controls how much information geth prints:

- `0` – silent (only fatal errors),
- `1` – errors,
- `2` – warnings,
- `3` – info (default for many setups),
- `4` – debug,
- `5` – trace (very verbose).

Use higher verbosity levels only when debugging, as they can generate
large amounts of log data.

## Log output

To direct logs to a file instead of stdout, use shell redirection, for example:

    geth ... --verbosity 3 >> geth.log 2>&1

Consider log rotation tools (such as `logrotate` or container log
rotation) for long-running nodes.

## Common patterns

- Keep default verbosity in production unless debugging a specific issue.
- When reporting a bug, include logs with a short time window around
  the problematic event at a reasonably high verbosity (3–4).
