# Security Policy

## Supported versions

`coverage` is distributed as a rolling build off `main` — there are no release
tags, and binaries are keyed to the commit SHA (see the README). Only the latest
`main` is supported; fixes land there and consumers move their pinned SHA
forward.

| Version | Supported |
|---|---|
| latest `main` | ✅ |
| older commits | ❌ |

## Reporting a vulnerability

Please **do not** report security vulnerabilities through public GitHub issues,
pull requests, or discussions.

Instead, use GitHub's private vulnerability reporting:

1. Go to the repository's **Security** tab.
2. Click **Report a vulnerability** (Security Advisories).
3. Provide a description, the affected commit/SHA, and a minimal reproduction if
   possible.

This keeps the report private until a fix is available.

## Scope

`coverage` reads local XML/JSON artifacts and writes a report; it makes no
network calls and executes no external commands. The most relevant classes of
issue are therefore:

- Parsing untrusted Cobertura/JUnit/summary input (malformed or hostile XML/JSON
  causing a crash, hang, or resource exhaustion).
- Path handling for `--input`, `--output`, `--config`, `--ignore`, and
  `--emit-json`.

Reports that fit this scope are especially useful. When you report, please
include the input that triggers the behavior so it can be reproduced.

## Response

We aim to acknowledge a valid report within a few days, agree on a disclosure
timeline, and credit reporters who wish to be named once a fix is released.
