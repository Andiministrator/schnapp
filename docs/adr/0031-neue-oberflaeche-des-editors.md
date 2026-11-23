# ADR-0031: Neue Oberfläche des Editors — Menüzeile, Werkzeugleiste links, rechte Leiste, Fußzeile

## Status
Akzeptiert (Claude Opus 5.5, 23.11.2026, Karte #187 in Sprint 12). Grundlage: Wunsch Andi vom 22.11.2026 („das menü
oben möglichst minimieren …, links eine leiste mit werkzeugen …, rechts eine leiste mit status-infos, den details …
des jeweiligen werkzeugs“), Muster a/b („das menu oben passt“) und Muster v2 (`tmp/sprint12/muster/bilder/
187-oberflaeche-v2.png`, Andi: „perfekt“). Ändert ADR-0026 Punkt 1 und 3 (Knopf „Menü“, Hinweise unter den Knöpfen);
die übrigen Regeln von ADR-0026 (jede Taste auch per Maus, Kürzel sichtbar, allein per Tastatur bedienbar,
Tastentabelle mit Test) gelten weiter.

## Kontext
Die Leiste oben trägt heute neun Werkzeuge als Textknöpfe, Farbe, Rückgängig/Wiederholen, Ganzes Bild, Einpassen,
1:1, Exportieren, „Menü“ und „Speichern“. Sie bricht schon bei mittlerer Fensterbreite um (F-148) und lässt keinen
Platz für die Eigenschaften der Werkzeuge (Schrift, Farbe mit Deckkraft, Strichstärke), für Ebenen und für
Bildangaben. Die Statusleiste unten ist eine einzige Textzeile. Folgekarten brauchen Platz: Farbwähler (#189), Schrift
(#190), Ebenen (#191), Schnellstile (#192), Bibliothek (#174), Einstellungen (#175).

Gio hat kein natives Menü und kein globales Menü (dbusmenu); Menüzeile, Aufklapplisten und Hinweise zeichnet Schnapp
selbst (wie schon das Menü nach ADR-0026).

## Entscheidung

1. **Menüzeile oben**, eine Textzeile wie unter KDE und Windows, selbst gezeichnet, Höhe 30 dp:
   - ganz links das „s“ des Logos (nur das Zeichen, aus `brand/werkzeug/` erzeugt, nie nachgesetzt); es ist kein
     Knopf;
   - danach die Menüs **Datei, Bearbeiten, Ansicht, Bild, Werkzeuge, Hilfe**; ein Klick klappt die Liste darunter auf,
     solange eine offen ist, wechselt das Überfahren eines anderen Titels die Liste (wie in Qt);
   - rechts der Dateiname (Nebentext, gekürzt in der Mitte).
   Die Einträge sind die heutigen aus ADR-0026 Punkt 2, nach Menüs geordnet:
   - **Datei:** Speichern (Strg+S), Exportieren … (Strg+Umschalt+E), Kopieren (Strg+C), Schließen (Strg+W)
   - **Bearbeiten:** Rückgängig (Strg+Z), Wiederholen (Strg+Umschalt+Z, Strg+Y), Text bearbeiten (Enter), Löschen
     (Entf)
   - **Ansicht:** Einpassen (Strg+0), 1:1 (Strg+1)
   - **Bild:** Ganzes Bild, Gestauchtes zurückholen, Kantenform (mit der aktuellen Form im Text, F-425)
   - **Werkzeuge:** die neun Werkzeuge mit ihrem Buchstaben (das aktive markiert)
   - **Hilfe:** Tastenkürzel (zeigt die Tabelle aus `kuerzel.go`), Über Schnapp (Version)
   „Drehen/Spiegeln“ und „Bildgröße“ aus Muster a sind Ideen, nicht beauftragt.
   **Tastatur:** F10 öffnet das erste Menü mit Fokus auf dem ersten Eintrag (KDE-Gewohnheit); Pfeil links/rechts
   wechselt das Menü, hoch/runter/Pos1/Ende wie ADR-0026 Nachträge 2 bis 4, Esc schließt. Alt+Buchstabe
   (Mnemonics) gibt es nicht: Sie kollidierten mit den Werkzeugbuchstaben und müssten je Sprache gepflegt werden.
   F10 und die Pfeile links/rechts sind Menü-Navigation im Sinn von ADR-0026 Nachtrag 4 (benannte Ausnahme, Test).
2. **Werkzeugleiste links**, eine Spalte, Breite 52 dp: die neun Werkzeuge als Symbole (Karte #188) in
   Knöpfen mit Rand und Grund (Andi: „mit Umrandung und Hintergrund“), das aktive grün hinterlegt; Gruppen durch Abstand
   (Auswählen | Pfeil, Rechteck, Text, Nummer | Schwärzen, Verpixeln | Zuschneiden, Stauchen); darunter das Farbfeld
   (Vorder-/Hintergrund wie im Muster, Klick öffnet die Farbe in der rechten Leiste). Hinweis mit Name und Kürzel beim
   Überfahren und bei Tastaturfokus **rechts neben** dem Knopf (ersetzt ADR-0026 Punkt 3 „unter dem Knopf“). Bis die
   Symbole abgenommen sind (#188), stehen Buchstaben-Platzhalter in den Knöpfen; Bedienung und Test hängen nicht am
   Bild.
3. **Rechte Leiste**, Breite 300 dp, einklappbar (Ansicht → „Seitenleiste“, Strg+Umschalt+B — Kürzel in der Tabelle):
   - oben **Werkzeug bzw. Auswahl:** Überschrift „Werkzeug: Text“ bzw. „Auswahl: Pfeil“, darunter die Eigenschaften
     des Werkzeugs oder des gewählten Objekts (heute: Farbe; bei Stauchen die Kantenform). Weitere Eigenschaften kommen
     mit ihren Karten (Schrift #190, Strichstärke);
   - darunter **Farbe** (heute die Farbfelder des Knopfs „Farbe“; Farbfläche, Hex und Deckkraft mit #189);
   - darunter **Ebenen** (Liste der Objekte von oben nach unten, mit #191; bis dahin nur die Überschrift nicht — der
     Abschnitt erscheint erst mit #191);
   - Abschnitte haben eine Überschrift in Versalien (Nebentext), getrennt durch 1-px-Linien; keine Karten, keine
     Schatten (Karte #186).
4. **Fußzeile unten**, Höhe 30 dp, ersetzt die Statusleiste und den Block „Bild“ aus Muster a:
   - links die **Meldung** (Hinweise, Warnungen, Fehler; haltende Meldungen nach ADR-0024 Nachtrag 7 unverändert),
     sie nimmt den übrigen Platz und wird gekürzt;
   - rechts **Felder** von links nach rechts: Auswahlgröße (nur mit Auswahl), Zeigerort im Bild, Farbe mit Deckkraft,
     Schriftgröße (nur Text), Bildgröße (mit „gestaucht“), Zoom, Speicherstand („ungesichert“ in Orange,
     „gespeichert“). Felder mit Auswahl (Farbe, Schriftgröße, Zoom) sind anklickbar und klappen eine kleine Liste auf;
     die übrigen sind reine Anzeige. Ein Feld verschwindet, wenn es nichts zu sagen hat.
   Die Logik der Felder liegt ohne Gio in `canvas` (wie heute `StatusZeile`), damit sie ohne Fenster testbar ist.
5. **Leinwand** ist die Fläche dazwischen; Einpassen und Zoom zum Zeiger beziehen sich auf sie (`canvas.SetzeFenster`).
   Die Rechte Leiste ein- und ausklappen passt die Ansicht wie eine Größenänderung des Fensters an.
6. **Aktionen bleiben eine Liste:** Menüzeile, Werkzeugleiste, Fußzeile und Tasten führen dieselben `aktion`en über
   `editor.fuehreAus` aus. Der Test aus ADR-0026 Punkt 6 verlangt für jede Taste weiter einen Eintrag in einem Menü
   oder einen Knopf; er prüft jetzt alle Menüs der Zeile.
7. **Fokus:** F10 öffnet die Menüzeile wie in KDE; die Menütitel sind zudem per Tab erreichbar (Gio-Knöpfe sind
   fokussierbar), Enter und Leertaste öffnen dann das Menü wie der frühere Menüknopf (ADR-0026 Nachträge 2 und 3).
   Fokusring 2 px orange, eckig.
8. **Gestaltung:** eckig (Karte #186: kein Radius an Knöpfen, Feldern, Listen, Hinweisen), Farben aus dem Corporate
   Design, Grün für das aktive Werkzeug und die Hauptaktion, Orange nur als Akzent (Fokus, „ungesichert“, „s“).
   Schrift wie bisher (Go-Schriften bis zum Wechsel auf Open Sans, eigener Schritt). Kein Speichern-Knopf mehr in einer
   Leiste: Speichern steht im Menü Datei und als Strg+S, der Stand steht in der Fußzeile; Exportieren ebenso.
9. **Umsetzung in Schritten**, jeder mit Tests und Sichtprobe im virtuellen KWin (`scripts/probe.sh`):
   (a) Rahmen: Menüzeile, Werkzeugleiste links mit Platzhaltern, Fußzeile mit Feldern, alte Leiste weg;
   (b) rechte Leiste mit Werkzeug/Auswahl und Farbe (heutige Farbwahl); (c) Symbole (#188) nach Abnahme;
   (d) Folgekarten füllen die Leiste (#189 Farbe, #190 Schrift, #191 Ebenen).
   Drehbücher mit festen Koordinaten (Menü, Rückfall, Kanten) werden in Schritt (a) neu gefahren und angepasst.

## Konsequenzen
- `internal/ui/editor.go` verliert `leistenKnoepfe`/`ordneLeiste`/`zeichneLeiste` und die Umbruch-Logik der Leiste
  (`canvas.LeisteAnordnen`, F-148); neu: `menuezeile.go`, `werkzeugleiste.go`, `fusszeile.go`, `seitenleiste.go` in
  `internal/ui`, Logik ohne Gio in `internal/canvas` (Fußzeilenfelder, Menüstruktur).
- Neue Oberflächentexte (Menütitel, Abschnitte, Felder) mit `de.po`.
- F-425 (aktuelle Kantenform sichtbar) wird mit dem Menü „Bild“ und der rechten Leiste gelöst; F-411 (Ort der
  Rettungskopie lesbar) hat seit ADR-0029 Nachtrag 8 eine kurze Form und braucht hier nichts mehr.
- Probe-Drehbücher mit festen Koordinaten brechen und werden nachgezogen (`knowledge/topics/lehren.md`: feste
  Koordinaten nach jeder Leistenänderung neu fahren).
- ADR-0026 Punkte 1 und 3 sind durch Punkte 1 und 2 hier ersetzt.

## Nachtrag 1 (23.11.2026): Schritt (a) umgesetzt

**Status:** Akzeptiert (Claude Opus 5.5, Karte #187).

- Menüzeile mit „s“ (Maße aus `internal/icons/logo.go`, gegen `brand/logo/svg/schnapp-s-dunkel.svg` getestet), sechs
  Menüs **Datei, Bearbeiten, Ansicht, Bild, Werkzeuge, Hilfe**; „Hilfe“ führt die Übersicht der Tastenkürzel (F1,
  aus Tastentabelle und Menüeinträgen erzeugt, Esc oder Klick schließt) und „Über Schnapp“ (Version als Meldung). Der Eintrag „Kantenform“ nennt die aktuelle Form (F-425, Teil 2). Werkzeugleiste
  links mit den Symbolen aus `internal/icons` (Karte #188, Muster noch nicht abgenommen; Änderungen dort wirken ohne
  Umbau). Fußzeile mit Feldern aus `canvas.FussFelder`; das Farbfeld nimmt bis Schritt (b) die nächste Farbe, das
  Zoomfeld wechselt zwischen Einpassen und 1:1 (statt einer Liste).
- Reihenfolge im Bild: Menü vor Leisten ausgewertet, damit ein Menüeintrag, der das Speichern startet, Klicks auf
  Leisten im selben Bild sperrt (F-280).
- `canvas.LeisteAnordnen` (Umbruch der alten Leiste, F-148) ist entfallen.
- Drehbücher mit festen Koordinaten der alten Leiste (`rueckfall-menue*.txt`, `rueckfall-leiste-schmal.txt` u. a.)
  sind noch nicht nachgezogen; Sichtprobe des neuen Rahmens: `scripts/drehbuecher/oberflaeche-187.txt`.

## Nachtrag 2 (23.11.2026): Stand nach der Sitzungs-QA, Abweichungen vom Entwurf (Karte #213, F-515)

**Status:** Akzeptiert (Claude Sonnet 5.5, Karte #213). Hält fest, was Schritt (a) tatsächlich tut, wo es vom Wortlaut
oben abweicht und was beim Prüfen als Fehler behoben wurde. Gilt neben Nachtrag 1.

**Dateien.** Die Konsequenzen oben nennen `menuezeile.go`, `werkzeugleiste.go`, `fusszeile.go`, `seitenleiste.go`; sie
sind **nicht** entstanden. Es gilt: Menüzeile, Werkzeugleiste, Fußzeile, Hinweise und das „s“ in
`internal/ui/oberflaeche.go`; Liste, Tastatur und Handlungen des Menüs in `internal/ui/menue.go`; Übersicht der Kürzel
in `internal/ui/hilfe.go`; Tastentabelle in `internal/ui/kuerzel.go`; Logik der Fußfelder ohne Gio in
`internal/canvas/fusszeile.go` (`FussFelder`, `ZeigerOrt`). Eine rechte Leiste gibt es noch nicht (Schritt b).

**Als Stand beschrieben (gewollt oder hingenommen):**
- **Überfahren wechselt die Liste nur bei einem per Maus geöffneten Menü.** Ein per Taste (F10, Enter, Leertaste)
  geöffnetes Menü bleibt, auch wenn der Zeiger ruhend über einem anderen Titel liegt; sonst sprängen Tastaturnutzer
  unfreiwillig um. Pfeil links/rechts wechselt dort das Menü.
- **Schriftfeld nicht anklickbar.** Oben steht „Felder mit Auswahl (Farbe, Schriftgröße, Zoom) sind anklickbar“. Heute
  sind es Farbe (nächste Farbe) und Zoom (Einpassen oder 1:1); die Schriftgröße ist nur Anzeige, bis es die Schrift-Liste
  gibt (Karte #190). Auch Farbe und Zoom klappen noch keine Liste auf (Schritt b).
- **Meldung vor Feldern.** Eine Meldung (Speichern, Warnung, Fehler) behält ihren Text bis zu zwei Dritteln der
  Breite, die Felder weichen von links her; der bloße Bedienhinweis des Werkzeugs weicht den Feldern (mindestens ein
  Drittel bleibt ihm). Ein Feld, das nicht mehr passt, fällt ganz weg, statt gekürzt zu werden.
- **Zeigerort in Pixeln des Originals** (wie Auswahlgröße und Objektkoordinaten), nicht der Ansicht: Bei Zuschnitt ab
  (10,10) liegt Ansichtspunkt (2,2) bei Original (12,12). Beim Zuschneiden, wo die Leinwand die Vollansicht zeigt, gilt
  deren Fläche, das Feld steht auch über dem abgedunkelten Teil. Das Feld rechnet bei jedem Bild aus der Fensterposition
  des Zeigers und veraltet nach Zoom per Taste nicht (F-513).
- **Werkzeugleiste in kleinen Fenstern** schneidet Knöpfe ab: eigenes Finding F-516, noch offen (Entscheidung mit
  Schritt b).

**Beim Prüfen als Fehler behoben:**
- **Enter im Bild eines Menüwechsels** (F-512): Pfeil rechts und Enter im selben Bild lösten den Eintrag des
  Nachbarmenüs an derselben Stelle aus, weil der Fokus noch auf der Klickfläche des alten Menüs lag. Nach einem Wechsel
  wirkt Enter erst im nächsten Bild (`menue.gewechselt`).
- **Klick in die Menüzeile** abseits der Titel („s“, leere Fläche, Dateiname) schließt eine offene Liste wie ein Klick
  daneben (bisher blieb sie offen); ein Klick auf einen Titel bleibt, was er war: öffnen, wechseln oder schließen.
- **Hinweis am Farbfeld** erscheint, wie die Hinweise der Werkzeuge, nicht über einer offenen Liste.
- **Tab-Reihenfolge der Fußfelder:** Die Felder werden von rechts her gelegt, die Klickflächen kommen dennoch von links
  nach rechts an die Reihe. Tab geht damit Menütitel, Werkzeuge, Farbfeld der Leiste, Fußfeld Farbe, Fußfeld Zoom;
  vorher besuchte er Zoom vor Farbe.
- Kommentare im Code sagen „Menütitel“ und „Fußzeile“ statt „Menüknopf“ und „Statusleiste“; F-438 steht dort als
  ADR-0023 Nachtrag 9.
