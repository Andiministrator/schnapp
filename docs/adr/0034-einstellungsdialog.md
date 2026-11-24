# ADR-0034: Einstellungsdialog im Editor

## Status
Akzeptiert (Claude Opus 5.5, 24.11.2026, Karte #175 in Sprint 12, im Auftrag Andis: „arbeite selbstständig“).
Gestaltung nach Muster `tmp/sprint12/muster/bilder/175-einstellungen-{a-aufnahme,b-agenten,c-bibliothek}.png`;
Andi nimmt sie mit der fertigen Umsetzung ab (gebündelt am Sprint-Ende), Änderungswünsche gehen als Nachtrag ein.
Grundlage: Wunsch Andis vom 22.11.2026 („die settings sollten auch als einstellungsdialog im programm einstellbar
sein“). Löst den Satz „nur ein künftiger Einstellungsdialog schreibt“ aus ADR-0018 ein; ergänzt ADR-0026 (Menü und
Kürzel), ADR-0030 (Einwilligung der Agenten) und ADR-0031 (Oberfläche).

## Kontext
Alle Einstellungen stehen bisher nur in `settings.ini` (ADR-0018 mit Nachträgen 1 bis 4): `[general] language`;
`[capture] screen`, `open-editor`, `seam`; `[library] cleanup`, `cleanup-days`; `[agents] capture`, `library`,
`confirm`, `image`, `edit`. Wer sie ändern will, muss die Datei kennen und von Hand bearbeiten. Die Datei darf dabei
nicht verlieren, was der Nutzer von Hand hineingeschrieben hat (Kommentare, Reihenfolge, unbekannte Schlüssel), und die
Einwilligungen der Agenten (`[agents]`) dürfen nur vom Menschen kommen (ADR-0030 Nachtrag 2 Punkt 7).

## Entscheidung

1. **Tafel im Editor-Fenster, kein eigenes Fenster.** Menü Datei → „Settings…“/„Einstellungen…“, Kürzel **Strg+,**
   (Komma; frei, in Bildprogrammen und Editoren üblich). Die Tafel liegt modal über dem Editor wie die Hilfe (F1,
   ADR-0031 Punkt 1); der Editor darunter ist abgedunkelt und nimmt keine Eingaben an. Eckig, Farben nach
   `brand/corporate-design.md`. Ein eigener Prozess oder ein zweites Fenster bringt nichts, was die Tafel nicht kann,
   und kostet unter Wayland eine zweite Oberfläche.
2. **Aufbau:** Kopf „Settings“ mit Schließen-Kreuz; links Reiter **General**, **Capture**, **Library**, **Agents**;
   rechts die Zeilen des Reiters (Beschriftung links, Feld rechts, kurze Erklärung darunter in Grau); Fuß mit dem Hinweis,
   wann die Änderung wirkt, und den Knöpfen „Cancel“ und „Save“. Felder: **Auswahl** (Liste über die Menü-Mechanik wie
   Schrift und Zoom), **Schalter** (eckiges Kästchen mit Haken), **Zahl** (− Wert +, Wert auch tippbar).
   | Reiter | Zeile | Schlüssel | Feld |
   |---|---|---|---|
   | General | Language | `[general] language` | Auswahl: „Like the system“ (`auto`) und die eingebauten Sprachen |
   | Capture | Capture screen | `[capture] screen` | Auswahl: pick / pointer / all |
   | Capture | After capture: open editor | `[capture] open-editor` | Schalter |
   | Capture | Seam edge | `[capture] seam` | Auswahl: tear / wave / zigzag |
   | Library | Clean up | `[library] cleanup` | Schalter |
   | Library | After days | `[library] cleanup-days` | Zahl 1 … 36500 |
   | Agents | Capture screen | `[agents] capture` | Schalter, mit Warnung |
   | Agents | Ask before each capture | `[agents] confirm` | Schalter |
   | Agents | Send the image | `[agents] image` | Schalter |
   | Agents | Read the library | `[agents] library` | Schalter, mit Warnung |
   | Agents | Annotate and redact files | `[agents] edit` | Schalter, mit Warnung |

   **Jeder neue Schlüssel in `settings.ini` bekommt im selben Commit seine Zeile in der Tafel** (Regel für Nachträge zu
   ADR-0018); ein Test vergleicht die Schlüssel des Lesers mit den Zeilen der Tafel.
3. **Wirkung:** Erst „Save“ übernimmt, „Cancel“, Esc und das Kreuz verwerfen ohne Rückfrage. Es gilt: Sprache nach
   dem nächsten Start (ein Wechsel mitten im Fenster ließe Texte halb alt; der Fuß sagt es); Aufnahme-Schlüssel ab der
   nächsten Aufnahme (die Kantenform gilt für neue Dokumente, das offene Dokument behält seine, ADR-0018 Nachtrag 3);
   Aufräumen beim nächsten Start; `[agents]` sofort, weil der MCP-Server die Datei bei jedem Aufruf liest.
4. **Tastatur:** Tab/Umschalt+Tab durch alle Felder und Knöpfe (Fokusrahmen orange wie in der rechten Leiste),
   Strg+Bild↑/↓ wechselt den Reiter, Leertaste schaltet einen Schalter oder öffnet eine Auswahl, Pfeiltasten wählen in
   der offenen Liste, − und + bzw. Pfeil auf/ab ändern eine Zahl, Enter speichert (auch aus einem Feld), Esc bricht ab
   (bei offener Liste schließt Esc erst die Liste). Solange die Tafel offen ist, greift keine Taste des Editors.
5. **Schreiben (`settings.Schreibe`, ohne Gio):** Die Tafel übergibt nur die Schlüssel, deren Wert sie geändert hat.
   - Die Datei wird beim Speichern **neu gelesen** (nicht der Stand beim Öffnen); so überlebt eine Handänderung, die
     zwischen Öffnen und Speichern geschah, sofern sie nicht denselben Schlüssel betrifft.
   - Steht der Schlüssel im Abschnitt, wird **nur seine letzte Zeile** im **letzten** gleichnamigen Abschnitt ersetzt
     (die letzte gilt, ADR-0018); Einrückung und ein Kommentar am Zeilenende bleiben nicht erhalten, die Form ist
     `schlüssel = wert`. Fehlt er, kommt er hinter die letzte Schlüsselzeile dieses Abschnitts; fehlt der Abschnitt, wird
     er am Dateiende angehängt (mit Leerzeile davor). Alle übrigen Zeilen bleiben Byte für Byte, auch ungültige,
     unbekannte und Kommentare; das Zeilenende (LF oder CRLF) folgt der Datei.
   - Steht ein Schlüssel nur in einem Abschnitt, den der Leser nicht kennt, wird nichts dort geändert.
   - Geschrieben werden nur Normalformen (`yes`/`no` in `[agents]`, `true`/`false` in `[capture] open-editor` und
     `[library] cleanup`, Zahlen ohne Vorzeichen).
   - **Atomar:** temporäre Datei `.settings.ini.schnapp-tmp-*` im selben Ordner (0600), `Sync`, umbenennen, Ordner `Sync`.
     Die Rechte einer vorhandenen Datei bleiben (nie lockerer als 0600 angelegt). Fehlt die Datei, wird sie mit 0600 in
     einem Ordner 0700 angelegt, mit einer Kopfzeile `# Schnapp settings (see the documentation); edited by hand or in
     Settings`.
   - **Verweis (Symlink):** Ist `settings.ini` ein Verweis (häufig bei Dotfile-Sammlungen), wird das Ziel geschrieben —
     atomar neben dem Ziel —, der Verweis bleibt. Ist das Ziel kein reguläres File, wird nicht geschrieben.
   - **Nicht schreiben, sondern melden:** Datei größer als 64 KiB (die Grenze des Lesers), unlesbar, kein reguläres File,
     kein gültiges UTF-8, Ordner nicht beschreibbar. Die Tafel bleibt dann offen und zeigt die Meldung im Fuß in Rot; es
     wird nichts halb geschrieben.
6. **Agenten:** Die Schalter in „Agents“ schreibt nur die Tafel des Editors — ein Mensch klickt „Save“. `schnapp mcp`
   bekommt keinen Weg zum Schreiben: Ein Test prüft, dass `cmd/schnapp/mcp.go` und `internal/mcp` `settings.Schreibe`
   nicht erreichen. Steht `capture`, `library` oder `edit` auf an, zeigt die Tafel unter dem Schalter eine Warnung, was ein
   Agent dann sieht oder darf (Text im Muster b). Voreinstellung bleibt „aus“ (ADR-0018 Nachtrag 4).
7. **Ungültige Zeilen:** Meldet der Leser Warnungen, nennt der Fuß ihre Zahl („2 lines in settings.ini are invalid and
   stay as they are“). Ein Feld zeigt dann den wirksamen Wert (Voreinstellung oder früherer gültiger Wert); speichert
   man es, ersetzt die neue Zeile die ungültige, wenn diese die letzte des Schlüssels ist.
8. **Was nicht in die Einstellungen kommt:** Was Schnapp sich selbst merkt (Zustand der rechten Leiste, Navigator,
   Einrasten, Fenstergröße) bleibt nach ADR-0018 „Abgrenzung“ Zustand und wird nicht gespeichert. F-539 (Drehbücher
   schalten die Leiste mit einem Umschalter ein) entfällt damit; wird der Zustand der Leiste je gespeichert, gilt
   F-539 wieder und die Drehbücher bekommen einen Befehl, der den Zustand setzt.
9. **Umsetzung:** `internal/settings/schreiben.go` (Schreiber, Tests mit eigenem `XDG_CONFIG_HOME`), Logik der Tafel
   ohne Gio (Felder, Fokus, Reiter, Änderungen, Warnungen) testbar über `input.Router` wie Menü und Hilfe, Zeichnen in
   `internal/ui/einstellungen.go`; Texte über `i18n.T` mit deutschem Eintrag; Mutanten-Gruppe `einstellungen`;
   Sichtprobe per Drehbuch `scripts/drehbuecher/einstellungen-175.txt`. Eine Bibliotheksleiste (#174) bringt ihren
   Schlüssel samt Tafelzeile selbst mit.

## Erwogene Alternativen
- **Eigenes Fenster/Prozess `schnapp --settings`:** auch ohne offenen Editor erreichbar, aber zweite Oberfläche,
  zweiter Start von Gio, und ohne Aufnahme hat Schnapp heute kein Hauptfenster. Kann später als Aufruf dazukommen, der
  dieselbe Tafel in einem leeren Fenster zeigt.
- **Datei ganz neu schreiben (aus den Werten):** einfach, verliert aber Kommentare und Handänderungen (gegen ADR-0018).
- **Sofort übernehmen ohne „Save“:** spart einen Klick, aber jede Zwischenstellung eines Schalters (etwa
  Agenten-Einwilligung) stünde kurz in der Datei, und der MCP-Server liest sie sofort.

## Konsequenzen
- ADR-0018: „Schreiben“ ist umgesetzt; neue Schlüssel brauchen eine Tafelzeile.
- ADR-0026: neuer Menüeintrag Datei → „Settings…“ mit Strg+, in der Tastentabelle.
- F-539 entfällt (Punkt 8).
- Andi nimmt die Gestaltung am Sprint-Ende an der Umsetzung ab.
