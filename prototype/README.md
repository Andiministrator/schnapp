# Prototyp Gio (Wegwerf-Code)

Prüft, ob Gio für Schnapp trägt (ADR-0003, Kriterien in Nachtrag 1). Kein Produktcode.

## Bauen

```bash
prototype/bauen.sh            # baut nach ~/.local/bin/schnapp-prototype
```

Gio kommt gepatcht aus `../third_party/gio` (`replace` in `go.mod`, ADR-0015); ein gewöhnliches
`go build -tags novulkan,nox11 .` geht also auch. Der Patch (`third_party/gio-ausgabe.patch`, siehe `third_party/README.md`):

- `app.FullscreenOn(name)` — Vollbildfenster auf einer bestimmten Wayland-Ausgabe (`xdg_toplevel.set_fullscreen`
  mit Ausgabe statt `nil`),
- `app.WaylandOutputs()` — Ausgaben mit Namen auflisten (`wl_output` Version 4 statt 2).
- **gebrochene Skalierung** (z. B. 1,6) über `wp_fractional_scale_v1` und `wp_viewporter`: Puffer in echter
  Pixelgröße statt Ganzzahl-Skala, die KWin herunterrechnet. Protokoll-Code mit wayland-scanner aus
  wayland-protocols 1.45 erzeugt (`knowledge/sources-visited.md`).

`bauen.sh` setzt Build-Tags und `-trimpath` (keine lokalen Pfade im Programm). Der Patch braucht
wayland-client ≥ 1.20 (`wl_output.name`).

Build-Tags: `novulkan` (keine Vulkan-Header nötig, OpenGL ES über EGL), `nox11` (nur Wayland, ADR-0007).
Gio braucht unter Linux cgo (libwayland-client ≥ 1.20, libwayland-egl, libwayland-cursor, libEGL, libxkbcommon).

## Aufrufe

| Aufruf | Was passiert |
|---|---|
| `schnapp-prototype` | jeden Bildschirm über KWin ScreenShot2 aufnehmen, je Monitor ein Vollbild-Overlay, Bereich aufziehen → Editor |
| `schnapp-prototype -bild datei.png` | Bild laden → Overlay → Editor |
| `schnapp-prototype -testbild` | synthetisches 4K-Bild im Editor |
| `schnapp-prototype -testbild -fenster` | dasselbe in einem verschiebbaren Fenster statt Vollbild (Zeigertest F-78) |
| `schnapp-prototype -messen 8` | 4K-Testbild, der Pfeil biegt sich 8 s selbst, Bildrate ins Protokoll |

Editor: Pfeil ziehen, am blauen Griff biegen, Enden verschieben, Strg+Z, Mausrad zoomt, Esc beendet.
Das Protokoll geht nach stderr.

`schnapp-prototype.desktop.in` ist die Vorlage für die Berechtigung über eine `.desktop`-Datei
(`@EXEC@` durch den absoluten Pfad ersetzen) — nötig nur bis Plasma 6.7; ab 6.8 prüft KWin nicht mehr (F-15).
