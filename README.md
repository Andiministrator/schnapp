<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="marke/logo/svg/schnapp-logo-dunkel.svg">
    <img alt="schnapp — screenshots for Linux" src="marke/logo/svg/schnapp-logo-hell.svg" width="520">
  </picture>
</p>

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

So far Schnapp captures an area, a monitor or a window on KDE Plasma (frozen full-screen overlay with magnifier) and saves the screenshot to the library; the editor is not there yet. Gio needs cgo (`libwayland-dev`, `libegl-dev`, `libxkbcommon-dev` or your distribution's equivalent) and the build tags `novulkan,nox11`, which you pass through `GOFLAGS` (no need to change your Go environment):

```sh
export GOFLAGS=-tags=novulkan,nox11
go build -trimpath -o schnapp ./cmd/schnapp
```

What exists:

| Path | What it is | How to check it |
|---|---|---|
| `.` (root) | Go module `github.com/Andiministrator/schnapp`: the program `cmd/schnapp` (options only so far), `aufnahme` (capture via KWin), `dokument` (layers, crop, redaction, undo), `datei` (working file and exports), `bibliothek` (library), `oberflaeche` (layer to Gio), `sprache` (language choice, gettext PO reader) and `po` (embedded translations, `po/de.po`); `data/` holds the desktop file template | `go vet ./... && go test ./...` (with `GOFLAGS` as above) |
| `zeichnen/` | Go module `github.com/Andiministrator/schnapp/zeichnen`: SVG parsing, an anti-aliased rasterizer, numeric helpers (standard library only) | `cd zeichnen && go vet ./... && go test ./...` |
| `marke/` | Logo, colours and corporate design ([`marke/corporate-design.md`](marke/corporate-design.md), German, with PDF) | `python3 marke/werkzeug/paket.py` rebuilds the logo files |
| `extern/gio/` | Gio with our patch (multi-monitor fullscreen, fractional scaling), used via `replace` in `go.mod` | `extern/gio-patch.sh pruefen`, see [`extern/README.md`](extern/README.md) (German) |
| `prototyp/` | Throw-away prototype that checked Gio (needs Go ≥ 1.27, cgo, `libwayland-client`, `libwayland-egl`, `libwayland-cursor`, `libEGL`, `libxkbcommon`) | `prototyp/bauen.sh`, see [`prototyp/README.md`](prototyp/README.md) |

## Documentation

User-facing documents (this README, `CONTRIBUTING.md`) exist in English and German. The project's working documents
— requirements, architecture decisions (`docs/adr/`), QA checklists, findings, research notes (`knowledge/`) and the
corporate design (`marke/`) — are written in German.

## AI disclosure

Most of the code and documentation in this repository is written by an AI model (Anthropic Claude) under the
direction and review of the maintainer. Decisions are recorded in `docs/adr/`. See [`CONTRIBUTING.md`](CONTRIBUTING.md).

## Licence

GPL-3.0-or-later, see [`LICENSE`](LICENSE). The logo uses outlines of Open Sans (SIL Open Font License 1.1). Bundled third-party code keeps its own licence: the Gio patch
(`extern/gio-ausgabe.patch`, applied in `extern/gio/`) applies to Gio (Unlicense OR MIT) and contains code generated from
wayland-protocols (MIT).
