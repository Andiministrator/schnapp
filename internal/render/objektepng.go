// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"bytes"
	"image"
	"image/png"
	"slices"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/errprefix"
)

// ObjektePNG rastert nur die Objekte ids von d auf durchsichtigen Grund (Objektebene eines Abzugs ohne die übrigen
// Objekte), schneidet auf ihre sichtbaren Pixel zu und kodiert als reines PNG wie PNG: das Bild einer Objekt-Kopie
// für fremde Programme (ADR-0045 Punkt 2). Eine Lupe zeigt darin ihren Inhalt. Sind keine Pixel sichtbar, ist es ein
// durchsichtiges Pixel. Läuft wie PNG auf einem Abzug in einer Goroutine; d selbst wird nicht verändert.
func ObjektePNG(d *document.Dokument, ids []int) (daten []byte, fehler error) {
	defer errprefix.Defer("render", &fehler)
	a, err := d.Abzug()
	if err != nil {
		return nil, err
	}
	var weg []int
	for _, o := range a.Objekte() {
		if !slices.Contains(ids, o.Kennung()) {
			weg = append(weg, o.Kennung())
		}
	}
	if len(weg) > 0 {
		if err := a.EntferneMehrere(weg); err != nil {
			return nil, err
		}
	}
	b, err := Objektebene(a)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, b.SubImage(deckung(b)).(*image.NRGBA)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// deckung ist das kleinste Rechteck um alle Pixel mit Alpha über 0; ohne solche ein Pixel oben links.
func deckung(b *image.NRGBA) image.Rectangle {
	x0, y0, x1, y1 := b.Rect.Max.X, b.Rect.Max.Y, b.Rect.Min.X, b.Rect.Min.Y
	for y := b.Rect.Min.Y; y < b.Rect.Max.Y; y++ {
		zeile := b.Pix[(y-b.Rect.Min.Y)*b.Stride:]
		for x := 0; x < b.Rect.Dx(); x++ {
			if zeile[4*x+3] != 0 {
				x0, x1 = min(x0, b.Rect.Min.X+x), max(x1, b.Rect.Min.X+x+1)
				y0, y1 = min(y0, y), max(y1, y+1)
			}
		}
	}
	if x1 <= x0 {
		return image.Rect(b.Rect.Min.X, b.Rect.Min.Y, b.Rect.Min.X+1, b.Rect.Min.Y+1)
	}
	return image.Rect(x0, y0, x1, y1)
}
