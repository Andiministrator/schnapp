module github.com/Andiministrator/schnapp/ocr

go 1.27

require (
	github.com/danlock/gogosseract v0.0.11-0ad3421
	golang.org/x/image v0.48.0
)

require (
	github.com/danlock/pkg v0.0.17-a9828f2 // indirect
	github.com/jerbob92/wazero-emscripten-embind v1.5.2 // indirect
	// wazero bleibt fest auf v1.7.3: ab v1.8 scheitert das WASM von gogosseract zur Laufzeit
	// (emscripten_stack_get_current; ADR-0035 Nachtrag 1 Punkt 6). Nur mit echter Erkennung im Test heben.
	github.com/tetratelabs/wazero v1.7.3 // indirect
	golang.org/x/exp v0.0.0-20231006140011-7918f672742d // indirect
	golang.org/x/sys v0.50.0 // indirect
	golang.org/x/text v0.44.0 // indirect
)
