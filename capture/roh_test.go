// SPDX-License-Identifier: GPL-3.0-or-later

package capture

import (
	"bytes"
	"strings"
	"testing"
)

// Zwei Pixel je Zeile, zwei Zeilen, Zeilenlänge 12 (4 Bytes Füllung am Zeilenende).
// Pixel (R,G,B,A): (200,100,3,128), (10,20,30,255) / (0,0,0,0), (255,255,255,255); bei vormultiplizierten
// Formaten stehen die vormultiplizierten Werte (100,50,2,128) für das erste Pixel.
func probeDaten(bgra, vormult bool) []byte {
	pix := [][4]byte{{200, 100, 3, 128}, {10, 20, 30, 255}, {0, 0, 0, 0}, {255, 255, 255, 255}}
	if vormult {
		pix[0] = [4]byte{100, 50, 2, 128}
	}
	var d []byte
	for z := 0; z < 2; z++ {
		for s := 0; s < 2; s++ {
			p := pix[z*2+s]
			if bgra {
				d = append(d, p[2], p[1], p[0], p[3])
			} else {
				d = append(d, p[0], p[1], p[2], p[3])
			}
		}
		d = append(d, 0xEE, 0xEE, 0xEE, 0xEE) // Füllbytes dürfen nicht im Bild landen
	}
	return d
}

func TestRohZuNRGBA(t *testing.T) {
	// Sollwerte von Hand. Nicht vormultipliziert: unverändert. Vormultipliziert (100,50,2,128) → c·255/128 mit
	// Rundung: 100 → 199,2 → 199 (nach +64 Rundung 199,7 → Ganzzahldivision 199); 50 → 100,4 → 100; 2 → 3,98 → 4.
	// Deckende Formate setzen Alpha 255, Farben bleiben.
	unveraendert := []byte{200, 100, 3, 128, 10, 20, 30, 255, 0, 0, 0, 0, 255, 255, 255, 255}
	zurueck := []byte{199, 100, 4, 128, 10, 20, 30, 255, 0, 0, 0, 0, 255, 255, 255, 255}
	deckend := []byte{200, 100, 3, 255, 10, 20, 30, 255, 0, 0, 0, 255, 255, 255, 255, 255}
	faelle := []struct {
		name    string
		format  uint32
		bgra    bool
		vormult bool
		soll    []byte
	}{
		{"RGB32", qFormatRGB32, true, false, deckend},
		{"ARGB32", qFormatARGB32, true, false, unveraendert},
		{"ARGB32Pre", qFormatARGB32Pre, true, true, zurueck},
		{"RGBX8888", qFormatRGBX8888, false, false, deckend},
		{"RGBA8888", qFormatRGBA8888, false, false, unveraendert},
		{"RGBA8888Pre", qFormatRGBA8888Pre, false, true, zurueck},
	}
	for _, f := range faelle {
		bild, err := rohZuNRGBA(probeDaten(f.bgra, f.vormult), 2, 2, 12, f.format)
		if err != nil {
			t.Fatalf("%s: %v", f.name, err)
		}
		if bild.Stride != 8 || !bytes.Equal(bild.Pix, f.soll) {
			t.Errorf("%s: Pixel %v, Soll %v", f.name, bild.Pix, f.soll)
		}
	}
}

func TestEntvormultipliziere(t *testing.T) {
	// c > a ist bei gültigen Daten unmöglich; das Ergebnis wird auf 255 begrenzt statt überzulaufen.
	if g := entvormultipliziere(200, 128); g != 255 {
		t.Errorf("200/128 = %d, Soll 255", g)
	}
	if g := entvormultipliziere(7, 0); g != 0 {
		t.Errorf("a=0: %d, Soll 0", g)
	}
	if g := entvormultipliziere(128, 128); g != 255 {
		t.Errorf("128/128 = %d, Soll 255", g)
	}
}

func TestRohZuNRGBAFehler(t *testing.T) {
	d := probeDaten(true, false)
	if _, err := rohZuNRGBA(d, 2, 2, 12, 99); err == nil {
		t.Error("unbekanntes Format ohne Fehler")
	}
	if _, err := rohZuNRGBA(d[:len(d)-5], 2, 2, 12, qFormatARGB32); err == nil {
		t.Error("zu kurze Daten ohne Fehler")
	}
	// Die letzte Zeile braucht keine Füllbytes: genau 12+8 Bytes reichen.
	if _, err := rohZuNRGBA(d[:20], 2, 2, 12, qFormatARGB32); err != nil {
		t.Errorf("letzte Zeile ohne Füllung abgelehnt: %v", err)
	}
	if _, err := rohZuNRGBA(d, 2, 2, 7, qFormatARGB32); err == nil {
		t.Error("Zeilenlänge < Breite×4 ohne Fehler")
	}
	if _, err := rohZuNRGBA(d, 0, 2, 12, qFormatARGB32); err == nil {
		t.Error("Breite 0 ohne Fehler")
	}
}

// F-17: unplausible Größen liefern einen Fehler statt Überlauf und Absturz in image.NewNRGBA.
func TestRohZuNRGBAUeberlauf(t *testing.T) {
	d := probeDaten(true, false)
	faelle := []struct {
		name                 string
		breite, hoehe, zeile uint64
	}{
		{"Breite 1<<62", 1 << 62, 2, 12},
		{"Breite 1<<62, riesige Zeile", 1 << 62, 1, 1 << 63},
		{"Höhe 1<<62", 2, 1 << 62, 12},
		{"Breite MaxSeite+1", MaxSeite + 1, 1, 4 * (MaxSeite + 1)},
		{"Höhe MaxSeite+1", 1, MaxSeite + 1, 4},
		{"Zeile 1<<63", 2, 2, 1 << 63},
		{"Zeile über maxZeile", 2, 2, maxZeile + 1},
		{"Höhe×Zeile über MaxBytes", MaxSeite, MaxSeite, 4 * MaxSeite}, // 4 GiB
		{"Breite 1<<30 (Breite×4 läuft in uint32 über)", 1 << 30, 1, 1 << 32},
	}
	for _, f := range faelle {
		if _, err := rohZuNRGBA(d, f.breite, f.hoehe, f.zeile, qFormatARGB32); err == nil {
			t.Errorf("%s: kein Fehler", f.name)
		}
		if _, err := groesseRoh(f.breite, f.hoehe, f.zeile); err == nil {
			t.Errorf("%s: groesseRoh ohne Fehler", f.name)
		}
	}
	// Obere Kante: MaxSeite×1 ist erlaubt.
	if n, err := groesseRoh(MaxSeite, 1, 4*MaxSeite); err != nil || n != 4*MaxSeite {
		t.Errorf("MaxSeite×1: %d, %v", n, err)
	}
}

func TestLiesBegrenzt(t *testing.T) {
	if d, err := liesBegrenzt(strings.NewReader("12345"), 5); err != nil || string(d) != "12345" {
		t.Errorf("genau am Limit: %q, %v", d, err)
	}
	if _, err := liesBegrenzt(strings.NewReader("123456"), 5); err == nil {
		t.Error("ein Byte über dem Limit ohne Fehler")
	}
	// Eine endlose Quelle muss am Limit enden, nicht den Speicher füllen.
	if _, err := liesBegrenzt(endlos{}, 1000); err == nil {
		t.Error("endlose Quelle ohne Fehler")
	}
}

type endlos struct{}

func (endlos) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 1
	}
	return len(p), nil
}

func TestZahl(t *testing.T) {
	if n, ok := zahl(int32(-1)); ok {
		t.Errorf("int32(-1) → %d akzeptiert", n)
	}
	if n, ok := zahl(int64(-5)); ok {
		t.Errorf("int64(-5) → %d akzeptiert", n)
	}
	if n, ok := zahl(uint32(1 << 31)); !ok || n != 1<<31 {
		t.Errorf("uint32(1<<31) → %d, %v", n, ok)
	}
	if _, ok := zahl("7"); ok {
		t.Error("String akzeptiert")
	}
}
