<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="brand/logo/svg/schnapp-logo-dunkel.svg">
    <img alt="schnapp — screenshots for Linux" src="brand/logo/svg/schnapp-logo-hell.svg" width="520">
  </picture>
</p>

# Schnapp

*English · [Deutsch](README.de.md)*

Screenshot and annotation tool for Linux/Wayland — KDE Plasma first. Inspired by the workflow of Snagit: capture
fast, annotate (straight and curved arrows, boxes, step numbers, stamps), redact for real, cut out the middle of long
pages, share. Also meant as a simple image editor.

## Status

Version 0.5.0 (KDE Plasma only) opens PNG and JPEG files from the file manager ("Open with", JPEG rotated as its EXIF data says), can join all monitors into one image, saves faster, and draws text the same in the editor, in PNG/JPG and in SVG. Since 0.4.0 closing or quitting waits as long as a slow save still makes progress. Since 0.3.0 Schnapp saves and copies in the background: the window no longer freezes with large images (editing pauses until the save is done, copying and closing still work), and the menu works entirely from the keyboard. Schnapp captures on KDE Plasma (area, monitor, window; **Print** key), saves every screenshot to
the library and opens it in an editor where you draw arrows (straight or bent), boxes, text labels and step numbers
and select, move and resize them, black out or pixelate areas for good, crop, cut out the middle of a long screenshot
and copy the result to the clipboard. It is written in Go with the [Gio](https://gioui.org) UI library,
plus a small patch to Gio (one full-screen overlay per monitor, fractional scaling, cursor at mixed scales,
images on the clipboard). A part that
reaches the edge of the image is cropped off and comes back with the *Whole image* button.

## Planned features

- Capture region, window or screen on KDE Plasma (KWin), with a frozen full-screen overlay on every monitor.
- Annotate with vector layers: arrows (also curved), boxes, text, step numbers, stamps.
- Redact and pixelate **for real** — redacted pixels are gone from every saved or exported file.
- Shorten long screenshots by cutting out the middle, without destroying the original.
- Library with automatic saving, search and tags.
- User interface in English and German; it follows the system language and falls back to English.

## Building

Gio needs cgo (`libwayland-dev`, `libegl-dev`, `libxkbcommon-dev` or your distribution's equivalent) and the build tags `novulkan,nox11`, which you pass through `GOFLAGS` (no need to change your Go environment):

```sh
export GOFLAGS=-tags=novulkan,nox11
go build -trimpath -o schnapp ./cmd/schnapp
```

What exists:

| Path | What it is | How to check it |
|---|---|---|
| `.` (root) | Go module `github.com/Andiministrator/schnapp`: the program `cmd/schnapp` and its packages under `internal/`, among them `capture` (capture via KWin), `selection` (overlay logic), `canvas` (editor logic), `document` (layers, crop, redaction, undo), `fileformat` (working file and exports), `render` (layers to pixels), `library` (library), `settings` (`settings.ini`), `ui` (overlay and editor in Gio), `wlclip` (keeps a copied image on the clipboard after the editor closes), `portal` (file dialog of the desktop portal), `export` (export from the editor), `i18n` (language choice, gettext PO reader) and `po` (embedded translations, `internal/po/de.po`); `data/` holds the desktop file template | `go vet ./... && go test ./...` (with `GOFLAGS` as above) |
| `draw/` | Go module `github.com/Andiministrator/schnapp/draw`: SVG parsing, an anti-aliased rasterizer, numeric helpers (standard library only) | `cd draw && go vet ./... && go test ./...` |
| `brand/` | Logo, colours and corporate design ([`brand/corporate-design.md`](brand/corporate-design.md), German, with PDF) | `python3 brand/werkzeug/paket.py` rebuilds the logo files |
| `third_party/gio/` | Gio with our patch (multi-monitor fullscreen, fractional scaling, cursor at mixed scales, images on the clipboard), used via `replace` in `go.mod` | `third_party/gio-patch.sh pruefen`, see [`third_party/README.md`](third_party/README.md) (German) |

## Usage

| Command | What it does |
|---|---|
| `schnapp` or `schnapp --region` | select an area in a frozen full-screen overlay (default, **Print**) |
| `schnapp --screen[=NAME]` | capture a monitor (without a name as set in the settings, `[capture] screen = all` joins all monitors into one image; `--list-screens` lists the names) |
| `schnapp --window` | select a window |
| `schnapp --repeat` | capture the last area again |
| `schnapp --no-editor …` | capture without opening the editor afterwards |
| `schnapp FILE` | open a PNG or JPEG in the editor (also what "Open with" in a file manager runs; a JPEG with EXIF orientation is rotated or mirrored as image viewers show it, the EXIF data itself is not kept) |
| `schnapp --ignore-limit FILE` | open the file even if it has so many or such large annotations that rendering exceeds the compute budget (for your own work that grew past it) |

Every capture is saved to the library (`~/Pictures/schnapp/YYYY-MM/`) and its path printed on standard output before
the editor opens. Settings live in `~/.config/schnapp/settings.ini`, for example `[capture] open-editor = false`.

In the editor, tools: **V** select, **A** arrow (drag; **Shift** snaps to 45°; drag the middle handle to bend it),
**R** box (**Shift**: square), **T** text (click, type, **Enter** confirms, **Shift+Enter** new line, **Esc** discards;
double-click a text to edit it), **N** step number (counts up by itself), **B** redact (drag over an area: black),
**P** pixelate (drag; a mosaic made only from the area's average colour and random noise, so it carries nothing of
what was there). Both overwrite the pixels of the image itself: they can be undone until you save (the undo history keeps at most 512 MiB of pixel copies, older steps fall away silently), after that they are
final and no saved or exported file contains the hidden content. An image copied with **Ctrl+C** before redacting
stays in the clipboard as it was. **C** crop (drag a frame; the *Whole image* button or a
double-click with the tool shows the whole image again), **X** cut out (drag across the part to remove — vertically to
remove rows, horizontally for columns; the seam shows a zigzag edge; double-click a seam to bring the part back; a part
reaching the edge of the image is cropped off instead and comes back with *Whole image*). Both only hide: the original stays complete in the working
file, so redact anything secret before cropping it away — exports and the clipboard contain only the visible part.
Annotations lying entirely in a hidden part are hidden too (the status bar says how many) and come back with it.
The *Colour* button cycles four colours and
recolours the selected annotation. A drawing tool stays active; **Esc** steps back: text input → drag → selection →
select tool. Drag an annotation or its handles to move, resize or bend it; it always stays partly inside the image.
The mouse wheel zooms at the pointer; with the select tool dragging empty space moves the view, the middle button does
so with every tool. Keys: **Ctrl+Z** undo, **Ctrl+Shift+Z**/**Ctrl+Y** redo, **Del** delete, **Ctrl+S** save,
**Ctrl+Shift+E** (button "Export…") export as PNG, JPG or SVG through the desktop file dialog (no editing data inside, redacted content stays gone; if replacing an existing file fails halfway and the old content cannot be restored, the status bar names the file as damaged),
**Ctrl+C** copy the finished image to the clipboard (on compositors with the data-control protocol, such as KDE Plasma, it stays after the editor closes until something else is copied, for at most 12 hours; elsewhere only while the editor is open, and closing then stops once and warns that the clipboard holds the image only while the window is open), **Ctrl+0** fit, **Ctrl+1** 100 %, **Ctrl+W** close. Every shortcut also works with the mouse (only holding Shift while dragging, for 45° arrows and squares, and Shift+Enter for a new line in text stay keys): the **Menu** button next to Save lists all actions with their shortcuts (including **Restore cut-out areas**, which brings back everything removed with Cut out), and hovering a tool button shows its key. The menu also works without a mouse: **Tab** to the Menu button, **Enter** or **Space** opens it, the arrow keys, **Home** and **End** choose an entry, **Enter** runs it, **Esc** closes.

Changes are saved when the editor closes; **Ctrl+S** without changes writes nothing. Saving runs in the background: the window stays responsive and shows "Saving…", editing pauses until it is done. A file from outside the library
becomes a new screenshot in the library on the first save (the file itself stays untouched; the status bar names the new
file, which is saved to from then on). If a library file cannot be updated (deleted, replaced or not writable), the
changes are saved as a new screenshot instead. If saving fails, the window stays open and says so; closing a second time
discards the changes. Ctrl+C in the terminal or logging out saves first. A file that is too costly to render is refused
when opening; `--ignore-limit FILE` opens it anyway.

## Install

`./install.sh` builds Schnapp (same requirements as under "Building"; the version comes from `git describe`, without a reachable tag it reads `0.0.0-dev+<hash>`, with changed files `0.0.0-dev+<hash>.dirty`) and installs it for the current user:

| File | Location |
|---|---|
| Program | `$PREFIX/bin/schnapp` (default `~/.local/bin/schnapp`) |
| Desktop file | `$XDG_DATA_HOME/applications/schnapp.desktop` (default `~/.local/share/applications/`) |
| Icon | `$XDG_DATA_HOME/icons/hicolor/scalable/apps/schnapp.svg` |

The desktop file lists PNG and JPEG, so file managers offer Schnapp under "Open with". It carries the absolute path of the program (KDE Plasma 6.7 and older bind the screenshot permission to it) and the default shortcut **Print** for "Capture Area". Spectacle uses **Print** as well: if both are installed, turn off Spectacle's shortcut in the system settings (Shortcuts). Make sure `~/.local/bin` is in your `PATH`.

Desktop tools may update their caches as well (menu database; the icon cache only if one exists). Remove it again with `./install.sh --uninstall` (deletes only these three files). `./install.sh --help` lists the options.

## Documentation

User-facing documents (this README, `CONTRIBUTING.md`) exist in English and German. The project's working documents
— requirements, architecture decisions (`docs/adr/`), QA checklists, findings, research notes (`knowledge/`) and the
corporate design (`brand/`) — are written in German.

## AI disclosure

Most of the code and documentation in this repository is written by an AI model (Anthropic Claude) under the
direction and review of the maintainer. Decisions are recorded in `docs/adr/`. See [`CONTRIBUTING.md`](CONTRIBUTING.md).

## Licence

GPL-3.0-or-later, see [`LICENSE`](LICENSE). The logo uses outlines of Open Sans (SIL Open Font License 1.1). Bundled third-party code keeps its own licence: the Gio patch
(`third_party/gio-ausgabe.patch`, applied in `third_party/gio/`) applies to Gio (Unlicense OR MIT) and contains code generated from
wayland-protocols (MIT); `internal/raster16` is taken from golang.org/x/image/vector (BSD-3-Clause, see
`internal/raster16/LICENSE`).
