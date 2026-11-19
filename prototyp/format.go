// SPDX-License-Identifier: GPL-3.0-or-later

// Umwandlung der rohen Pixel, die KWin ScreenShot2 über die Pipe liefert.
package main

import (
	"fmt"
	"image"
)

// Nummern aus QImage::Format, die KWin für ScreenShot2 liefern kann.
const (
	qFormatRGB32       = 4  // 0xffRRGGBB als 32-Bit-Zahl, im Speicher (little endian) B G R ff
	qFormatARGB32      = 5  // wie RGB32, Alpha nicht vormultipliziert
	qFormatARGB32Pre   = 6  // wie ARGB32, vormultipliziert
	qFormatRGBX8888    = 16 // Byte-Reihenfolge R G B ff
	qFormatRGBA8888    = 17 // R G B A, nicht vormultipliziert
	qFormatRGBA8888Pre = 18 // R G B A, vormultipliziert
)

// rohZuRGBA deutet die rohen Bytes nach dem QImage-Format und liefert ein Bild mit
// vormultipliziertem Alpha (wie image.RGBA es verlangt). Nur little-endian-Rechner.
func rohZuRGBA(daten []byte, breite, hoehe, zeile int, format uint32) (*image.RGBA, error) {
	if breite <= 0 || hoehe <= 0 {
		return nil, fmt.Errorf("ungültige Bildgröße %dx%d", breite, hoehe)
	}
	if zeile < breite*4 {
		return nil, fmt.Errorf("Zeilenlänge %d kleiner als Breite×4 (%d)", zeile, breite*4)
	}
	if noetig := zeile*(hoehe-1) + breite*4; len(daten) < noetig {
		return nil, fmt.Errorf("zu wenig Pixeldaten: %d Bytes, nötig %d", len(daten), noetig)
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
		return nil, fmt.Errorf("QImage-Format %d wird nicht unterstützt", format)
	}
	bild := image.NewRGBA(image.Rect(0, 0, breite, hoehe))
	for y := 0; y < hoehe; y++ {
		quelle := daten[y*zeile : y*zeile+breite*4]
		ziel := bild.Pix[y*bild.Stride : y*bild.Stride+breite*4]
		for x := 0; x < breite*4; x += 4 {
			r, g, b, a := quelle[x], quelle[x+1], quelle[x+2], quelle[x+3]
			if bgra {
				r, b = b, r
			}
			switch {
			case deckend:
				a = 255
			case !vormult && a != 255:
				r, g, b = vormultipliziere(r, a), vormultipliziere(g, a), vormultipliziere(b, a)
			}
			ziel[x], ziel[x+1], ziel[x+2], ziel[x+3] = r, g, b, a
		}
	}
	return bild, nil
}

// vormultipliziere rechnet c·a/255 mit Rundung zur nächsten ganzen Zahl.
func vormultipliziere(c, a uint8) uint8 {
	return uint8((uint32(c)*uint32(a) + 127) / 255)
}
