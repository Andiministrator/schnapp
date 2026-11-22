// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
)

// jpegGrenze ist der höchste geschätzte Speicherbedarf beim Öffnen eines JPEG (F-361): Dekoder samt Ausgabeebenen und
// das NRGBA-Bild des Dokuments. 2 GiB lassen jedes Baseline-JPEG in YCbCr oder Grau bis document.MaxPixel zu (höchstens
// 7 Byte je Pixel); ein progressives JPEG kostet in Go zusätzlich 4 Byte je Pixel und Komponente für die Koeffizienten
// und ist deshalb kleiner begrenzt (4:4:4 rund 113 Megapixel, 4:2:0 rund 186). RGB-JPEG kosten 4 Byte je Pixel mehr.
const jpegGrenze = 2 << 30

// jpegKomponente ist eine Komponente aus dem SOF-Segment: Kennung und Abtastfaktoren waagrecht und senkrecht.
type jpegKomponente struct {
	id   byte
	h, v int
}

// jpegKopf sind die Angaben aus dem ersten SOF-Segment; rgb setzt pruefeJPEGBedarf (Go dekodiert dann über ein RGBA).
type jpegKopf struct {
	breite, hoehe int
	progressiv    bool
	rgb           bool
	komponenten   []jpegKomponente
}

// leseJPEGKopf sucht das erste SOF-Segment (SOF0, SOF1, SOF2) vor dem ersten Scan und überspringt dabei, was auch
// image/jpeg still überspringt: Fremdbytes vor einem Marker, „FF 00“, Füllbytes „FF“ und RST-Marker (ohne Länge).
// ok ist false, wenn keins da ist oder die Angaben unbrauchbar sind; pruefeJPEGBedarf rechnet dann mit dem
// ungünstigsten Fall.
func leseJPEGKopf(daten []byte) (k jpegKopf, ok bool) {
	if len(daten) < 2 || daten[0] != 0xFF || daten[1] != 0xD8 {
		return k, false
	}
	i := 2
	for {
		for i < len(daten) && daten[i] != 0xFF { // Fremdbytes
			i++
		}
		if i+1 >= len(daten) {
			return k, false
		}
		marker := daten[i+1]
		i += 2
		if marker == 0x00 { // „FF 00“ gilt Go als Fremdbyte
			continue
		}
		for marker == 0xFF { // Füllbytes
			if i >= len(daten) {
				return k, false
			}
			marker = daten[i]
			i++
		}
		if marker == 0xD9 || marker == 0xDA { // Ende oder Scan vor einem SOF
			return k, false
		}
		if 0xD0 <= marker && marker <= 0xD7 { // RST ohne Länge
			continue
		}
		if i+2 > len(daten) {
			return k, false
		}
		laenge := int(daten[i])<<8 | int(daten[i+1])
		if laenge < 2 || i+laenge > len(daten) {
			return k, false
		}
		switch marker {
		case 0xC0, 0xC1, 0xC2:
			return sofLesen(daten[i+2:i+laenge], marker == 0xC2)
		}
		i += laenge
	}
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
		k.komponenten = append(k.komponenten, jpegKomponente{inhalt[6+3*j], h, v})
	}
	return k, true
}

// flexModus sagt, ob image/jpeg die drei Farbebenen ohne Unterabtastung anlegt (scan.go makeImg, „flex mode“): wenn
// Cb und Cr verschieden abgetastet sind, Y nicht die größten Faktoren trägt oder das Verhältnis keins der sechs
// Standardverhältnisse ist.
func flexModus(c []jpegKomponente, hmax, vmax int) bool {
	if c[1].h != c[2].h || c[1].v != c[2].v || c[0].h != hmax || c[0].v != vmax {
		return true
	}
	switch (hmax/c[1].h)<<4 | vmax/c[1].v {
	case 0x11, 0x12, 0x21, 0x22, 0x41, 0x42:
		return false
	}
	return true
}

// bedarf schätzt die Bytes, die Öffnen anfordert, wie image/jpeg sie anlegt: je Komponente eine Ausgabeebene von
// 1 Byte je Abtastwert (auf ganze MCU gerundet; im Flex-Modus die drei Farbebenen in voller Größe; Grau immer 1×1),
// bei progressiven Dateien zusätzlich die Koeffizienten (ein [64]int32 je 8×8-Block), bei vier Komponenten das CMYK-
// bzw. RGBA-Bild, bei RGB das RGBA-Bild von convertToRGB, und das NRGBA-Bild des Dokuments. Gemessen (F-361):
// progressiv 4:4:4 8192² und 16384² genau 19 Byte je Pixel, Baseline 7; RGB 4 Byte mehr; Flex Y 1×1 mit Cb/Cr 2×2 7.
func (k jpegKopf) bedarf() int64 {
	komp := k.komponenten
	if len(komp) == 1 {
		komp = []jpegKomponente{{h: 1, v: 1}}
	}
	hmax, vmax := 1, 1
	for _, c := range komp {
		hmax, vmax = max(hmax, c.h), max(vmax, c.v)
	}
	mxx := int64((k.breite + 8*hmax - 1) / (8 * hmax))
	myy := int64((k.hoehe + 8*vmax - 1) / (8 * vmax))
	voll := 8 * int64(hmax) * mxx * 8 * int64(vmax) * myy // eine Ebene ohne Unterabtastung
	flex := len(komp) >= 3 && flexModus(komp, hmax, vmax)
	pixel := int64(k.breite) * int64(k.hoehe)
	var summe int64
	for j, c := range komp {
		bloecke := mxx * myy * int64(c.h*c.v)
		if flex && j < 3 {
			summe += voll
		} else {
			summe += bloecke * 64
		}
		if k.progressiv {
			summe += bloecke * 64 * 4
		}
	}
	if len(komp) == 4 {
		summe += pixel * 4
	}
	if k.rgb && len(komp) == 3 {
		summe += pixel * 4
	}
	return summe + pixel*4
}

// adobeOhneTransform sagt, ob irgendwo in der Datei ein Adobe-Segment (APP14) mit Transformation 0 steht. Go wertet
// es auch hinter dem ersten Scan aus, DecodeConfig sieht es dann nicht; die Suche über die ganze Datei schätzt im
// Zweifel zu hoch, nie zu niedrig.
func adobeOhneTransform(daten []byte) bool {
	for i := 0; ; {
		j := bytes.Index(daten[i:], []byte{0xFF, 0xEE})
		if j < 0 {
			return false
		}
		i += j + 2
		if i+14 <= len(daten) && int(daten[i])<<8|int(daten[i+1]) >= 14 && bytes.HasPrefix(daten[i+2:], []byte("Adobe")) &&
			daten[i+13] == 0 {
			return true
		}
	}
}

// schaetzeJPEG liefert den Kopf, mit dem pruefeJPEGBedarf rechnet, und seine Art für die Meldung. cfg ist das Ergebnis
// von jpeg.DecodeConfig. Die Schätzung fällt geschlossen aus: Kann der eigene Leser den Kopf nicht lesen oder weicht er
// von cfg ab, wird mit den Maßen aus cfg und dem ungünstigsten Fall gerechnet (progressiv, alle Komponenten voll und auf
// 32 Pixel gerundet, bei drei Komponenten RGB).
func schaetzeJPEG(daten []byte, cfg image.Config) (k jpegKopf, art string) {
	n := 3
	switch cfg.ColorModel {
	case color.GrayModel:
		n = 1
	case color.CMYKModel:
		n = 4
	}
	k, ok := leseJPEGKopf(daten)
	art = "Baseline"
	if ok && k.breite == cfg.Width && k.hoehe == cfg.Height && len(k.komponenten) == n {
		// Go nimmt RGB an ohne JFIF bei Adobe-Transformation 0 oder Kennungen R, G, B. JFIF kann ein späteres
		// APP0-Segment wieder aufheben; deshalb zählen Kennungen und Adobe-Segment auch neben cfg.
		k.rgb = n == 3 && (cfg.ColorModel == color.RGBAModel || adobeOhneTransform(daten) ||
			k.komponenten[0].id == 'R' && k.komponenten[1].id == 'G' && k.komponenten[2].id == 'B')
		if k.progressiv {
			art = "progressiv"
		}
	} else {
		k = jpegKopf{breite: cfg.Width, hoehe: cfg.Height, progressiv: true, rgb: n == 3}
		for range n {
			k.komponenten = append(k.komponenten, jpegKomponente{h: 4, v: 4})
		}
		art = "Kopf nicht lesbar, ungünstigster Fall"
	}
	return k, art
}

// pruefeJPEGBedarf lehnt ein JPEG ab, dessen Öffnen mehr als jpegGrenze anforderte (F-361); cfg wie bei schaetzeJPEG.
func pruefeJPEGBedarf(daten []byte, cfg image.Config) error {
	k, art := schaetzeJPEG(daten, cfg)
	if b := k.bedarf(); b > jpegGrenze {
		return fmt.Errorf("%w: JPEG %dx%d (%s, %d Komponenten) braucht beim Öffnen etwa %d MiB, höchstens %d MiB",
			ErrZuGross, k.breite, k.hoehe, art, len(k.komponenten), b>>20, jpegGrenze>>20)
	}
	return nil
}
