# ADR-0036: Bibliothek in der Oberfläche — Leiste der letzten Aufnahmen und volle Ansicht

## Status
Akzeptiert (Claude Opus 5.5, 24.11.2026, Karte #174 in Sprint 12, im Auftrag Andis: „arbeite selbstständig“).
Gestaltung nach Muster `tmp/sprint12/muster/bilder/174-bibliothek-{a-leiste,b-ansicht}.png`; Andi nimmt sie mit der
fertigen Umsetzung ab (gebündelt am Sprint-Ende). Grundlage: Wunsch Andis vom 22.11.2026 („die bibliothek sollte per
button aufgehen, aber auch (per schalter de-/aktivierbar) unten als leiste zu sehen sein (so wie bei snagit)“), dazu
aus Karte #173: Agenten-Aufnahmen per Schalter einblendbar. Setzt A-02 in der Oberfläche um; ergänzt ADR-0016
(Bibliothek), ADR-0018/ADR-0034 (Einstellungen), ADR-0026 (Menü und Kürzel), ADR-0031 (Oberfläche).

## Kontext
Die Bibliothek (ADR-0016) legt jede Aufnahme ab, führt einen Index und Vorschaubilder, kann suchen und Schlagworte
setzen — aber nur im Code. Wer eine frühere Aufnahme wieder öffnen will, muss den Dateimanager nehmen. Snagit zeigt
die letzten Aufnahmen als Leiste („Tray“) unter dem Bild und eine volle Bibliothek.

## Entscheidung

1. **Leiste der letzten Aufnahmen** unten im Editor zwischen Leinwand und Fußzeile, so breit wie die Leinwand
   (Werkzeugleiste und rechte Leiste bleiben voll hoch), Höhe 104 dp. Links der Knopf „Library“ (öffnet Punkt 3),
   daneben die Vorschaubilder der Bibliothek, **neueste links**, je 120 × 68 dp mit Zeitangabe darunter (heute: Uhrzeit,
   gestern: „yesterday“, sonst Wochentag und Datum, Sprache über `date format`). Die offene Aufnahme ist grün umrandet.
   Mausrad rollt waagerecht, Tastatur über die Fokusfahrt. Agenten-Aufnahmen stehen nicht in der Leiste.
2. **Schalter:** Ansicht → „Recent captures“/„Letzte Aufnahmen“, **Strg+Umschalt+R**, gemerkt in
   `[library] bar = yes|no` (Voreinstellung **yes**; Nachtrag zu ADR-0018 und Zeile in der Einstellungstafel nach
   ADR-0034 Punkt 2, Reiter Library). In Fenstern unter 420 dp Leinwandhöhe blendet sich die Leiste von selbst aus
   (wie die rechte Leiste unter 752 dp Breite), ohne die Einstellung zu ändern.
3. **Volle Ansicht** im Editor-Fenster: Menü Datei → „Library“, **Strg+Umschalt+L**, oder der Knopf in der Leiste.
   Sie ersetzt Werkzeugleiste, Leinwand und rechte Leiste; Menüzeile und Fußzeile bleiben. Kopf: Suchfeld
   (Schlagwort, `library.Anfrage.Schlagwort`), Zeitraum (Today, 7 days, 30 days, This year, All), Schalter
   „Show agent captures“ (aus bei jedem Öffnen, nicht gemerkt; `Anfrage.OhneAgenten`), Knopf „Back to image“. Darunter
   ein Raster nach Monaten (Überschrift mit Zahl der Aufnahmen), neueste zuerst; je Kachel Vorschau, Datum und Uhrzeit,
   Größe und Schlagworte. Grün umrandet die offene Aufnahme, orange der Tastaturfokus.
4. **Bedienung der Ansicht:** Pfeiltasten wandern im Raster, Pos1/Ende springen, Bild↑/↓ blättern; Enter oder
   Doppelklick öffnet; **Entf** legt die Aufnahme in den Papierkorb (wie das Aufräumen, freedesktop, rückholbar; die
   offene Aufnahme ist ausgenommen, der Eintrag ist dann grau); **F2** bearbeitet die Schlagworte (Feld unter der
   Kachel, Komma trennt, Enter übernimmt über `SetzeSchlagworte`); Esc oder „Back to image“ zurück. Strg+F setzt den
   Fokus ins Suchfeld. Solange die Ansicht offen ist, wirken die Werkzeugtasten nicht.
5. **Öffnen einer anderen Aufnahme (Leiste oder Ansicht) im selben Fenster:** Zuerst wird die offene Aufnahme
   gespeichert wie beim Schließen (Speichern im Hintergrund mit den Fristen aus ADR-0020 Nachträgen 6 bis 9; scheitert
   es, bleibt sie offen und die Statuszeile meldet es — gewechselt wird dann nicht). Danach lädt der Editor die gewählte
   Datei wie `schnapp DATEI` (`ladeFuerEditor` mit Budget-Prüfung, Rettungskopie nach ADR-0029, eigene Sitzung,
   eigener Rückgängig-Verlauf; der Verlauf der vorigen ist danach weg). Fenster, Zoomstufe-Voreinstellung, Zustand
   der Leisten bleiben. Ist eine Datei beschädigt oder neuer, öffnet sie als einfaches Bild wie bisher.
6. **Vorschaubilder** nur aus dem Vorschau-Cache (`library.VorschauPfad`), nie aus dem Original; fehlt eine, wird sie
   im Hintergrund erzeugt (wie heute beim Ablegen) und bis dahin ein graues Feld gezeigt. Geladen werden nur die
   sichtbaren Kacheln (plus eine Zeile Vorlauf); höchstens 64 MiB dekodierte Vorschauen im Speicher, ältere fallen
   heraus. Index und Vorschauen werden im Hintergrund gelesen; die Oberfläche wartet nie auf die Platte.
7. **Aktualität:** Die Leiste liest den Index beim Öffnen des Editors, nach jedem eigenen Speichern und beim Öffnen der
   Ansicht neu (`library.Laden`/`Suchen`); kein Dateiwächter (inotify) — eine Aufnahme aus einem zweiten Fenster
   erscheint beim nächsten dieser Anlässe.
8. **Nicht in diesem Schritt:** Ordner oder Sammlungen, mehrere auswählen, Ziehen in andere Programme, Umbenennen,
   Vorschau einer fremden Datei außerhalb der Bibliothek, Suche im Text (OCR, ADR-0035).
9. **Umsetzung:** Logik ohne Gio in `internal/canvas` oder einem eigenen Paket (Raster, Fokus, Zeitraum, Zeitangabe,
   Speicher der Vorschauen), Zeichnen in `internal/ui/bibliothek.go` und `internal/ui/leiste_unten.go`, Wechsel des
   Dokuments in `cmd/schnapp/editor.go` und `internal/ui` (der Editor bekommt eine Anfrage „öffne Pfad“, die die
   Hauptschleife nach dem Speichern ausführt). Texte über `i18n`, Mutanten-Gruppe `bibliothek-ui`, Sichtproben per
   Drehbuch.

## Erwogene Alternativen
- **Neues Fenster je geöffneter Aufnahme (`schnapp DATEI` als eigener Prozess):** einfacher, aber viele Fenster und
  kein Bezug zur Leiste; Snagit wechselt im Fenster.
- **Bibliothek als eigenes Programmfenster:** wie oben, dazu eine zweite Oberfläche ohne Bild.
- **Dateiwächter (inotify) für die Leiste:** sofort aktuell, aber ein weiterer Hintergrundvorgang für einen seltenen
  Fall.

## Konsequenzen
- ADR-0018 Nachtrag 5: `[library] bar`; ADR-0034: Zeile in der Tafel; ADR-0026: zwei Menüeinträge mit Kürzeln.
- Die Leiste nimmt der Leinwand 104 dp Höhe; Drehbücher mit festen Koordinaten setzen sie am Anfang aus (wie die
  rechte Leiste, ADR-0031 Nachtrag 3) — über den Menüeintrag, nicht über einen Umschalter, sobald die Einstellung
  gemerkt wird (Lehre aus F-539).
