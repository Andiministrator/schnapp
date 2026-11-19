package main

import (
	"bytes"
	"testing"
)

// Zwei Pixel je Zeile, zwei Zeilen, Zeilenlänge 12 (4 Bytes Füllung am Zeilenende).
// Pixel: (200,100,3,128), (10,20,30,255) / (0,0,0,0), (255,255,255,255) als R,G,B,A.
func probeDaten(bgra bool) []byte {
	pix := [][4]byte{{200, 100, 3, 128}, {10, 20, 30, 255}, {0, 0, 0, 0}, {255, 255, 255, 255}}
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

func TestRohZuRGBA(t *testing.T) {
	// Sollwerte von Hand: 200·128/255 = 100,39 → 100; 100·128/255 = 50,2 → 50; 3·128/255 = 1,506 → 2 (Rundung, nicht Abschneiden).
	nichtVormult := []byte{100, 50, 2, 128, 10, 20, 30, 255, 0, 0, 0, 0, 255, 255, 255, 255}
	unveraendert := []byte{200, 100, 3, 128, 10, 20, 30, 255, 0, 0, 0, 0, 255, 255, 255, 255}
	deckend := []byte{200, 100, 3, 255, 10, 20, 30, 255, 0, 0, 0, 255, 255, 255, 255, 255}
	faelle := []struct {
		name   string
		format uint32
		bgra   bool
		soll   []byte
	}{
		{"RGB32", qFormatRGB32, true, deckend},
		{"ARGB32", qFormatARGB32, true, nichtVormult},
		{"ARGB32Pre", qFormatARGB32Pre, true, unveraendert},
		{"RGBX8888", qFormatRGBX8888, false, deckend},
		{"RGBA8888", qFormatRGBA8888, false, nichtVormult},
		{"RGBA8888Pre", qFormatRGBA8888Pre, false, unveraendert},
	}
	for _, f := range faelle {
		bild, err := rohZuRGBA(probeDaten(f.bgra), 2, 2, 12, f.format)
		if err != nil {
			t.Fatalf("%s: %v", f.name, err)
		}
		if bild.Stride != 8 || !bytes.Equal(bild.Pix, f.soll) {
			t.Errorf("%s: Pixel %v, Soll %v", f.name, bild.Pix, f.soll)
		}
	}
}

func TestRohZuRGBAFehler(t *testing.T) {
	d := probeDaten(true)
	if _, err := rohZuRGBA(d, 2, 2, 12, 99); err == nil {
		t.Error("unbekanntes Format ohne Fehler")
	}
	if _, err := rohZuRGBA(d[:len(d)-5], 2, 2, 12, qFormatARGB32); err == nil {
		t.Error("zu kurze Daten ohne Fehler")
	}
	// Die letzte Zeile braucht keine Füllbytes: genau 12+8 Bytes reichen.
	if _, err := rohZuRGBA(d[:20], 2, 2, 12, qFormatARGB32); err != nil {
		t.Errorf("letzte Zeile ohne Füllung abgelehnt: %v", err)
	}
	if _, err := rohZuRGBA(d, 2, 2, 7, qFormatARGB32); err == nil {
		t.Error("Zeilenlänge < Breite×4 ohne Fehler")
	}
	if _, err := rohZuRGBA(d, 0, 2, 12, qFormatARGB32); err == nil {
		t.Error("Breite 0 ohne Fehler")
	}
}
