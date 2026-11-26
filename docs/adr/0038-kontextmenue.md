# ADR-0038: Kontextmenü (rechte Maustaste) auf der Leinwand und an Vorschauen der Bibliothek

## Status
Akzeptiert (Claude Opus 5.5, 26.11.2026, Karte #313 in Sprint 14, im Auftrag Andis: „arbeite selbstständig und ziehe
auch selbstständig karten“). Grundlage: Wunsch Andis vom 25.11.2026 („Rechtsklick-Menü für Kopieren, Teilen,
Umbenennen usw.“). Ergänzt ADR-0026 (Menü und Kürzel), ADR-0031 (Oberfläche), ADR-0032 (Ebenen), ADR-0036
(Bibliothek).

## Kontext
Alle Handlungen stehen heute in der Menüzeile, mit Kürzel. Wer mit der Maus arbeitet, erwartet sie zusätzlich am
Objekt unter dem Zeiger: Snagit, Krita, Inkscape und Dolphin zeigen auf Rechtsklick die Handlungen für das, was darunter
liegt. Die Menü-Mechanik (`internal/ui/menue.go`) kann schon Aufklapplisten an beliebigen Ankern (Farbe, Zoom,
Pfeil-Stil, Strichart) samt Tastatur, Klick daneben und Sperre beim Speichern.

## Entscheidung
1. **Eine Liste mehr in der vorhandenen Mechanik** (`listeKontext`), kein eigenes Widget: dieselbe Tastatur (Pfeile,
   Pos1/Ende, Enter, Esc), derselbe Klick daneben, dieselbe Sperre beim Speichern, dieselben Einträge (`menueEintrag`
   mit `aktion`), also dieselben Wirkungen und Kürzelanzeigen wie im Menü der Zeile. Anker ist der Punkt des
   Rechtsklicks; die Liste klappt rechts unten davon auf und rückt ins Fenster, wenn sie nicht passt.
2. **Leinwand (Teil 1):** Ein Rechtsklick bestätigt eine laufende Texteingabe und wählt das Objekt unter dem Zeiger mit
   derselben Trefferregel wie das Auswählen (sichtbar, nicht gesperrt; unabhängig vom aktiven Werkzeug, das Werkzeug
   bleibt; Zuschneiden und Stauchen wählen nie, dort gilt das Menü dem Bild und die Auswahl bleibt). Trifft er nichts,
   hebt er die Auswahl auf. Während eines Zugs (Ziehen, Regler, Farbzug) wirkt er nicht. Pipette und Bereichsmodus der
   Texterkennung enden wie mit Esc. Ein zweiter Rechtsklick bei offenem Kontextmenü öffnet es am neuen Punkt für das
   neue Ziel (wie Qt/KDE); ein Linksdruck hinter der offenen Liste wirkt nicht.
   - **Mit Objekt:** Text bearbeiten (nur Text und Sprechblase) · Doppeln · Löschen │ Eine Ebene nach vorn · Eine Ebene
     nach hinten │ Ausblenden · Sperren · Umbenennen │ Als Schnellstil sichern (grau mit Erklärung, wenn „+“ der
     Schnellstile grau wäre) │ Farben tauschen (nur mit Füllung).
   - **Ohne Objekt:** Kopieren · Exportieren… · Text aus dem Bild kopieren │ Einpassen · 1:1 │ Ganzes Bild (nur, wenn
     zugeschnitten oder gestaucht ist).
   - Ausschneiden/Einfügen von Objekten fehlt, bis es sie gibt (Karte „Objekte kopieren/einfügen“); das Menü bekommt
     sie dann dazu.
3. **Bibliothek (Teil 2):** Ein Rechtsklick auf eine Vorschau (Leiste und volle Ansicht) setzt den Fokus auf sie und
   zeigt: Öffnen · Im Dateimanager zeigen │ Schlagworte… (nur volle Ansicht, wie F2) │ In den Papierkorb. „Im
   Dateimanager zeigen“ ruft `org.freedesktop.FileManager1.ShowItems` auf dem Sitzungsbus (Dolphin und Nautilus kennen
   es); fehlt der Dienst, sagt die Fußzeile es. Kopieren und Exportieren einer nicht offenen Aufnahme, und alles für
   mehrere Vorschauen, kommen mit der Mehrfachmarkierung (Karte #361), die dieselbe Liste erweitert.
4. **Kein „Teilen“ im ersten Schritt.** KDEs Purpose ist kein Portal, das Portal „Email“ verschickt nur Mail. „Im
   Dateimanager zeigen“ führt zum Teilen-Menü des Dateimanagers; ein eigener Teilen-Eintrag braucht eine eigene ADR.
5. **Kein „Umbenennen“ in der Bibliothek:** Die Dateinamen sind Zeitstempel und bleiben es (ADR-0016); was eine Aufnahme
   benennt, sind ihre Schlagworte. „Umbenennen“ im Kontextmenü der Leinwand benennt die Ebene (ADR-0032).
6. **Tastatur:** Umschalt+F10 öffnet das Kontextmenü für die Auswahl (Anker: Mitte ihres Rahmens, sonst Mitte der
   Leinwand) bzw. in der vollen Ansicht für die Vorschau mit Fokus. Die Menütaste der Tastatur meldet Gio unter
   Wayland nicht (`xkb_unix.go` kennt `XKB_KEY_Menu` nicht); sie kommt mit einem Gio-Patch dazu, wenn es sich lohnt.
7. **Bestehende Rechtsklicks bleiben:** An Schnellstil-Kacheln und eigenen Farbfeldern entfernt der Rechtsklick weiter
   einen eigenen Eintrag (ADR-0037, ADR-0021 Nachtrag 24) — dort gibt es keine weiteren Handlungen, ein Menü mit einem
   einzigen Eintrag wäre ein Umweg. Das Overlay der Aufnahme behält seinen Rechtsklick (Abbruch).

## Erwogene Alternativen
- **Eigenes schwebendes Menü-Widget:** doppelte Tastatur- und Fokuslogik, die in `menue.go` schon getestet ist.
- **Rechtsklick wählt nicht:** wie manche Editoren, die das Menü für die alte Auswahl zeigen — verwirrt, wenn der
  Zeiger auf einem anderen Objekt steht.
- **Alle Menüeinträge der Zeile:** zu lang; das Kontextmenü zeigt nur, was zum Ziel unter dem Zeiger passt.

## Konsequenzen
- `internal/canvas` bekommt `KontextBei(p)` (Treffer und Auswahl ohne Zug); `internal/ui` die Liste `listeKontext`
  mit Anker am Klickpunkt und die Einträge je Ziel.
- Tests: Treffer und Auswahl (canvas), Einträge je Ziel, Rechtsklick öffnet am Punkt, Umschalt+F10, Wirkung eines
  Eintrags (ui), Mutanten, Sichtprobe.

## Umsetzung Teil 1 (26.11.2026, 5052dae, Nacharbeit nach Sitzungs-Kritik F-860 bis F-866)
Leinwand umgesetzt wie Punkte 1, 2, 6 und 7. In der vollen Ansicht der Bibliothek ist Umschalt+F10 bis Teil 2
wirkungslos (öffnet nicht das Menü Datei, F-860). Die Übersicht F1 nennt Umschalt+F10 unter „Other keys“ (F-861).
Punkte 3 bis 5 (Bibliothek) folgen mit Teil 2. Tests `internal/canvas/kontext_test.go`, `internal/ui/kontext_test.go`,
Mutanten `scripts/mutanten/kontextmenue.json` (KM1–KM21), Sichtprobe `scripts/drehbuecher/kontextmenue-313.txt`.

## Umsetzung Teil 2 (26.11.2026)
Bibliothek umgesetzt wie Punkte 3 bis 6. Die Einträge gelten der Vorschau, an der die Liste aufging (Pfad relativ zur
Ablage, `kontextMenue.vorschau`), nicht der Vorschau mit Fokus; liest der Index inzwischen neu, sucht der Eintrag sie
wieder, fehlt sie, tut er nichts. Eigene Handlungen `aktVorschau…` neben `aktBib…` (Menü Bearbeiten der vollen Ansicht).
Entscheidungen im Rahmen des Auftrags:
- **Öffnen an der offenen Aufnahme:** in der Leiste grau mit Erklärung (ein Klick darauf bleibt dort ohnehin still), in
  der vollen Ansicht führt es zurück zum Bild wie Enter.
- **Umschalt+F10 auch in der Leiste:** Hat eine Vorschau der Leiste den Fokus, öffnet Umschalt+F10 deren Menü statt des
  Menüs der Auswahl. In der vollen Ansicht nur, wenn das Raster den Fokus hat (nicht im Suchfeld, nicht beim Bearbeiten
  der Schlagworte); Anker ist die Mitte der Vorschau.
- **Fokus:** Rechtsklick setzt ihn auf die Vorschau; Esc und ein Eintrag geben ihn in der Leiste an die Vorschau zurück,
  in der vollen Ansicht an die Ansicht.
- **Dateimanager:** `org.freedesktop.FileManager1.ShowItems` in einer Goroutine (Frist 10 s, höchstens ein Aufruf zugleich,
  der Eintrag ist solange grau), eigene kurze Busverbindung (`internal/portal/dateimanager.go`). Fehlt der Dienst
  (ServiceUnknown, NameHasNoOwner, Unknown*), sagt die Fußzeile es; andere Fehler allgemein.
- Ein zweiter Rechtsklick bei offenem Menü öffnet es auch an einer anderen Vorschau bzw. von der Vorschau auf der
  Leinwand neu (`kontextErneut`).
Tests `internal/ui/kontext_vorschau_test.go`, `internal/portal/dateimanager_test.go`, Mutanten KV1–KV21 in
`scripts/mutanten/kontextmenue.json`, Sichtprobe `scripts/drehbuecher/kontextmenue-313-bibliothek.txt`. Gegen einen echten
Dolphin ist ShowItems noch ungeprüft (Echtlauf Andi).
