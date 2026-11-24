# Third-party licenses of schnapp-ocr

`schnapp-ocr` (this module, `ocr/`) compiles in the following third-party parts. Their license texts and notices are
in this folder; ship the whole folder together with the `schnapp-ocr` binary (for example in a Flatpak or a release
archive). The list follows `go list -deps .` in `ocr/` (modules with a dot in the path) plus the parts inside the
WebAssembly module that gogosseract embeds.

| Part | Version | License | Text / notice in this folder |
|---|---|---|---|
| github.com/danlock/gogosseract | v0.0.11-0ad3421 | Apache-2.0 | `APACHE-2.0.txt` |
| github.com/danlock/pkg | v0.0.17-a9828f2 | Apache-2.0 (the module version has no license file; the repository states Apache-2.0) | `APACHE-2.0.txt` |
| github.com/tetratelabs/wazero | v1.7.3 | Apache-2.0, with NOTICE | `APACHE-2.0.txt`, `wazero-NOTICE.txt` |
| github.com/jerbob92/wazero-emscripten-embind | v1.5.2 | MIT | `wazero-emscripten-embind-LICENSE.txt` |
| golang.org/x/exp | v0.0.0-20231006140011-7918f672742d | BSD-3-Clause | `golang.org-x-exp-LICENSE.txt` |
| golang.org/x/text | v0.44.0 | BSD-3-Clause | `golang.org-x-text-LICENSE.txt` |
| tesseract-wasm (build of `tesseract-core.wasm`, via the gogosseract fork) | as embedded in gogosseract | BSD-2-Clause | `tesseract-wasm-LICENSE.md` |
| Tesseract (inside the WASM module) | as embedded | Apache-2.0 (no NOTICE file upstream) | `APACHE-2.0.txt` |
| Leptonica (inside the WASM module) | as embedded | BSD-2-Clause | `leptonica-license.txt` |
| Emscripten runtime (inside the WASM module) | as embedded | MIT (dual MIT / UIUC) | `emscripten-LICENSE.txt` |
| musl libc (part of the Emscripten runtime) | as embedded | MIT | not yet included, see below |
| tessdata_fast `deu`, `eng` | 87416418 (2024-08-01) | Apache-2.0 | `../tessdata/LICENSE` |

Sources: the module texts are copied unchanged from the Go module cache (`go env GOMODCACHE`); the texts of
tesseract-wasm (`LICENSE.md` of github.com/robertknight/tesseract-wasm), Leptonica (`leptonica-license.txt` of
github.com/DanBloomberg/leptonica) and Emscripten (`LICENSE` of github.com/emscripten-core/emscripten) were taken from
the public repositories on 2026-11-24.

Open: the exact set of runtime libraries inside `tesseract-core.wasm` (musl libc, possibly libc++ and compiler-rt, both
Apache-2.0 WITH LLVM-exception) is not verified per file; the musl copyright notice (`COPYRIGHT` in the musl sources,
MIT) still has to be added before a binary is distributed.
