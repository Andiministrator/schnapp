# KWin-Screenshot-Schnittstelle (ScreenShot2)

Stand: 2026-11-19 — gemessen auf dem Laptop: KWin/Plasma 6.8.3. Quellen in `../sources-visited.md` (19.11.2026).
Entscheidung: ADR-0007 Nachtrag 1.

## Schnittstelle

D-Bus (Session-Bus) `org.kde.KWin`, Objekt `/org/kde/KWin/ScreenShot2`, Interface `org.kde.KWin.ScreenShot2`,
Property `Version` (u).

| Methode | Parameter außer `options a{sv}`, `pipe h` | seit |
|---|---|---|
| `CaptureArea` | `x, y i`, `width, height u` (logische Koordinaten, Ursprung oben links) | v1 |
| `CaptureScreen` | `name s` (Bildschirmname) | v1 |
| `CaptureActiveScreen` | — | v2 |
| `CaptureWindow` | `handle s` | v1 |
| `CaptureActiveWindow` | — | v2 |
| `CaptureInteractive` | `kind u` (0 Fenster, 1 Bildschirm) — KWin lässt den Nutzer klicken | v1 |
| `CaptureWorkspace` | — (alle Bildschirme zusammen) | v3 |

Optionen: `include-cursor` (false), `include-decoration` (false), `include-shadow` (true), `native-resolution`
(false), `hide-caller-windows` (true, ab v5).

Rückgabe `results a{sv}`: `type` = `"raw"`, `width`, `height`, `stride`, `format` (Zahl eines `QImage::Format`),
`scale` (devicePixelRatio, ab v4), `windowId` bzw. `screen` (ab v4). **Die Pixel kommen über die übergebene Pipe**,
roh, `stride * height` Bytes; der Aufrufer liest die Pipe bis EOF und deutet sie nach `format`.

Fehler u. a. `org.kde.KWin.ScreenShot2.Error.InvalidArea`, `.FileDescriptor`, `.Cancelled`; ohne Berechtigung
lehnt KWin den Aufruf ab („not authorized").

## Berechtigung

KWin erlaubt ScreenShot2 nur Programmen, deren **ausführbare Datei** (über die PID des D-Bus-Aufrufers ermittelt) in
einer installierten `.desktop`-Datei mit **absolutem** `Exec=`-Pfad steht und dort
`X-KDE-DBUS-Restricted-Interfaces=org.kde.KWin.ScreenShot2` trägt. Vorbild auf dem Laptop:
`/usr/share/applications/org.kde.spectacle.desktop` (`Exec=/usr/bin/spectacle`, genau diese Zeile).

Folgen für Schnapp:
- `go run` bekommt keine Aufnahme (Binärdatei liegt in einem Cache-Ordner). Für Entwicklung: bauen nach festen Pfad
  (z. B. `~/.local/bin/schnapp`) und `~/.local/share/applications/<id>.desktop` mit diesem absoluten Pfad. **Im
  Prototyp prüfen**, ob KWin `.desktop`-Dateien unter `~/.local/share/applications` dafür anerkennt (es gab
  Diskussionen, den Ort auf root-eigene Pfade zu beschränken).
- Wrapper-Skripte zählen nicht, es zählt die tatsächlich laufende Binärdatei.
- Flatpak später: eigener Weg (Portal), siehe ADR-0007.

## Folgerungen für den Entwurf

- **Erst aufnehmen, dann auswählen** (wie Spectacle): `CaptureWorkspace` oder je Bildschirm `CaptureScreen` mit
  `native-resolution`, dann das eingefrorene Bild im Vollbild-Overlay zeigen und darin den Bereich wählen. Damit muss
  das Overlay nicht durchsichtig über laufenden Fenstern liegen, und Lupe/Pixelraster arbeiten auf echten Pixeln.
- „Fenster per Klick“: `CaptureInteractive` mit `kind=0` (KWin übernimmt die Auswahl) — kein eigener Fensterzugriff
  nötig, Wayland gibt Programmen keine Fensterliste.
- Mehrere Monitore mit unterschiedlicher Skalierung: `scale` je Bild auswerten; Koordinaten von `CaptureArea` sind
  logisch.
- D-Bus aus Go: `github.com/godbus/dbus/v5` (Unix-FD-Übergabe für `h` wird unterstützt) — Fremdpaket nach ADR-0011.

## Globales Tastenkürzel (E-10)

Einfachster Weg: `.desktop`-Aktion (z. B. „Bereich aufnehmen“), die der Nutzer in den Systemeinstellungen an eine Taste
bindet; Schnapp wird mit Argument gestartet. Später xdg-desktop-portal `GlobalShortcuts`. Nicht weiter recherchiert.
