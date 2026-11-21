// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"encoding/binary"
	"image"
	"image/color"
)

// Kleiner eigener Leser für die EXIF-Ausrichtung (Tag 0x0112) eines JPEG (F-113, ADR-0011: kein Fremdpaket für ein
// einziges Feld). Gelesen wird nur diese eine Zahl, sonst nichts aus den Metadaten; sie wird nie ins Dokument oder in
// eine gespeicherte Datei übernommen (Prinzip „keine Spuren“). Jede unbrauchbare Angabe ergibt 1 (unverändert).

// exifAusrichtung liefert die Ausrichtung 1 bis 8 aus dem ersten brauchbaren Exif-Segment, sonst 1. Alle Zugriffe sind
// gegen die Länge geprüft; es gibt keine Schleife ohne Fortschritt.
func exifAusrichtung(jpg []byte) int {
	if len(jpg) < 4 || jpg[0] != 0xFF || jpg[1] != 0xD8 {
		return 1
	}
	i := 2
	for i+4 <= len(jpg) {
		if jpg[i] != 0xFF {
			return 1
		}
		m := jpg[i+1]
		if m == 0xFF { // Füllbyte
			i++
			continue
		}
		if m == 0x01 || (m >= 0xD0 && m <= 0xD7) { // Marken ohne Länge
			i += 2
			continue
		}
		if m == 0xD9 || m == 0xDA { // Ende bzw. Bilddaten: danach kommt kein APP1 mehr
			return 1
		}
		n := int(binary.BigEndian.Uint16(jpg[i+2:]))
		if n < 2 || i+2+n > len(jpg) {
			return 1
		}
		if m == 0xE1 {
			if o := tiffAusrichtung(jpg[i+4 : i+2+n]); o != 0 {
				return o
			}
		}
		i += 2 + n
	}
	return 1
}

// tiffAusrichtung wertet den Inhalt eines APP1-Segments aus. 0 heißt: kein Exif oder keine brauchbare Angabe.
func tiffAusrichtung(seg []byte) int {
	if len(seg) < 6+8 || string(seg[:6]) != "Exif\x00\x00" {
		return 0
	}
	t := seg[6:]
	var bo binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 0
	}
	if bo.Uint16(t[2:]) != 42 {
		return 0
	}
	ifd := uint64(bo.Uint32(t[4:]))
	if ifd+2 > uint64(len(t)) {
		return 0
	}
	n := uint64(bo.Uint16(t[ifd:]))
	for k := uint64(0); k < n; k++ {
		e := ifd + 2 + 12*k
		if e+12 > uint64(len(t)) {
			return 0
		}
		if bo.Uint16(t[e:]) != 0x0112 {
			continue
		}
		// Typ SHORT (3), Anzahl 1; der Wert steht in den ersten zwei Bytes des Wertfeldes.
		if bo.Uint16(t[e+2:]) != 3 || bo.Uint32(t[e+4:]) != 1 {
			return 0
		}
		if v := int(bo.Uint16(t[e+8:])); v >= 1 && v <= 8 {
			return v
		}
		return 0
	}
	return 0
}

// richteAus dreht oder spiegelt bild gemäß der EXIF-Ausrichtung o (1 bis 8) und liefert ein NRGBA-Bild mit Ursprung
// (0, 0). Bei 1 oder einem anderen Wert kommt bild unverändert zurück.
//
//	2 gespiegelt links-rechts   3 um 180 Grad gedreht   4 gespiegelt oben-unten
//	5 an der Hauptdiagonale gespiegelt   6 um 90 Grad im Uhrzeigersinn   7 an der Gegendiagonale gespiegelt
//	8 um 90 Grad gegen den Uhrzeigersinn
//
// Neben der Quelle entsteht genau eine Vollbild-Kopie (das Ergebnis); die Quelle wird direkt gelesen, nicht erst in
// ein NRGBA-Zwischenbild umkopiert (F-323 a).
func richteAus(bild image.Image, o int) image.Image {
	if o < 2 || o > 8 {
		return bild
	}
	b := bild.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 1 || h < 1 {
		return bild
	}
	dw, dh := w, h
	if o >= 5 {
		dw, dh = h, w
	}
	ziel := image.NewNRGBA(image.Rect(0, 0, dw, dh))
	lies := pixelLeser(bild)
	for dy := 0; dy < dh; dy++ {
		for dx := 0; dx < dw; dx++ {
			var sx, sy int
			switch o {
			case 2:
				sx, sy = w-1-dx, dy
			case 3:
				sx, sy = w-1-dx, h-1-dy
			case 4:
				sx, sy = dx, h-1-dy
			case 5:
				sx, sy = dy, dx
			case 6:
				sx, sy = dy, h-1-dx
			case 7:
				sx, sy = w-1-dy, h-1-dx
			case 8:
				sx, sy = w-1-dy, dx
			}
			d := ziel.PixOffset(dx, dy)
			lies(ziel.Pix[d:d+4:d+4], b.Min.X+sx, b.Min.Y+sy)
		}
	}
	return ziel
}

// pixelLeser liefert eine Funktion, die das Pixel (x, y) der Quelle (in deren Koordinaten) als NRGBA in dst schreibt,
// mit demselben Ergebnis wie draw.Draw mit draw.Src in ein NRGBA-Bild. Die häufigen Typen gehen ohne Schnittstellenaufruf.
func pixelLeser(bild image.Image) func(dst []byte, x, y int) {
	switch q := bild.(type) {
	case *image.NRGBA:
		return func(dst []byte, x, y int) {
			s := q.PixOffset(x, y)
			copy(dst, q.Pix[s:s+4])
		}
	case *image.YCbCr:
		return func(dst []byte, x, y int) {
			yi := q.YOffset(x, y)
			ci := q.COffset(x, y)
			dst[0], dst[1], dst[2] = color.YCbCrToRGB(q.Y[yi], q.Cb[ci], q.Cr[ci])
			dst[3] = 255
		}
	}
	return func(dst []byte, x, y int) {
		c := color.NRGBAModel.Convert(bild.At(x, y)).(color.NRGBA)
		dst[0], dst[1], dst[2], dst[3] = c.R, c.G, c.B, c.A
	}
}
