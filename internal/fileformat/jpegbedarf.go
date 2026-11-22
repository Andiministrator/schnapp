// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import "fmt"

// jpegGrenze ist der höchste geschätzte Speicherbedarf beim Öffnen eines JPEG (F-361): Dekoder samt Ausgabeebenen und
// das NRGBA-Bild des Dokuments. 2 GiB lassen jedes Baseline-JPEG in YCbCr oder Grau bis document.MaxPixel zu (höchstens
// 7 Byte je Pixel); ein progressives JPEG kostet in Go zusätzlich 4 Byte je Pixel und Komponente für die Koeffizienten
// und ist deshalb kleiner begrenzt (4:4:4 rund 113 Megapixel, 4:2:0 rund 186).
const jpegGrenze = 2 << 30

// jpegKomponente ist eine Komponente aus dem SOF-Segment: Abtastfaktoren waagrecht und senkrecht.
type jpegKomponente struct{ h, v int }

// jpegKopf sind die Angaben aus dem ersten SOF-Segment.
type jpegKopf struct {
	breite, hoehe int
	progressiv    bool
	komponenten   []jpegKomponente
}

// leseJPEGKopf sucht das erste SOF-Segment (SOF0, SOF1, SOF2). ok ist false, wenn keins da ist oder die Angaben unbrauchbar
// sind: Dann entscheidet der Dekoder (er lehnt solche Dateien ab).
func leseJPEGKopf(daten []byte) (k jpegKopf, ok bool) {
	if len(daten) < 4 || daten[0] != 0xFF || daten[1] != 0xD8 {
		return k, false
	}
	i := 2
	for i+4 <= len(daten) {
		if daten[i] != 0xFF {
			return k, false
		}
		marker := daten[i+1]
		if marker == 0xFF { // Füllbyte
			i++
			continue
		}
		if marker == 0x01 || 0xD0 <= marker && marker <= 0xD8 { // ohne Länge
			i += 2
			continue
		}
		laenge := int(daten[i+2])<<8 | int(daten[i+3])
		if laenge < 2 || i+2+laenge > len(daten) {
			return k, false
		}
		inhalt := daten[i+4 : i+2+laenge]
		switch marker {
		case 0xC0, 0xC1, 0xC2:
			return sofLesen(inhalt, marker == 0xC2)
		case 0xDA, 0xD9: // Scan oder Ende vor einem SOF
			return k, false
		}
		i += 2 + laenge
	}
	return k, false
}

func sofLesen(inhalt []byte, progressiv bool) (k jpegKopf, ok bool) {
	if len(inhalt) < 6 {
		return k, false
	}
	k.hoehe = int(inhalt[1])<<8 | int(inhalt[2])
	k.breite = int(inhalt[3])<<8 | int(inhalt[4])
	n := int(inhalt[5])
	if n < 1 || n > 4 || len(inhalt) < 6+3*n || k.breite < 1 || k.hoehe < 1 {
		return k, false
	}
	k.progressiv = progressiv
	for j := 0; j < n; j++ {
		hv := inhalt[6+3*j+1]
		h, v := int(hv>>4), int(hv&0x0F)
		if h < 1 || h > 4 || v < 1 || v > 4 {
			return k, false
		}
		k.komponenten = append(k.komponenten, jpegKomponente{h, v})
	}
	return k, true
}

// bedarf schätzt die Bytes, die Öffnen anfordert, wie image/jpeg sie anlegt: je Komponente eine Ausgabeebene von
// 1 Byte je Abtastwert (auf ganze MCU gerundet), bei progressiven Dateien zusätzlich die Koeffizienten (ein [64]int32 je
// 8×8-Block), bei vier Komponenten das CMYK-Bild, und das NRGBA-Bild des Dokuments. Gemessen (F-361): progressiv 4:4:4
// 8192² und 16384² genau 19 Byte je Pixel, Baseline 7.
func (k jpegKopf) bedarf() int64 {
	hmax, vmax := 1, 1
	for _, c := range k.komponenten {
		hmax, vmax = max(hmax, c.h), max(vmax, c.v)
	}
	mxx := int64((k.breite + 8*hmax - 1) / (8 * hmax))
	myy := int64((k.hoehe + 8*vmax - 1) / (8 * vmax))
	pixel := int64(k.breite) * int64(k.hoehe)
	var summe int64
	for _, c := range k.komponenten {
		bloecke := mxx * myy * int64(c.h*c.v)
		summe += bloecke * 64
		if k.progressiv {
			summe += bloecke * 64 * 4
		}
	}
	if len(k.komponenten) == 4 {
		summe += pixel * 4
	}
	return summe + pixel*4
}

// pruefeJPEGBedarf lehnt ein JPEG ab, dessen Öffnen mehr als jpegGrenze anforderte (F-361).
func pruefeJPEGBedarf(daten []byte) error {
	k, ok := leseJPEGKopf(daten)
	if !ok {
		return nil
	}
	if b := k.bedarf(); b > jpegGrenze {
		art := "Baseline"
		if k.progressiv {
			art = "progressiv"
		}
		return fmt.Errorf("%w: JPEG %dx%d (%s, %d Komponenten) braucht beim Öffnen etwa %d MiB, höchstens %d MiB",
			ErrZuGross, k.breite, k.hoehe, art, len(k.komponenten), b>>20, jpegGrenze>>20)
	}
	return nil
}
