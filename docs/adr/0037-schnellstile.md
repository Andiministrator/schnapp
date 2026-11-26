# ADR-0037: Schnellstile — gespeicherte Werkzeug-Einstellungen mit einem Klick

## Status
Akzeptiert (Claude Opus 5.5, 24.11.2026, Karte #192 in Sprint 12, im Auftrag Andis: „arbeite selbstständig“).
Gestaltung nach Muster `tmp/sprint12/muster/bilder/192-schnellstile.png`; Andi nimmt sie mit der fertigen Umsetzung
ab (gebündelt am Sprint-Ende). Grundlage: Wunsch Andis vom 22.11.2026 (Ideenliste, Idee 1). Baut auf ADR-0021
Nachträgen 7, 10 und 11 (Strichstärke mit Bildgröße, Schrift, Farbpaar je Werkzeug) auf; ergänzt ADR-0018 (Ort der
Konfiguration) und ADR-0031 (rechte Leiste).

## Kontext
Wer oft dieselben Kombinationen nutzt (schwarze Schrift auf gelbem Schild, dicker roter Rahmen, halbdurchsichtige
gelbe Markierung), stellt heute Farbe, Füllung, Strich und Schrift jedes Mal einzeln ein. Snagit hat dafür
„Quick Styles“.

## Entscheidung

1. **Ein Stil** gehört zu einer Objektart und trägt: Vordergrund, Füllung (bei Kasten, Text und Sprechblase), Strichstärke
   (Pfeil, Kasten, Lupe), beim Text dazu Schrift, Schriftgrad, Fett, Kursiv, Unterstrichen, Durchgestrichen und
   Ausrichtung, bei der Nummer den Radius. Größen sind auf **Faktor 1** gespeichert und wirken mal
   `document.KantenFaktor` des Bilds (wie die Voreinstellung, ADR-0021 Nachträge 7 und 8), damit ein Stil auf einer
   4K-Aufnahme gleich aussieht wie auf Full HD.
2. **Abschnitt „Quick styles“/„Schnellstile“** oben in der rechten Leiste (über den Eigenschaften), sichtbar bei einem
   Werkzeug mit Stilen (Pfeil, Kasten, Text, Sprechblase, Nummer, Lupe) oder einem gewählten Objekt dieser Art; einklappbar
   wie „Farbe“. Kacheln 40 × 40 dp in Reihen zu sechs, auf hellem Schachbrett (Schwarz, Weiß und Durchsichtiges bleiben
   sichtbar), je Kachel eine Mini-Darstellung des Objekts im Stil. Orange umrandet die Kachel, die zum aktuellen Stand
   passt (gleiche Werte), sonst keine.
3. **Anwenden:** Klick (oder Leertaste mit Fokus): ohne gewähltes Objekt wird der Stil zum Stand des Werkzeugs (sein
   Farbpaar und seine Maße, ADR-0021 Nachtrag 11 Punkt 2); mit gewähltem Objekt ändert er das Objekt in **einem**
   Rückgängig-Schritt und, wie eine Farbe am Objekt, auch den Stand des Werkzeugs dieser Art.
4. **Eingebaute Stile** (fest, übersetzt benannt im Tooltip/Statustext): Pfeil und Kasten je Rot, Blau, Grün, Schwarz,
   Rot dick, Weiß auf Schwarz (Kasten: weiße Linie mit schwarzer Füllung), Kasten zusätzlich „Marker“ (gelbe Linie,
   gelbe Füllung 40 %); Text Weiß auf Rot (Voreinstellung), Schwarz auf Gelb, Weiß auf Schwarz, Weiß auf Blau, Rot ohne
   Schild, Schwarz ohne Schild; Nummer Rot, Blau, Grün, Schwarz; Lupe Rot, Schwarz, Weiß. Die Werte stehen in einer
   Tabelle in `internal/canvas/schnellstile.go`.
5. **Eigene Stile:** Knopf „+“ hinter den Kacheln speichert den aktuellen Stand (des Werkzeugs bzw. des gewählten
   Objekts) als eigenen Stil am Ende der Reihe; eigene Kacheln tragen eine graue Ecke. Rechtsklick oder Entf auf einer
   eigenen Kachel mit Fokus entfernt sie (ohne Rückfrage, Statuszeile „Style removed“; kein Rückgängig, weil Stile
   nicht zum Dokument gehören). Höchstens 16 eigene je Objektart; ist die Grenze erreicht, ist „+“ grau.
   Umbenennen und Umordnen gibt es nicht.
6. **Speicherort:** `$XDG_CONFIG_HOME/schnapp/styles.ini` (sonst `~/.config/schnapp/styles.ini`), Ordner 0700, Datei
   0600, atomar geschrieben wie `settings.ini` (ADR-0034 Punkt 5), aber als **Datei des Programms**: Schnapp schreibt sie
   jeweils ganz neu (Kopfkommentar sagt das). Ein Abschnitt je Stil, Name `[<art>]` (`arrow`, `box`, `text`, `callout`,
   `step`, `magnifier`), Reihenfolge der Abschnitte = Reihenfolge der Kacheln; Schlüssel englisch (`color`, `fill`,
   `width`, `font`, `size`, `bold`, `italic`, `underline`, `strike`, `align`, `radius`), Farben `#RRGGBBAA`. Unbekannte
   Schlüssel und ungültige Werte: der Stil fällt weg, mit Warnung auf stderr wie bei `settings.ini`; die Datei ist
   höchstens 64 KiB. Gelesen beim Öffnen des Editors, geschrieben bei jedem „+“ und Entfernen. Eigene Stile gehören
   nicht in die Einstellungstafel (ADR-0034 Punkt 2 betrifft `settings.ini`).
7. **Tastatur:** Tab erreicht die erste Kachel, Pfeiltasten wandern, Leertaste/Enter wendet an, Entf entfernt einen eigenen
   Stil; kein globales Kürzel je Stil (Ziffern bleiben frei).
8. **Nicht in diesem Schritt:** Stile für Schwärzen/Verpixeln (haben keine Farbe), Stile über mehrere Objektarten,
   Austausch von Stil-Dateien, Stile im Dateiformat.
9. **Umsetzung:** Logik ohne Gio in `internal/canvas/schnellstile.go` (Tabelle, Anwenden als ein Schritt, Passt-zu,
   Faktor), Lesen/Schreiben in `internal/settings/stile.go` (Tests mit eigenem `XDG_CONFIG_HOME`), Leiste in
   `internal/ui/schnellstile.go`; Texte über `i18n`; Mutanten-Gruppe `schnellstile`; Sichtprobe per Drehbuch.

## Erwogene Alternativen
- **Eigene Stile in `settings.ini`:** eine Datei weniger, aber eine Liste von Abschnitten passt nicht zu „jeder Schlüssel
  hat eine Tafelzeile“ und vermischt Handbearbeitetes mit Programmdaten.
- **Stile im Zustand (`$XDG_STATE_HOME`):** Stile sind bewusst gewählt und gehören damit nach ADR-0018 zur Konfiguration.

## Konsequenzen
- Neue Datei im Konfigurationsordner (README nennt sie neben `settings.ini`).
- Die Sprechblase (ADR-0021 Nachtrag 13) bekommt eigene Stile; bis sie umgesetzt ist, entfällt die Art `callout`.

## Ergänzt bei der Umsetzung (Teil 1: Logik und Speicher, Karte #192)
- **Pfeil „Weiß auf Schwarz“:** Ein Pfeil hat keine Füllung, ein weißer Pfeil „auf Schwarz“ wäre irreführend benannt. Der
  sechste Pfeil-Stil heißt deshalb **Weiß** (weiße Linie); Kasten und Text behalten „Weiß auf Schwarz“. „Schwarz“ ist
  überall das Dunkelgrau der Palette (`canvas.Beschriftungsfarben[3]`).
- **„+“ bei einem Stand, der schon als Kachel da ist:** Der Stand ist dann eingebaut oder bereits eigen; ein zweiter,
  gleicher Stil wird nicht angelegt (`HinzuDoppelt`, „+“ ist wie bei 16 Stilen grau, `KannSchnellstilHinzufuegen`).
- **Ein Stil der anderen Art** als das gewählte Objekt ändert das Objekt nicht, wohl aber den Stand seines Werkzeugs.
- **Texteingabe:** Ein Stil wirkt auf den Text in der Eingabe und gehört zu ihr (kein eigener Rückgängig-Schritt).
- **Fehlende Schlüssel** in `styles.ini` nehmen die Voreinstellung bei Faktor 1 (Strich 6 seit ADR-0021 Nachtrag 16, vorher 4; Schriftgrad 20, Radius 14,
  Füllung durchsichtig); nur `color` ist Pflicht. Ein Schlüssel, der für die Art nicht vorgesehen ist (`fill` beim Pfeil),
  macht den Stil ungültig. Stile der Sprechblase (`[callout]`) bleiben beim Lesen und Schreiben erhalten, auch wenn es das
  Werkzeug noch nicht gibt.
- **Zuordnung:** Das Paket `settings` kennt `canvas` nicht; `settings.Stil` ist ein neutraler Datentyp, `canvas`
  wandelt (`Schnellstil.Einstellung`, `SchnellstilAus`).

## Ergänzt bei der Umsetzung (Teil 2: Leiste in der Oberfläche, Karte #192)
- **Name als Hinweis in der Statuszeile**, nicht als Tooltip: Die Oberfläche kennt keine Tooltips; wie bei den
  Werkzeugknöpfen zeigt die Fußzeile den Namen der Kachel mit Tastaturfokus, sonst der unter dem Zeiger. Eine eigene
  Kachel nennt „Custom style“ samt Weg zum Entfernen; der Knopf „+“ sagt, was er tut oder warum er grau ist (Stand schon
  eine Kachel, 16 eigene erreicht).
- **Entf** an einer eingebauten Kachel tut nichts und löscht **nicht** das gewählte Objekt dahinter; an einer eigenen
  entfernt es den Stil. Die Rücktaste bleibt bei „Objekt löschen“ wie an jedem Knopf der Leiste. Nach dem Entfernen der
  letzten Kachel wandert der Fokus auf „+“, sonst ginge er mit der Kachel verloren.
- **Pfeile, Pos1 und Ende** wandern durch die Kacheln und „+“ (Pfeile hoch und runter eine Reihe von sechs); Tab fährt
  die Kacheln wie jeden Knopf der Leiste ab (Überschrift, Kacheln, „+“, danach Schrift und die übrigen Eigenschaften).
- **Fehlende Schriftfamilie:** Die Mini-Darstellung setzt „Aa“ blass und schräg (wie ein Ersatz), der Hinweis nennt die
  Familie. Geprüft wird erst, wenn die Suche nach Systemschriften fertig ist; vorher gilt keine als fehlend.
- **Ein Speicherfehler** (styles.ini nicht schreibbar) bleibt als Fehlermeldung in der Fußzeile; der Stil gilt in der
  laufenden Sitzung trotzdem.
- **Einklappen** der Überschrift gilt für die Sitzung des Editors und wird nicht gespeichert (wie „Farbe“).
- **Höhe:** Der Abschnitt macht die Leiste bei Pfeil, Kasten, Text und Sprechblase um 143 dp höher (zwei Reihen, „+“ wandert
  mit den sechs bis sieben Kacheln in die zweite), bei Nummer und Lupe um 98 dp. Wo das Fenster nicht reicht, rollt die
  Leiste (ADR-0031 Nachtrag 8); der Abschnitt klappt nicht von selbst ein.

## Ergänzt (Sitzungs-QA 24.11., Karte #244, F-627, F-628, F-629 b)
- **Zusammenführen statt Ersetzen (F-627):** „+“ und Entfernen schreiben nicht mehr die Sitzungsliste, sondern lesen `styles.ini` unter einer flock-Sperre (`styles.ini.lock` im selben Ordner, 0600, ohne flock-Unterstützung ohne Sperre, Wartezeit höchstens 3 s) neu und wenden nur die eigene Änderung an (`settings.AendereStile`, `StilHinzu`: anhängen, wenn kein gleicher Stil da ist und die Art Platz hat; `StilEntfernt`: genau diesen Stil). Danach gilt in der Sitzung der Stand der Datei. Zwei Fenster überschreiben einander also nicht, und ein entfernter Stil kommt nicht zurück. Die Sperrdatei bleibt liegen (kein Löschen, keine Wettlauf-Prüfung nötig).
- **Nur speicherbare Stände (F-628):** `neuerStil` prüft mit demselben `settings.PruefeStil` wie der Schreiber; ein Stand mit Strich 0 oder einem Schriftnamen mit Leerraum am Rand wird abgelehnt (`HinzuUnbrauchbar`), „+“ ist grau, die Statuszeile erklärt es. Abgelehnt, nicht gekürzt: Der Stil soll den Stand wiedergeben, den man sieht.
- **Unlesbare Datei (F-629 b):** Ist `styles.ini` nicht lesbar (keine reguläre Datei, über 64 KiB, Lesefehler), bleibt sie unberührt und die Fußzeile nennt den Fehler. Hat das Lesen Stile oder Zeilen verworfen, legt der erste Schreibzug zuerst `styles.ini.bak` an (exklusiv, 0600) und sagt es einmal in der Fußzeile; gibt es die Sicherung schon, wird **nichts** geschrieben (`ErrStileSicherung`). Gewählt statt „immer verweigern“, weil der Nutzer so mit einer lesbaren Datei weiterarbeiten kann, ohne dass etwas verloren geht; gewählt statt „Sicherung überschreiben“, weil die erste Sicherung das Einzige sein kann, was von handgeschriebenen Stilen übrig ist. Der Teil a von F-629 (Ergebnisgröße von `settings.ini`) gehört zu einem anderen Strang.

## Nachtrag 1 (26.11.2026): Schatten und Strichart in Schnellstilen (Karte #363 Teil 2)

**Status:** Akzeptiert (Claude Opus 5.5, im Auftrag Andis: „arbeite selbstständig“). Setzt ADR-0021 Nachtrag 26 Punkt 8
um und entscheidet die offene Frage aus ADR-0021 Nachtrag 25 Punkt 8.

1. **Schatten:** Ein Schnellstil trägt einen Schatten, wenn er einen hat (`canvas.Schnellstil.Schatten`, `An` = ja;
   Länge und Weich auf Faktor 1, beim Anwenden mal Faktor, gedeckelt auf die Grenzen des Dokuments). Er entsteht nur
   über „+“ aus einem Stand mit Schatten (Objekt mit Schatten oder Werkzeug mit eingeschaltetem Schatten). Anwenden setzt
   ihn am Werkzeug und am gewählten Objekt, im selben Rückgängig-Schritt wie die übrigen Werte. Ein Stil **ohne**
   Schatten lässt den des Werkzeugs und des Objekts stehen und **passt unabhängig davon** (`SchnellstilPasst` sieht den
   Schatten des Stands dann nicht an); sonst verlöre ein Objekt mit Schatten den orangen Rand seiner eingebauten Kachel.
   Für „schon vorhanden“ (`HinzuDoppelt`) zählt der Schatten dagegen mit: ein Stand mit Schatten neben einer
   eingebauten Kachel ohne ist ein neuer Stil. Die eingebauten Stile haben keinen Schatten.
2. **`styles.ini`:** freiwilliger Schlüssel `shadow = WINKEL LÄNGE WEICH #RRGGBBAA` bei jeder Art, nur geschrieben, wenn
   der Stil einen Schatten hat. Lesen: vier Werte, Zahlen endlich, Länge 0 bis 1000, Weich 0 bis 200, der Winkel wird
   auf 0 bis 360 gebracht; sonst ist der Stil ungültig wie bei jedem anderen kaputten Wert. `PruefeStil` lehnt einen
   Schatten ab, der sich so nicht schreiben ließe.
3. **Strichart:** Schnellstile tragen sie weiterhin nicht (wie die Spitzen des Pfeils): Sie ist eine Eigenschaft der
   Form, nicht der Ansicht, und eine gestrichelte Kachel neben einer durchgezogenen verdoppelte die Reihe für wenig
   Nutzen. Wer sie braucht, wählt sie im Klappfeld; ein späterer Wunsch kann sie wie den Schatten freiwillig aufnehmen.
4. **Kacheln:** Die Mini-Darstellung zeigt den Schatten nicht; der Hinweis in der Statuszeile bleibt der Name.

Tests: `internal/canvas/schnellstile_test.go` (`TestSchnellstilSchatten`), `internal/settings/stile_test.go`
(Rundlauf, Parse-Regeln, `TestStileSchattenGrenzen`).
