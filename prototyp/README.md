# Prototyp Gio (Wegwerf-Code)

Prüft, ob Gio für Schnapp trägt (ADR-0003, Kriterien in Nachtrag 1). Kein Produktcode.

## Bauen

```bash
prototyp/bauen.sh            # baut nach ~/.local/bin/schnapp-prototyp
```

Das Skript legt eine Kopie von Gio unter `tmp/gio-gepatcht` an und wendet `gio-ausgabe.patch` an:

- `app.FullscreenOn(name)` — Vollbildfenster auf einer bestimmten Wayland-Ausgabe (`xdg_toplevel.set_fullscreen`
  mit Ausgabe statt `nil`),
- `app.WaylandOutputs()` — Ausgaben mit Namen auflisten (`wl_output` Version 4 statt 2).

Build-Tags: `novulkan` (keine Vulkan-Header nötig, OpenGL ES über EGL), `nox11` (nur Wayland, ADR-0007).
Gio braucht unter Linux cgo (libwayland-client, libEGL, libxkbcommon).

## Aufrufe

| Aufruf | Was passiert |
|---|---|
| `schnapp-prototyp` | jeden Bildschirm über KWin ScreenShot2 aufnehmen, je Monitor ein Vollbild-Overlay, Bereich aufziehen → Editor |
| `schnapp-prototyp -bild datei.png` | Bild laden → Overlay → Editor |
| `schnapp-prototyp -testbild` | synthetisches 4K-Bild im Editor |
| `schnapp-prototyp -messen 8` | 4K-Testbild, der Pfeil biegt sich 8 s selbst, Bildrate ins Protokoll |

Editor: Pfeil ziehen, am blauen Griff biegen, Enden verschieben, Strg+Z, Mausrad zoomt, Esc beendet.
Das Protokoll geht nach stderr.

`schnapp-prototyp.desktop.in` ist die Vorlage für die Berechtigung über eine `.desktop`-Datei
(`@EXEC@` durch den absoluten Pfad ersetzen), falls KWin die Aufnahme ohne sie ablehnt.
