# ADR-0007: Nur Wayland, KDE zuerst, Aufnahme als austauschbare Schicht

## Status
Akzeptiert (Andi, 18.11.2026; übernommen am 19.11.2026)

## Kontext
Andi nutzt KDE Plasma unter Wayland. Unter Wayland darf ein Programm den Bildschirm nicht frei auslesen; Wege sind
die KWin-eigene Screenshot-Schnittstelle (D-Bus, ohne Dialog, für berechtigte Programme) und xdg-desktop-portal
(desktopübergreifend, mit Dialog bzw. Rückfrage).

## Entscheidung
- **Nur Wayland**, kein X11.
- **KDE zuerst** über die KWin-Schnittstelle: Bereich, Fenster, Bildschirm ohne Dialog.
- Die Aufnahme ist eine **austauschbare Schicht** (Go-Schnittstelle), der Rest des Programms kennt nur sie.
- Später ein Weg über **xdg-desktop-portal** für GNOME und andere.
- Die Einzelheiten der KWin-Schnittstelle (Version, Berechtigung über `.desktop`-Datei, Rückgabeformat, Fenster
  per Klick, Mauszeiger) werden zu Projektbeginn geprüft und als Nachtrag hier festgehalten (offene Entscheidung E-02).

## Konsequenzen
- Für die Berechtigung braucht Schnapp vermutlich eine installierte `.desktop`-Datei; `go run` allein reicht für die
  Aufnahme dann nicht (im Prototyp prüfen).
- Das Vollbild-Overlay zur Auswahl ist eine eigene Wayland-Frage (Layer-Shell oder Vollbildfenster über allen
  Monitoren) und Teil des Gio-Prototyps (ADR-0003).

## Nachtrag 1 (19.11.2026): KWin-Schnittstelle geprüft (E-02)

Einzelheiten: `knowledge/topics/kwin-screenshot.md`. Festgelegt:
- Aufnahme über `org.kde.KWin.ScreenShot2` (KWin 6.8.3 auf dem Laptop), Pixel roh über eine Pipe.
- **Erst aufnehmen, dann auswählen:** ganzer Arbeitsbereich bzw. je Bildschirm aufnehmen, Auswahl im eingefrorenen Bild
  (Vollbild-Overlay). Fenster per Klick über `CaptureInteractive`.
- Berechtigung über eine installierte `.desktop`-Datei mit absolutem `Exec=` und
  `X-KDE-DBUS-Restricted-Interfaces=org.kde.KWin.ScreenShot2`. Ob ein Eintrag unter `~/.local/share/applications`
  genügt, klärt der Prototyp; sonst Installation nach `/usr/local` (Andi, root).
- D-Bus über `github.com/godbus/dbus/v5` (ADR-0011: spart FD-Übergabe und Protokoll).
