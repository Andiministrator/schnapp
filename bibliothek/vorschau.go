// SPDX-License-Identifier: GPL-3.0-or-later

package bibliothek

import (
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
)

// VorschauSeite ist die längste Seite eines Vorschaubilds in Pixeln.
const VorschauSeite = 256

// VorschauPfad ist der Ort des Vorschaubilds zu einem Eintrag-Pfad: ein Hash des relativen Pfads, damit der
// Name weder Pfad noch Zeit verrät und Unterordner entfallen.
func (b *Bibliothek) VorschauPfad(rel string) string {
	h := sha256.Sum256([]byte(rel))
	return filepath.Join(b.Orte.Vorschau, hex.EncodeToString(h[:12])+".png")
}

func (b *Bibliothek) schreibeVorschau(rel string, bild image.Image) error {
	if err := os.MkdirAll(b.Orte.Vorschau, rechteOrdner); err != nil {
		return err
	}
	klein := verkleinere(bild, VorschauSeite)
	return schreibeAtomar(b.VorschauPfad(rel), func(w io.Writer) error { return png.Encode(w, klein) })
}

// verkleinere skaliert das Bild so, dass die längste Seite höchstens seite Pixel misst (nie vergrößern), durch
// Mittelwert über das zugehörige Quellrechteck.
func verkleinere(bild image.Image, seite int) *image.NRGBA {
	b := bild.Bounds()
	w, h := b.Dx(), b.Dy()
	nw, nh := w, h
	if max(w, h) > seite {
		if w >= h {
			nw, nh = seite, max(1, h*seite/w)
		} else {
			nw, nh = max(1, w*seite/h), seite
		}
	}
	aus := image.NewNRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		y0, y1 := y*h/nh, max(y*h/nh+1, (y+1)*h/nh)
		for x := 0; x < nw; x++ {
			x0, x1 := x*w/nw, max(x*w/nw+1, (x+1)*w/nw)
			var r, g, bl, a, n uint64
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					cr, cg, cb, ca := bild.At(b.Min.X+sx, b.Min.Y+sy).RGBA() // vormultipliziert, 16 Bit
					r, g, bl, a, n = r+uint64(cr), g+uint64(cg), bl+uint64(cb), a+uint64(ca), n+1
				}
			}
			// Mittelwert (vormultipliziert) und zurück nach NRGBA
			aus.Set(x, y, color.RGBA64{uint16(r / n), uint16(g / n), uint16(bl / n), uint16(a / n)})
		}
	}
	return aus
}
