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
- **Rechte:** Ordner 0700, Dateien 0600; atomar schreiben (temporäre Datei im selben Ordner, dann umbenennen).
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
- Offen: F-48 (zwei Prozesse gleichzeitig), schneller Neuaufbau nur aus `snAP` (Karte #45).

## Nachtrag 2 (19.11.2026): Wann aufgeräumt wird (F-88)

**Status:** Akzeptiert (Andi, 19.11.2026: „ja“).

Das automatische Aufräumen läuft, wenn Schnapp zum Aufnehmen startet, **nach dem Speichern** der neuen Aufnahme und
**höchstens einmal am Tag**; den Zeitpunkt des letzten Laufs merkt sich Schnapp im Zustandsordner
(`$XDG_STATE_HOME/schnapp/`, wie der letzte Bereich). Kein Timer, kein Hintergrunddienst. Einstellungen
`[library] cleanup` und `cleanup-days` (ADR-0018). Scheitert das Aufräumen, gibt es nur eine Warnung — die Aufnahme ist
dann schon gespeichert. Umsetzung: Karte #54.

## Nachtrag 3 (20.11.2026): Symlink-Ablage, leerer Einhängepunkt, liegen gebliebene temporäre Dateien (F-270, F-271, F-274)

**Status:** Akzeptiert (Umsetzung Karte #114).

1. **Ablage wird vor dem Ablaufen aufgelöst** (`filepath.EvalSymlinks`, wie `Relativ`): Neuaufbau und
   `VerwaisteVorschauen` laufen sonst bei einer Ablage als Symlink gar nicht ab (F-271, F-270).
2. **`VerwaisteVorschauen` liefert `ErrAblageLeer` statt einer Liste**, wenn die Ablage keine einzige Aufnahme enthält,
   der Cache aber Vorschaubilder (leerer Einhängepunkt, verschobener Ordner). Gelöscht wird dort nichts.
3. **Temporäre Dateien (`.schnapp-tmp-*`)** sind nach dem Link-Weg ein zweiter Name der Aufnahme (ADR-0022 Punkt 4):
   Ein Fehler beim Entfernen nach dem Link wird als `ErrTempRest` gemeldet (Aufnahme bleibt gültig); Neuaufbau und
   Speichern räumen eigene Reste im Ordner, die älter als eine Minute sind (nur reguläre Dateien mit eigenem Präfix,
   nie Symlinks, nie Aufnahmen oder Vorschaubilder). Gewählt wurde das Entfernen nach Alter statt nur der Reste mit
   Aufnahme-Inode, weil auch ein halb geschriebener Rest Bildpunkte einer Aufnahme trägt. Nach Link und rename wird der
   Ordner gesynct (Best Effort).

