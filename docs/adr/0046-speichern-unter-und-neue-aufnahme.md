# ADR-0046: Speichern unter… und Neue Aufnahme im Menü Datei

## Status
Akzeptiert (Claude Opus 5.5, 28.11.2026, Karte #341 in Sprint 14; abgespalten von #326; Andi 25.11.2026: „keinen
[Punkt] wie speichern unter“, 28.11.: „arbeite selbstständig“). Ergänzt ADR-0016 (Bibliothek), ADR-0025 (Export),
ADR-0026 (Menü und Kürzel, Nachtrag 12) und ADR-0042 (Verzögerung).

## Kontext
Das Menü Datei kennt Öffnen…, Neu aus der Zwischenablage, Speichern, Export… und Bild kopieren. Es fehlen zwei
gewohnte Punkte:

- **Speichern unter…**: die bearbeitbare Datei an einem frei gewählten Ort ablegen, etwa in einem Projektordner oder
  für eine zweite Person mit Schnapp. Der Export schreibt nur ein flaches Bild (PNG, JPG, SVG ohne Bearbeitungsstand).
- **Neue Aufnahme**: aus dem offenen Editor heraus aufnehmen, ohne den globalen Kurzbefehl oder ein Terminal.

Schnapp ist um die Bibliothek gebaut (Kernprinzip 4, ADR-0016): Jede Aufnahme liegt dort, Speichern schreibt dorthin,
Rettungskopie (ADR-0029), Dateistand (F-834), Verlaufsmerker (Karte #356) und Leiste hängen an der Bibliotheksdatei.
Eine Datei von außen wird mit der ersten Änderung als Aufnahme übernommen (ADR-0016 Nachtrag 11); ein Fenster, das
nach „Speichern unter“ eine Datei außerhalb der Bibliothek führte, widerspräche dem.

Overlay und Editor laufen heute als Abfolge in einem Prozess: `ui.Starte` hält die Gio-Hauptschleife, das Overlay
entsteht je Monitor vor dem Editor, danach legt `cmd/schnapp` ab und öffnet den Editor.

## Entscheidung
1. **Speichern unter… schreibt eine Kopie als Arbeitsdatei.** Datei → „Save As…“ (deutsch „Speichern unter…“),
   **Strg+Umschalt+S**, öffnet den Speichern-Dialog des Desktop-Portals (`portal.Dialog.Speichern`, Filter „Schnapp
   image“ `*.png`, vorgeschlagen der Name der Aufnahme). Geschrieben wird eine vollständige Arbeitsdatei (PNG mit snAP
   und snAO, `fileformat.SchreibeArbeitsdatei`), die Schnapp wieder bearbeiten kann. **Das Fenster bleibt bei der
   Aufnahme in der Bibliothek**: Titel, Pfad, Rettungskopie, ungesicherter Stand und Rückgängig-Verlauf ändern sich
   nicht. Die Kopie ist eine bewusste Ausgabe wie ein Export: Was danach geändert wird, auch eine spätere Schwärzung,
   erreicht sie nicht. Die Fußzeile sagt das: „Saved a copy as NAME. This window keeps editing the capture in the
   library.“
2. **Metadaten** wie beim Speichern in die Bibliothek: Autor, Aufnahmezeit und Schlagworte aus der Datei, die den
   Stand trägt (`fileformat.LeseMetadaten`), dazu „bearbeitet“. Ist sie keine Arbeitsdatei (ein einfaches Bild von
   außen), ohne Autor und Schlagworte, mit der Zeit des Schreibens. Keine Pfade, keine Rechnernamen (Kernprinzip 3).
3. **Endung:** Fehlt `.png`, wird sie angehängt; ein so entstandener Name ersetzt keine vorhandene Datei
   (`export.PruefeAngehaengt`, `export.SchreibeOhneErsetzen`), sonst atomar ersetzend wie der Export
   (`export.SchreibeAtomar` samt Rückfall ohne Schreibrecht im Ordner, ADR-0025 Nachträge 3 und 4).
4. **Nicht die offene Aufnahme selbst.** Ist das Ziel dieselbe Datei wie die, die den Stand trägt (`os.SameFile`),
   wird abgelehnt mit Verweis auf Strg+S: Ein Schreiben an der Speicherfunktion vorbei sähe beim nächsten Speichern wie
   eine Änderung von außen aus (`library.ErrGeaendert`). *(Der Verweis auf Strg+S entfällt seit Nachtrag 1 Punkt 5;
   dazu kommen Ziele unter der Ablage.)*
5. **Zuschnitt und Stauchen bleiben in der Kopie umkehrbar** (Kernprinzip 1): Die Arbeitsdatei trägt das ganze
   Original. Hat das Dokument einen Zuschnitt oder Streifen, hängt die Fußzeile an: „The copy contains the whole
   original; to share only the visible image, use Export.“ Schwärzungen sind in jedem Fall endgültig (Kernprinzip 2).
6. **Im Hintergrund.** Geschrieben wird ein Abzug (`document.Abzug`) in einer Goroutine wie beim Kopieren; das
   Bearbeiten läuft weiter. Ein laufender Zug oder eine Texteingabe wird wie beim Export vorher bestätigt bzw.
   verschiebt das Schreiben. Über dem Rechenbudget wird geschrieben und gewarnt wie beim Speichern (Speichern bricht
   nie ab). Nur ein Speichern unter zugleich; nicht während Export- oder Öffnen-Dialog und nicht während eine andere
   Aufnahme lädt. Das Schließen wartet auf ein laufendes Schreiben (*seit Nachtrag 1 Punkt 2 nach dem
   Notspeichern bis `unterFrist` = 2 min ab Beginn, Nachtrag 2: ein weiteres Signal beendet das Warten; zuerst waren
   es `wechselFrist`, 5 s*), danach steht ein
   Hinweis auf der Fehlerausgabe. Bei offener Bibliotheksansicht geht es wie Speichern.
7. **Neue Aufnahme startet einen eigenen Prozess.** Datei → „New capture“ (deutsch „Neue Aufnahme“), **Strg+N**,
   startet dasselbe Programm (`os.Executable`; *seit Nachtrag 1 mit dem internen Argument `--internal-new-capture`*),
   abgelöst (eigene Sitzung, Standardein- und -ausgabe
   leer, Fehlerausgabe nur mit `SCHNAPP_DEBUG`). Es nimmt wie der globale Kurzbefehl auf — Bereich als Voreinstellung,
   `[capture] delay` (ADR-0042) und `[capture] cursor` (ADR-0043) gelten, weil der neue Prozess die Einstellungen
   selbst liest — legt in der Bibliothek ab und öffnet ein **eigenes Editorfenster**. Das bisherige Fenster bleibt
   offen und unverändert; ein Abbruch im Overlay (Esc) lässt nichts zurück.
   - Warum kein Overlay im selben Fenster: `ui.Starte` und die Overlays sind für den Programmstart gebaut, der
     Wechsel ins Overlay und zurück müsste ungesicherte Arbeit, Rettungskopie und Verlauf der offenen Aufnahme
     verwalten. Ein zweiter Prozess braucht nichts davon und ist derselbe Weg wie der Kurzbefehl.
   - Das offene Fenster ist in der Aufnahme zu sehen, wenn es im Bereich liegt. Es vorher zu minimieren (Gio kann
     `xdg_toplevel.set_minimized`) braucht eine Messung in der echten Sitzung (Zeit bis KWin es ausgeblendet hat) und
     kommt als eigene Karte; bis dahin hilft die Verzögerung.
   - Mit geerbtem `WAYLAND_SOCKET` fände der neue Prozess den Compositor nicht sicher (wie der Halter der
     Zwischenablage, ADR-0024 Nachtrag 5): Der Eintrag ist dann grau mit Erklärung. Scheitert der Start, sagt es die
     Fußzeile. Bei offener Bibliotheksansicht geht es.
8. **Menü und Kürzel** (ADR-0026 Nachtrag 12): Datei = Öffnen…, Neue Aufnahme, Neu aus der Zwischenablage │ Speichern,
   Speichern unter… │ Export…, Bild kopieren │ Einstellungen…, Bibliothek │ Schließen. Die Übersicht F1 fasst
   „Speichern / Speichern unter“ und „Öffnen / Neue Aufnahme“ je auf eine Zeile (`hilfePaare`), damit sie weiter ins
   Startfenster passt (F-705).

## Konsequenzen
- Wer eine Aufnahme dauerhaft woanders führen will, hat mit „Speichern unter“ nur eine Kopie; die Bibliothek bleibt
  der eine Ort, an dem Schnapp weiterarbeitet. Öffnet man die Kopie später (Datei → Öffnen… oder „Öffnen mit“), ist sie
  eine Datei von außen und wird bei der ersten Änderung als neue Aufnahme übernommen.
- Die Kopie trägt das Original ohne Zuschnitt; das sagt die Fußzeile. Zum Teilen bleibt der Export der Weg.
- Neue Aufnahme öffnet ein zweites Fenster statt das erste zu ersetzen; mehrere Fenster laufen schon heute
  nebeneinander (Rettungskopie mit Sperre je Datei, ADR-0029).
- Den Portal-Dialog und den Start des neuen Prozesses aus der echten Plasma-Sitzung prüft Andi am Sprintende
  (Echtprobenliste an der Abschlusskarte); der virtuelle KWin hat kein Portal.

## Nachtrag 1 (28.11.2026): Kritik Fable zu a6adc91 (F-1007 bis F-1017)

**Status:** Akzeptiert (Claude Opus 5.5).

1. **Rechte:** Eine neue Kopie entsteht mit 0600 wie die Bibliothek (`export.SchreibeAtomarPrivat`,
   `SchreibeOhneErsetzenPrivat`), denn sie trägt das ganze Original samt Weggeschnittenem; ein ersetztes Ziel behält
   seine Rechte (F-1007).
2. **Schließen:** Das Zerstören wartet erst **nach** dem Notspeichern und bis zu `unterFrist` = 2 min auf ein laufendes
   Schreiben (wie die Frist eines Speicherns), damit keine halbe temporäre Datei und im Rückfall kein gekürztes Ziel
   liegen bleiben; das Fenster ist dann schon zu, der Prozess endet danach (F-1009, F-1016). Ersetzt die 5 s aus
   Punkt 6.
3. **Wechsel:** Solange Dialog oder Schreiben laufen, nimmt die Leiste keinen Wechsel an und sagt es in der Fußzeile;
   kam der Dialog an der Sperre vorbei, nennt die Meldung ihn (F-1011). Damit liest das Schreiben die Metadaten immer
   von der Aufnahme, für die der Dialog aufging.
4. **Meldung:** „Saving a copy…“ geht über `meldeNebenbei` und verdrängt keine haltende Meldung (F-1010).
5. **Ziel:** Abgelehnt werden die Datei, die das Fenster offen hat (Text ohne Verweis auf Strg+S, der bei einer Datei
   von außen nicht dorthin schreibt), und jeder Ort unter der Ablage der Bibliothek (`inOrdner`, nach Auflösen von
   Verweisen): Dort verwaltet die Bibliothek Index, Vorschau und Rettung selbst (F-1017).
6. **Metadaten:** Autor, Aufnahmezeit und Schlagworte gehen mit, das Quellprogramm (`s:quelle`) nicht; README sagt,
   dass Autor und Schlagworte in der Kopie stehen (F-1015).
7. **Neue Aufnahme:** Nach einem Start nimmt Strg+N 3 s lang keinen weiteren an (Tastenwiederholung, Doppelklick;
   F-1008). Der neue Prozess bekommt das interne Argument `--internal-new-capture`: `cmd/schnapp` schließt damit vor
   allem anderen die ohne CLOEXEC geerbten Deskriptoren ab 3 (`wlclip.SchliesseGeerbteAb`, wie der Halter, F-189) und
   nimmt dann auf wie ohne Argumente (F-1013). Die Fehlerausgabe erbt er nur mit `SCHNAPP_DEBUG` und nie als Pipe
   (`wlclip.IstPipe`, F-325, F-1012). Er bleibt in der systemd-Scope des ersten Fensters (wie der Halter, F-320).
8. **Tests** (F-1014): Rechte, Metadaten aus der Quelle, große Endung, Ziel in der Ablage, gegenseitige Dialogsperren,
   haltende Meldung, Wechsel während des Schreibens, Mehrfachstart, echter Start ohne Fenster, internes Argument.
9. **Offen (eigene Karte):** Die Wahl der Verzögerung beim Start aus dem Menü (ADR-0042 Punkt 5) und eine Aktion
   „Abbrechen“ am Countdown-Hinweis (ADR-0042 Nachtrag 1 Punkt 3, F-941) kommen nicht mit dieser Karte; heute gilt
   `[capture] delay`. — Erledigt mit Karte #396, ADR-0042 Nachtrag 2.

## Nachtrag 2 (28.11.2026): Verify und Doku-QA der Sitzung (F-1018 bis F-1022)

**Status:** Akzeptiert (Claude Opus 5.5).

1. **Verweis als Ziel (F-1018):** Ein vorhandenes Ziel wird vor der Prüfung auf die Ablage aufgelöst
   (`filepath.EvalSymlinks`); ein Verweis von außen auf eine Aufnahme der Bibliothek wird abgelehnt wie ein Ziel in der
   Ablage selbst.
2. **Warten beim Schließen (F-1019):** Notspeichern, Rettung und Werkzeugstile kommen vor dem Warten auf die Kopie;
   die Frist `unterFrist` zählt ab Beginn des Schreibens (`unterZustand.start`), ein weiteres Signal beendet das
   Warten sofort mit dem Hinweis auf der Fehlerausgabe.
3. **Rechte (F-1021):** Eine neue Kopie entsteht mit 0600; ein bestätigt ersetztes Ziel behält seine Rechte. Das README
   sagt beides.
4. Testlücken des Verify (F-1020) und Doku-Nachzug (F-1022) — ohne Änderung der Entscheidung.
