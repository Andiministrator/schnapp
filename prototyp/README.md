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
- **gebrochene Skalierung** (z. B. 1,6) über `wp_fractional_scale_v1` und `wp_viewporter`: Puffer in echter
  Pixelgröße statt Ganzzahl-Skala, die KWin herunterrechnet. Protokoll-Code mit wayland-scanner aus
  wayland-protocols 1.45 erzeugt (`knowledge/sources-visited.md`).

Die Kopie trägt eine Marke mit Gio-Version und Prüfsumme des Patches; passt sie nicht (Patch geändert, Lauf
abgebrochen), verschiebt `bauen.sh` die alte Kopie nach `tmp/papierkorb/` und legt sie neu an. Der Patch braucht
wayland-client ≥ 1.20 (`wl_output.name`).

Build-Tags: `novulkan` (keine Vulkan-Header nötig, OpenGL ES über EGL), `nox11` (nur Wayland, ADR-0007).
Gio braucht unter Linux cgo (libwayland-client ≥ 1.20, libwayland-egl, libwayland-cursor, libEGL, libxkbcommon).

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
