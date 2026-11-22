// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

// jpegBauen baut ein gültiges JPEG ohne Bildinhalt (alle Koeffizienten 0): DQT, SOF (0xC0 Baseline oder 0xC2
// progressiv), Huffman-Tabellen mit dem einzigen Code „0“, ein Scan über alle Komponenten. Progressiv ist es nur der
// DC-Scan — Go schreibt keine progressiven JPEG, also von Hand (F-361). hv sind die Abtastfaktoren je Komponente.
func jpegBauen(w, h int, sof byte, hv ...byte) []byte {
	var b bytes.Buffer
	seg := func(marker byte, inhalt ...byte) {
		b.Write([]byte{0xFF, marker})
		_ = binary.Write(&b, binary.BigEndian, uint16(len(inhalt)+2))
		b.Write(inhalt)
	}
	b.Write([]byte{0xFF, 0xD8})
	seg(0xE0, []byte("JFIF\x00\x01\x01\x00\x00\x01\x00\x01\x00\x00")...) // ein APP-Segment vor dem SOF
	seg(0xDB, append([]byte{0}, bytes.Repeat([]byte{1}, 64)...)...)
	if len(hv) == 4 { // CMYK verlangt in Go das Adobe-Segment (Transformation 0: keine)
		seg(0xEE, []byte("Adobe\x00\x64\x00\x00\x00\x00\x00")...)
	}
	kopf := []byte{8, byte(h >> 8), byte(h), byte(w >> 8), byte(w), byte(len(hv))}
	hmax, vmax := 1, 1
	for i, f := range hv {
		kopf = append(kopf, byte(i+1), f, 0)
		hmax, vmax = max(hmax, int(f>>4)), max(vmax, int(f&0x0F))
	}
	seg(sof, kopf...)
	tabelle := func(klasse byte) []byte { // ein Code der Länge 1 für das Symbol 0
		return append(append([]byte{klasse, 1}, make([]byte, 15)...), 0)
	}
	seg(0xC4, tabelle(0x00)...)
	scan := []byte{byte(len(hv))}
	for i := range hv {
		scan = append(scan, byte(i+1), 0x00)
	}
	bloecke := 0
	for _, f := range hv {
		bloecke += int(f>>4) * int(f&0x0F)
	}
	mcus := ((w + 8*hmax - 1) / (8 * hmax)) * ((h + 8*vmax - 1) / (8 * vmax))
	if len(hv) == 1 { // nicht verschachtelt: Blöcke der Komponente
		mcus, bloecke = ((w+7)/8)*((h+7)/8), 1
	}
	bits := mcus * bloecke
	if sof == 0xC2 {
		seg(0xDA, append(scan, 0, 0, 0)...)
	} else {
		seg(0xC4, tabelle(0x10)...) // AC: Symbol 0 ist EOB
		seg(0xDA, append(scan, 0, 63, 0)...)
		bits *= 2
	}
	b.Write(make([]byte, (bits+7)/8))
	b.Write([]byte{0xFF, 0xD9})
	return b.Bytes()
}

// F-361: Die Schätzung trifft, was Dekoder und Dokument anfordern, für Baseline und progressiv, 4:4:4, 4:2:0, Grau und
// CMYK (vier Komponenten).
func TestJPEGBedarfSchaetzungTrifft(t *testing.T) {
	for _, fall := range []struct {
		name string
		sof  byte
		hv   []byte
	}{
		{"Baseline 4:4:4", 0xC0, []byte{0x11, 0x11, 0x11}},
		{"progressiv 4:4:4", 0xC2, []byte{0x11, 0x11, 0x11}},
		{"progressiv 4:2:0", 0xC2, []byte{0x22, 0x11, 0x11}},
		{"Baseline 4:2:0", 0xC0, []byte{0x22, 0x11, 0x11}},
		{"progressiv Grau", 0xC2, []byte{0x11}},
		{"Baseline CMYK", 0xC0, []byte{0x11, 0x11, 0x11, 0x11}},
	} {
		daten := jpegBauen(1024, 768, fall.sof, fall.hv...)
		k, ok := leseJPEGKopf(daten)
		if !ok || k.breite != 1024 || k.hoehe != 768 || k.progressiv != (fall.sof == 0xC2) || len(k.komponenten) != len(fall.hv) {
			t.Fatalf("%s: Kopf %+v, ok %v", fall.name, k, ok)
		}
		var fehler error
		ist := int64(angefordert(func() {
			bild, err := jpeg.Decode(bytes.NewReader(daten))
			if err != nil {
				fehler = err
				return
			}
			_, fehler = neuesDokument(richteAus(bild, 1))
		}))
		if fehler != nil {
			t.Fatalf("%s: %v", fall.name, fehler)
		}
		soll := k.bedarf()
		t.Logf("%s: angefordert %d KiB, geschätzt %d KiB", fall.name, ist>>10, soll>>10)
		// Gemessen trifft die Schätzung auf wenige KiB (Kleinkram des Dekoders); 2 % und 64 KiB Spiel.
		if d := ist - soll; d > soll/50+64<<10 || d < -soll/50 {
			t.Errorf("%s: angefordert %d KiB, geschätzt %d KiB", fall.name, ist>>10, soll>>10)
		}
	}
}

// F-361: Ein präpariertes progressives JPEG von 1,5 MB mit 16384×16384 Pixeln (vorher fast 5 GiB, gemessen) wird
// abgelehnt, bevor der Dekoder Speicher anfordert.
func TestOeffnenLehntTeuresProgressivesJPEGAb(t *testing.T) {
	daten := jpegBauen(16384, 16384, 0xC2, 0x11, 0x11, 0x11)
	pfad := filepath.Join(t.TempDir(), "teuer.jpg")
	if err := os.WriteFile(pfad, daten, 0o600); err != nil {
		t.Fatal(err)
	}
	var err error
	ist := angefordert(func() { _, err = Oeffne(pfad) })
	if !errors.Is(err, ErrZuGross) {
		t.Fatalf("Öffnen: %v, erwartet ErrZuGross", err)
	}
	if ist > 64<<20 {
		t.Errorf("vor der Ablehnung %d MiB angefordert", ist>>20)
	}
	t.Log(err)
}

// Grenzen der Schätzung: Baseline YCbCr und Grau bis document.MaxPixel bleiben erlaubt (wie vorher); progressiv 4:4:4 bis
// etwa 113 Megapixel; CMYK in voller Größe nicht.
func TestJPEGBedarfGrenzen(t *testing.T) {
	for _, fall := range []struct {
		name    string
		w, h    int
		sof     byte
		hv      []byte
		erlaubt bool
	}{
		{"Baseline 4:4:4 16384²", 16384, 16384, 0xC0, []byte{0x11, 0x11, 0x11}, true},
		{"Baseline 4:2:0 16384²", 16384, 16384, 0xC0, []byte{0x22, 0x11, 0x11}, true},
		{"Baseline Grau 16384²", 16384, 16384, 0xC0, []byte{0x11}, true},
		{"Baseline CMYK 16384²", 16384, 16384, 0xC0, []byte{0x11, 0x11, 0x11, 0x11}, false},
		{"progressiv 4:4:4 10000²", 10000, 10000, 0xC2, []byte{0x11, 0x11, 0x11}, true},
		{"progressiv 4:4:4 11000²", 11000, 11000, 0xC2, []byte{0x11, 0x11, 0x11}, false},
		{"progressiv 4:2:0 13000²", 13000, 13000, 0xC2, []byte{0x22, 0x11, 0x11}, true},
		{"progressiv 4:2:0 14000²", 14000, 14000, 0xC2, []byte{0x22, 0x11, 0x11}, false},
	} {
		err := pruefeJPEGBedarf(jpegBauen(fall.w, fall.h, fall.sof, fall.hv...))
		if (err == nil) != fall.erlaubt || err != nil && !errors.Is(err, ErrZuGross) {
			t.Errorf("%s: %v, erlaubt soll %v", fall.name, err, fall.erlaubt)
		}
	}
}

// Kaputte Köpfe: keine Schätzung (der Dekoder entscheidet), kein Absturz.
func TestLeseJPEGKopfKaputt(t *testing.T) {
	gut := jpegBauen(64, 48, 0xC2, 0x11, 0x11, 0x11)
	sof := bytes.Index(gut, []byte{0xFF, 0xC2})
	faelle := map[string][]byte{
		"leer":          nil,
		"kein SOI":      []byte{0xFF, 0xD9, 0, 0},
		"abgeschnitten": gut[:sof+5],
		"Länge zu groß": append(append([]byte(nil), gut[:sof+2]...), 0xFF, 0xFF, 8),
		"Scan vor SOF":  {0xFF, 0xD8, 0xFF, 0xDA, 0, 2},
		"kein Marker":   {0xFF, 0xD8, 0x00, 0x00, 0x00},
	}
	for _, f := range []func(b []byte){
		func(b []byte) { b[sof+9] = 0 },              // null Komponenten
		func(b []byte) { b[sof+11] = 0x51 },          // Abtastfaktor 5
		func(b []byte) { b[sof+11] = 0x10 },          // Abtastfaktor 0
		func(b []byte) { b[sof+5], b[sof+6] = 0, 0 }, // Höhe 0
	} {
		b := append([]byte(nil), gut...)
		f(b)
		faelle[string(rune('A'+len(faelle)))] = b
	}
	for name, daten := range faelle {
		if k, ok := leseJPEGKopf(daten); ok {
			t.Errorf("%s: Kopf %+v angenommen", name, k)
		}
		if err := pruefeJPEGBedarf(daten); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	// Füllbytes vor einem Marker sind erlaubt.
	mitFuell := append([]byte{0xFF, 0xD8, 0xFF, 0xFF}, gut[2:]...)
	if _, ok := leseJPEGKopf(mitFuell); !ok {
		t.Error("Füllbyte vor dem Marker nicht übersprungen")
	}
}
