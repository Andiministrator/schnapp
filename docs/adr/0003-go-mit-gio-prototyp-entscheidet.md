# ADR-0003: Go mit Gio, ein früher Prototyp entscheidet

## Status
Akzeptiert (Andi, 18.11.2026; übernommen am 19.11.2026)

## Kontext
Schnapp braucht eine schnelle, flüssig zeichnende Oberfläche (Vollbild-Overlay, Ziehen und Biegen von Pfeilen,
Ebenen) unter Wayland. Andi hat bereits Go-Code für SVG und Rasterung (Zeichenpakete, ADR-0012). Gio ist eine reine
Go-Oberfläche mit eigenem Rendering und Wayland-Unterstützung, aber jung und mit wenig fertigen Bausteinen.

## Entscheidung
- Sprache **Go**, Oberfläche **Gio** (gioui.org).
- Ein **früher Prototyp** prüft, ob Gio trägt. Er muss zeigen:
  1. einen echten Screenshot laden und anzeigen,
  2. einen Pfeil ziehen und mit einem Griff **biegen** (Bézier), flüssig,
  3. ein **Vollbild-Overlay** zur Bereichsauswahl über allen Monitoren (Wayland, KDE),
  4. eine **Aufnahme** über KWin anstoßen und das Ergebnis im Editor öffnen.
- Trägt Gio nicht (Overlay unter Wayland nicht machbar, Zeichnen ruckelt, Eingabe unzuverlässig), wird **früh auf
  Rust** gewechselt — vor der ersten Fassung, als neue ADR.

## Konsequenzen
- Bis der Prototyp entschieden hat, entsteht kein Produktcode außerhalb des Prototyps und der Zeichenpakete.
- Das Prototyp-Ergebnis wird als Nachtrag hier festgehalten (trägt / trägt nicht, mit Messwerten und Begründung).
- Bei einem Wechsel zu Rust bleibt das Go-Modul mit `svg`/`raster`/`geo` bestehen (ADR-0012); Schnapp
  müsste dann eine eigene Zeichenschicht bekommen.

## Nachtrag 1 (19.11.2026): Kriterien für den Prototyp (E-01) — Akzeptiert (Andi, 19.11.2026: „ok“)

Gio trägt, wenn alle Muss-Kriterien erfüllt sind. Gemessen auf Andis Laptop (KDE Plasma 6.8, Wayland), Ergebnis mit
Zahlen als Nachtrag 2.

**Muss**
1. **Aufnahme:** Schnapp holt über KWin ScreenShot2 ein Bild aller Bildschirme ohne Dialog (installierte
   `.desktop`-Datei erlaubt) und zeigt es pixelgenau (Skalierung beachtet).
2. **Overlay:** Ein randloses Vollbildfenster je Monitor (alle Monitore, auch mit unterschiedlicher Skalierung) zeigt
   das eingefrorene Bild; Bereich aufziehen mit Maus, Esc bricht ab. Kein Fenster-Rahmen, keine Lücke, keine Taskleiste
   darüber.
3. **Zeichnen:** Pfeil ziehen und über einen Griff zur Bézier-Kurve biegen, mit Kantenglättung, auf einem 4K-Bild
   **ohne sichtbares Ruckeln** (Bildrate beim Ziehen ≥ 50 fps, gemessen im Programm).
4. **Reaktionszeit:** Programmstart bis sichtbares Overlay ≤ 500 ms (Ziel; Snagit/Spectacle-Gefühl).
5. **Eingabe:** Maus, Tastatur (Esc, Enter, Strg+Z) und Mausrad-Zoom funktionieren zuverlässig unter Wayland.
6. **Ohne cgo** baubar (ADR-0011), oder cgo-Bedarf ist begründet und klein.

**Soll** (kein Ausschlussgrund, aber notiert): Text mit Systemschrift, Zwischenablage-Schreiben (Wayland), Speicherbedarf
im Leerlauf ≤ 150 MB.

**Abbruch zu Rust**, wenn Muss 2 oder 3 trotz eines Tages Arbeit nicht erreichbar ist.

## Nachtrag 2 (19.11.2026): Ergebnis des Prototyps — Gio trägt, mit Patch

**Status:** Akzeptiert (Andi, 19.11.2026: Variante „a“ — gebrochene Skalierung in den Gio-Patch; Handtest „hat alles
funktioniert“, „kein Stocken“).

Gemessen mit `prototyp/` (Karte #4) auf Andis Laptop: KDE Plasma 6.8, Wayland, KWin ScreenShot2 Version 5, drei
Monitore (zwei mit Skala 1, einer mit Skala 1,6). Gio v0.10.3, Go 1.27. Protokolle der Läufe lagen unter `tmp/`
(nicht versioniert); die Zahlen stehen hier.

| Kriterium | Ergebnis |
|---|---|
| Muss 1 Aufnahme | ✅ `CaptureScreen` je Bildschirm 17–38 ms, ohne Dialog, native Auflösung; Bild 1:1 auf allen drei Monitoren (Fenster in Pixeln = Aufnahme). |
| Muss 2 Overlay | ✅ **nur mit Patch**: ein randloses Vollbildfenster je Monitor, auch bei Skala 1,6 pixelgenau ab dem ersten Frame; Sichtprüfung Andi: alles da, keine Lücke, keine Taskleiste. |
| Muss 3 Zeichnen | ✅ 4K-Testbild, Pfeil biegt sich selbst: 73,5 fps, p95 13,9 ms. Von Hand: Ops-Aufbau ≤ 2,5 ms je Frame, höchstens 3–4 Mausbewegungen je Frame (Zusammenfassen bei ~95 Hz Maus und 60 Hz Bild); Andi: „kein Stocken“. |
| Muss 4 Reaktionszeit | ✅ Programmstart → erster Frame 122–131 ms, einschließlich drei Aufnahmen (Ziel ≤ 500 ms). |
| Muss 5 Eingabe | ✅ Maus, Esc, Enter, Strg+Z, Mausrad-Zoom unter Wayland. |
| Muss 6 cgo | ✅ begründet und klein: Gio braucht unter Linux cgo für libwayland-client, libEGL, libxkbcommon (alle in Flatpak-Laufzeiten). Mit Build-Tags `nox11,novulkan` keine weiteren Bibliotheken. |
| Soll | nicht geprüft: Systemschrift, Zwischenablage, Speicher im Leerlauf. |

### Was Gio nicht kann und der Patch nachrüstet

`prototyp/gio-ausgabe.patch` (rund 120 Zeilen von Hand, dazu erzeugter Protokoll-Code):

1. **Monitor wählen:** Gio ruft `xdg_toplevel.set_fullscreen` ohne Ausgabe auf und bindet `wl_output` nur in
   Version 2 (ohne Namen). Patch: `app.FullscreenOn(name)`, `app.WaylandOutputs()`.
2. **Gebrochene Skalierung:** Gio kennt nur ganzzahlige Skalen; bei 1,6 rendert es in 2 und KWin rechnet herunter
   (unscharf, nicht pixelgenau). Patch: `wp_fractional_scale_v1` + `wp_viewporter`; ohne diese Protokolle
   verhält sich Gio wie vorher.

### Weitere Befunde

- Die Aufnahme klappte **ohne** installierte `.desktop`-Datei. Ursache offen (F-15); nicht darauf verlassen.
- `CaptureWorkspace` rechnet bei gemischter Skalierung alles auf die größte Skala hoch — je Bildschirm einzeln
  aufnehmen (`knowledge/topics/kwin-screenshot.md`).

### Entscheidung

Schnapp bleibt bei **Go mit Gio**. Kein Wechsel zu Rust.

### Konsequenzen

- Schnapp trägt einen **Gio-Patch**, bis Gio die beiden Fähigkeiten selbst hat. Bau über eine gepatchte Kopie und
  `replace` (Muster: `prototyp/bauen.sh`); vor der ersten Fassung festlegen, wie der Patch ins Repo und in die CI
  kommt (Karte), und den Patch Gio anbieten (Karte).
- Jede Gio-Aktualisierung prüft, ob der Patch noch passt und noch nötig ist.
- Der Prototyp-Code ist Wegwerf-Code. Übernommen werden Erkenntnisse und Teile mit Tests (`format.go`, `pfeil.go`),
  nicht die Oberfläche.
