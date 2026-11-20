# Fremder Code mit eigenen Änderungen

## `gio/` — Gio mit Patch

[Gio](https://gioui.org) (Unlicense OR MIT, siehe `gio/LICENSE`) in der Version aus `gio-patch.sh` (`GIO_VERSION`),
erweitert um `gio-ausgabe.patch`:

- `app.FullscreenOn(name)` und `app.WaylandOutputs()` — Vollbild auf einer bestimmten Wayland-Ausgabe,
- gebrochene Skalierung (z. B. 1,6) über `wp_fractional_scale_v1` und `wp_viewporter`; Protokoll-Code mit
  wayland-scanner aus wayland-protocols 1.45 erzeugt (MIT),
- Mauszeiger bei wechselnder Skala (F-78): Das Cursor-Thema wird bei Skalenwechsel neu geladen
  (`reloadCursorTheme`, Felder `cursor.name/size/scale/failedScale`), der Hotspot durch die Skala des Themas geteilt;
  ein Bild, dessen Größe nicht durch die Skala teilbar ist, bekommt Puffer-Skala 1 (sonst Wayland-Protokollfehler).
  Beim Gio-Update prüfen, ob Gio das inzwischen selbst kann (`wp_cursor_shape_v1`).

- **Bild-Typ in der Zwischenablage (Karte #39, ADR-0015 Nachtrag 2):** `wlDisplay.writeClipboard` in `app/os_wayland.go`
  wertet `clipboard.WriteCmd.Type` aus. Bei `image/…` bietet die `wl_data_source` genau diesen Typ an und liefert die Bytes
  bei `send`; Text bleibt unverändert. Der Inhalt gehört dem Fenster: Nach dem Schließen bleibt er nur erhalten, wenn ein
  Zwischenablage-Verwalter (unter KDE Klipper) ihn übernommen hat. Beim Gio-Update prüfen, ob Gio das selbst kann.

Warum eine Kopie im Repo und kein Fork: `docs/adr/0015-gepatchtes-gio-im-repo.md`. Hintergrund:
`knowledge/topics/gio-wayland.md`.

| Aufruf | Wofür |
|---|---|
| `third_party/gio-patch.sh pruefen` | Gio aus dem Modulcache + Patch = `gio/`? (läuft in der CI) |
| `third_party/gio-patch.sh erzeugen` | nach einer Änderung in `gio/` den Patch neu schreiben — beides im selben Commit |
| `third_party/gio-patch.sh anlegen` | `gio/` neu aufbauen (nach einem Gio-Update; vorher die alte Kopie nach `tmp/papierkorb/`) |

Module, die Gio nutzen, tragen `replace gioui.org => <relativer Pfad>/third_party/gio`. Bauen mit `-trimpath` und den
Tags `novulkan,nox11`; cgo-Bibliotheken: libwayland-client ≥ 1.20, libwayland-egl, libwayland-cursor, libEGL,
libxkbcommon.
