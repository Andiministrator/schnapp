# KWin-Screenshot-Schnittstelle (ScreenShot2)

Stand: 2026-11-19 — gemessen auf dem Laptop: KWin/Plasma 6.8.3. Quellen in `../sources-visited.md` (19.11.2026);
Berechtigung geklärt am 19.11.2026 (F-15, `../sources/kwin-screenshot-berechtigung.md`).
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

Fehler u. a. `org.kde.KWin.ScreenShot2.Error.InvalidArea`, `.FileDescriptor`, `.Cancelled`; bis KWin 6.7 ohne
Berechtigung auch `.NoAuthorized` („The process is not authorized to take a screenshot“).

## Berechtigung

**Ab KWin 6.7.90 (Plasma 6.8.0) gibt es keine Prüfung mehr.** KWin-Commit `0ea31ae7` vom 18.06.2026
(„plugins/screenshot: Drop X-KDE-DBUS-Restricted-Interfaces“) hat sie entfernt, Begründung des Autors sinngemäß:
`.desktop`-Dateien kann der Nutzer ändern oder überschreiben, die Prüfung bringe daher keinen Schutz; Schutz biete
eher eine Sandbox wie Flatpak. Enthalten in den Tags v6.7.90 bis v6.8.3. Jedes Programm in der Sitzung darf
ScreenShot2 aufrufen. Damit ist geklärt, warum die Aufnahme im Prototyp ohne `.desktop`-Datei ging (F-15).

**Bis KWin 6.7** (aus dem entfernten Code `checkPermissions`): KWin ermittelte die PID des D-Bus-Aufrufers und
erlaubte den Aufruf nur, wenn die ausführbare Datei in einer installierten `.desktop`-Datei mit **absolutem**
`Exec=`-Pfad steht und dort `X-KDE-DBUS-Restricted-Interfaces=org.kde.KWin.ScreenShot2` trägt (Vorbild:
`org.kde.spectacle.desktop`, `Exec=/usr/bin/spectacle`). Wrapper-Skripte zählen nicht. Ausnahme: Umgebungsvariable
`KWIN_SCREENSHOT_NO_PERMISSION_CHECKS=1` im KWin-Prozess schaltete die Prüfung ab.

Folgen für Schnapp:
- Unter Plasma ≥ 6.8 braucht die Aufnahme keine Berechtigung; `go run` geht.
- Für Plasma 6.7 und älter installiert Schnapp seine `.desktop`-Datei mit absolutem `Exec=` und der Zeile
  `X-KDE-DBUS-Restricted-Interfaces=org.kde.KWin.ScreenShot2` — die Datei braucht es ohnehin (Programmmenü,
  Tastenkürzel E-10), der Eintrag schadet ab 6.8 nicht. Vorlage: `prototyp/schnapp-prototyp.desktop.in`.
- Fehlerweg `org.kde.KWin.ScreenShot2.Error.NoAuthorized` behandeln: verständliche Meldung („Schnapp ist nicht als
  Aufnahmeprogramm eingetragen …“) statt stillem Scheitern.
- Flatpak später: eigener Weg (Portal), siehe ADR-0007.

## ScreenShot3 mit memfd (nur KWin-master, Stand 19.11.2026; umgesetzt in `aufnahme/`)

KWin-Commit `5be04288` vom 17.11.2026 („plugins/screenshot: Share screenshots with memfd“): Objekt
`/org/kde/KWin/ScreenShot3`, Interface `org.kde.KWin.ScreenShot3`, Property `Version` (u). Dieselben Methoden und
Optionen wie ScreenShot2, aber ohne `pipe h` als Eingabe — die Methoden **geben** `fd h` (memfd) und danach
`results a{sv}` zurück, in dieser Reihenfolge; Schlüssel wie bei ScreenShot2. Laut Commit spart das 4–13 ms Kopieren
je Aufnahme. **In keinem Release-Tag enthalten** (vermutlich Plasma 6.9); auf dem Laptop (Plasma 6.8.3) fehlt das
Objekt, ScreenShot2 meldet Version 5.

Umsetzung (`aufnahme/`, Karte #27): beim ersten Aufruf `Version` von ScreenShot3 abfragen, sonst ScreenShot2; memfd
per fstat prüfen und genau stride×(height−1)+width×4 Bytes per ReadAt lesen (kein mmap). Grenzen: Seite ≤ 32768,
Zeile ≤ 262144 Bytes, Rohdaten ≤ 1 GiB (die Pipe wird vor der Antwort gelesen, darum feste Obergrenze); Zeitgrenze
10 s je Aufruf, 5 min bei `CaptureInteractive` (der Nutzer klickt). Fehlerwerte: `ErrNichtBerechtigt` (NoAuthorized),
`ErrAbgebrochen` (`org.kde.KWin.ScreenShot2.Error.Cancelled`), `ErrKeineSchnittstelle`. ScreenShot3 meldet dieselben
Fehler unter eigenem Namensraum (`org.kde.KWin.ScreenShot3.Error.Cancelled`, laut `screenshotdbusinterface3.cpp`);
die Schicht erkennt beide (F-49). Gegen echtes KWin noch nicht
gelaufen (Stand Karte #27).

## Gemessen im Prototyp (19.11.2026, Plasma 6.8, drei Monitore, einer mit Skala 1,6)

- **Aufnahme ging ohne installierte `.desktop`-Datei** — erklärt durch den Wegfall der Prüfung in Plasma 6.8
  (Abschnitt Berechtigung, F-15 erledigt).
- `CaptureScreen` je Bildschirm 17–38 ms, `CaptureWorkspace` über alle drei 127 ms. Mit `native-resolution`
  liefert `CaptureWorkspace` bei gemischter Skalierung ein Bild in der **größten** Skala (Monitore mit Skala 1
  werden hochgerechnet) — für das Overlay deshalb je Bildschirm einzeln aufnehmen.
- Das Format kam als QImage-Format 16 (RGBX8888, Byte-Reihenfolge R G B x); Umwandlung aller sechs
  32-Bit-Formate in `prototyp/format.go`.

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
