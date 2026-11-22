// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

// jpegBauen baut ein gültiges JPEG ohne Bildinhalt (alle Koeffizienten 0): DQT, SOF (0xC0 Baseline oder 0xC2
// progressiv), Huffman-Tabellen mit dem einzigen Code „0“, ein Scan über alle Komponenten. Progressiv ist es nur der
// DC-Scan — Go schreibt keine progressiven JPEG, also von Hand (F-361). hv sind die Abtastfaktoren je Komponente.
func jpegBauen(w, h int, sof byte, hv ...byte) []byte {
	return jpegBauenMit(jpegArt{adobe: -1}, w, h, sof, hv...)
}

// jpegArt steuert die Farbangaben von jpegBauenMit: ohneJFIF lässt das JFIF-Segment weg, kennungen setzt die
// Komponentenkennungen (sonst 1, 2, 3 …), adobe ≥ 0 schreibt ein Adobe-Segment mit dieser Transformation (bei vier
// Komponenten immer, dann mit 0), hinten sind fertige Segmente zwischen Scan und Ende.
type jpegArt struct {
	ohneJFIF  bool
	kennungen string
	adobe     int
	hinten    []byte
}

func jpegBauenMit(art jpegArt, w, h int, sof byte, hv ...byte) []byte {
	var b bytes.Buffer
	seg := func(marker byte, inhalt ...byte) {
		b.Write([]byte{0xFF, marker})
		_ = binary.Write(&b, binary.BigEndian, uint16(len(inhalt)+2))
		b.Write(inhalt)
	}
	b.Write([]byte{0xFF, 0xD8})
	if !art.ohneJFIF {
		seg(0xE0, []byte("JFIF\x00\x01\x01\x00\x00\x01\x00\x01\x00\x00")...) // ein APP-Segment vor dem SOF
	}
	seg(0xDB, append([]byte{0}, bytes.Repeat([]byte{1}, 64)...)...)
	if len(hv) == 4 && art.adobe < 0 { // CMYK verlangt in Go das Adobe-Segment (Transformation 0: keine)
		art.adobe = 0
	}
	if art.adobe >= 0 {
		seg(0xEE, append([]byte("Adobe\x00\x64\x00\x00\x00\x00"), byte(art.adobe))...)
	}
	kennung := func(i int) byte {
		if i < len(art.kennungen) {
			return art.kennungen[i]
		}
		return byte(i + 1)
	}
	kopf := []byte{8, byte(h >> 8), byte(h), byte(w >> 8), byte(w), byte(len(hv))}
	hmax, vmax := 1, 1
	for i, f := range hv {
		kopf = append(kopf, kennung(i), f, 0)
		hmax, vmax = max(hmax, int(f>>4)), max(vmax, int(f&0x0F))
	}
	seg(sof, kopf...)
	tabelle := func(klasse byte) []byte { // ein Code der Länge 1 für das Symbol 0
		return append(append([]byte{klasse, 1}, make([]byte, 15)...), 0)
	}
	seg(0xC4, tabelle(0x00)...)
	scan := []byte{byte(len(hv))}
	for i := range hv {
		scan = append(scan, kennung(i), 0x00)
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
	b.Write(art.hinten)
	b.Write([]byte{0xFF, 0xD9})
	return b.Bytes()
}

// F-361: Die Schätzung trifft, was Dekoder und Dokument anfordern, für Baseline, SOF1 und progressiv, 4:4:4, 4:2:0, Grau
// und CMYK (vier Komponenten), RGB (Adobe-Transformation 0 oder Kennungen R, G, B ohne JFIF; Go legt dafür ein RGBA
// an), den Flex-Modus (Go legt die Farbebenen dann voll an) und Größen, die kein Vielfaches der MCU sind.
func TestJPEGBedarfSchaetzungTrifft(t *testing.T) {
	ohneArt := jpegArt{adobe: -1}
	for _, fall := range []struct {
		name string
		art  jpegArt
		w, h int
		sof  byte
		hv   []byte
	}{
		{"Baseline 4:4:4", ohneArt, 1024, 768, 0xC0, []byte{0x11, 0x11, 0x11}},
		{"progressiv 4:4:4", ohneArt, 1024, 768, 0xC2, []byte{0x11, 0x11, 0x11}},
		{"progressiv 4:2:0", ohneArt, 1024, 768, 0xC2, []byte{0x22, 0x11, 0x11}},
		{"Baseline 4:2:0", ohneArt, 1024, 768, 0xC0, []byte{0x22, 0x11, 0x11}},
		{"progressiv Grau", ohneArt, 1024, 768, 0xC2, []byte{0x11}},
		{"Baseline CMYK", ohneArt, 1024, 768, 0xC0, []byte{0x11, 0x11, 0x11, 0x11}},
		{"SOF1 4:2:0", ohneArt, 1024, 768, 0xC1, []byte{0x22, 0x11, 0x11}},
		{"Baseline RGB Adobe 0", jpegArt{ohneJFIF: true, adobe: 0}, 1024, 768, 0xC0, []byte{0x11, 0x11, 0x11}},
		{"progressiv RGB Adobe 0", jpegArt{ohneJFIF: true, adobe: 0}, 1024, 768, 0xC2, []byte{0x11, 0x11, 0x11}},
		{"Baseline RGB Kennungen", jpegArt{ohneJFIF: true, kennungen: "RGB", adobe: -1}, 1024, 768, 0xC0, []byte{0x11, 0x11, 0x11}},
		// DecodeConfig hält beim ersten Scan an und sieht dahinter nichts; Decode wertet Adobe und JFIF dort noch aus.
		{"progressiv RGB Adobe hinter dem Scan", jpegArt{ohneJFIF: true, adobe: -1, hinten: adobeSegment}, 1024, 768, 0xC2, []byte{0x11, 0x11, 0x11}},
		{"progressiv RGB Kennungen, JFIF hinten aufgehoben", jpegArt{kennungen: "RGB", adobe: -1, hinten: app0OhneJFIF}, 1024, 768, 0xC2, []byte{0x11, 0x11, 0x11}},
		{"Baseline Flex Y 1×1", ohneArt, 1024, 768, 0xC0, []byte{0x11, 0x22, 0x22}},
		{"progressiv Flex Y 1×1", ohneArt, 1024, 768, 0xC2, []byte{0x11, 0x22, 0x22}},
		{"Baseline Flex Cb≠Cr", ohneArt, 1024, 768, 0xC0, []byte{0x22, 0x12, 0x21}},
		{"Baseline Flex Verhältnis 1:2", ohneArt, 1024, 768, 0xC0, []byte{0x24, 0x11, 0x11}},
		{"Baseline 4:2:0 1023×767", ohneArt, 1023, 767, 0xC0, []byte{0x22, 0x11, 0x11}},
		{"progressiv 4:2:0 1023×767", ohneArt, 1023, 767, 0xC2, []byte{0x22, 0x11, 0x11}},
		{"progressiv Grau 2×2 1023×767", ohneArt, 1023, 767, 0xC2, []byte{0x22}},
	} {
		daten := jpegBauenMit(fall.art, fall.w, fall.h, fall.sof, fall.hv...)
		k, ok := leseJPEGKopf(daten)
		if !ok || k.breite != fall.w || k.hoehe != fall.h || k.progressiv != (fall.sof == 0xC2) || len(k.komponenten) != len(fall.hv) {
			t.Fatalf("%s: Kopf %+v, ok %v", fall.name, k, ok)
		}
		soll := schaetzung(t, daten)
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
		t.Logf("%s: angefordert %d KiB, geschätzt %d KiB", fall.name, ist>>10, soll>>10)
		// Gemessen trifft die Schätzung auf wenige KiB (Kleinkram des Dekoders); 2 % und 64 KiB Spiel.
		if d := ist - soll; d > soll/50+64<<10 || d < -soll/50 {
			t.Errorf("%s: angefordert %d KiB, geschätzt %d KiB", fall.name, ist>>10, soll>>10)
		}
	}
}

var (
	adobeSegment = append([]byte{0xFF, 0xEE, 0, 14}, []byte("Adobe\x00\x64\x00\x00\x00\x00\x00")...)
	app0OhneJFIF = append([]byte{0xFF, 0xE0, 0, 7}, []byte("JFXX\x00")...)
)

// F-361 QA-7: Die Ebenen werden auf ganze MCU gerundet, bei Grau auf 8 Pixel (Go rechnet Grau immer mit 1×1). Die
// Messung in TestJPEGBedarfSchaetzungTrifft ist dafür zu grob, deshalb hier genau.
func TestJPEGBedarfAufMCUGerundet(t *testing.T) {
	for _, fall := range []struct {
		name string
		k    jpegKopf
		soll int64
	}{
		// 4:2:0: MCU 16×16, 64×48 MCU; Y 12288 Blöcke, Cb und Cr je 3072; je Block 64 + 256 Byte.
		{"progressiv 4:2:0 1023×767", jpegKopf{breite: 1023, hoehe: 767, progressiv: true,
			komponenten: []jpegKomponente{{1, 2, 2}, {2, 1, 1}, {3, 1, 1}}}, 18432*320 + 1023*767*4},
		// Grau 2×2 nominal: Blöcke 8×8, 127×97.
		{"Baseline Grau 2×2 1009×769", jpegKopf{breite: 1009, hoehe: 769,
			komponenten: []jpegKomponente{{1, 2, 2}}}, 127*97*64 + 1009*769*4},
	} {
		if b := fall.k.bedarf(); b != fall.soll {
			t.Errorf("%s: %d, soll %d", fall.name, b, fall.soll)
		}
	}
}

// F-361 QA-1: Go überspringt Fremdbytes vor einem Marker und „FF 00“ still; die Vorabprüfung darf daran nicht
// vorbeigehen. Die präparierte 16384²-Datei mit solchen Bytes wird abgelehnt, bevor der Dekoder Speicher anfordert.
func TestOeffnenLehntTeuresJPEGMitFremdbytesAb(t *testing.T) {
	gut := jpegBauen(16384, 16384, 0xC2, 0x11, 0x11, 0x11)
	sof := bytes.Index(gut, []byte{0xFF, 0xC2})
	einfuegen := func(an int, b ...byte) []byte {
		return append(append(append([]byte(nil), gut[:an]...), b...), gut[an:]...)
	}
	for name, daten := range map[string][]byte{
		"Byte nach SOI":     einfuegen(2, 0x00),
		"Byte vor dem SOF":  einfuegen(sof, 0x42),
		"FF 00 vor dem SOF": einfuegen(sof, 0xFF, 0x00),
	} {
		if _, err := jpeg.DecodeConfig(bytes.NewReader(daten)); err != nil {
			t.Fatalf("%s: Go liest den Kopf nicht: %v", name, err)
		}
		pfad := filepath.Join(t.TempDir(), "teuer.jpg")
		if err := os.WriteFile(pfad, daten, 0o600); err != nil {
			t.Fatal(err)
		}
		var err error
		ist := angefordert(func() { _, err = Oeffne(pfad) })
		if !errors.Is(err, ErrZuGross) {
			t.Errorf("%s: Öffnen: %v, erwartet ErrZuGross", name, err)
		}
		if ist > 64<<20 {
			t.Errorf("%s: vor der Ablehnung %d MiB angefordert", name, ist>>20)
		}
	}
}

// F-361 QA-1: Der Leser überspringt, was Go überspringt: Fremdbytes, „FF 00“, Füllbytes und RST-Marker ohne Länge.
func TestLeseJPEGKopfNachsichtigWieGo(t *testing.T) {
	gut := jpegBauen(64, 48, 0xC2, 0x11, 0x11, 0x11)
	sof := bytes.Index(gut, []byte{0xFF, 0xC2})
	einfuegen := func(an int, b ...byte) []byte {
		return append(append(append([]byte(nil), gut[:an]...), b...), gut[an:]...)
	}
	for name, daten := range map[string][]byte{
		"Byte nach SOI":       einfuegen(2, 0x00),
		"Bytes vor dem SOF":   einfuegen(sof, 0x42, 0x17),
		"FF 00 vor dem SOF":   einfuegen(sof, 0xFF, 0x00),
		"RST vor dem SOF":     einfuegen(sof, 0xFF, 0xD3),
		"Füllbyte vor Marker": einfuegen(2, 0xFF),
	} {
		cfg, err := jpeg.DecodeConfig(bytes.NewReader(daten))
		if err != nil {
			t.Fatalf("%s: Go liest den Kopf nicht: %v", name, err)
		}
		k, ok := leseJPEGKopf(daten)
		if !ok || k.breite != cfg.Width || k.hoehe != cfg.Height || !k.progressiv || len(k.komponenten) != 3 {
			t.Errorf("%s: Kopf %+v, ok %v", name, k, ok)
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
		// knapp an 2 GiB (19 Byte je Pixel): 10624² liegt 2,8 MiB darunter, 10632² 0,3 MiB darüber
		{"progressiv 4:4:4 10624² knapp darunter", 10624, 10624, 0xC2, []byte{0x11, 0x11, 0x11}, true},
		{"progressiv 4:4:4 10632² knapp darüber", 10632, 10632, 0xC2, []byte{0x11, 0x11, 0x11}, false},
	} {
		err := pruefe(t, jpegBauen(fall.w, fall.h, fall.sof, fall.hv...))
		if (err == nil) != fall.erlaubt || err != nil && !errors.Is(err, ErrZuGross) {
			t.Errorf("%s: %v, erlaubt soll %v", fall.name, err, fall.erlaubt)
		}
	}
}

// Kaputte Köpfe: der Leser lehnt ab (pruefeJPEGBedarf rechnet dann mit cfg), kein Absturz. Was Go nicht überspringt,
// überspringt auch der Leser nicht: ein SOF hinter dem ersten Scan oder hinter dem Ende zählt nicht.
func TestLeseJPEGKopfKaputt(t *testing.T) {
	gut := jpegBauen(64, 48, 0xC2, 0x11, 0x11, 0x11)
	sof := bytes.Index(gut, []byte{0xFF, 0xC2})
	faelle := map[string][]byte{
		"leer":                   nil,
		"kein SOI":               []byte{0xFF, 0xD9, 0, 0},
		"abgeschnitten":          gut[:sof+5],
		"Länge zu groß":          append(append([]byte(nil), gut[:sof+2]...), 0xFF, 0xFF, 8),
		"Scan vor SOF":           {0xFF, 0xD8, 0xFF, 0xDA, 0, 2},
		"SOF hinter leerem Scan": append([]byte{0xFF, 0xD8, 0xFF, 0xDA, 0, 2}, gut[2:]...),
		"SOF hinter dem Ende":    append([]byte{0xFF, 0xD8, 0xFF, 0xD9, 0, 2}, gut[2:]...),
		"kein Marker":            {0xFF, 0xD8, 0x00, 0x00, 0x00},
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
		if err := pruefeJPEGBedarf(daten, image.Config{}); err != nil { // ohne Maße: nichts zu schätzen, kein Absturz
			t.Errorf("%s: %v", name, err)
		}
	}
	// Füllbytes vor einem Marker sind erlaubt.
	mitFuell := append([]byte{0xFF, 0xD8, 0xFF, 0xFF}, gut[2:]...)
	if _, ok := leseJPEGKopf(mitFuell); !ok {
		t.Error("Füllbyte vor dem Marker nicht übersprungen")
	}
}

// schaetzung liefert den geschätzten Bedarf so, wie Oeffne ihn prüft (mit DecodeConfig).
func schaetzung(t *testing.T, daten []byte) int64 {
	t.Helper()
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(daten))
	if err != nil {
		t.Fatalf("DecodeConfig: %v", err)
	}
	k, _ := schaetzeJPEG(daten, cfg)
	return k.bedarf()
}

// pruefe ruft pruefeJPEGBedarf mit dem cfg von DecodeConfig auf.
func pruefe(t *testing.T, daten []byte) error {
	t.Helper()
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(daten))
	if err != nil {
		t.Fatalf("DecodeConfig: %v", err)
	}
	return pruefeJPEGBedarf(daten, cfg)
}

// F-361 QA-1: Die Prüfung fällt geschlossen aus. Kann der Leser den Kopf nicht lesen oder passt er nicht zu dem, was
// DecodeConfig gelesen hat, wird mit den Maßen aus cfg und dem ungünstigsten Fall gerechnet.
func TestJPEGBedarfGeschlossen(t *testing.T) {
	ycbcr := image.Config{ColorModel: color.YCbCrModel, Width: 10000, Height: 10000}
	// Baseline 4:2:0 10000² ist erlaubt (rund 5,5 Byte je Pixel); ohne lesbaren Kopf nicht mehr.
	klein := jpegBauen(64, 48, 0xC0, 0x22, 0x11, 0x11)
	for name, daten := range map[string][]byte{
		"kein Kopf":               nil,
		"Kopf passt nicht zu cfg": klein,
	} {
		if err := pruefeJPEGBedarf(daten, ycbcr); !errors.Is(err, ErrZuGross) {
			t.Errorf("%s: %v, erwartet ErrZuGross", name, err)
		}
	}
	// Ungünstigster Fall ohne Kopf: progressiv, alle Komponenten voll (auf 32 gerundet), RGB bzw. CMYK.
	for _, fall := range []struct {
		cfg  image.Config
		soll int64
	}{
		{image.Config{ColorModel: color.GrayModel, Width: 1001, Height: 999}, 1008*1000*5 + 1001*999*4},
		{image.Config{ColorModel: color.YCbCrModel, Width: 1001, Height: 999}, 1024*1024*15 + 1001*999*8},
		{image.Config{ColorModel: color.RGBAModel, Width: 1001, Height: 999}, 1024*1024*15 + 1001*999*8},
		{image.Config{ColorModel: color.CMYKModel, Width: 1001, Height: 999}, 1024*1024*20 + 1001*999*8},
	} {
		if k, _ := schaetzeJPEG(nil, fall.cfg); k.bedarf() != fall.soll {
			t.Errorf("%v: geschätzt %d, soll %d", fall.cfg.ColorModel, k.bedarf(), fall.soll)
		}
	}
}
