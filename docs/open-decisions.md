# Offene Entscheidungen

Stand 19.11.2026. Je Eintrag: Frage · Impact · Tendenz · muss entschieden sein vor.

## E-01: Wann hat der Gio-Prototyp bestanden? — ENTSCHIEDEN 19.11.2026 (ADR-0003 Nachtrag 1)
- **Frage:** Messbare Kriterien für „Gio trägt" (ADR-0003): z. B. Pfeil biegen ohne sichtbares Ruckeln auf 4K, Overlay
  über alle Monitore unter KWin, Reaktionszeit Tastenkürzel → Overlay.
- **Impact:** hoch — entscheidet Go oder Rust.
- **Tendenz:** Kriterien vor dem Prototyp festschreiben (Nachtrag ADR-0003), Andi bestätigt.
- **Vor:** Beginn des Prototyps.

## E-02: KWin-Schnittstelle im Einzelnen — ENTSCHIEDEN 19.11.2026 (ADR-0007 Nachtrag 1, Prototyp-Ergebnis Nachtrag 2; Berechtigung: F-15)
- **Frage:** Welche D-Bus-Schnittstelle (`org.kde.KWin.ScreenShot2`), welche Berechtigung (`.desktop`-Eintrag
  `X-KDE-DBUS-Restricted-Interfaces`?), welches Rückgabeformat, wie „Fenster per Klick", Mauszeiger, mehrere Monitore
  und Skalierung? Ab welcher Plasma-Version?
- **Impact:** hoch — bestimmt, ob Aufnahme ohne Dialog geht und wie installiert werden muss.
- **Tendenz:** aus dem öffentlichen KWin-Quelltext und der Doku prüfen (Spurensicherung), Ergebnis als Nachtrag zu
  ADR-0007.
- **Vor:** Prototyp Schritt 4 (Aufnahme).

## E-03: Name und Aufbau des PNG-Blocks — ENTSCHIEDEN 19.11.2026 (ADR-0008 Nachtrag 1)
- Zwei Blöcke: `snAP` (SVG, komprimiert), `snAO` (Original binär); nicht kopiersicher (4. Buchstabe groß).
- **Frage:** Chunk-Name (privat, nicht sicherheitskritisch, kopierbar?: Groß-/Kleinschreibung der vier Buchstaben
  bestimmt das), komprimiert oder nicht, Original als eingebettetes PNG im SVG oder als eigener Chunk?
- **Impact:** mittel — einmal veröffentlicht, schwer zu ändern.
- **Tendenz:** eigener privater Chunk, SVG komprimiert, Original als `<image>` mit Daten-URI im SVG (ein Block).
- **Vor:** erste Fassung, Dateiformat.

## E-04: Modulpfad und Ort der Zeichenpakete — ENTSCHIEDEN 19.11.2026 (ADR-0012)
- Eigenes Modul `github.com/Andiministrator/schnapp/zeichnen` im Ordner `zeichnen/`.

## E-05: CI — ENTSCHIEDEN 19.11.2026: GitHub Actions (ADR-0010)
- Eingerichtet 19.11.2026: `.github/workflows/test.yml` (gofmt, vet, test für `zeichnen/`), grün.

## E-06: Lizenzvariante — ENTSCHIEDEN 19.11.2026: GPL-3.0-or-later (ADR-0009)
- Rechte an den Zeichenpaketen liegen beim Maintainer; Veröffentlichung unter GPL-3.0 erlaubt (Andi, 19.11.2026).

## E-07: Englische Oberfläche und README — ENTSCHIEDEN 19.11.2026 (ADR-0014)
- Oberfläche Englisch + Deutsch ab Start, Systemsprache erkennen, sonst Englisch; Nutzer-Doku zweisprachig.

## E-08: Namensprüfung vor Veröffentlichung — GEPRÜFT 19.11.2026, kein Hindernis
- **Ergebnis:** AUR und Arch-Paketquellen: kein `schnapp`. GitHub: 129 Treffer mit „schnapp“ im Namen, davon ein
  ähnliches Werkzeug `pricerc/schnapp` (periodische Screenshots mit Web-Annotation, 0 Sterne, ohne Paket);
  `schnapps` ist ein Snapshot-Werkzeug von Turris OS (Router, anderes Feld). Flathub-Suche über die API nicht
  auswertbar, Websuche ohne Treffer. Marke nicht geprüft. Vor einer Flathub-Einreichung dort erneut suchen.
- **Frage:** Gibt es „Schnapp" auf GitHub, Flathub, in Debian/Arch/Fedora, als Marke?
- **Impact:** mittel — Umbenennung nach Veröffentlichung ist teuer.
- **Tendenz:** Prüfung kurz vor dem ersten öffentlichen Push, Ergebnis in Spurensicherung.
- **Vor:** Flathub-Einreichung.

## E-09: Bibliothek — Ort und Format — ENTSCHIEDEN 19.11.2026 (ADR-0016)
- `~/Bilder/schnapp/JJJJ-MM/`, Index `~/.local/share/schnapp/bibliothek.jsonl` als Cache, Aufräumen nach einstellbaren
  Tagen in den Papierkorb (abschaltbar, Voreinstellung 90 Tage, nur ohne Schlagworte/Bearbeitung). `schnapp` in Pfaden klein.
- **Frage:** Wo liegen Aufnahmen (`~/Bilder/Schnapp`? XDG-Datenverzeichnis?), Index als Datei oder SQLite (cgo!),
  wie werden Quellprogramm und Schlagworte gespeichert, Aufbewahrung/Aufräumen?
- **Impact:** mittel — Datenschutz (Aufnahmen enthalten oft Vertrauliches), Rechte 0600/0700.
- **Tendenz:** Bilder als Arbeitsdateien unter XDG-Bilderordner, Index als JSON-Zeilen-Datei, kein cgo.
- **Vor:** erste Fassung, Bibliothek.

## E-10: Globales Tastenkürzel unter KDE — ENTSCHIEDEN 19.11.2026 (ADR-0017)
- `.desktop`-Aktionen, Voreinstellung Druck für „Bereich“ (Spectacle-Kürzel abschalten); Portal später.
- **Frage:** Über KGlobalAccel (D-Bus), das Portal `GlobalShortcuts` oder eine `.desktop`-Aktion, die der Nutzer in
  den Systemeinstellungen bindet?
- **Impact:** mittel — Kern der Bedienung („schnell zur Hand").
- **Tendenz:** zunächst `.desktop`-Aktion + Kurzbefehl in den Systemeinstellungen (am einfachsten), später Portal.
- **Vor:** erste Fassung.

## E-11: Board-Dateien im öffentlichen Repo? — ENTSCHIEDEN 19.11.2026: nein
- Board-Dateien und interne Angaben (`CLAUDE.local.md`) stehen in `.gitignore`; das Board ist über die Sicherung des
  Arbeitsrechners gesichert.

## E-12: Lizenz und Nutzungsregeln für Logo und Symbol — ENTSCHIEDEN 19.11.2026 (Andi folgt der Tendenz)
- Code GPL; Logo mit kurzer Markenregel. Umsetzung: Markenregel in `marke/` und README vor der ersten Paket-Veröffentlichung.
- **Frage:** Bleiben Logo und Symbol unter GPL-3.0 wie der Code (jeder darf sie ändern und weiterverwenden), oder
  bekommen sie eine eigene Lizenz (z. B. CC BY-SA) bzw. Regeln zur Markennutzung (Abwandlungen dürfen nicht
  „Schnapp“ heißen)?
- **Impact:** mittel — betrifft Forks und Pakete in Distributionen; nachträglich schwer zurückzuholen.
- **Tendenz:** Code GPL, Logo mit kurzer Markenregel (frei nutzbar, um auf Schnapp zu verweisen; Abwandlungen nicht
  als Schnapp ausgeben) — wie bei vielen freien Projekten.
- **Vor:** erste Veröffentlichung eines Pakets (Flathub).

## E-13: Ansprache in der deutschen Oberfläche — ENTSCHIEDEN 19.11.2026 (Andi folgt der Tendenz)
- Möglichst ohne Anrede formulieren; wo nötig „du“.
- **Frage:** Spricht die deutsche Übersetzung den Nutzer mit „du“ oder „Sie“ an?
- **Impact:** niedrig, aber durchgängig; spätere Umstellung betrifft jeden Text.
- **Tendenz:** möglichst ohne Anrede formulieren („Bild gespeichert“, „Bereich wählen“); wo nötig „du“ (freundlich,
  passt zur Haltung im Corporate Design).
- **Vor:** erste Oberflächentexte (Karte Übersetzbarkeit).
