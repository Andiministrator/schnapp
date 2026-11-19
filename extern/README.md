# Fremder Code mit eigenen Änderungen

## `gio/` — Gio mit Patch

[Gio](https://gioui.org) (Unlicense OR MIT, siehe `gio/LICENSE`) in der Version aus `gio-patch.sh` (`GIO_VERSION`),
erweitert um `gio-ausgabe.patch`:

- `app.FullscreenOn(name)` und `app.WaylandOutputs()` — Vollbild auf einer bestimmten Wayland-Ausgabe,
- gebrochene Skalierung (z. B. 1,6) über `wp_fractional_scale_v1` und `wp_viewporter`; Protokoll-Code mit
  wayland-scanner aus wayland-protocols 1.45 erzeugt (MIT).

Warum eine Kopie im Repo und kein Fork: `docs/adr/0015-gepatchtes-gio-im-repo.md`. Hintergrund:
`knowledge/topics/gio-wayland.md`.

| Aufruf | Wofür |
|---|---|
| `extern/gio-patch.sh pruefen` | Gio aus dem Modulcache + Patch = `gio/`? (läuft in der CI) |
| `extern/gio-patch.sh erzeugen` | nach einer Änderung in `gio/` den Patch neu schreiben — beides im selben Commit |
| `extern/gio-patch.sh anlegen` | `gio/` neu aufbauen (nach einem Gio-Update; vorher die alte Kopie nach `tmp/papierkorb/`) |

Module, die Gio nutzen, tragen `replace gioui.org => <relativer Pfad>/extern/gio`. Bauen mit `-trimpath` und den
Tags `novulkan,nox11`; cgo-Bibliotheken: libwayland-client ≥ 1.20, libwayland-egl, libwayland-cursor, libEGL,
libxkbcommon.
