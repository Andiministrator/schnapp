# Third-party licenses of schnapp-ocr

`schnapp-ocr` (this module, `ocr/`) compiles in the following third-party parts. Their license texts and notices are
in this folder; ship the whole folder together with the `schnapp-ocr` binary (for example in a Flatpak or a release
archive). The list follows `go list -deps .` in `ocr/` (modules with a dot in the path) plus the parts inside the
WebAssembly module that gogosseract embeds.

**Notice required by the IJG license:** this software is based in part on the work of the Independent JPEG Group.

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
| Emscripten runtime incl. dlmalloc and embind (inside the WASM module) | as embedded | MIT / University of Illinois/NCSA (dual) | `emscripten-LICENSE.txt` |
| musl libc (part of the Emscripten runtime) | as embedded | MIT | `musl-COPYRIGHT.txt` |
| libc++, libc++abi, compiler-rt (LLVM, Clang 17, via Emscripten) | as embedded | Apache-2.0 WITH LLVM-exception; older parts UIUC | `llvm-libcxx-LICENSE.txt`, `llvm-libcxxabi-LICENSE.txt`, `llvm-compiler-rt-LICENSE.txt` |
| libjpeg (Emscripten port, used by Leptonica) | 9c | IJG | `ijg-libjpeg-README.txt` (section LEGAL ISSUES) |
| libpng (Emscripten port, used by Leptonica) | 1.6.39 | PNG Reference Library License version 2 | `libpng-LICENSE.txt` |
| zlib (Emscripten port, used by libpng and Leptonica) | 1.2.13 | zlib | `zlib-LICENSE.txt` |
| tessdata_fast `deu`, `eng` | 87416418 (2024-08-01) | Apache-2.0 | `../tessdata/LICENSE` |

How the WASM contents were determined: the source paths in `internal/wasm/tesseract-core-debug.wasm` of the gogosseract
module (Emscripten `system/lib/libc/musl`, `system/lib/libcxx`, `system/lib/libcxxabi`, `system/lib/compiler-rt`,
`system/lib/dlmalloc.c`, `system/lib/embind`, ports `libjpeg/jpeg-9c`, `libpng/libpng-1.6.39`, `zlib/zlib-1.2.13`;
compiler `clang version 17.0.0`), cross-checked against messages in the embedded release module `tesseract-core.wasm`
(libjpeg, libpng, zlib and libc++ messages are present there as well). The sanitizer and ubsan runtime appear only
in the debug module.

Sources: the module texts are copied unchanged from the Go module cache (`go env GOMODCACHE`). Taken unchanged from
the public repositories on 2026-11-24: tesseract-wasm (`LICENSE.md` of github.com/robertknight/tesseract-wasm),
Leptonica (`leptonica-license.txt` of github.com/DanBloomberg/leptonica), Emscripten (`LICENSE`), musl
(`system/lib/libc/musl/COPYRIGHT`), libc++ / libc++abi / compiler-rt (`system/lib/{libcxx,libcxxabi,compiler-rt}/LICENSE.TXT`)
from github.com/emscripten-core/emscripten (branch main; the exact Emscripten version of the build is not recorded in
the module), libpng (`LICENSE` at tag v1.6.39 of github.com/pnggroup/libpng), zlib (`LICENSE` at tag v1.2.13 of
github.com/madler/zlib), libjpeg (`README` of `jpegsrc.v9c.tar.gz` from ijg.org).
