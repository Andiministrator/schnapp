# Schnapp

*English · [Deutsch](README.de.md)*

Screenshot and annotation tool for Linux/Wayland — KDE Plasma first. Inspired by the workflow of Snagit: capture
fast, annotate (straight and curved arrows, boxes, step numbers, stamps), redact for real, cut out the middle of long
pages, share. Also meant as a simple image editor.

## Status

Early work — **nothing usable yet**. A prototype has confirmed the toolkit: Schnapp is written in Go with the
[Gio](https://gioui.org) UI library, plus a small patch to Gio (one full-screen overlay per monitor, fractional
scaling). The first version (0.1.0) is being planned.

## Planned features

- Capture region, window or screen on KDE Plasma (KWin), with a frozen full-screen overlay on every monitor.
- Annotate with vector layers: arrows (also curved), boxes, text, step numbers, stamps.
- Redact and pixelate **for real** — redacted pixels are gone from every saved or exported file.
- Shorten long screenshots by cutting out the middle, without destroying the original.
- Library with automatic saving, search and tags.
- User interface in English and German; it follows the system language and falls back to English.

## Building

There is no program to build yet. What exists:

| Path | What it is | How to check it |
|---|---|---|
| `zeichnen/` | Go module `github.com/Andiministrator/schnapp/zeichnen`: SVG parsing, an anti-aliased rasterizer, numeric helpers (standard library only) | `cd zeichnen && go vet ./... && go test ./...` |
| `prototyp/` | Throw-away prototype that checked Gio (needs Go ≥ 1.27, cgo, `libwayland-client`, `libEGL`, `libxkbcommon`) | `prototyp/bauen.sh`, see [`prototyp/README.md`](prototyp/README.md) |

## Documentation

User-facing documents (this README, `CONTRIBUTING.md`) exist in English and German. The project's working documents
— requirements, architecture decisions (`docs/adr/`), QA checklists, findings and research notes (`knowledge/`) — are
written in German.

## AI disclosure

Most of the code and documentation in this repository is written by an AI model (Anthropic Claude) under the
direction and review of the maintainer. Decisions are recorded in `docs/adr/`. See [`CONTRIBUTING.md`](CONTRIBUTING.md).

## Licence

GPL-3.0-or-later, see [`LICENSE`](LICENSE). Bundled third-party code keeps its own licence: the Gio patch
(`prototyp/gio-ausgabe.patch`) applies to Gio (Unlicense OR MIT) and contains code generated from
wayland-protocols (MIT).
