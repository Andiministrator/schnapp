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
  der Farbe.
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
  und das Farbfeld. Erst unter 244 dp Leistenhöhe (Fenster unter 304 dp hoch, also unter der Mindesthöhe von 320 dp, F-252) schneidet sie unten ab; Werkzeuge und Farbe bleiben dann über das
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
  Plätze bleiben leer) und als letztes Feld die **Pipette**. Darunter über die volle Breite der **Farbton-Regler** und
  der **Deckkraft-Regler** (Schachbrett unter einem Verlauf von durchsichtig zur Farbe), darunter die Felder **Hex**
  (`#RRGGBB`, bei Deckkraft unter 100 % `#RRGGBBAA`) und **Deckkraft** (`100 %`).
- **Maus:** Drücken und Ziehen in Fläche oder Regler setzt die Farbe laufend (Marke folgt dem Zeiger, auch außerhalb
  begrenzt auf den Rand), Loslassen schließt den Schritt ab. Die gewählte Farbe hat einen grünen Rand; ein Feld, das
  halbdurchsichtig ist, zeigt das Schachbrett darunter.
- **Eingabefelder:** Ein Klick oder Tab fokussiert das Feld (orangefarbener Rand, Schreibmarke am Ende, der Text wird
  beim Fokussieren ersetzt, sobald getippt wird). Enter übernimmt, Esc verwirft und gibt den Fokus an die Leinwand, Tab
  oder ein Klick woanders übernimmt ebenfalls — der Klick übernimmt, **bevor** er selbst wirkt (eine durchlässige
  Fläche über dem Fenster sieht jeden Druck; danach liegt der Fokus an der Leinwand), so dass ein Klick auf ein
  Farbfeld oder die Pipette danach gilt (Sitzungs-QA A-2). Verliert nur das Fenster den Fokus, bleibt die Eingabe
  unverändert stehen. Bei offenem Menü oder offener Hilfetafel gehören Esc und Enter diesen (A-3, A-7); Umschalt+Enter
  wirkt im Feld wie Enter (A-5); die Rücktaste auf dem unberührten Feld beginnt beim Wert ohne Einheit (A-8). Rücktaste löscht ein Zeichen. Solange ein Feld den Fokus hat, sind
  Buchstaben, Ziffern, Entf und die Rücktaste Eingabe, keine Kürzel (wie in der Texteingabe, ADR-0021 Nachtrag 1); eine
  ungültige Eingabe zeigt den Rand rot, bis sie korrigiert oder verworfen ist.
- **Tastatur:** Farbfläche und Regler sind nur mit der Maus zu bedienen; per Tastatur gehen Farbfelder, Pipette (Enter)
  und die beiden Eingabefelder. **Tab-Reihenfolge** (ersetzt die Liste aus Nachtrag 4 ab „Farben der rechten Leiste“):
  … Farbfelder der Leiste (Beschriftungsfarben, dann zuletzt benutzte), Pipette, Hex, Deckkraft, Fußfeld Farbe, Fußfeld
  Zoom.
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
