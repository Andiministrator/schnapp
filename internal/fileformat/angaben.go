// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"image"
	"io"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/errprefix"
)

// Angaben sind Objekte, Zuschnitt, Streifen und Kantenform eines Dokuments ohne Pixel — der Inhalt des Blocks snAP
// ohne Metadaten. Die Rettungskopie (ADR-0029) trägt sie in derselben Form wie die Arbeitsdatei.
type Angaben struct {
	Groesse   image.Point // Größe des Originals
	Zuschnitt image.Rectangle
	Streifen  []document.Streifen
	Kante     document.Kante // fehlt s:kante (ältere Rettungskopie), KanteZacken wie beim Laden einer Arbeitsdatei
	Objekte   []document.Objekt
}

// AngabenSVG schreibt die Angaben von d als SVG wie den Block snAP, ohne Metadaten. Es liest keine Pixel.
func AngabenSVG(d *document.Dokument) []byte { return svgArbeitsdatei(d, Metadaten{}) }

// LeseAngaben liest ein SVG aus AngabenSVG (oder den Block snAP) mit denselben Grenzen wie das Laden einer
// Arbeitsdatei (document.PruefeInhalt).
func LeseAngaben(r io.Reader) (_ *Angaben, fehler error) {
	defer errprefix.Defer("fileformat", &fehler)
	in, err := leseSVG(r)
	if err != nil {
		return nil, err
	}
	g := image.Pt(in.breite, in.hoehe)
	if _, err := document.PruefeInhalt(g, in.zuschnitt, in.streifen, in.objekte); err != nil {
		return nil, err
	}
	return &Angaben{Groesse: g, Zuschnitt: in.zuschnitt, Streifen: in.streifen, Kante: in.kante, Objekte: in.objekte}, nil
}
