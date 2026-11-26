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

## Nachtrag 3 (23.11.2026): Schritt (b) umgesetzt — rechte Leiste, Listen der Fußzeile, kleine Fenster (F-516)

**Status:** Akzeptiert (Claude Opus 5.5, Karte #187).

- **Rechte Leiste** (`internal/ui/seitenleiste.go`, Inhalt ohne Gio in `canvas.Sitzung.Seitenleiste`), 300 dp:
  - Kopf in Versalien: „Auswahl: <Art>“, wenn ein Objekt gewählt ist (es geht dem Werkzeug vor, denn die Farbwahl färbt
    es um), sonst „Werkzeug: <Name>“.
  - Eigenschaften heute: beim Stauchen die **Kantenform** als drei Knöpfe Abriss/Welle/Zacken (aktive grün; ohne Naht
    grau wie der Menüeintrag, F-426), gesetzt über `canvas.Sitzung.SetzeKante` als ein Rückgängig-Schritt, dieselbe
    Form erneut ist kein Schritt. Darunter ein Hinweis im Nebentext: nichts ausgewählt (Auswählen), Pixel endgültig
    (Schwärzen, Verpixeln), Weggenommenes bleibt in der Arbeitsdatei (Zuschneiden, Stauchen), Kantenform wirkt erst mit
    einer Naht.
  - Abschnitt **Farbe** nur, wo eine Farbe wirkt (Pfeil, Kasten, Text, Nummer oder ein gewähltes solches Objekt): die
    vier Beschriftungsfarben als Felder (gewählte mit grünem Rand), Hex und Deckkraft als Anzeige. Farbfläche, Hex-Eingabe
    und Deckkraft-Regler kommen mit #189.
  - Ebenen (#191), Schrift (#190) und Strichstärke fehlen noch; die Leiste wächst mit ihren Karten.
- **Einklappen:** Ansicht → „Seitenleiste“ (Haken als grüner Text) und Strg+Umschalt+B (Tastentabelle; Strg+Umschalt
  unterscheidet es vom Werkzeug B). Die Leinwand wird um die Breite der Leiste schmaler oder breiter, Einpassen folgt.
  **Abweichung:** Ist das Fenster schmaler als Werkzeugleiste + 300 dp + 400 dp Leinwand (752 dp), bleibt die Leiste
  eingeklappt, ohne die Einstellung zu ändern; wird das Fenster breiter, erscheint sie wieder. Die Einstellung gilt je
  Fenster und wird nicht gespeichert (später mit #175).
- **Farbfeld der Werkzeugleiste:** Ein Klick klappt die eingeklappte Leiste auf, wenn sie Platz hat und die Farbe zeigt;
  sonst (Leiste schon offen, Fenster zu schmal, Werkzeug ohne Farbe) öffnet er die Farbliste am Farbfeld. Per Tastatur
  (Enter) öffnet es immer die Liste mit Fokus auf dem ersten Eintrag. Die Liste ersetzt das bisherige Weiterschalten
  der Farbe. *Seit Nachtrag 12:* per Maus zeigt der Klick den Farbwähler der Leiste, die Liste nur noch, wo die Leiste
  die Farbe nicht zeigen kann. *Seit ADR-0021 Nachtrag 11 (#219):* Das Farbfeld ist ein Farbpaar (Vordergrund oben links, Füllung unten
  rechts, ⇄ oben rechts); ein Klick auf ein Feld macht es aktiv und wirkt dann wie hier beschrieben, das graue Feld
  Füllung tut nichts. Tab: Vordergrund, Füllung, ⇄; in der rechten Leiste steht unter der Überschrift „Farbe“ die Zeile
  Vordergrund, Füllung, ⇄, „Keine“ vor den Farbfeldern. *Seit ADR-0021 Nachtrag 23:* ohne Füllung nur das Feld des
  Vordergrunds, „Keine“ als Feld mit Schrägstrich.
- **Listen der Fußzeile:** Farbe und Zoom klappen eine Liste über dem Feld auf (▾ am Feld): Farbe mit Farbquadrat,
  Name und Hex; Zoom mit Einpassen (Strg+0), 25, 50, 100 (Strg+1), 200, 400 % (`canvas.ZoomStufen`,
  `canvas.Sitzung.SetzeZoom` um die Fenstermitte). Die aktuelle Wahl ist grün. Die Listen laufen über dieselbe
  Liste wie die Menüs (`menue.welches` = `listeFarbe`/`listeZoom` hinter den Menüs): Tastatur (hoch/runter/Pos1/Ende,
  Enter, Esc gibt den Fokus an das Feld zurück), Klick daneben, Sperre beim Speichern. Pfeil links/rechts wechselt dort
  kein Menü. Enter auf dem fokussierten Feld öffnet die Liste mit Fokus auf dem ersten Eintrag. Damit ist
  Nachtrag 2 „Farbe und Zoom klappen noch keine Liste auf“ erledigt; die Schriftgröße bleibt Anzeige bis #190.
- **Tab-Reihenfolge:** Menütitel, Werkzeuge, Farbfeld links, Farben (und Kantenknöpfe) der rechten Leiste, Fußfeld Farbe,
  Fußfeld Zoom.
- **Kleine Fenster (F-516):** Passt die Werkzeugleiste nicht zwischen Menüzeile und Fußzeile, werden Knöpfe, Abstände,
  Symbol und Farbquadrat gleichmäßig kleiner, bis zu Knöpfen von 20 dp; bei 480 × 320 passen damit alle neun Werkzeuge
  und das Farbfeld. *Seit Karte #194 (elf Werkzeuge mit Lupe und Sprechblase) bis zu 18 dp (`werkzeugKnopfMinDp`),
  ADR-0021 Nachtrag 13 „Ergänzt (Teil 2)“.* Erst unter 244 dp Leistenhöhe (Fenster unter 304 dp hoch, also unter der Mindesthöhe von 320 dp, F-252) schneidet sie unten ab; Werkzeuge und Farbe bleiben dann über das
  Menü „Werkzeuge“, die Buchstaben und die Fußzeile erreichbar. Hohe Fenster behalten die Maße aus Schritt (a).
- **Drehbücher:** Die Drehbücher mit festen Koordinaten klappen die Leiste am Anfang mit Strg+Umschalt+B ein; damit gilt
  die Lage der Leinwand aus Schritt (a) weiter. Sichtprobe von Schritt (b): `scripts/drehbuecher/oberflaeche-187b.txt`
  und `oberflaeche-187b-klein.txt` (480 × 320).

## Nachtrag 4 (23.11.2026): Strichstärke, Tastatur und Kleinkram nach der Sitzungs-QA (Kritiker A, A-1 bis A-11)

**Status:** Akzeptiert (Claude Opus 5.5). Ergänzt Nachtrag 3.

- **Strichstärke** steht seit #216 in der Leiste (− / + für Pfeil und Kasten, ADR-0021 Nachträge 7 und 8); der Satz in
  Nachtrag 3 „Ebenen, Schrift und Strichstärke fehlen noch“ gilt nur noch für Ebenen (#191) und Schrift (#190).
- **Tab-Reihenfolge:** Menütitel, Werkzeuge, Farbfeld links, Strich − und + (Pfeil, Kasten), Kantenknöpfe (Stauchen),
  Farben der rechten Leiste, Fußfeld Farbe, Fußfeld Zoom.
- **Anker der Listen (A-1):** Die Zoomliste hängt immer am Zoomfeld; nur die Farbliste kann am Farbfeld links hängen.
- **Tastatur (A-5, A-6):** Ein grauer Stufenknopf und ein grauer Kantenknopf bleiben fokussierbar (ein Klick tut dort nichts; Kantenknöpfe seit dem Verify, N-9), damit der Fokus
  nach dem Schritt auf die letzte Stufe nicht ins Leere fällt. Strg+Umschalt+B gibt beim Einklappen den Fokus an die
  Leinwand. Die Leertaste an den Feldern mit Liste (Fußfelder Farbe und Zoom, Farbfeld links) öffnet die Liste wie
  Enter, mit Fokus auf dem ersten Eintrag; Esc gibt ihn ans Feld zurück (`editor.leertastenZiele`).
- **Hilfetafel (A-8):** Solange die Übersicht der Tastenkürzel offen ist, wirken Klicks in die rechte Leiste nicht.
- **Sichtbarkeit (A-9):** Die Farbfelder der Leiste haben einen hellen 1-px-Rand (Dunkelgrau auf grau-800).
- **Drehbücher (A-11):** Die Zeile „taste Strg+Umschalt+B“ am Anfang der Drehbücher ist ein Umschalter und gilt, weil
  die Leiste beim Start immer offen ist (die Einstellung wird nicht gespeichert). Speichert #175 die Einstellung, muss
  sie durch ein Zustandssetzen ersetzt werden (Karte #175).

## Nachtrag 5 (23.11.2026): Farbwähler in der rechten Leiste (Karte #189)

**Status:** Akzeptiert (Claude Opus 5.5). Füllt den Abschnitt „Farbe“ aus Nachtrag 3 nach dem Muster v2; Bedeutung der
Farbe und der Deckkraft in ADR-0021 Nachtrag 9.

- **Aufbau** (von oben): links die **Farbfläche** (Sättigung nach rechts, Hellwert nach oben, zum Farbton des Reglers),
  rechts daneben ein Raster aus 2 × 4 Feldern: die vier Beschriftungsfarben, bis zu drei zuletzt benutzte Farben (leere
  Plätze bleiben leer) und als letztes Feld die **Pipette** (*seit ADR-0021 Nachtrag 17:* vier zuletzt benutzte, die
  Pipette rechts in der Zeile des Farbpaars, Hex immer sechsstellig; *seit ADR-0021 Nachtrag 24:* das Raster zeigt nur
  die acht zuletzt benutzten, eigene Farbfelder stehen unter Hex und Deckkraft). Darunter über die volle Breite der **Farbton-Regler** und
  der **Deckkraft-Regler** (Schachbrett unter einem Verlauf von durchsichtig zur Farbe), darunter die Felder **Hex**
  (`#RRGGBB`, bei Deckkraft unter 100 % `#RRGGBBAA`) und **Deckkraft** (`100 %`).
- **Maus:** Drücken und Ziehen in Fläche oder Regler setzt die Farbe laufend (Marke folgt dem Zeiger, auch außerhalb
  begrenzt auf den Rand), Loslassen schließt den Schritt ab. Die gewählte Farbe hat einen grünen Rand; ein Feld, das
  halbdurchsichtig ist, zeigt das Schachbrett darunter.
- **Eingabefelder:** Ein Klick oder Tab fokussiert das Feld (orangefarbener Rand, Schreibmarke am Ende, der Text wird
  beim Fokussieren ersetzt, sobald getippt wird). Enter übernimmt, Esc verwirft und gibt den Fokus an die Leinwand, Tab
  oder ein Klick woanders übernimmt ebenfalls — der Klick übernimmt, **bevor** er selbst wirkt (eine durchlässige
  Fläche über dem Fenster sieht jeden Druck; danach liegt der Fokus an der Leinwand). Ein Klick auf ein Farbfeld, die
  Pipette, die Farbfläche oder einen Regler **verwirft** die schwebende Eingabe dagegen, denn diese setzen selbst eine
  Farbe; übernommen rückte die Eingabe vorn in die zuletzt benutzten Farben und verschöbe das Feld unter dem Klick
  (Sitzungs-QA A-2, Verify V-1). Verliert nur das Fenster den Fokus, bleibt die Eingabe
  unverändert stehen. Bei offenem Menü oder offener Hilfetafel gehören Esc und Enter diesen (A-3, A-7); Umschalt+Enter
  wirkt im Feld wie Enter (A-5); die Rücktaste auf dem unberührten Feld beginnt beim Wert ohne Einheit (A-8). Rücktaste löscht ein Zeichen. Solange ein Feld den Fokus hat, sind
  Buchstaben, Ziffern, Entf und die Rücktaste Eingabe, keine Kürzel (wie in der Texteingabe, ADR-0021 Nachtrag 1); eine
  ungültige Eingabe zeigt den Rand rot, bis sie korrigiert oder verworfen ist.
- **Tastatur:** Farbfläche und Regler sind nur mit der Maus zu bedienen; per Tastatur gehen Farbfelder, Pipette (Enter)
  und die beiden Eingabefelder. **Tab-Reihenfolge** (ersetzt die Liste aus Nachtrag 4 ab „Farben der rechten Leiste“):
  … Farbfelder der Leiste (Beschriftungsfarben, dann zuletzt benutzte), Pipette, Hex, Deckkraft, Fußfeld Farbe, Fußfeld
  Zoom. *Seit ADR-0021 Nachtrag 17:* Pipette nach „Keine“, vor den Farbfeldern. *Seit ADR-0021 Nachtrag 24 Punkt 5:*
  Farbpaar, Pipette, zuletzt benutzte, Hex, Deckkraft, eigene Farbfelder, „+“.
- **Pipette:** Solange sie an ist, ist ihr Feld grün, der Zeiger ein Fadenkreuz und die Statuszeile sagt „Click in the
  image to pick its color. Esc cancels.“.
- **Eine Farbe überall:** Leiste, Farbfeld links, Farbliste und Fußzeile zeigen dieselbe Farbe — die des gewählten
  Objekts, sonst die des Werkzeugs (`canvas.Sitzung.AktuelleFarbe`). Vorher zeigten Farbfeld und Fußzeile immer die
  Werkzeugfarbe; nach Rückgängig an einem gewählten Objekt wichen sie von der Leiste ab (Sichtprobe #189,
  `scripts/drehbuecher/farbwaehler-189.txt`).

## Nachtrag 6 (23.11.2026): Schrift in der rechten Leiste (Karte #190)

**Status:** Akzeptiert (Claude Opus 5.5). Füllt den Block „Werkzeug: Text“ des Musters v2; Bedeutung in ADR-0021
Nachtrag 10.

- **Aufbau** (beim Text-Werkzeug oder gewählten Text, über der Farbe): Zeile **Schrift** mit einem Feld (Name, ▾;
  fehlt die Familie hier, „Name (missing)“), Zeile **Größe** mit − / + und der Größe in px wie bei der Strichstärke,
  Zeile **Stil** mit vier Umschaltern (F, K, U, S auf Deutsch; B, I, U, S auf Englisch; an = grün) und Zeile
  **Ausrichtung** mit drei Knöpfen (links, Mitte, rechts). Abweichung vom Muster: Die Ausrichtung steht in einer
  eigenen Zeile, weil sieben Knöpfe neben der Beschriftungsspalte nicht in 300 dp passen; die Größe hat keine Liste
  (− / + genügt, die Stufen stehen in `canvas.SchriftStufen`). „Schild: gefüllt, eckig“ aus dem Muster gehört zur
  zweiten Farbe (#219).
- **Schriftliste:** Ein Klick oder Enter auf das Feld öffnet unter ihm eine Liste mit allen Familien (Go und Go Mono
  zuerst, dann die gefundenen nach Namen), höchstens 320 dp hoch, mit dem Mausrad zu blättern; die gezeigte Familie
  ist grün markiert. Ein Klick auf einen Eintrag wählt ihn, ein Klick daneben schließt ohne Wahl. Per Tastatur:
  Pfeil hoch und runter, Pos1, Ende, Enter wählt, Esc schließt; getippte Buchstaben springen zur ersten Familie mit
  diesem Anfang (nach einer Sekunde Pause beginnt ein neuer Anfang). Solange die Liste offen ist, wirken andere
  Tasten nicht (kein Werkzeugkürzel hinter der Liste); danach liegt der Fokus wieder auf dem Feld, während einer
  Texteingabe an der Leinwand (Sitzungs-QA B-1). Verschwindet die Leiste (schmales Fenster), schließt die Liste (B-6). Die Namen stehen in
  der Schrift der Oberfläche, nicht in ihrer eigenen.
- **Tab-Reihenfolge** (beim Text): … Farbfeld links, Schrift, Größe − und +, die vier Umschalter, die drei
  Ausrichtungen, dann die Farbe wie in Nachtrag 5.
- **Suche:** Der Editor startet die Suche nach Systemschriften beim Öffnen im Hintergrund und zeichnet neu, wenn sie
  fertig ist; bis dahin enthält die Liste nur Go und Go Mono.
- Sichtprobe: `scripts/drehbuecher/schrift-190.txt`.

## Nachtrag 7 (23.11.2026): Abschnitt „Ebenen“ und Menü „Ebene“ (Karte #191)

**Status:** Akzeptiert (Claude Opus 5.5). Einzelheiten in ADR-0032. Ändert Punkt 1 und Punkt 3:

1. Die Menüzeile hat **sieben** Menüs: „Ebene“ steht zwischen „Werkzeuge“ und „Hilfe“ (die Titel davor behalten ihre
   Lage).
2. Die rechte Leiste trägt unter der Farbe den Abschnitt **Ebenen**; er hängt am unteren Rand der Leiste, damit die
   Liste beim Wählen nicht springt. Die Überschriften „Farbe“ und „Ebenen“ sind Knöpfe, die ihren Abschnitt ein- und
   ausklappen (Tab-Reihenfolge: Überschrift vor dem Inhalt).

## Nachtrag 8 (23.11.2026): Die rechte Leiste rollt bei wenig Höhe (Karte #225, F-598)

**Status:** Akzeptiert (Claude Opus 5.5). Ergänzt Nachtrag 3, Nachtrag 5 und Nachtrag 7.

**Problem.** In niedrigen Fenstern mit sichtbarer Leiste lagen Hex und Deckkraft des Farbwählers unter der Fußzeile
(Fenster 320 px hoch, Text gewählt: Hex bis y 504, Fußzeile ab 290). Die Einklapp-Logik kann nur die Ebenen einklappen;
die Farbe von selbst einzuklappen sperrte den Farbwähler aus (F-592) und ist ausgeschlossen.

**Erwogen.**
1. *Kompakter Wähler* (kleinere Farbfläche bei wenig Höhe): spart höchstens rund 70 dp. Der Inhalt über dem Wähler
   (Schrift, Größe, Stil, Ausrichtung, Strichstärke) und der Wähler selbst ergeben bei gewähltem Text über 480 dp; das
   Mindestmaß wäre nie erreicht, und die Farbfläche würde für genaue Wahl zu klein.
2. *Eigener rollbarer Abschnitt nur für die Farbe*: zwei verschachtelte Rolllisten (Farbe, Ebenen, Schriftliste) mit
   getrennten Fokus- und Zeigerräumen; mehr Zustand, mehr Sonderfälle.
3. *Die ganze Leiste rollt* (gewählt): Alles wird wie bisher in Inhaltskoordinaten gezeichnet und mit einem Versatz
   verschoben; nichts wird ein- oder ausgeblendet. Ein kompakter Wähler bleibt bei Bedarf später möglich, ändert aber
   nichts an dieser Regel.

**Entscheidung.**
- Reicht die Höhe nicht (Inhalt endet unter der Leiste), rollt die Leiste senkrecht: Mausrad über der Leiste, dazu ein
  schmaler eckiger Balken am rechten Rand. Passt alles, ist der Versatz 0 und nichts ist sichtbar anders (hohe Fenster
  zeichnen wie zuvor). Die Listen der Ebenen und der Schriften behalten ihr eigenes Mausrad.
- Die Farbe klappt nie von selbst ein (F-592); der Vorrang der Ebenen (ADR-0032 Punkt 6, Klick auf „Ebenen“ klappt die
  Farbe ein, Klick auf „Farbe“ gibt ihn zurück) und die Schwelle `farbwaehlerHoeheDp + ebenenMindestDp` bleiben. Das
  Rollen ist das Netz darunter: Was trotz Einklappen nicht passt, ist erreichbar.
- **Tab-Weg:** Die Reihenfolge der Felder und Knöpfe bleibt (`editor.eingaben()`, `fokusFeld`, `seitenKnoepfe.alle`).
  Wechselt der Tastaturfokus auf ein Element, das nicht ganz sichtbar ist, rollt die Leiste es mit 8 dp Rand ins Bild
  (`rollZeige`, nur beim Wechsel, damit das Mausrad nicht dagegen arbeitet).
- **Koordinaten:** Die Lagen in den Zustandsstrukturen (`hex.rahmen`, `knopfRahmen` usw.) und die Lagen der Zeigerereignisse
  in der Leiste sind Inhaltskoordinaten (Versatz 0 = Fensterkoordinaten); wer Fensterkoordinaten dagegen hält
  (`inFarbwahl`, die Fokuswache), addiert `roll.rollen`. Rechnung ohne Gio in `internal/ui/seitenrollen.go`
  (`rollEnde`, `rollBegrenze`, `rollZeige`, `rollDaumen`).
- **Mausrad-Anmeldung (Sichtprobe):** Die Anmeldung der Leiste für das Mausrad steht im Ausschnitt der Leiste selbst
  (`zeichneSeitenleiste`, vor dem Inhalt), nicht in einem eigenen, wieder abgelegten Ausschnitt. Gio überspringt
  Bereiche unter einem undurchlässigen Treffer, es sei denn, der Eintrag ist ein Vorfahre; in einem eigenen Ausschnitt
  stand das Rad deshalb nach 75 px still, sobald Knopf, Feld oder Farbfläche unter den Zeiger wanderte. So behalten
  Ebenen- und Schriftliste ihr Rad (tiefere Einträge werden zuerst bedient), alles andere rollt die Leiste.
- **Nachbesserung (Karte #235, F-611, F-612):** Die Ebenenliste meldet als Rad-Bereich nur den Weg, den sie selbst noch
  blättern kann (`Min: -versatz`, `Max: inhaltH - Höhe - versatz`); am Anschlag oder ohne Überstand reicht Gio den
  Rest an die Leiste weiter, die Schriftliste (`layout.List`) tut das ohnehin. Radwege unter 1 px (Touchpad) sammelt
  `rollRad` als Rest (`rollZustand.rest`), am Anschlag verfällt er. Die Schriftliste liegt außerhalb der Leiste in
  Fensterkoordinaten: `zeichneSchriftliste` rechnet das Feld um den Versatz zurück, hängt die Liste an das sichtbare Feld
  und schließt sie, wenn das Feld ganz aus der Sicht ist oder unter ihm keine Zeile mehr Platz hat. Offen: die echte
  Schrittweite eines Mausrads (Gio multipliziert diskrete Schritte mit 10, rund 150 px je Raste) steht in Andis gebündelter
  Echtprobe.
- Tests: `internal/ui/seitenrollen_test.go` (Sonde aus der Kritik zu #219 über alle Höhen 320 bis 800 in 2-px-Schritten,
  Tab-Weg, Klicks im gerollten Bild, Ebenen-Vorrang, Zeichnen ohne Fenster), Mutanten KU225-1 bis KU225-8 und KU235-1 bis KU235-13
  (Nachbesserung, Karte #235) in `scripts/mutanten/ui-seitenleiste.json`.

## Nachtrag 9 (24.11.2026): Navigator — Mini-Übersicht bei starkem Zoom (Karte #197)

Wunsch des Maintainers: Bei starkem Zoom soll eine kleine Übersicht des ganzen Bilds zeigen, welcher Ausschnitt gerade
im Fenster liegt, und sich per Klick und Ziehen verschieben lassen. Entscheidung des Orchestrators:

- **Ort:** schwebend unten rechts auf der Leinwand, über dem Bild (nicht in der rechten Leiste). Eckiger Kasten mit
  1-dp-Rand in `farbeLinie`, Grund `farbeLeisteVoll`, 12 dp Abstand zu den Leinwandkanten (die Fußzeile liegt unter der
  Leinwand), längste Seite 160 dp, Seitenverhältnis der gezeigten Bildfläche (`Sitzung.Bildflaeche`, beim Zuschneiden
  die Vollansicht).
- **Wann:** nur, wenn das Bild bei der Zoomstufe nicht ganz ins Fenster passt (eine Achse ragt hinaus, Toleranz ein
  halbes Pixel); beim Einpassen und bei kleinerem Zoom verschwindet er. Er verschwindet auch, wenn er nicht in die
  Leinwand passt. Menü Ansicht → „Navigator“ (Schalter, Voreinstellung an, nicht gespeichert, ohne Kürzel) blendet ihn
  ganz aus.
- **Inhalt:** das Leinwandbild samt Objekten und Nähten, auf die Kastengröße verkleinert (dieselben Zeichenaufrufe wie
  die Leinwand unter einer Verkleinerung, keine zweite Renderkette). Darüber ein 2-dp-Rechteck in `farbeAkzent` (orange
  wie die Hilfslinien) für den sichtbaren Ausschnitt, auf das Bild begrenzt.
- **Bedienung:** Klick zentriert die Ansicht auf die Stelle, Ziehen führt die Mitte des Ausschnitts live mit dem Zeiger
  (der Zeiger darf den Kasten verlassen, die Stelle wird auf den Kasten begrenzt); die Ansicht bleibt dabei so
  begrenzt, dass der Ausschnitt im Bild liegt (ist das Bild in einer Achse kleiner als das Fenster, steht es dort
  mittig). Das Mausrad über dem Navigator zoomt wie über der Leinwand, nur um die Fenstermitte statt um den Zeiger (der
  Zeiger steht in der Ecke, ein Zoom um ihn schöbe das Bild aus dem Fenster). Der Navigator meldet sich mit eigenem
  Zeigerziel über der Leinwand an: Klicks, Ziehen und Rad erreichen die Werkzeuge nicht. Ein Klick darin nimmt dem
  laufenden Zug nichts weg; während eines Zugs (Ziehen eines Objekts, Zuschneiden, Verschieben mit der mittleren
  Taste) ändert der Navigator die Ansicht nicht (wie Zoom und Einpassen).
- **Umsetzung:** Logik ohne Gio in `internal/canvas/navigator.go` (Sichtbarkeitsregel, Größe, Lage, Abbildung Bild ↔
  Kasten, Ausschnitt, Zentrieren mit Begrenzung; `Sitzung.NavigatorZu`, `Sitzung.NavigatorZoom`); Zeichnen und
  Ereignisse in `internal/ui/navigator.go`. Tests `internal/canvas/navigator_test.go`, `internal/ui/navigator_test.go`,
  Mutanten `scripts/mutanten/navigator.json`, Sichtprobe `scripts/drehbuecher/navigator-197.txt`.
- **Ergänzung — flache und schmale Bilder, Zeiger (F-613, Karte #236):** Die kurze Seite des Kastens ist mindestens
  16 dp dick (`NavigatorMinSeite`, ein Zehntel der langen Seite, bei Skala 2 also 32 px). Das Bild behält seinen
  Maßstab (Faktor aus der langen Seite, Seitenverhältnis gewahrt) und steht in der dünnen Achse mittig im Kasten
  (`Navigator.Inhalt`; `ZuKasten` und `ZuAnsicht` rechnen mit dieser Lage, `Enthaelt` und das Zeigerziel mit dem ganzen
  Kasten). Deckt der Ausschnitt das Bild in der dünnen Achse ganz, füllt das orange Rechteck dort den Kasten, statt als
  Strich im Bildstreifen zu verschwinden. Für die Zeigerereignisse gilt: Nur die linke Taste führt den Ausschnitt (jeder
  andere Druck beendet ein Ziehen); Loslassen und Abbruch (`Cancel`) gelten immer, auch während des Speicherns, denn sie
  ändern nichts am Dokument, während Druck und Ziehen dann ruhen. Tests `internal/canvas/navigator_flach_test.go`,
  `internal/ui/navigator_flach_test.go`, `internal/ui/navigator_grenzfaelle_test.go`.

## Nachtrag 10 (24.11.2026): Tafeln rollen in kleinen Fenstern — Übersicht F1 und Einstellungstafel (Karte #272, F-705, F-709)

**Befund.** Die Übersicht der Tastenkürzel zeigte im Startfenster 1200×800 nur 31 von 48 Zeilen (eine Spalte, weil zwei
Spalten 1004 dp Leinwandbreite brauchten, die Leinwand aber 848 dp breit ist), und die Einstellungstafel ließ in
kleinen Fenstern Felder im Fuß verschwinden. Beide schnitten ab, ohne dass man den Rest erreichen konnte.

**Regel.** Eine modale Tafel zeigt nie weniger, als sie hat: **Passt der Inhalt nicht, rollt er** — nicht die Tafel als
Ganzes, sondern ihr Inhaltsbereich, der Kopf (Überschrift bzw. Titel und Reiter) und der Fuß (Save/Cancel, Fehlertext)
bleiben stehen. Dieselbe Rechnung wie die rechte Leiste (Nachtrag 8: `rollEnde`, `rollZeige`, `rollRad`, `rollDaumen` in
`internal/ui/seitenrollen.go`), die gemeinsamen Teile der Tafeln in `internal/ui/tafelrollen.go`:

- **Bedienung:** Mausrad über dem Inhalt (die Anmeldung steht im Ausschnitt des Inhalts vor diesem, wie in Nachtrag 8),
  Bild auf/ab (eine Seite minus eine Zeile), in der Übersicht zusätzlich Pfeil auf/ab, Pos1 und Ende (die Einstellungstafel
  braucht Pfeile und Pos1/Ende für Zahlfelder); ein eckiger Balken rechts im Inhalt, wenn gerollt werden kann
  (`rollZustand.balken`). Ein Klick oder Esc schließt die Übersicht weiter; Rolltasten schließen sie nicht.
- **Tastaturfokus rollt ins Bild:** in der Einstellungstafel rollt Tab das fokussierte Feld samt 8 dp Rand in den
  sichtbaren Bereich (nur bei einem Fokuswechsel, damit das Mausrad nichts bekämpft). Die Lagen in `tafelRahmen` sind
  Fensterpixel **mit** Versatz (`fensterR`), Klicks, Tests und Drehbücher treffen also, was man sieht; ein
  weggerolltes Feld hat eine Lage außerhalb der Inhaltsfläche. Jeder Reiter und jedes Öffnen beginnt oben.
- **Übersicht dichter:** Zeilen 20 dp, Schrift 12 sp, Spalten 220 + 170 dp, Rand 12 dp; zwei Spalten ab 800 dp
  Leinwandbreite (`hilfeAufteilung`), wenn eine nicht in die Höhe passt. Im Startfenster stehen damit alle 48 Zeilen
  ohne Rollen da, mit und ohne Leiste der letzten Aufnahmen (848 × 636 dp). Der Text „Cancel, deselect, back to Select“
  wurde zu „Cancel, deselect, Select tool“ gekürzt (die Spalte kürzt sonst mit „…“).
- **Untergrenze:** Das Fenster ist mindestens 480×320 dp (`app.MinSize`); bis dahin sind alle Zeilen und Felder
  erreichbar. Was kleiner ist, ist nicht unterstützt.

Tests `internal/ui/hilfe_rollen_test.go`, `internal/ui/einstellungen_rollen_test.go` (Felder per Tab erreichbar bei
1200×800 bis 480×320, Fuß bleibt, Klick auf ein gerolltes Feld, Reiterwechsel), Mutanten
`scripts/mutanten/tafeln-rollen.json` (T1–T7, E1–E7), Sichtprobe `scripts/drehbuecher/tafeln-klein-272.txt` (1200×800,
640×480, 480×320). ADR-0034 gilt unverändert; die Tafel rollt, ihre Maße (800×640) bleiben.

## Nachtrag 11 (24.11.2026): Die Übersicht F1 ist modal wie die Einstellungstafel (Karte #274, F-707)

**Status:** Akzeptiert (Claude Sonnet 5.5). Ersetzt aus Nachtrag 4 den Satz „Klicks in der rechten Leiste ruhen bei offener Hilfetafel“.

**Befund.** Das Zeigerziel der Übersicht deckte nur die Leinwand. Linke Werkzeugleiste, Farbfelder samt ⇄, Fußfelder und
Menüzeile blieben bedienbar, ein Klick auf ⇄ tauschte die Farben des Objekts hinter der Tafel (Abschluss-Kritik C-3).

**Regel.** Die Übersicht ist **konsequent modal**: ihr Zeigerziel deckt das ganze Fenster (`zeichneHilfe`, wie die
Einstellungstafel, ADR-0034). Menüzeile, Werkzeugleiste, Farbfelder, rechte Leiste, Leiste der letzten Aufnahmen und Fußfelder
bekommen keinen Klick; **ein Klick irgendwo schließt nur die Tafel** und löst dahinter nichts aus. **Menüzeile:** auch sie
schließt nur die Tafel (kein Menü öffnet sich im selben Klick). Begründung: eine Ausnahme für die Menüzeile bräuchte eine
zweite Regel, F10 öffnet das Menü weiter über die Tastatur, und ein Klick, der zwei Dinge tut, ist schwerer vorherzusagen.
Das Mausrad rollt die Tafel weiter (Nachtrag 10), die Tastatur bleibt unverändert (Esc/F1 schließen, Rolltasten rollen).
Tests `internal/ui/hilfe_modal_test.go`, Mutanten `scripts/mutanten/hilfe-modal.json`, Sichtprobe
`scripts/drehbuecher/hilfe-modal-274.txt`.

## Nachtrag 12 (25.11.2026): Ein Klick aufs Farbpaar zeigt den Farbwähler (Karte #310, F-817)

**Status:** Akzeptiert (Claude Opus 5.5). Ersetzt aus Nachtrag 3 den Teilsatz „sonst (Leiste schon offen, …) öffnet er
die Farbliste am Farbfeld“ für die Maus; die Tastatur bleibt, wie sie ist.

**Befund.** Andi (25.11.2026): „die farbauswahl unter der werkzeugleiste funktioniert nicht“. Die Klicks kamen an
(Sichtprobe im virtuellen KWin bei Skala 1 und 1,25), aber bei offener Leiste öffnete ein Klick nur die kleine Liste mit
den vier Beschriftungsfarben statt des Farbwählers daneben, und ein eingeklappter Abschnitt „Farbe“ oder eine nach unten
gerollte Leiste blieben, wie sie waren. Der Klick wirkte also, als täte er nichts Brauchbares.

**Regel.** Ein **Mausklick** auf Vordergrund oder Füllung des Farbpaars macht das Feld aktiv und zeigt den
**Farbwähler der rechten Leiste**, sobald die Leiste die Farbe zeigen kann (Platz für die Leiste, Werkzeug oder Auswahl mit
Farbe): Leiste aufklappen, Abschnitt „Farbe“ aufklappen, den Ebenen den Vorrang nehmen (Platzmangel, ADR-0032 Punkt 6),
den Abschnitt von der Überschrift bis unter Hex/Deckkraft einmal ins Bild rollen (`seitenKnoepfe.farbeZeigen`,
`rollZustand.ziel`; ist er höher als die Leiste, gilt seine Oberkante). Die **Farbliste** am Farbfeld öffnet ein Klick
nur noch, wo die Leiste die Farbe nicht zeigen kann (Fenster unter 752 dp, Werkzeug ohne Farbe). **Enter/Leertaste** am
Farbfeld öffnen weiter die Liste (Nachträge 3 und 4): Die Tastatur bleibt bei der Werkzeugleiste, statt in die rechte
Leiste zu springen.

Tests `internal/ui/farbpaar_klick_test.go`, Mutanten `scripts/mutanten/farbpaar-klick.json` (FK1–FK8) und die
nachgezogenen S187-11, QA-U4a/b in `ui-seitenleiste.json`. Die deutlichere Darstellung von Vorder- und Füllfarbe folgt
mit Karte #317.

## Nachtrag 13 (26.11.2026): Farbziel als Auswahl, Schatten als Kästchen, Richtung als Scheibe (Karte folgt)

**Status:** Akzeptiert (Claude Opus 5.5 im Auftrag Andis: „arbeite selbstständig“). Ändert die Bedienung aus ADR-0021
Nachträge 11, 23 und 26 Punkt 7; das Farbmodell (Felder Vordergrund, Füllung, Schatten in `canvas`) bleibt.

**Befund.** Andi (26.11.2026): „Das mit Pfeil-Farbe und Schatten-Farbe ist usabilitytechnisch sehr ungünstig“. Heute
bearbeitet der Farbwähler ein unsichtbares „aktives Feld“: Wer im Abschnitt Schatten auf dessen Farbfeld klickt, färbt
ab dann den Schatten, obwohl oben im Abschnitt Farbe weiter das rote Farbpaar steht und nur ein kleines Wort
(„Schatten“) neben dem Paar den Wechsel verrät. Der Schalter „Schatten“ sieht aus wie ein Farbfeld, die Richtung
verstellt man nur mit − / + in 15°.

**Regel.**
1. **Farbziel sichtbar wählen.** Der Abschnitt „Farbe“ beginnt mit einer Reihe gleich breiter Reiter, je Feld einer mit
   Farbmuster und Namen: **Strich** (bei Text „Schrift“) · **Füllung** (nur, wo es eine gibt) · **Schatten** (nur, wenn
   er an ist). Der Reiter des Felds, das der Farbwähler bearbeitet, ist hervorgehoben (Fläche `farbeAuswahl`, orange
   Unterkante); Hex, Deckkraft, Fläche, Ton und Farbfelder darunter gelten genau ihm. Tauschen (Strg+Umschalt+X) steht als
   kleines Symbol zwischen Strich und Füllung, „Keine“ bleibt ein Farbfeld der Palette. Das Wort neben dem Farbpaar
   entfällt. Pfeile links/rechts wechseln den Reiter, wenn er den Fokus hat.
2. **Schatten an/aus ist ein Kästchen** („☐ Schatten“, Haken wie „Agenten-Aufnahmen zeigen“ der Bibliothek), Klick auf
   Kästchen oder Wort, Leertaste schaltet. Das Farbfeld im Abschnitt Schatten bleibt als Kurzweg: Ein Klick wählt den
   Reiter „Schatten“ und zeigt den Farbwähler (wie Nachtrag 12).
3. **Richtung als Scheibe.** Eine runde Scheibe (Durchmesser 56 dp) zeigt in der Mitte ein kleines Quadrat und seinen
   Schatten in der eingestellten Richtung, dazu einen Punkt am Rand; Klicken oder Ziehen setzt die Richtung zum Zeiger hin
   (der Schatten fällt dorthin, wo man zeigt), rastet in 15°-Schritten ein, mit Umschalt frei in 1°. Mausrad und Pfeile
   ±15° (mit Umschalt ±1°), ein Ziehen ist ein Rückgängig-Schritt (`BeginneSchattenZug`). Daneben steht die Zahl; sie
   bleibt der Winkel des Lichts wie in Photoshop und im Format (`s:schatten`), also Fallrichtung + 180°. Die Knöpfe
   − / + der Richtung entfallen.
4. **Feinabstimmung** (Andis Frage, ob ein Kästchen „Feinabstimmung“ bei vielen Werkzeugen hilft): offen bis zu Andis
   Antwort, siehe Karte.

**Erwogene Alternativen.** Farbwähler je Feld in jedem Abschnitt (doppelter Platz, drei Hex-Felder); Ziel nur über das
Farbpaar links (unsichtbar, wenn die Leiste offen ist); Richtung als Liste von acht Pfeilen (kein Zwischenwinkel).
