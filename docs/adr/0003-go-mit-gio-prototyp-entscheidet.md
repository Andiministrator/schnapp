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
