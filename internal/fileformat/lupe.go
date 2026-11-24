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
// Schwärzung den Klartext weiter. Der Export bettet die sichtbaren Pixel als PNG ein, vergrößert ohne Glätten.
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

// lupenBild schreibt das Bild im Ziel für den Export (ADR-0033 Punkt 6, Nachtrag 2 Punkte 1 und 3): ein <svg> in der
// Lage des Ziels, das auf das Ziel beschneidet, darin ein PNG mit image-rendering pixelated. Eingebettet wird nur, was
// das Ziel braucht, und davon das Kleinere: die Ansichtspixel unter der Quelle (viewBox = Quelle in der Ansicht) oder,
// bei weniger Pixeln (Vergrößerung kleiner 1), das fertige Bild auf dem Pixelraster des Ziels (viewBox = Ziel). Beide
// rechnet document.LupenQuelle, so zeigt jedes Zielpixel dasselbe wie im Renderer; Unsichtbares (vor dem Zuschnitt, im
// Spalt einer Naht) ist durchsichtig. Liegt das Ziel in keiner Kachel der Ansicht, entfällt das Bild.
func (s *schreiber) lupenBild(d *document.Dokument, o document.Lupe) {
	if !d.Sichtbar(o.Ziel) {
		return
	}
	q := d.LupenQuelle(o)
	box := q.InAnsicht()
	if !(box.Breite() > 0 && box.Hoehe() > 0) {
		return // Quelle ganz in einem Streifen: das Ziel bleibt leer
	}
	px := q.AnsichtsAusschnitt()
	z := o.Ziel
	zr := image.Rect(int(math.Floor(z.Min.X)), int(math.Floor(z.Min.Y)), int(math.Ceil(z.Max.X)), int(math.Ceil(z.Max.Y))).
		Intersect(d.Zuschnitt())
	if flaeche(zr) < flaeche(px.Rect) {
		px, box = q.Zielbild(zr), z
	}
	if px.Rect.Empty() {
		return
	}
	var kodiert bytes.Buffer
	if err := png.Encode(&kodiert, px); err != nil {
		return // ein NRGBA im Speicher kodiert immer; ohne Bild bleibt das Ziel leer
	}
	s.WriteString(`<svg`)
	s.attr("x", zahl(z.Min.X))
	s.attr("y", zahl(z.Min.Y))
	s.attr("width", zahl(z.Breite()))
	s.attr("height", zahl(z.Hoehe()))
	s.attr("viewBox", zahl(box.Min.X)+" "+zahl(box.Min.Y)+" "+zahl(box.Breite())+" "+zahl(box.Hoehe()))
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

// flaeche ist die Pixelzahl von r.
func flaeche(r image.Rectangle) int { return r.Dx() * r.Dy() }
