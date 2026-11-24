# schnapp-ocr

Hilfsprogramm für „Text aus dem Bild kopieren“, wenn kein `tesseract` installiert ist (ADR-0035 Nachtrag 1).
tesseract läuft hier als WebAssembly im Go-Programm (ohne cgo), die Sprachdaten `deu` und `eng` sind eingebaut.
Eigenes Go-Modul, damit die Fremdpakete nicht im Hauptmodul liegen.

    schnapp-ocr --list-langs
    schnapp-ocr - - -l deu [--psm 3] [--progress] < bild.png > text.txt

Bauen: `go build -trimpath -o schnapp-ocr .` (macht `install.sh`). Testen: `go vet ./... && go test ./...`
(der Test erkennt ein Bild in beiden Sprachen, einige Sekunden).

## Fremde Bestandteile

| Teil | Herkunft | Lizenz |
|---|---|---|
| `tessdata/deu.traineddata`, `tessdata/eng.traineddata` | [tessdata_fast](https://github.com/tesseract-ocr/tessdata_fast), Stand 87416418 (2024-08-01) | Apache-2.0 (`tessdata/LICENSE`) |
| gogosseract (Go-Modul, enthält `tesseract-core.wasm`) | [github.com/danlock/gogosseract](https://github.com/danlock/gogosseract), aus einem Fork von [tesseract-wasm](https://github.com/robertknight/tesseract-wasm) | Apache-2.0 (Modul), BSD-2-Clause (tesseract-wasm) |
| im WASM: Tesseract, Leptonica | [tesseract-ocr/tesseract](https://github.com/tesseract-ocr/tesseract), [DanBloomberg/leptonica](https://github.com/DanBloomberg/leptonica) | Apache-2.0, BSD-2-Clause |
| wazero | [github.com/tetratelabs/wazero](https://github.com/tetratelabs/wazero), fest auf 1.7.3 (ADR-0035 Nachtrag 1 Punkt 6) | Apache-2.0 |

Alle sind mit der GPL-3.0 von Schnapp verträglich.
