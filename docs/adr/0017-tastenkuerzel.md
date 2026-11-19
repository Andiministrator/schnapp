# ADR-0017: Globale Tastenkürzel über die Desktop-Datei

## Status
Akzeptiert (Andi, 19.11.2026: E-10 Empfehlung, Taste Variante „a“ = Druck).

## Kontext
„Schnell zur Hand“ (Kernprinzip 4) heißt: ein Tastendruck, sofort das Overlay. Unter KDE gibt es drei Wege:
KGlobalAccel über D-Bus (Schnapp müsste laufen), das Portal `GlobalShortcuts` (ebenfalls laufender Prozess, dafür
auch GNOME), oder Aktionen in der `.desktop`-Datei, die KDE mit Kürzeln verbindet. Der Prototyp startet in rund
130 ms bis zum ersten Frame — ein neuer Prozess je Tastendruck ist schnell genug.

## Entscheidung
- **Erste Fassung:** `schnapp.desktop` mit Aktionen, jede startet Schnapp mit Argument:
  `Bereich` (`schnapp --bereich`), `Bildschirm`, `Fenster`, `Letzten Bereich wiederholen`.
- **Voreinstellung Druck** für „Bereich“ (Andi: Schnapp ist das Haupt-Werkzeug); Spectacle belegt Druck
  standardmäßig — dort wird das Kürzel abgeschaltet (Hinweis in README/Installation). Weitere Kürzel nur als
  Vorschlag, alles in den Systemeinstellungen änderbar.
- Ob KDE Voreinstellungen aus der `.desktop`-Datei übernimmt (`X-KDE-Shortcuts` je Aktion, vermutlich wie bei
  Spectacle), wird vor der Umsetzung am Spectacle-Quelltext belegt; sonst richtet der Nutzer das Kürzel einmal ein.
- **Später:** Portal `GlobalShortcuts` für andere Desktops (ADR-0007).

## Konsequenzen
- Kein Hintergrunddienst, kein eigener Kürzel-Code in der Ersten Fassung.
- Die `.desktop`-Datei trägt zugleich die Berechtigungszeile für Plasma ≤ 6.7 (ADR-0007 Nachtrag 3) und die
  Übersetzungen `Name[de]=…` (ADR-0014).
- „Letzten Bereich wiederholen“ braucht den zuletzt gewählten Bereich auf Platte (`~/.local/state/schnapp/`).
