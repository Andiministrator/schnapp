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


## Nachtrag 6 (24.11.2026): Index gegen zweiten Prozess, Abgleich bei fehlender Ablage, Wiederherstellen (F-673, F-675, F-678, F-685, F-686, F-689, F-690)

**Status:** Akzeptiert (Claude Opus 5.5, Sitzungs-QA zu Karte #174, Kritikberichte A und B vom 24.11.2026).

1. **Sperre des Index (F-686).** `b.mu` gilt nur im eigenen Prozess; ein zweites Editor-Fenster oder der MCP-Server
   konnte eine angehängte Indexzeile verlieren, wenn ein anderer Prozess den Index gerade gelesen hatte und danach per
   Umbenennen ersetzte. Jedes Lesen-Ändern-Schreiben (Abgleich, Speichern und SetzeSchlagworte über `ersetzeImIndex`,
   InPapierkorb, Entfernen, Aufräumen, Neuaufbau) und jedes Anhängen (`haengeAn`) läuft jetzt zusätzlich unter einer
   `flock`-Sperre auf `<Index>.lock` neben dem Index (0600, Ordner wie der Index, `internal/library/indexsperre.go`).
   Die Sperrdatei wird nie gelöscht. Lesen allein sperrt nicht. Kann das Dateisystem kein flock (ENOLCK,
   ENOTSUP/EOPNOTSUPP), geht es ohne Sperre weiter; hält ein anderer die Sperre länger als 10 s, scheitert nur der
   Indexschritt (`ErrIndex`), die Aufnahme bleibt. Aufräumen liest den Index unter der Sperre frisch, statt den Stand vom
   Anfang zu schreiben. Der Neuaufbau hält die Sperre über Ablaufen, Metadaten und
   Schreiben des Index; fehlende Vorschaubilder heilt er erst danach, nur unter `b.mu` (F-691: das volle Dekodieren je
   Bild hielt die Sperre bei großer Ablage sonst über die Frist; 10 000 Aufnahmen rund 0,8 s, `BenchmarkNeuaufbau10000`).
2. **Abgleich bei fehlender oder leerer Ablage (F-685).** Fehlen Einträge und enthält die Ablage keine einzige PNG-Datei
   (nicht eingehängt, leerer Einhängepunkt, umbenannt, Verweis ins Leere), schreibt `Abgleich` nichts und meldet
   `ErrAblageLeer`, wie `VerwaisteVorschauen` (Nachtrag 3 Punkt 2). Preis: Hat der Nutzer wirklich jede Aufnahme im
   Dateimanager gelöscht, bleiben ihre Einträge stehen, bis die nächste Aufnahme in der Ablage liegt; dann fallen sie
   beim nächsten Abgleich weg.
3. **Wiederherstellen des Index.** Fehlt die Indexdatei ganz und enthält die Ablage Aufnahmen, baut `Abgleich` den Index
   neu auf (`Neuaufbau`, unter b.mu und der Sperre; ein leerer, aber vorhandener Index löst nichts aus, ein erster Start
   ohne Ablage legt nichts an). Damit gibt es einen Weg für den Nutzer: **Indexdatei
   (`$XDG_DATA_HOME/schnapp/bibliothek.jsonl`, sonst `~/.local/share/schnapp/bibliothek.jsonl`) löschen oder beiseitelegen
   und Schnapp öffnen** — der Editor gleicht beim Start ab (ADR-0036 Zusatz F-672). Ergänzt Nachtrag 5 Punkt 4: Der
   Neuaufbau hat damit einen Aufrufer; `NeuaufbauErgebnis.TempReste` liest er weiterhin nicht.
4. **Abgleich prüft ohne Sperren (F-690).** Die Lstat je Eintrag laufen ohne `b.mu` und ohne die Sperre des Index. Danach
   wird unter beiden frisch gelesen; gestrichen wird nur, was beim Prüfen fehlte und auch jetzt noch fehlt, neu
   Hinzugekommenes bleibt. Messung `BenchmarkAbgleich10000` (10 000 vorhandene Dateien, nichts zu streichen, lokale
   SSD, drei Läufe): rund 31 ms je Abgleich, davon rund 15 ms das Lesen des Index (vgl. `BenchmarkSuchen10000`); `b.mu`
   ist davon nur für dieses Lesen belegt.
5. **SetzeSchlagworte (F-673, F-675).** Wie `Speichern`: Stand der Datei vor dem Lesen merken, direkt vor dem Umbenennen
   vergleichen, sonst `ErrGeaendert` (die Datei behält den neueren Stand, etwa eine Schwärzung aus einem zweiten
   Fenster; Kernprinzip 2). Ein Pfad, dessen Ordner als Verweis aus der Ablage führt, ist `ErrPfad`
   (`b.Relativ(voll) == rel`, Kritik A-11). Dieselbe Verweisprüfung hat jetzt `InPapierkorb`.
6. **InPapierkorb: Erfolg mit Indexwarnung (F-678).** `ErrIndex` kommt nur, wenn die Datei schon im Papierkorb liegt;
   dann ist auch `Verschoben` gefüllt. Jeder andere Fehler heißt „nicht verschoben“. Aufrufer unterscheiden mit
   `errors.Is(err, ErrIndex)`.
7. **Teilwort (F-689).** Die Teilwortsuche faltet wie `strings.EqualFold` (jedes Zeichen auf die kleinste Form seiner
   `unicode.SimpleFold`-Klasse): Schluss-Sigma, langes s und Kelvin-Zeichen treffen beim Teilwort genau dann, wenn sie
   beim ganzen Schlagwort treffen. Ein Teilwort nur aus Leerraum gilt als keins.

Tests in `internal/library/haertung_test.go`, Mutanten `scripts/mutanten/bibliothek-haertung.json` (HB1–HB23).

## Nachtrag 7 (24.11.2026): Aufnahmen ohne Indexzeile werden ergänzt (F-700)

**Status:** Akzeptiert (Claude Opus 5.5, Karte #269, Abschluss-Kritik Sprint 12 B-3).

1. **Ablegen an einen fehlenden Index.** Fehlt die Indexdatei, legte das Anhängen sie mit nur der neuen Zeile an; der
   Abgleich danach sah einen vorhandenen Index, und alle älteren Aufnahmen blieben unsichtbar. Der übliche Start ist die
   Aufnahme, der Weg „Index löschen, Schnapp öffnen“ aus Nachtrag 6 Punkt 3 trug also nicht. Jetzt baut `haengeAn` den
   Index in diesem Fall unter derselben Sperre aus der Ablage neu auf (die neue Datei liegt schon dort); scheitert der
   Neuaufbau, wird wie bisher nur angehängt. Fehlende Vorschaubilder heilt es danach ohne die Sperre (F-691), außer der
   neuen, die das Ablegen selbst schreibt. Die allererste Aufnahme kostet damit ein Ablaufen einer Ablage mit einer
   Datei.
2. **Abgleich ergänzt.** `Abgleich` läuft die Ablage zusätzlich ab (`ohneIndexzeile`, ohne `b.mu` und ohne Sperre wie
   die Dateiprüfung, Punkt 4 von Nachtrag 6) und liest die Metadaten jeder PNG-Datei, die keine Zeile hat. Unter beiden
   Sperren werden nur die ergänzt, die noch keine Zeile haben und deren Stand (Gerät, Inode, Größe, Zeit) seit dem Lesen
   gleich ist; eine inzwischen geänderte holt der nächste Abgleich. Der Index wird danach sortiert wie beim Neuaufbau
   geschrieben, fehlende Vorschaubilder der ergänzten heilen nach der Freigabe der Sperre. Eigene temporäre Reste,
   fremde und beschädigte PNG-Dateien zählen nicht. Damit holt der nächste Start einer Leiste oder Ansicht Aufnahmen
   nach, deren Zeile fehlt (Absturz zwischen Datei und Zeile, `ErrIndex` beim Ablegen, aus einer Sicherung
   zurückgespielter Ordner), und das Aufräumen sieht sie wieder. Der Rückgabewert bleibt die Zahl der entfernten
   Einträge.
3. **Leerer Index.** Ersetzt den Satz „ein leerer, aber vorhandener Index löst nichts aus“ in Nachtrag 6 Punkt 3: Ein
   leerer Index wird wie jeder andere ergänzt, einmal — danach ist er nicht mehr leer. Den Neuaufbau aus
   `neuaufbauOhneIndex` löst weiterhin nur das Fehlen der Datei aus.
4. **Kosten.** Das Ablaufen liest je Monatsordner das Verzeichnis, Metadaten nur für Dateien ohne Zeile. Fremde
   PNG-Dateien in der Ablage werden bei jedem Abgleich erneut gelesen (nur `snAP`-Suche und Kopf). Auf einer
   Netz-Ablage kostet das Ablaufen zusätzlich zu den Lstat je Eintrag; die Widerlegungsbedingung von F-700 (spürbar
   langsamer) ist mit einer lokalen Messung nicht zu prüfen und bleibt beobachtet.

Tests in `internal/library/index_ergaenzen_test.go`, Mutanten HB27–HB31 in `scripts/mutanten/bibliothek-haertung.json`
(HB4 seitdem äquivalent).

## Nachtrag 8 (25.11.2026): haengeAn hängt nicht doppelt an und heilt keine Vorschauen (F-755, F-756)

**Status:** Akzeptiert (Claude Sonnet 5.5, Karte #287, Sitzungs-Kritik 25.11.).

1. **Keine doppelte Zeile.** Hat ein zweiter Prozess die Datei einer neuen Aufnahme schon abgelegt, seine Zeile aber noch
   nicht angehängt, ergänzt der Abgleich des ersten sie (Nachtrag 7 Punkt 2); danach hängte `haengeAn` des zweiten sie
   nochmal an. Jetzt prüft `haengeAn` unter der Sperre des Index, ob der Pfad schon eine Zeile hat, und ersetzt sie
   dann mit den Angaben des neuen Eintrags, statt anzuhängen. Ersetzen statt Überspringen, weil die Zeile auch zu einer
   verschwundenen Datei gehören kann, deren Namen die neue Aufnahme wieder trägt (Speichern nach Wegfall der Datei,
   `TestSpeichernDateiWegWaehrendEditor`): Sie bliebe sonst mit den alten Angaben stehen (Nachbesserung Claude Opus 5.5).
2. **Kein Heilen im Vordergrund.** Ersetzt in Nachtrag 7 Punkt 1 den Satz über das Heilen: Baut `haengeAn` den fehlenden
   Index neu auf, heilt es die fehlenden Vorschaubilder der älteren Aufnahmen nicht mehr (41 Aufnahmen kosteten 1,58 s
   statt 53 ms). Leiste und Ansicht erzeugen Vorschauen bei Bedarf (`library.ErzeugeVorschau`); `Neuaufbau` und
   `Abgleich` heilen weiterhin nach der Freigabe der Sperre.

Tests in `internal/library/index_zweiter_prozess_test.go`, Mutanten HB32 bis HB35.

## Nachtrag 9 (25.11.2026): Abgleich heilt doppelte Indexzeilen (F-773, Karte #294)

**Status:** Akzeptiert (Claude Sonnet 5.5; Befund B-5 der Sitzungs-Kritik 25.11.).

- **Ergänzt Nachtrag 8.** Doppelte Zeilen, die `haengeAn` vor F-755 hinterlassen hat, blieben stehen: `Abgleich` stieg ohne Fehlendes früh aus, und `ersetzeImIndexGesperrt` ersetzt beide Doppel gleich. `Abgleich` erkennt nun mehrfach vorkommende Pfade, behält unter der Index-Sperre je Pfad nur die **letzte** Zeile (die jüngste Fassung) und schreibt den Index neu; `lade` bleibt lesend. Ein zweiter Abgleich findet nichts mehr zu tun.
- Test `internal/library/abgleich_doppel_test.go`, Mutanten HB36 bis HB38 in `scripts/mutanten/bibliothek-haertung.json`.

## Nachtrag 10 (26.11.2026): Eine Datei von außen wird beim Öffnen eine Aufnahme (Karte #345)

*Durch Nachtrag 11 abgelöst: Punkt 1 (Übernahme beim Öffnen, `importiere` gibt es nicht mehr), Punkt 3 (Scheitern der Übernahme beim Öffnen; heute scheitert höchstens das erste Speichern, Nachtrag 11 Punkt 2), Punkt 4 (Rettungskopie) und beide „Verworfen“. Es gelten weiter Punkt 2 (keine Doppelten, gemerkt erst beim ersten Speichern), Punkt 5 und Punkt 6.*

**Status:** Akzeptiert (Claude Opus 5.5, Andi: „arbeite selbstständig“).

Andi (25.11.2026, nach Datei → Öffnen…, Karte #326): „sie sollte aber dann auch unten in der bibliotheksleiste
auftauchen.“ Bisher wurde eine Datei außerhalb der Ablage erst beim ersten Speichern eine neue Aufnahme (Punkt 5,
E-16); bis dahin fehlte sie in der Leiste, und die Leiste umrandete nichts.

1. **Übernahme beim Öffnen:** Öffnet der Editor eine Datei außerhalb der Ablage (`schnapp DATEI`, „Öffnen mit“,
   Datei → Öffnen…), legt Schnapp zuerst eine Kopie als Aufnahme der Bibliothek ab und öffnet diese (`importiere` in
   `cmd/schnapp/importieren.go`). Zeitpunkt der Aufnahme ist der des Öffnens, sie steht also vorn in der Leiste und ist
   grün umrandet. Ein einfaches Bild bekommt die Kantenform aus `[capture] seam`; eine Arbeitsdatei behält Objekte,
   Kantenform und Schlagworte. **Die Datei von außen bleibt unberührt**, wie bisher.
2. **Keine Doppelten:** Die Bibliothek merkt sich je übernommene Datei die SHA-256-Summe ihres Inhalts und den Pfad
   der Aufnahme (`imports.jsonl` neben dem Index, 0600, nur lokal, nie in einer Bilddatei). Öffnet man dieselbe
   unveränderte Datei wieder, öffnet Schnapp die vorhandene Aufnahme samt ihren Beschriftungen, solange sie noch in der
   Ablage liegt; sonst wird neu übernommen. Geht die Liste verloren, entsteht höchstens eine weitere Kopie.
3. **Scheitert die Übernahme** (Ablage nicht beschreibbar, Datei über dem Rechenbudget, unlesbar, keine reguläre Datei
   oder größer als `fileformat.MaxDatei` — beides prüft sie vor dem Hashen), öffnet der Editor die Datei wie bisher.
   Mit `--ignore-limit` wird auch eine Datei über dem Budget übernommen; die Aufnahme öffnet sich später aus der Leiste
   nicht (Budgetmeldung), nur wieder mit `--ignore-limit`; das erste Speichern legt sie dann ab (Punkt 5 bleibt als Rückfall, auch für eine Aufnahme, die
   sich nicht zurückschreiben lässt). Einen Lade- oder Budgetfehler meldet der gewöhnliche Weg.
4. Die Rettungskopie einer früher außerhalb bearbeiteten Datei (ADR-0029) wird beim Übernehmen nicht angewandt; sie
   bleibt liegen und verfällt nach 30 Tagen. Neue Arbeit läuft immer an der Aufnahme.
5. Agenten-Wege (`--export`, `--annotate`, MCP) öffnen keinen Editor und übernehmen nichts. Im Drehbuch-Lauf des
   virtuellen KWin übernimmt der Probe-Build `--bild` nicht (`SCHNAPP_PROBE_LEISTE_AUS`), sonst öffnete der zweite Lauf
   die beschriftete Aufnahme des ersten (Kritik D-5).
6. **Schwärzen (ADR-0022 Punkt 4, F-221):** Die Datei von außen bleibt eine liegen gebliebene Fassung. Wird in der
   übernommenen Aufnahme geschwärzt, warnt das Speichern einmal je neuer Schwärzung mit ihrem Namen ohne Ordner („The
   original file %s is unchanged and still shows the unredacted content.“), auch wenn die Aufnahme über die Summe
   wiedergefunden wurde (`herkunftVon`, Sitzungs-Kritik 26.11. K-1/D-1). Gilt für die Laufzeit des Programms; nach
   einem Neustart weiß die Aufnahme nichts mehr von ihrer Herkunft.

Verworfen: die Datei nur als Verweis in der Leiste zeigen (die Leiste liest den Index der Ablage; ein Verweis bräuchte
eine zweite Liste und verschwände beim nächsten Start); die übernommene Kopie beim Schließen ohne Änderung wieder
entfernen (Andi hat sie geöffnet, um damit zu arbeiten; Löschen geht über ✕, Karte #347).

## Nachtrag 11 (26.11.2026): Übernahme erst mit der ersten Änderung (Karte #348, F-825)

**Status:** Akzeptiert (Andi, 26.11.2026: „vielleicht nur dann in die bibliothek übernehmen, wenn wir änderungen daran
durchführen, oder?“; Ausgestaltung Claude Opus 5.5, Andi: „arbeite selbstständig“).

Kritik K-2 (F-825): Nach Nachtrag 10 lag jede geöffnete Datei von außen danach still als Kopie samt Vorschau in der
Ablage, auch eine, die man nur ansehen wollte (etwa vom USB-Stick).

1. **Öffnen kopiert nicht.** Eine Datei außerhalb der Ablage öffnet sich selbst (`vorOeffnen` in
   `cmd/schnapp/importieren.go`, `ui.Geoeffnet.Aussen`); die Fußzeile sagt beim Öffnen, dass sie nicht in der
   Bibliothek liegt und mit der ersten Änderung hineinkommt. Wird sie ohne Änderung geschlossen oder gewechselt, bleibt
   die Bibliothek, wie sie war.
2. **Die erste Änderung speichert sofort.** Sobald das Dokument zum ersten Mal geändert ist, stößt der Editor ein
   Speichern wie Strg+S an (`uebernahmePruefen` in `internal/ui/uebernahme.go`), aber erst, wenn kein Zug, keine
   Texteingabe (Strg+S bestätigte sie), kein Bereich für den Text aus dem Bild, kein anderes Speichern, kein Wechsel und
   kein Schließen läuft. Das Speichern ist der bisherige Weg „erstes Speichern legt ab“ (Punkt 5, E-16): neue Aufnahme
   mit dem Zeitpunkt des Speicherns, vorn in der Leiste, grün umrandet; die Fußzeile nennt sie. Angestoßen wird einmal je
   geöffneter Datei; scheitert es, gilt das gewöhnliche Speichern (Strg+S, Schließen) mit seinen Meldungen. Während des
   Speicherns ruht das Bearbeiten wie bei Strg+S (ADR-0020 Nachtrag 6).
3. **Keine Doppelten** bleibt (Nachtrag 10 Punkt 2): Beim Öffnen wird die Summe des Inhalts gebildet; ist er schon
   übernommen und liegt die Aufnahme noch in der Ablage, öffnet sich diese. Gemerkt wird die Summe vom Öffnen erst, wenn
   das erste Speichern abgelegt hat (`merkeUebernahme` in `ablegenIn`, nicht für Agenten-Wege).
4. Die Rettungskopie (ADR-0029) gilt bis zur ersten Änderung für die Datei von außen, danach für die Aufnahme — wie vor
   Nachtrag 10. Punkt 4 von Nachtrag 10 entfällt damit.
5. Die Warnung beim Schwärzen (Nachtrag 10 Punkt 6) bleibt: beim ersten Speichern als „The original file is unchanged
   and still shows the unredacted content“, nach dem Wiederfinden über die Summe mit dem Namen der Datei.
6. Agenten-Wege und der Drehbuch-Lauf bleiben ohne Übernahme (Nachtrag 10 Punkt 5); mit `importBeimOeffnen` aus speichert
   der Editor auch nicht bei der ersten Änderung.
7. **Keine Übernahme von selbst nach einer Schwärzung** (Sitzungs-Kritik 26.11., K-1, K-2): Solange der Verlauf eine
   Schwärzung oder Verpixelung trägt, stößt die Änderung kein Speichern an. Sonst wäre eine versehentliche Schwärzung
   sofort endgültig (das Speichern leert den Verlauf, ADR-0022 Punkt 3), erneutes Öffnen führte zur geschwärzten
   Aufnahme, und die Warnung vor dem ungeschwärzten Original käme nur als flüchtige Meldung. Es gilt dann das
   gewöhnliche Speichern (Strg+S, Schließen mit haltender Warnung); wird die Schwärzung zurückgenommen, stößt die
   nächste Änderung wieder an.
8. Jedes gelungene Speichern hebt den Merker auf (K-4): Hat Strg+S die Datei schon abgelegt, stößt die nächste Änderung
   nichts mehr an. Eine Meldung der Rettungskopie beim Start geht dem Hinweis „not in the library“ vor (K-3).
9. `--ignore-limit DATEI` von außen: öffnet sich ohne Kopie; die erste Änderung legt sie wie jede andere als Aufnahme
   ab. Diese Aufnahme öffnet sich später aus der Leiste nicht (Budgetmeldung), nur wieder mit `--ignore-limit` (wie
   Nachtrag 10 Punkt 3; aus dem Codepfad `oeffneImEditor` → `vorOeffnen`, ohne eigenen Test, F-831).

Verworfen: eine vorläufige Kachel der offenen Datei in der Leiste (gestrichelt, „nicht in der Bibliothek“). Die Leiste
zählt Stellen, Fokus, ✕ und Papierkorb über Einträge des Index; eine Kachel ohne Eintrag hätte jeden dieser Wege
berührt. Die Fußzeile sagt dasselbe, und mit der ersten Änderung steht die echte Kachel da. Übernahme erst beim
Speichern (Strg+S oder Schließen): Dann fehlte die Aufnahme während der Arbeit in der Leiste, worum es in Karte #345 ging.

