# ADR-0041: Stil je Werkzeug, über Sitzungen gemerkt

## Status
Akzeptiert (Claude Opus 5.5, 27.11.2026, Karte #383 in Sprint 14; Andi: „arbeite selbstständig“). Grundlage: Andi,
27.11.2026: „könnte man pro werkzeug die letzte einstellung merken? was ich nicht gut finde, dass die auswahl eines
werkzeugs beim wechsel auf das andere übergeht“. Ergänzt ADR-0021 Nachtrag 11 (Farbpaar je Werkzeug) und Nachtrag 26
(Schatten, bisher „ein Wert für alle Werkzeuge“), ADR-0018 (Konfigurationsdateien).

## Kontext
`canvas.Sitzung.Stil` hält die Einstellungen für neue Objekte: Farbe und Füllung, Strichstärke, Strichart, Spitzen,
Eckradius, Schrift mit Grad, Schnitt und Ausrichtung, Nummernradius, Schatten. Je Werkzeug gemerkt werden bisher nur
die zwei Farben (Farbpaar, ADR-0021 Nachtrag 11); alles andere teilen sich die Werkzeuge. Wer am Pfeil 12 px und
gestrichelt einstellt, zeichnet danach auch Kästen und Lupen so; wer beim Text einen Schatten einschaltet, hat ihn auch
am nächsten Pfeil. Beim Schließen geht alles verloren, jedes Fenster beginnt mit der Voreinstellung. Vorbilder: Snagit,
Greenshot und Photoshop merken die Werkzeugoptionen je Werkzeug über Sitzungen hinweg.

## Entscheidung
1. **Ein vollständiger Stil je Werkzeug.** Pfeil, Kasten, Text, Nummer und Lupe haben je einen eigenen `Stil`; Text und
   Sprechblase teilen sich einen (wie bisher das Farbpaar, `paarSchluessel`). Das Farbpaar je Werkzeug geht darin auf:
   Die Sitzung hält statt `paare` die Stile der übrigen Werkzeuge, der des zuletzt gewählten Zeichenwerkzeugs steht in
   `Stil`. Ein Wechsel des Werkzeugs legt `Stil` ab und holt den des neuen; ein Werkzeug, das noch keinen hat, beginnt
   mit der Voreinstellung (`StilMitFaktor`, Farben aus `standardPaar`). Auswählen, Schwärzen, Verpixeln, Zuschneiden und
   Stauchen haben keinen Stil; der Wechsel zu ihnen lässt `Stil` stehen.
2. **Änderungen an einem gewählten Objekt merkt das Werkzeug dieses Objekts**, wie bisher bei der Farbe („ein
   umgefärbter Kasten färbt auch die nächsten Kästen“): Strichstärke, Strichart, Spitzen, Schrift, Schatten,
   Schnellstil und Farben einer gewählten Beschriftung gehen in den Stil ihres Werkzeugs, nicht in den des gerade
   aktiven (`zielStil`). Der Schatten ist damit je Werkzeug (Nachtrag 26 sagte „ein Wert für alle Werkzeuge“).
3. **Datei `tools.ini`** im Konfigurationsordner neben `settings.ini`, `styles.ini` und `colors.ini`; sie gehört dem
   Programm und wird ganz neu geschrieben (atomar, 0600, Ordner 0700), Lesen wie `styles.ini` (höchstens 64 KiB, ohne
   Blockieren, Warnungen statt Abbruch). Ein Abschnitt je Werkzeug (`[arrow]`, `[box]`, `[text]`, `[step]`,
   `[magnifier]`), Schlüssel wie `styles.ini` (`color`, `fill`, `width`, `font`, `size`, `bold`, `italic`, `underline`,
   `strike`, `align`, `radius`, `shadow`), dazu `line_style` (solid/dashed/dotted), `start_head`/`end_head` (Namen wie
   im Format), `corner_radius` und `shadow_on` (yes/no; `shadow` trägt die Werte auch, wenn er aus ist). Ein ungültiger
   oder unbekannter Schlüssel fällt mit einer Warnung allein weg, der Rest des Abschnitts gilt; ein fehlender Schlüssel
   behält die Voreinstellung. Größen stehen auf Faktor 1 (wie `styles.ini`, ADR-0037 Punkt 1): Wer auf einem
   4K-Bild (Faktor 2) 12 px wählt, bekommt auf einem 1080p-Bild 6 px.
4. **Wann gelesen und geschrieben wird:** gelesen beim Öffnen des Editors (`neuerEditor`), geschrieben beim Schließen
   des Fensters. Beim Wechsel der Aufnahme im selben Fenster nimmt die neue Sitzung die Stile der alten mit (auf
   Faktor 1 umgerechnet). Mehrere Fenster: Es gewinnt, wer zuletzt schließt — es sind Vorlieben, keine Daten.
   Geschrieben wird nur, wenn sich seit dem Lesen etwas geändert hat. Schreibt das Schließen nicht (Fehler), sagt die
   Fehlerausgabe es; das Fenster schließt trotzdem.
5. **Schnellstile** bleiben, was sie sind (ADR-0037): Ein Klick auf eine Kachel setzt den Stil des Werkzeugs bzw.
   des gewählten Objekts, und das Werkzeug merkt ihn damit.

## Erwogene Alternativen
- **Nur in der Sitzung merken** (wie bisher die Farbpaare): erfüllt „beim Wechsel nicht übergehen“, aber nicht „die
  letzte Einstellung merken“ über das Schließen hinaus.
- **In `settings.ini`:** Die Datei gehört dem Nutzer und der Einstellungstafel (ADR-0034); ein Programm, das bei jedem
  Schließen hineinschreibt, verdrängte Kommentare und Hand-Änderungen. Wie `styles.ini` eine eigene Datei.
- **Bei jeder Änderung schreiben:** viele Schreibvorgänge beim Ziehen eines Reglers; ein Absturz verliert höchstens
  die Vorlieben dieser Sitzung.
- **Größen in Pixeln ohne Faktor:** Auf einem 4K-Bild gewählte 12 px wären auf einem kleinen Ausschnitt riesig.

## Konsequenzen
- `internal/canvas/werkzeugstil.go` (Stile je Werkzeug, `zielStil`, `WerkzeugStile`/`SetzeWerkzeugStile` auf
  Faktor 1, Umwandlung), `farbpaar.go` liest die Farben aus diesen Stilen.
- `internal/settings/werkzeugstile.go` (`LadeWerkzeugStile`, `SchreibeWerkzeugStile`), der Graph-Test
  `TestMCPErreichtDenSchreiberNicht` führt den neuen Schreiber als Ausnahme für die Datei, die ihn ruft.
- README en/de, CLAUDE.md; Tests und Mutanten.

## Nachtrag 1 (27.11.2026, Kritik #383, F-927 bis F-933)
1. **Faktor der Stile (F-927):** Die Sitzung merkt den Faktor, zu dem ihre Stile gerechnet sind (`stilFaktor`,
   `KantenFaktor` beim Anlegen). Zuschneiden und Stauchen ändern den Faktor des Dokuments, nicht die gemerkten Stile;
   `WerkzeugStile` teilt durch `stilFaktor`. Vorher schrieb ein Zuschnitt einer 4K-Aufnahme `width = 12` statt 6.
2. **Schatten einschalten an einem Objekt (F-928)** nimmt die Werte des Werkzeugs dieses Objekts, nicht die des aktiven;
   ebenso der Stand der Schnellstile.
3. **Grenzen (F-929):** Ein Wert aus `tools.ini`, der mal dem Faktor eine Grenze überschritte, wird auf die Grenze
   begrenzt statt verworfen.
4. **Rundung (F-930):** `WerkzeugStile` rundet Größen auf Tausendstel, damit ein Wechsel zwischen Faktoren die Datei
   nicht ohne Änderung neu schreibt.
5. **Abbruch eines Zugs (F-931):** Ein mit Esc oder Zeiger-Cancel abgebrochener Regler- oder Farbzug stellt auch den
   Stil des Werkzeugs her (vorher behielt das Werkzeug den gezogenen Wert). Abgebrochen heißt: nichts geschehen, auch
   nicht in `tools.ini`.
6. **Probe-Läufe (F-932):** Im Drehbuch-Lauf (`SCHNAPP_PROBE_LEISTE_AUS`, nur Probe-Build) wird `tools.ini` weder
   gelesen noch geschrieben.
