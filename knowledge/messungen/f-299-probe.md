# Probelauf F-299: Schließen während des Speicherns (Nacharbeit 30b3b7b)

Datum 21.11.2026, Basis cf016f6. Umgebung: virtueller KWin über `scripts/probe.sh`, Probe-Build, Software-Zeichner.
Testbild synthetisch 3840×2160 (Verläufe, Farbflächen, Rauschen, nicht im Repo), Bildschirm 3840×2160, `--zeit 250`.
Drehbuch `scripts/drehbuecher/rueckfall-schliessen-waehrend-speichern.txt`: Schwärzen-Zug (B, ziehen), Strg+S,
sofort Strg+W, 45 s warten.

## Lauf 1: Strg+W während des Speicherns, Bild außerhalb der Bibliothek (`--bild` fremdes PNG)

Das Speichern dauerte 19,0 s. Log: „Schließen-Anfrage: 3 (SchlussWarten), Speichern beginnt false“ (das Speichern
lief schon), „Speichern im Hintergrund: 19.032s“, „Speichern zurück … Ende {Schliessen:false Schluss:2 …}“. Das Fenster
blieb offen, bis probe.sh 23 s später SIGTERM schickte (Exit 0). Ursache ist keine Lücke von F-299, sondern der
Hinweis „Als neue Aufnahme in der Bibliothek gespeichert …“: Ein Bild außerhalb der Bibliothek wird als neue Aufnahme
abgelegt, der Hinweis hält das Fenster offen (SchlussHinweis, ADR-0020 Nachtrag 4). Strg+W wurde angenommen und
vorgemerkt; der Hinweis verlangt eine zweite Schließen-Anfrage.

## Lauf 2: Strg+W während des Speicherns, Bild in der Bibliothek (Belegt F-299)

Aufruf: `scripts/probe.sh --drehbuch … --bild <Bibliotheksdatei, 4K> --breite 3840 --hoehe 2160 --zeit 250`, 75 s.
Log (Zeiten Sekunde des Tages):

| Zeit | Zeile |
|---|---|
| 03:08:34 | Zug: 12 Bilder, Bildzeit mittel 12,66 ms |
| 03:08:44 | Schließen über Taste oder Menü; Schließen-Anfrage: 3, Speichern beginnt false |
| 03:09:07 | Speichern im Hintergrund: 23.663s, nil |
| 03:09:07 | Speichern zurück: nil, Ende {Schliessen:true Schluss:0 …}; Fenster zu: Fehler nil, ungesichert false |

Ergebnis: probe.sh meldet „schnapp hat sich selbst beendet“, Exit 0, kein SIGTERM, kein SIGKILL. Foto 02 (0,5 s nach
Strg+W) zeigt das Fenster weiter offen, Statusleiste „Speichern …“, die Schwärzung im Bild. Erwartet erfüllt: Strg+W
während des Speicherns wird vorgemerkt und schließt das Fenster nach dem Ende des Speicherns.

## Lauf 3: Menü „Schließen“ per Tastatur während des Speicherns

Nicht wie vorgesehen herstellbar, mehrere Varianten (Umschalt+Tab zum Menüknopf, Enter, Pfeile; Eintrag „Schließen“ ist
nach 6 Mal Runter erreicht, ausgegraute Einträge werden übersprungen, die Liste läuft im Kreis):

- Strg+S, dann Menü per Tastatur öffnen: Der Menüknopf bleibt wirkungslos, die Liste öffnet sich nicht (Foto zeigt
  den Editor unverändert, Statusleiste „Speichern …“). Das Bearbeiten ruht während des Speicherns, Enter auf dem
  Menüknopf wird verworfen (`ui/editor.go` `tasten`, nur Enter auf einem schon offenen Eintrag „Schließen“ wirkt).
- Menü offen, dann Strg+S oder Strg+W: wirkungslos, es beginnt kein Speichern (Log ohne „Speichern“ bis zum Signal).
  Bei offenem Menü gelten die Kürzel nicht.
- Das Menü offen und das Speichern gleichzeitig zu haben, geht mit echter Tastatur-Eingabe also nicht; der Fall
  „Menü offen, dann beginnt ein Speichern“ (F-282, `TestMenueOffenWaehrendSpeichern`) entsteht nur über die
  Schließen-Anfrage des Compositors (Fensterknopf), die die Fernbedienung nicht auslösen kann.

Ersatzprobe: Menü per Tastatur auf „Schließen“ bei geändertem Dokument, kein Speichern vorher (Lauf mit Bibliotheksdatei):
„Schließen-Anfrage: 3, Speichern beginnt true“, „Speichern im Hintergrund: 23.749s“, „Speichern zurück … Schliessen:true“,
„Fenster zu“, Exit 0, kein Signal. Foto 01 zeigt den Fokus auf „Schließen Strg+W“. Das Menü-Schließen mit Änderung
wartet also bis zum Ende des Speicherns und schließt dann. (Ein erster Versuch dieser Probe zeigte „Hinweis“ und blieb
offen, weil im Bibliotheksordner ein von einem früheren SIGKILL liegen gebliebenes `.schnapp-tmp-*` lag: „liegen
gebliebene temporäre Kopie ließ sich nicht entfernen“ — erwartetes Verhalten, ADR-0016 Nachtrag 3.)

## Lauf 4: Gegenprobe Strg+W ohne Änderung

Log: „Schließen-Anfrage: 0, Speichern beginnt false“, „Fenster zu“, Exit 0, Fenster nach 3 s Drehbuchzeit zu
(„schnapp hat sich selbst beendet“).

## Fazit

F-299 für Strg+W belegt (Lauf 2), Gegenprobe sauber. Der Menüweg während des Speicherns ist mit echter Eingabe nicht
erreichbar, der Eintrag wirkt nur nach dem Öffnen vor dem Speichern (Test deckt ihn). Hinweise für Drehbücher: Bild
muss in der Bibliothek liegen, sonst hält der Hinweis das Fenster offen; 4K-Speichern dauert 19 bis 24 s.
