# schnapp-ocr

Hilfsprogramm für „Text aus dem Bild kopieren“, wenn kein `tesseract` installiert ist (ADR-0035 Nachtrag 1).
tesseract läuft hier als WebAssembly im Go-Programm (ohne cgo), die Sprachdaten `deu` und `eng` sind eingebaut.
Eigenes Go-Modul, damit die Fremdpakete nicht im Hauptmodul liegen.

    schnapp-ocr --list-langs
    schnapp-ocr - - -l deu [--psm 3] [--progress] < bild.png > text.txt

Bild als PNG oder JPEG, höchstens 40 Mio. Pixel und 32767 je Seite (wie `internal/ocr`; geprüft am Kopf, vor dem
Laden) und 192 MiB. Exit 0: erkannt, 1: Bild oder Erkennung gescheitert (kurze Meldung auf stderr, ohne Go-Interna),
2: falscher Aufruf. Ohne `--progress` schreibt das Programm im Erfolgsfall nichts auf stderr (ADR-0035 Nachtrag 2).

Bauen: `go build -trimpath -o schnapp-ocr .` (macht `install.sh`; scheitert der Bau, installiert es schnapp trotzdem
und warnt). Testen: `go vet ./... && go test ./...` (die Tests erkennen ein Bild in beiden Sprachen und fahren das
Programm als eigenen Prozess, einige Sekunden).

## Fremde Bestandteile

Lizenztexte und Vermerke liegen in [`THIRD-PARTY-LICENSES/`](THIRD-PARTY-LICENSES/README.md); der Ordner gehört bei
jeder Weitergabe des Programms dazu (etwa im Flatpak, ADR-0035 Nachtrag 1 Punkt 7). Einkompiliert sind
(`go list -deps .`, Module mit Punkt im Pfad, dazu der Inhalt des WASM-Moduls):

| Teil | Herkunft | Lizenz |
|---|---|---|
| `tessdata/deu.traineddata`, `tessdata/eng.traineddata` | [tessdata_fast](https://github.com/tesseract-ocr/tessdata_fast), Stand 87416418 (2024-08-01) | Apache-2.0 (`tessdata/LICENSE`) |
| gogosseract v0.0.11-0ad3421 (Go-Modul, enthält `tesseract-core.wasm`) | [github.com/danlock/gogosseract](https://github.com/danlock/gogosseract), WASM aus einem Fork von [tesseract-wasm](https://github.com/robertknight/tesseract-wasm) | Apache-2.0 (Modul), BSD-2-Clause (tesseract-wasm) |
| danlock/pkg v0.0.17-a9828f2 | [github.com/danlock/pkg](https://github.com/danlock/pkg) (die Modulfassung hat keine Lizenzdatei, das Repo nennt Apache-2.0) | Apache-2.0 |
| im WASM: Tesseract, Leptonica | [tesseract-ocr/tesseract](https://github.com/tesseract-ocr/tesseract), [DanBloomberg/leptonica](https://github.com/DanBloomberg/leptonica) | Apache-2.0, BSD-2-Clause |
| im WASM: Laufzeit von Emscripten mit musl libc, dlmalloc und embind | [emscripten-core/emscripten](https://github.com/emscripten-core/emscripten) | MIT/UIUC, musl MIT |
| im WASM: libc++, libc++abi, compiler-rt (LLVM, Clang 17) | über Emscripten | Apache-2.0 WITH LLVM-exception (UIUC für ältere Teile) |
| im WASM: libjpeg 9c, libpng 1.6.39, zlib 1.2.13, giflib 5.2.1 (Emscripten-Ports, von Leptonica eingebunden) | [ijg.org](https://www.ijg.org/), [pnggroup/libpng](https://github.com/pnggroup/libpng), [madler/zlib](https://github.com/madler/zlib), [giflib](https://sourceforge.net/projects/giflib/) | IJG, PNG Reference Library License v2, zlib, MIT |
| wazero v1.7.3 | [github.com/tetratelabs/wazero](https://github.com/tetratelabs/wazero), fest (ADR-0035 Nachtrag 1 Punkt 6) | Apache-2.0 mit NOTICE |
| wazero-emscripten-embind v1.5.2 | [github.com/jerbob92/wazero-emscripten-embind](https://github.com/jerbob92/wazero-emscripten-embind) | MIT |
| golang.org/x/exp, golang.org/x/text | [go.googlesource.com](https://go.googlesource.com/) | BSD-3-Clause |

`golang.org/x/image` braucht nur der Test. Alle sind mit der GPL-3.0 von Schnapp verträglich.

Vermerk nach der IJG-Lizenz (Pflicht bei Weitergabe des Programms): *this software is based in part on the work of
the Independent JPEG Group.* Welche Teile im WASM stecken, ist an den Quellpfaden und Meldungstexten von
`tesseract-core-debug.wasm` und `tesseract-core.wasm` aus dem gogosseract-Modul abgelesen (F-667).
