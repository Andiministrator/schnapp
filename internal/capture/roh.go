// SPDX-License-Identifier: GPL-3.0-or-later

package capture

import (
	"fmt"
	"image"
	"io"
	"math"
)

// Nummern aus QImage::Format, die KWin liefern kann (32-Bit-Formate).
const (
	qFormatRGB32       = 4  // 0xffRRGGBB als 32-Bit-Zahl, im Speicher (little endian) B G R ff
	qFormatARGB32      = 5  // wie RGB32, Alpha nicht vormultipliziert
	qFormatARGB32Pre   = 6  // wie ARGB32, vormultipliziert
	qFormatRGBX8888    = 16 // Byte-Reihenfolge R G B ff
	qFormatRGBA8888    = 17 // R G B A, nicht vormultipliziert
	qFormatRGBA8888Pre = 18 // R G B A, vormultipliziert
)

// maxZeile begrenzt die Zeilenlänge (Bytes): Breite×4 plus großzügig Füllung.
const maxZeile = 8 * MaxSeite

// groesseRoh prüft Breite, Höhe und Zeilenlänge und liefert die Zahl der nötigen Rohbytes (die letzte Zeile braucht
// keine Füllbytes). Gerechnet wird in uint64 mit vorheriger Obergrenze, es gibt keinen Überlauf (F-17).
func groesseRoh(breite, hoehe, zeile uint64) (uint64, error) {
	if breite == 0 || hoehe == 0 || breite > MaxSeite || hoehe > MaxSeite {
		return 0, fmt.Errorf("capture: ungültige Bildgröße %dx%d", breite, hoehe)
	}
	if zeile < breite*4 { // breite ≤ 2^15, kein Überlauf
		return 0, fmt.Errorf("capture: Zeilenlänge %d kleiner als Breite×4 (%d)", zeile, breite*4)
	}
	if zeile > maxZeile {
		return 0, fmt.Errorf("capture: Zeilenlänge %d über der Grenze %d", zeile, maxZeile)
	}
	noetig := zeile*(hoehe-1) + breite*4 // ≤ 2^18·2^15 + 2^17, passt in uint64
	if noetig > MaxBytes {
		return 0, fmt.Errorf("capture: Bild mit %d Bytes über der Grenze %d", noetig, int64(MaxBytes))
	}
	return noetig, nil
}

// rohZuNRGBA deutet die rohen Bytes nach dem QImage-Format und liefert ein Bild mit nicht vormultipliziertem
// Alpha. Nur little-endian-Rechner (wie die Formatnummern oben beschrieben).
func rohZuNRGBA(daten []byte, breite, hoehe, zeile uint64, format uint32) (*image.NRGBA, error) {
	noetig, err := groesseRoh(breite, hoehe, zeile)
	if err != nil {
		return nil, err
	}
	if uint64(len(daten)) < noetig {
		return nil, fmt.Errorf("capture: zu wenig Pixeldaten: %d Bytes, nötig %d", len(daten), noetig)
	}
	var bgra, deckend, vormult bool
	switch format {
	case qFormatRGB32:
		bgra, deckend = true, true
	case qFormatARGB32:
		bgra = true
	case qFormatARGB32Pre:
		bgra, vormult = true, true
	case qFormatRGBX8888:
		deckend = true
	case qFormatRGBA8888:
	case qFormatRGBA8888Pre:
		vormult = true
	default:
		return nil, fmt.Errorf("capture: QImage-Format %d wird nicht unterstützt", format)
	}
	b, h, z := int(breite), int(hoehe), int(zeile) // alle ≤ Grenzen, passen in int
	bild := image.NewNRGBA(image.Rect(0, 0, b, h))
	for y := 0; y < h; y++ {
		quelle := daten[y*z : y*z+b*4]
		ziel := bild.Pix[y*bild.Stride : y*bild.Stride+b*4]
		for x := 0; x < b*4; x += 4 {
			r, g, bl, a := quelle[x], quelle[x+1], quelle[x+2], quelle[x+3]
			if bgra {
				r, bl = bl, r
			}
			switch {
			case deckend:
				a = 255
			case vormult && a != 255:
				r, g, bl = entvormultipliziere(r, a), entvormultipliziere(g, a), entvormultipliziere(bl, a)
			}
			ziel[x], ziel[x+1], ziel[x+2], ziel[x+3] = r, g, bl, a
		}
	}
	return bild, nil
}

// entvormultipliziere rechnet c·255/a mit Rundung zur nächsten ganzen Zahl, höchstens 255 (fehlerhafte Daten mit
// c > a). Bei a = 0 ist die Farbe 0.
func entvormultipliziere(c, a uint8) uint8 {
	if a == 0 {
		return 0
	}
	v := (uint32(c)*255 + uint32(a)/2) / uint32(a)
	if v > 255 {
		return 255
	}
	return uint8(v)
}

// liesBegrenzt liest r bis zum Ende, höchstens aber max Bytes; mehr ist ein Fehler (F-17: kein unbegrenztes
// io.ReadAll auf eine Pipe, deren Gegenstelle wir nicht kontrollieren).
func liesBegrenzt(r io.Reader, max int64) ([]byte, error) {
	d, err := io.ReadAll(io.LimitReader(r, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(d)) > max {
		return nil, fmt.Errorf("mehr als %d Bytes Pixeldaten", max)
	}
	return d, nil
}

// zahl liest eine nichtnegative Ganzzahl aus einem Antwortwert; Vorzeichenfehler (negativer int32) sind ein Fehler.
func zahl(v any) (uint64, bool) {
	switch n := v.(type) {
	case uint32:
		return uint64(n), true
	case uint64:
		return n, true
	case int32:
		if n < 0 {
			return 0, false
		}
		return uint64(n), true
	case int64:
		if n < 0 {
			return 0, false
		}
		return uint64(n), true
	}
	return 0, false
}

// skalaAus liest die Skala; fehlt sie oder ist sie unsinnig (≤ 0, NaN, > 64), gilt 1.
func skalaAus(v any) float64 {
	s, ok := v.(float64)
	if !ok || math.IsNaN(s) || s <= 0 || s > 64 {
		return 1
	}
	return s
}
