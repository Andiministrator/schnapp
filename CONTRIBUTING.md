# Contributing to Schnapp

*English · [Deutsch](CONTRIBUTING.de.md)*

Thanks for your interest. Schnapp is at an early stage; please open an issue before larger changes so we can agree on
the direction first.

## How this project is made

Most code and documentation is written by an AI model (Anthropic Claude) under the direction and review of the
maintainer. Contributions written by people are just as welcome. If you use AI tools yourself, say so in the pull
request.

## Ground rules

- **Wayland only.** No X11 code paths (`docs/adr/0007-*`).
- **Redaction is final.** Redacted or pixelated content must not be recoverable from any saved or exported file.
  Changes in this area need a test that proves it.
- **No traces.** Generated files must not contain local paths, user, host or program names.
- **No real screenshots** in tests or documentation — they show private data. Generate test images instead.
- **New dependencies** only with a clear benefit; name the purpose, the licence (GPL-3.0 compatible) and the
  alternative in the commit message (`docs/adr/0011-*`).
- **No cgo** beyond what Gio needs, unless agreed in an architecture decision.

## Languages

- User interface texts: English is the source language; every new text also needs a German translation (`po/de.po`
  once it exists). Further languages are welcome.
- User documents (README, CONTRIBUTING) exist in English and German — update both in the same pull request.
- Working documents (`docs/`, `knowledge/`), commit messages and code comments are in German. English is fine in your
  pull request; the maintainer will adapt it.

## Design

Logo, colours, type and UI rules: [`marke/corporate-design.md`](marke/corporate-design.md) (German, PDF alongside);
logo files in [`marke/logo/`](marke/logo/README.md).

## Commits and checks

- Conventional Commits: `<type>(<scope>): <description>`, types `feat fix docs refactor chore style test perf`.
- Before a pull request: `gofmt`, `go vet ./...` and `go test ./...` in every module you touched.
- Every source file starts with `// SPDX-License-Identifier: GPL-3.0-or-later`.

By contributing you agree that your contribution is licensed under GPL-3.0-or-later.
