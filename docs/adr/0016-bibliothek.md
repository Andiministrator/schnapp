# ADR-0016: Bibliothek — Ablage, Index, Aufräumen

## Status
Akzeptiert (Andi, 19.11.2026: E-09 „ja, passt so“; Kleinschreibung in Pfaden; „automatisch aufräumen ja (tage
einstellbar und die option komplett abschaltbar)“).

## Kontext
Jede Aufnahme landet sofort in einer Bibliothek (Anforderung A-02: Vorschau, Datum, Quellprogramm, Suche,
Schlagworte). Aufnahmen zeigen oft Vertrauliches. SQLite bräuchte cgo (ADR-0011), das Projekt meidet es.

## Entscheidung
- **Ablage sichtbar im Bilderordner:** `<XDG-Bilderordner>/schnapp/<JJJJ-MM>/<JJJJ-MM-TT_hh-mm-ss>.png`, z. B.
  `~/Bilder/schnapp/2026-11/2026-11-19_14-35-12.png`. Der Bilderordner kommt aus `user-dirs.dirs` (XDG), sonst
  `~/Pictures`. Zwei Aufnahmen in derselben Sekunde bekommen `-2`, `-3` angehängt. Die Datei ist die Arbeitsdatei
  nach ADR-0008 (PNG mit Blöcken).
- **Kleinschreibung:** In Pfaden und in Dateinamen, die Schnapp oder das Projekt anlegt, steht immer `schnapp`
  (Andi, 19.11.2026). Im Fließtext und in der Oberfläche bleibt der Name „Schnapp“.
- **Index als Cache:** `~/.local/share/schnapp/bibliothek.jsonl` (XDG-Datenverzeichnis), eine JSON-Zeile je
  Aufnahme (Pfad relativ zur Ablage, Datum, Größe, Schlagworte, Quellprogramm, Bearbeitet-Kennung), Vorschaubilder
  unter `~/.cache/schnapp/vorschau/`. Alle Angaben stehen auch im `snAP`-Block der Datei; fehlt oder zerbricht der
  Index, baut Schnapp ihn aus den Dateien neu auf. Kein cgo.
- **Rechte:** Ordner 0700, Dateien 0600; atomar schreiben (temporäre Datei im selben Ordner, dann umbenennen). *Neue Aufnahmen seit Nachtrag 4 exklusiv per Link.*
- **Quellprogramm:** nur verlässlich bei der Fenster-Aufnahme (KWin liefert das Fenster); bei Bereich und Bildschirm
  leer, von Hand ergänzbar.
- **Automatisch aufräumen:** einschaltbar und komplett abschaltbar, Frist in Tagen einstellbar. Voreinstellung: an,
  90 Tage. Erfasst nur Aufnahmen **ohne Schlagworte und ohne Bearbeitung**; sie gehen in den **Papierkorb**
  (freedesktop-Trash, wiederherstellbar), nie direkt gelöscht. Läuft beim Programmstart, nicht als Hintergrunddienst.

## Konsequenzen
- Tests: Index-Neuaufbau aus Dateien, Namenskollision in derselben Sekunde, Rechte 0600/0700, Aufräumen nur der
  passenden Aufnahmen (Schlagwort oder Bearbeitung schützt), Abschalten wirkt.
- Wer Aufnahmen per Dateimanager verschiebt oder löscht, bringt den Index nicht durcheinander: Fehlende Dateien
  fallen beim nächsten Abgleich heraus.
- Der Papierkorb-Weg braucht die freedesktop-Trash-Spezifikation (Ordner `~/.local/share/Trash/` mit `.trashinfo`);
  Aufnahmen auf anderen Dateisystemen erst bei Bedarf.

## Nachtrag 1 (19.11.2026): Umsetzung und Kritiker-Befunde (Sprint 3, `bibliothek/`)

**Status:** Akzeptiert (Umsetzung Karte #33, Nacharbeit Sitzungs-QA F-50 bis F-55).

- **Frist** 1 bis 36500 Tage (darüber liefe die Zeitrechnung über, F-50). Ob eine Aufnahme erfasst wird, entscheidet
  die Datei selbst (Schlagworte, Bearbeitet, Erstellt aus dem `snAP`-Block, sonst Änderungszeit), nie allein der Index
  (F-54).
- **Papierkorb nach freedesktop-Spezifikation:** zuerst die `.trashinfo` exklusiv anlegen (reserviert den Namen),
  dann die Aufnahme verschieben; scheitert das, nimmt Schnapp nur die eigene Reservierung zurück, die Aufnahme bleibt
  liegen. Fremde Einträge werden nie überschrieben (F-53).
- **Vorschaubilder** verlassen die Bibliothek mit ihrer Aufnahme (Aufräumen, Abgleich) und werden beim Neuaufbau
  nachgezogen, wenn sie fehlen (F-51, F-52). Das Löschen trifft nur die eigene Cache-Datei.
- **Rechte:** auch schon vorhandene eigene Ordner (Ablage, Index, Vorschau) werden auf 0700 gebracht (F-55); Bilder-
  und Papierkorb-Ordner des Nutzers bleiben unberührt.
- Offen: F-48 (zwei Prozesse gleichzeitig), schneller Neuaufbau nur aus `snAP` (Karte #45). *Erledigt 20.11.2026: Nachtrag 4.*

## Nachtrag 2 (19.11.2026): Wann aufgeräumt wird (F-88)

**Status:** Akzeptiert (Andi, 19.11.2026: „ja“).

Das automatische Aufräumen läuft, wenn Schnapp zum Aufnehmen startet, **nach dem Speichern** der neuen Aufnahme und
**höchstens einmal am Tag**; den Zeitpunkt des letzten Laufs merkt sich Schnapp im Zustandsordner
(`$XDG_STATE_HOME/schnapp/`, wie der letzte Bereich). Kein Timer, kein Hintergrunddienst. Einstellungen
`[library] cleanup` und `cleanup-days` (ADR-0018). Scheitert das Aufräumen, gibt es nur eine Warnung — die Aufnahme ist
dann schon gespeichert. Umsetzung: Karte #54.

## Nachtrag 3 (20.11.2026): Symlink-Ablage, leerer Einhängepunkt, liegen gebliebene temporäre Dateien (F-270, F-271, F-274)

**Status:** Akzeptiert (Claude Sonnet 5.5, Umsetzung Karte #114; Claudes Entscheidung, das Entfernen nach Alter ist als offene Frage an Andi vermerkt: E-22).

1. **Ablage wird vor dem Ablaufen aufgelöst** (`filepath.EvalSymlinks`, wie `Relativ`): Neuaufbau und
   `VerwaisteVorschauen` laufen sonst bei einer Ablage als Symlink gar nicht ab (F-271, F-270).
2. **`VerwaisteVorschauen` liefert `ErrAblageLeer` statt einer Liste**, wenn die Ablage keine einzige Aufnahme enthält,
   der Cache aber Vorschaubilder (leerer Einhängepunkt, verschobener Ordner). Gelöscht wird dort nichts.
3. **Temporäre Dateien (`.schnapp-tmp-*`)** sind nach dem Link-Weg ein zweiter Name der Aufnahme (Kernprinzip 2; ADR-0022 Nachtrag 2: liegen gebliebene Fassungen):
   Ein Fehler beim Entfernen nach dem Link wird als `ErrTempRest` gemeldet (Aufnahme bleibt gültig); Neuaufbau und
   Speichern räumen eigene Reste im Ordner, die älter als eine Minute sind (nur reguläre Dateien mit eigenem Präfix,
   nie Symlinks, nie Aufnahmen oder Vorschaubilder). Gewählt wurde das Entfernen nach Alter statt nur der Reste mit
   Aufnahme-Inode, weil auch ein halb geschriebener Rest Bildpunkte einer Aufnahme trägt. Nach Link und rename wird der
   Ordner gesynct (Best Effort).

## Nachtrag 4 (20.11.2026, nachgetragen zu c0d5238 und 5ee6d05): Exklusives Ablegen, Neuaufbau aus Metadaten (F-48, F-272, F-273)

**Status:** Akzeptiert (Claude, Umsetzung Karten #45 und #115; Verify Fable).

- **Neue Aufnahmen werden exklusiv angelegt, nie überschrieben** — auch nicht gegen einen zweiten Schnapp-Prozess
  (F-48). Die Datei wird einmal vollständig in eine temporäre Datei im selben Ordner geschrieben; der endgültige Name
  entsteht per Hardlink (`os.Link`), der bei belegtem Namen scheitert, dann folgt `-2`, `-3` …. Kann das Dateisystem
  oder die Stelle keine Hardlinks (FAT/exFAT, manche Netzfreigaben), reserviert ein mit `O_EXCL` angelegter leerer
  Platzhalter den Namen, und die temporäre Datei wird darüber umbenannt; zwischen Reservierung und Umbenennen ist
  kurz eine leere Datei sichtbar, ein Absturz genau dort lässt sie liegen (der Neuaufbau überspringt sie als
  beschädigt, die Aufnahme steckt noch in der temporären Datei). Das Zurückschreiben einer bestehenden Aufnahme bleibt
  beim Umbenennen.
- **Folge des Link-Wegs:** Bis zum Entfernen ist die temporäre Datei ein zweiter Name der Aufnahme (Nachtrag 3 Punkt 3).
- **Der Neuaufbau liest nur Metadaten** (`fileformat.LeseMetadaten`): `snAP` und die Köpfe, keine Pixel. Urteil und
  Größe sind dieselben wie beim vollen Lesen (`document.PruefeInhalt`, F-272); was nur das Dekodieren zeigt (kaputte
  Bilddaten), fällt erst beim Öffnen auf. Angekündigte Blocklängen reservieren keinen Speicher im Voraus (F-273).
- **`VerwaisteVorschauen` listet nur**, zu welchen Vorschaubildern keine Aufnahme mehr gehört; gelöscht wird dort
  nichts, ein Aufräumen ist eine eigene, freizugebende Aktion.

## Nachtrag 5 (20.11.2026): Temporäre Reste ohne Altersgrenze, beim Entfernen und im Vorschau-Ordner (Verify zu #114)

**Status:** Akzeptiert (Claude Sonnet 5.5, Nacharbeit Verify Fable zu #114). Ergänzt Nachtrag 3 Punkt 3.

1. **Gleicher Inode geht ohne Altersgrenze (Z-1).** Ein Rest aus dem Link-Weg trägt die Änderungszeit der Aufnahme: Wer
   binnen einer Minute nach dem Ablegen schwärzt, hätte ihn behalten, bei verstellter Uhr nie. `Speichern` und
   `Entfernen` entfernen deshalb jede eigene temporäre Datei im Ordner, die denselben Inode hat wie die Datei, die
   ersetzt oder entfernt wird (`os.SameFile`); sonst bleibt es bei „älter als eine Minute“. Weiterhin nur reguläre Dateien
   mit dem Präfix `.schnapp-tmp-` im selben Ordner, nie ein Symlink, nie eine Aufnahme, nie ein Vorschaubild. Was bleibt,
   weil es jünger ist, oder eine Zeit weit in der Zukunft trägt, wird nicht gelöscht, sondern als `ErrTempRest` gemeldet;
   der Neuaufbau meldet zukunftsdatierte Reste in `NeuaufbauErgebnis.TempReste`.
2. **`Entfernen` räumt mit (Z-2).** Nach dem Löschen der Aufnahme wären die Kopien mit gleichem Inode ihr letzter Name.
   `Entfernen` räumt sie nach Punkt 1 und meldet, was bleibt, als `ErrTempRest`; die Datei gilt dann trotzdem als entfernt.
   Der Editor nennt die Kopie zusätzlich im Hinweis (auch im Rückblick nach ADR-0022 Nachtrag 2).
3. **Vorschau-Ordner (Z-3).** `erneuereVorschau` (also Ablegen und Speichern) räumt dort eigene temporäre Dateien nach
   denselben Regeln (Altersgrenze, gleicher Inode wie das zu ersetzende Vorschaubild); ein Absturz zwischen Schreiben und
   Umbenennen ließ sonst ein verkleinertes Bild des alten Stands für immer liegen. Anders als im Ordner der Aufnahmen
   werden junge fremde Reste dort **nicht** gemeldet (jedes Ablegen eines zweiten Prozesses schreibt kurz eine solche
   Datei); gemeldet werden Reste mit Zeit weit in der Zukunft und solche, die sich nicht entfernen lassen (`ErrTempRest`).
4. **Aufrufer melden (Z-4).** Die Aufnahme (`cmd/schnapp`) warnt auf stderr, der Editor nennt die Kopie beim Speichern
   (auch zusammen mit der alten Vorschau), beim Ausweichen und beim Entfernen. `NeuaufbauErgebnis.TempReste` liest kein
   Programmteil: Es gibt keinen Befehl, der den Neuaufbau auslöst (nur die Bibliothek, für Tests und eine spätere
   Prüfroutine); bewusst kein Verbraucher, bis es einen Aufrufer gibt.

