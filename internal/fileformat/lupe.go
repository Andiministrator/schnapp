// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"math"
	"strings"

	"github.com/Andiministrator/schnapp/internal/document"
)

// rahmenPfad ist der eckige Rahmen um r als gefüllter Pfad: genau die Rechtecke des Renderers (document.EckigerRahmen,
// F-498), wie beim Kasten ohne Fläche.
func rahmenPfad(r document.Rechteck, breite float64) string {
	var pfad strings.Builder
	for i, q := range document.EckigerRahmen(r, breite) {
		if i > 0 {
			pfad.WriteString(" ")
		}
		pfad.WriteString("M" + punkt(q.Min) + " L" + punkt(document.Pt(q.Max.X, q.Min.Y)) + " L" + punkt(q.Max) +
			" L" + punkt(document.Pt(q.Min.X, q.Max.Y)) + " Z")
	}
	return pfad.String()
}

// lupe schreibt eine Lupe (ADR-0033 Punkte 4 bis 6, docs/format.md 8.5): die Daten s:quelle, s:ziel, s:strich und die
// Darstellung von unten nach oben — Verbindungslinie, Rahmen um die Quelle, Bild im Ziel, Rahmen um das Ziel. In der
// Arbeitsdatei (rundlauf) steht kein Bild: Die Pixel liegen im Original (snAO), eine Kopie trüge nach einer späteren
// Schwärzung den Klartext weiter. Der Export bettet die sichtbaren Pixel der Quelle als PNG ein, vergrößert ohne Glätten.
func (s *schreiber) lupe(d *document.Dokument, o document.Lupe) {
	s.WriteString(`<g`)
	s.attr("s:typ", "lupe")
	s.ebene(o.Ebenenwerte())
	s.attr("s:quelle", punkt(o.Quelle.Min)+" "+punkt(o.Quelle.Max))
	s.attr("s:ziel", punkt(o.Ziel.Min)+" "+punkt(o.Ziel.Max))
	s.attr("s:strich", strich(o.Strich))
	s.WriteString(`>`)
	strichDa := o.Strich.Breite > 0 && o.Strich.Farbe.A > 0
	rahmen := func(r document.Rechteck) {
		s.WriteString(`<path`)
		s.attr("d", rahmenPfad(r, o.Strich.Breite))
		s.farbeSVG("fill", o.Strich.Farbe)
		s.WriteString(`/>`)
	}
	if strichDa {
		if a, b, ok := o.Linie(); ok {
			s.WriteString(`<path`)
			s.attr("d", "M"+punkt(a)+" L"+punkt(b))
			s.attr("fill", "none")
			s.farbeSVG("stroke", o.Strich.Farbe)
			s.attr("stroke-width", zahl(o.Strich.Breite))
			s.attr("stroke-linecap", "round")
			s.WriteString(`/>`)
		}
		rahmen(o.Quelle)
	}
	if !s.rundlauf {
		s.lupenBild(d, o)
	}
	if strichDa {
		rahmen(o.Ziel)
	}
	s.WriteString(`</g>`)
}

// lupenBild schreibt das Bild im Ziel für den Export: ein <svg> in der Lage des Ziels, dessen viewBox die Quelle ist
// (preserveAspectRatio none, beschneidet auf das Ziel), darin die ganzen Pixel unter der Quelle als PNG mit
// image-rendering pixelated. So nimmt jedes Zielpixel das Quellpixel floor(Quelle.Min + (z − Ziel.Min) · Quelle/Ziel)
// wie im Renderer (ADR-0033 Punkt 3). Was die Ansicht nicht zeigt, ist im PNG durchsichtig (Punkt 2).
func (s *schreiber) lupenBild(d *document.Dokument, o document.Lupe) {
	q := o.Quelle
	r := image.Rect(int(math.Floor(q.Min.X)), int(math.Floor(q.Min.Y)), int(math.Ceil(q.Max.X)), int(math.Ceil(q.Max.Y)))
	px := d.SichtbarerAusschnitt(r)
	if px.Rect.Empty() {
		return
	}
	var kodiert bytes.Buffer
	if err := png.Encode(&kodiert, px); err != nil {
		return // ein NRGBA im Speicher kodiert immer; ohne Bild bleibt das Ziel leer
	}
	s.WriteString(`<svg`)
	s.attr("x", zahl(o.Ziel.Min.X))
	s.attr("y", zahl(o.Ziel.Min.Y))
	s.attr("width", zahl(o.Ziel.Breite()))
	s.attr("height", zahl(o.Ziel.Hoehe()))
	s.attr("viewBox", zahl(q.Min.X)+" "+zahl(q.Min.Y)+" "+zahl(q.Breite())+" "+zahl(q.Hoehe()))
	s.attr("preserveAspectRatio", "none")
	s.WriteString(`><image`)
	s.attr("x", zahl(float64(px.Rect.Min.X)))
	s.attr("y", zahl(float64(px.Rect.Min.Y)))
	s.attr("width", zahl(float64(px.Rect.Dx())))
	s.attr("height", zahl(float64(px.Rect.Dy())))
	s.attr("preserveAspectRatio", "none")
	s.attr("style", "image-rendering:pixelated")
	s.attr("href", "data:image/png;base64,"+base64.StdEncoding.EncodeToString(kodiert.Bytes()))
	s.WriteString(`/></svg>`)
}
