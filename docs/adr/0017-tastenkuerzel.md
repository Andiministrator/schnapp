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

## Nachtrag 1 (19.11.2026): Voreinstellung Druck belegt, Umsetzung der Desktop-Datei (Karte #30)

- **Belegt:** Spectacle trägt seine Voreinstellungen als `X-KDE-Shortcuts` in der `.desktop`-Datei, im Hauptteil
  (`X-KDE-Shortcuts=Print,Meta+Shift+S`) und je Aktion (etwa `[Desktop Action FullScreenScreenShot]` mit
  `X-KDE-Shortcuts=Shift+Print`). Quelle: `desktop/org.kde.spectacle.desktop.cmake` im Spectacle-Quelltext
  (https://invent.kde.org/plasma/spectacle/-/raw/master/desktop/org.kde.spectacle.desktop.cmake, gelesen am 19.11.2026);
  `src/ShortcutActions.cpp` benennt die globalen Kürzel ausdrücklich „to match the jumplist actions in our .desktop
  file“. Dass KDE (nicht Spectacle) die Zeile auswertet, folgt daraus, dass Spectacles Code keine Voreinstellung
  setzt; den Quelltext von KGlobalAccel haben wir nicht gelesen. Ein Test auf einer echten Plasma-Sitzung steht aus
  (Andi, bei der ersten Installation).
- **Umgesetzt:** `data/schnapp.desktop.in` mit den Aktionen `Bereich`, `Bildschirm`, `Fenster`, `LetztenBereich`;
  `X-KDE-Shortcuts=Print` nur an `Bereich`. `@EXEC@` ist der absolute Pfad der Programmdatei (KWin bindet die
  Aufnahme-Berechtigung daran, ADR-0007 Nachtrag 3). Spectacle belegt `Print` ebenfalls: gleichzeitig installiert,
  meldet KDE einen Konflikt, der Nutzer entscheidet in den Systemeinstellungen (Hinweis in README/Installation, sobald
  es ein Installationsskript gibt).
- **Vorläufig:** Die Aktion `Bildschirm` ruft `schnapp --bildschirm` ohne Wert; das gibt die Monitorliste aus. Mit der
  Umsetzung der Aufnahme wählt der Aufruf den Monitor selbst (Karte im Board).
