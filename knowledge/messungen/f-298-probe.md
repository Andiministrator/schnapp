# Probelauf F-298 und F-296: SIGTERM während eines langen Speicherns

Datum 21.11.2026, Basis 76718cb plus die Umsetzung von Karte #119 (Frist erkennt Fortschritt, ADR-0020 Nachtrag 9).
Umgebung: virtueller KWin über `scripts/probe.sh`, Probe-Build mit `-race` (Race-Zahlen, Normalbau rund 1,5 s,
[`f-297.md`](f-297.md)), Bildschirm 3840×2160. Testbild synthetisch 3840×2160 (Rauschen, Seed 298, nicht im Repo).
Drehbuch `scripts/drehbuecher/rueckfall-sigterm-langes-speichern.txt`: Schwärzen-Zug, danach Ende des Drehbuchs;
`probe.sh` schickt SIGTERM, Frist `--ende 90` (F-296), `--zeit 120`.

## Lauf 1: Bild außerhalb der Bibliothek (Ablegen als neue Aufnahme)

`[speichern]`-Marken: gerendert +586 ms, bild-kodiert +7,969 s, original-kodiert +8,029 s, datei-geschrieben +267 ms,
budget-geprueft +991 ms; „Speichern im Hintergrund: 17.842s, <nil>“, Hinweis „Als neue Aufnahme … gespeichert“ auf
stderr, „Fenster zu: Fehler <nil>, ungesichert false“. `probe.sh`: SIGTERM, „schnapp exit 0“, kein SIGKILL.

## Lauf 2: dieselbe Aufnahme aus der Bibliothek (Überschreiben)

| Marke | seit voriger | gesamt |
|---|---|---|
| gerendert | 578 ms | 0,578 s |
| alte-datei-gelesen | 4,196 s | 4,775 s |
| bild-kodiert | 8,066 s | 12,841 s |
| original-kodiert | 8,007 s | 20,848 s |
| datei-geschrieben | 253 ms | 21,101 s |
| ersetzt | 23 ms | 21,124 s |
| index | 2 ms | 21,126 s |
| vorschau | 932 ms | 22,058 s |
| budget-geprueft | 0 | 22,058 s |

„Speichern im Hintergrund: 22.058s, <nil>“, „Fenster zu: Fehler <nil>, ungesichert false“, `probe.sh`: SIGTERM,
„schnapp exit 0“, kein SIGKILL.

## Bewertung

- Das Speichern dauerte 17,8 und 22,1 s, also länger als die 15-s-Frist. Es gab keine Meldung „Speichern antwortet
  nicht“ und keinen Abbruch: Zwischen den Marken liegen bis zu 8 s, doch die PNG-Kodierung meldet je geschriebenem
  Stück Fortschritt (`speicherschritt.Schreiber`), das Lesen je gelesenem Stück (`Leser`).
- Mit der alten 5-s-Frist hätte `probe.sh` hier SIGKILL geschickt und ein ungesichertes Dokument hinterlassen (F-296);
  mit `--ende 90` endet der Lauf geordnet.
- Ein weiteres Signal während des Speicherns kann `probe.sh` nicht schicken; diesen Weg decken die Tests in
  `ui/speichern_fortschritt_test.go`.
