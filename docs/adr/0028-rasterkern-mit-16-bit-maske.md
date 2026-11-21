# ADR-0028: Rasterkern mit 16-Bit-Maske (internal/raster16)

## Status
Akzeptiert (Claude, Andi informiert). Karte #78, F-145 und F-147, 21.11.2026.

## Kontext
`internal/render` füllte über `golang.org/x/image/vector` mit `draw.Over` auf `*image.NRGBA`. Für dieses Ziel kennt
`vector` nur seinen allgemeinen Weg: je Pixel `At` und `Set` über Schnittstellen. Der eigene `einmischer` (F-130) hatte
das um Faktor 4 verkürzt, der Rest blieb in `vector` (F-145). Den Weg über eine 8-Bit-Maske (`*image.Alpha`, schnell in
`vector`) gab es auch, er ist aber nicht bitgleich (Texthintergrund weicht um bis zu 4 ab). Glyphen gingen über
`font.Drawer` und `draw.DrawMask`, das für `*image.NRGBA` ebenfalls nur den allgemeinen Weg hat (F-147).

ADR-0011 verlangt für Fremdcode spürbaren Nutzen. Ein Fork ist Fremdcode im Repo.

## Entscheidung
1. **Rasterkern übernommen:** `internal/raster16` enthält den Rasterkern von `golang.org/x/image/vector` v0.26.0 (BSD,
   Lizenztext `internal/raster16/LICENSE`, Herkunft im Kopf jeder übernommenen Datei): Strecken, Festkomma- und Gleitkomma-
   Akkumulation. Entfallen sind Kurven, das Zeichnen auf Bilder und die 8-Bit-Ziele. Neu ist `Maske()` mit der
   16-Bit-Deckung, die `vector` intern seinem allgemeinen Weg übergibt; `render` mischt damit selbst ein, mit denselben
   uint32-Ausdrücken wie `vector` und `image/draw`.
2. **Bitgleich zu `vector`:** Auf amd64 akkumuliert `vector` mit SSE4.1 in Viererblöcken (`acc_amd64.s`). Weil
   float32-Addition nicht assoziativ ist, rechnet `raster16` unter derselben Build-Bedingung (`acc_amd64.go`) dieselbe
   Summenfolge in Go nach, fest eingeschaltet, ohne eigene CPU-Abfrage und ohne neue Abhängigkeit: Eine Abfrage über
   `golang.org/x/sys/cpu` folgte `GODEBUG=cpu.sse41=off`, die CPUID-Abfrage von `vector` nicht (Sitzungs-QA, K-3).
   Abweichung bewusst angenommen: auf amd64-CPUs ohne SSE4.1 (Intel vor 2008, AMD vor 2011) rechnet `vector` skalar,
   `raster16` nicht.
   Außerhalb dieser Bedingung rechnen beide skalar. Im Festkommaweg (bis 512 px) nimmt `vector` auf amd64 ebenfalls
   eine SIMD-Fassung, `raster16` die skalare; Ganzzahlen ergeben dieselben Werte (bis auf einen Überlauf bei
   Windungszahl ±8192, aus `render` nicht erreichbar). `TestMaskeGleichVector` (beide Rechenwege, Breiten um 512) und `TestMaskeGleichVectorZufall`
   (600 Zufallsfälle im Gleitkommaweg) vergleichen die Maske Wert für Wert mit `vector`, `TestFuellenBitgleich` das gerenderte Bild.
3. **Blockweise:** `Bloecke(n, f)` liefert die Maske je 64 Zeilen mit fortgeführtem Übertrag (gleiche Werte,
   `TestBloeckeGleichMaske`) und leert den Gleitkommapuffer gleich nach dem Lesen; `Reset` spart sich dann das Leeren
   (`TestWiederverwendet`). Vier Werte ohne Kante setzen den Übertrag fort, ohne zu rechnen (exakt, `0+acc == acc`).
4. **Glyphen:** `render.glyphMischen` ersetzt `draw.DrawMask` für `*image.Alpha`-Masken, bitgleich (gleicher Zuschnitt,
   Deckung 0 lässt das Pixel unverändert; `TestGlyphMischenBitgleich`, `TestGlyphMischenKleineMaske`). Andere Masken
   gehen weiter über `draw.DrawMask`.
5. **Pflege:** Ein Update von `x/image` ändert `raster16` nicht mit. Weicht `vector` später ab, zeigt das
   `TestMaskeGleichVector`; dann den Kern nachziehen oder die Abweichung bewusst annehmen. Auf amd64-CPUs ohne SSE4.1
   schlagen die Vergleichstests fehl (Punkt 2); das Bild ist dort dasselbe wie auf allen anderen amd64-Rechnern.

## Konsequenzen
Messung (Normalbau, amd64, 16 Kerne, `-count 2`, main mit `-benchtime 1x`, nachher `2x`, `./internal/render`):

| Benchmark | main (147265d) | nachher |
|---|---|---|
| AufwandZweiKaesten8000 | 1,92 s | 0,49 s |
| AufwandNaheBudgetFlaeche | 34,9 s | 7,0 s |
| AufwandNaheGlyphBudget | 4,91 s | 1,23 s |
| AufwandAlltag | 10,7 s | 1,94 s |
| AufwandPfeilDiagonal8000 | 0,56 s | 0,21 s |
| AufwandNaheKantenBudget | 1,55 s | 1,51 s |

Die Budgets (ADR-0020 Nachtrag 3) bleiben unverändert; ein präpariertes Dokument knapp unter `Budget` speichert jetzt
in rund 7 s statt rund 35 s; zusammen mit Glyphen und Kanten knapp unter ihren Budgets rund 9,7 s statt gemessen 41,4 s (Nachtrag 3 schätzte 45 s).
663 Zeilen übernommener Code (`wc -l internal/raster16/raster.go internal/raster16/raster_fixed.go
internal/raster16/raster_floating.go`), geprüft gegen das Original über die Tests oben.
