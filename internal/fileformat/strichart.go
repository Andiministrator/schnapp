// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"image/color"
	"strings"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Gemusterte Striche im SVG (ADR-0021 Nachtrag 25 Punkt 5): ausdrücklich als Stücke aus document, nicht über
// stroke-dasharray — die Länge einer Bézier-Kurve misst jeder Betrachter anders, die Muster lägen verschoben.

// strichNaeherung schreibt in der Arbeitsdatei (rundlauf) für einen gemusterten Strich nur eine Näherung über
// stroke-dasharray an der vollen Darstellung (F-841): Die Stücke auszuschreiben machte das SVG der Arbeitsdatei bei
// großen Mustern so groß, dass der Leser es ablehnte; der Leser braucht nur s:strichart. Der Export schreibt die Stücke
// aus (musterSVG). Gepunktet: Striche der Länge 0 mit runden Enden (rundSchon: das Element hat sie schon).
func (s *schreiber) strichNaeherung(st document.Strich, rundSchon bool) {
	if !s.rundlauf || st.Art == document.StrichVoll {
		return
	}
	strich, luecke := document.StrichMasse(st.Art, st.Breite)
	if st.Art == document.StrichGepunktet {
		s.attr("stroke-dasharray", "0 "+zahl(luecke))
		if !rundSchon {
			s.attr("stroke-linecap", "round")
		}
		return
	}
	s.attr("stroke-dasharray", zahl(strich)+" "+zahl(luecke))
}

// strichart schreibt s:strichart, wenn der Strich gemustert ist (docs/format.md 8).
func (s *schreiber) strichart(st document.Strich) {
	if st.Art != document.StrichVoll {
		s.attr("s:strichart", st.Art.Name())
	}
}

// musterSVG schreibt die Stücke m: Striche als ein Pfad mit geraden Enden und runden Gelenken (wie der Linienstrich
// des Renderers), Punkte als ein Pfad aus Kreisen vom Durchmesser der Strichstärke.
func (s *schreiber) musterSVG(m document.Strichmuster, breite float64, farbe color.NRGBA) {
	if len(m.Striche) > 0 {
		var d strings.Builder
		for _, z := range m.Striche {
			for i, p := range z {
				if i == 0 {
					d.WriteString(" M" + punkt(p))
				} else {
					d.WriteString(" L" + punkt(p))
				}
			}
		}
		s.WriteString(`<path`)
		s.attr("d", strings.TrimSpace(d.String()))
		s.attr("fill", "none")
		s.farbeSVG("stroke", farbe)
		s.attr("stroke-width", zahl(breite))
		s.attr("stroke-linecap", "butt")
		s.attr("stroke-linejoin", "round")
		s.WriteString(`/>`)
	}
	if len(m.Punkte) > 0 {
		s.WriteString(`<path`)
		s.attr("d", kreisePfad(m.Punkte, breite/2))
		s.farbeSVG("fill", farbe)
		s.WriteString(`/>`)
	}
}

// kreisePfad sind Kreise vom Radius r um die Punkte pts als ein Pfad (je Kreis zwei Halbbögen).
func kreisePfad(pts []document.Punkt, r float64) string {
	var d strings.Builder
	for i, p := range pts {
		if i > 0 {
			d.WriteString(" ")
		}
		d.WriteString("M" + punkt(document.Pt(p.X-r, p.Y)) + " a" + zahl(r) + " " + zahl(r) + " 0 1 0 " + zahl(2*r) +
			" 0 a" + zahl(r) + " " + zahl(r) + " 0 1 0 " + zahl(-2*r) + " 0 Z")
	}
	return d.String()
}

// rahmenSVG schreibt den eckigen Rahmen um r mit dem Strich st: Rechtecke als gefüllter Pfad (voll und gestrichelt,
// document.EckigerRahmenMuster, genau die Rechtecke des Renderers), Punkte als Kreise.
func (s *schreiber) rahmenSVG(r document.Rechteck, st document.Strich) {
	balken, punkte := document.EckigerRahmenMuster(r, st.Breite, st.Art)
	if len(balken) > 0 {
		s.WriteString(`<path`)
		s.attr("d", rechteckePfad(balken))
		s.farbeSVG("fill", st.Farbe)
		s.WriteString(`/>`)
	}
	if len(punkte) > 0 {
		s.WriteString(`<path`)
		s.attr("d", kreisePfad(punkte, st.Breite/2))
		s.farbeSVG("fill", st.Farbe)
		s.WriteString(`/>`)
	}
}

// rechteckePfad sind die Rechtecke rs als ein Pfad.
func rechteckePfad(rs []document.Rechteck) string {
	var pfad strings.Builder
	for i, q := range rs {
		if i > 0 {
			pfad.WriteString(" ")
		}
		pfad.WriteString("M" + punkt(q.Min) + " L" + punkt(document.Pt(q.Max.X, q.Min.Y)) + " L" + punkt(q.Max) +
			" L" + punkt(document.Pt(q.Min.X, q.Max.Y)) + " Z")
	}
	return pfad.String()
}

// kastenGemustert schreibt einen Kasten mit gemustertem Strich: eine Gruppe trägt die Daten (wie kastenOhneFlaeche, der
// Leser unterscheidet die Elemente nicht), darin die Füllung als Rechteck und der Strich in Stücken.
func (s *schreiber) kastenGemustert(o document.Kasten) {
	b := o.Bereich
	rad := document.WirksamerEckradius(o.Eckradius, b.Breite(), b.Hoehe())
	s.WriteString(`<g`)
	s.attr("s:typ", "kasten")
	s.ebene(o.Ebenenwerte())
	s.attr("s:bereich", punkt(b.Min)+" "+punkt(b.Max))
	s.attr("s:strich", strich(o.Strich))
	s.attr("s:fuellung", farbeHex(o.Fuellung))
	s.attr("s:eckradius", zahl(o.Eckradius))
	s.strichart(o.Strich)
	s.WriteString(`>`)
	if o.Fuellung.A > 0 {
		s.WriteString(`<rect`)
		s.attr("x", zahl(b.Min.X))
		s.attr("y", zahl(b.Min.Y))
		s.attr("width", zahl(b.Breite()))
		s.attr("height", zahl(b.Hoehe()))
		s.attr("rx", zahl(rad))
		s.farbeSVG("fill", o.Fuellung)
		s.WriteString(`/>`)
	}
	if o.Strich.Breite > 0 && o.Strich.Farbe.A > 0 {
		if rad == 0 {
			s.rahmenSVG(b, o.Strich)
		} else {
			s.musterSVG(document.RundRahmenAbschnitte(b, o.Eckradius, o.Strich.Art, o.Strich.Breite), o.Strich.Breite,
				o.Strich.Farbe)
		}
	}
	s.WriteString(`</g>`)
}

// pfeilGemustert schreibt die Gestalt eines Pfeils mit gemustertem Schaft (ADR-0021 Nachtrag 25 Punkt 4): der Schaft in
// Stücken wie im Renderer (render.ZerlegePfeil), die Köpfe wie beim vollen Pfeil mit Fuge (pfeilDarstellung).
// Gepunktet steht unter einem Kopf statt des letzten Punkts die halbe Scheibe hinter ihm.
func (s *schreiber) pfeilGemustert(o document.Pfeil, f document.PfeilForm) {
	hw := o.Strich.Breite / 2
	farbe := o.Strich.Farbe
	m := document.StrichAbschnitte(f.Schaft, o.Strich.Art, o.Strich.Breite)
	if len(f.Koepfe) == 0 {
		s.musterSVG(m, o.Strich.Breite, farbe)
		return
	}
	if farbe.A > 0 && farbe.A < 255 {
		s.WriteString(`<g`)
		s.attr("opacity", zahl(float64(farbe.A)/255))
		s.WriteString(`>`)
		farbe.A = 255
		defer s.WriteString(`</g>`)
	}
	schaft := f.TB > f.TA && (len(m.Striche) > 0 || len(m.Punkte) >= 2) // ein einzelner Punkt: nur die Köpfe (Renderer)
	fuge := 0.0
	if schaft {
		fuge = min(pfeilFuge, linienLaenge(f.Schaft))
		if o.Strich.Art == document.StrichGepunktet && !m.Ersatz {
			for _, k := range f.Koepfe {
				// Der Punkt unter dem Kopf entfällt, an seiner Stelle die halbe Scheibe nach hinten.
				if k.AmAnfang {
					m.Punkte = m.Punkte[1:]
				} else {
					m.Punkte = m.Punkte[:len(m.Punkte)-1]
				}
				n := k.SchaftNormale
				vorn := k.Schaftende.Plus(document.Pt(n.Y, -n.X)) // nach außen, zur Spitze hin
				s.halbeScheibe(k.Schaftende, vorn, hw, 0, farbe)
			}
		}
		s.musterSVG(m, o.Strich.Breite, farbe)
	}
	for _, k := range f.Koepfe {
		u := k.Umriss(hw)
		if fuge > 0 {
			n := k.SchaftNormale
			zurueck := document.Pt(-n.Y, n.X).Mal(fuge)
			u = append(append([]document.Punkt{u[0].Plus(zurueck)}, u...), u[len(u)-1].Plus(zurueck))
		}
		d := "M" + punkt(u[0])
		for _, q := range u[1:] {
			d += " L" + punkt(q)
		}
		s.WriteString(`<path`)
		s.attr("d", d+" Z")
		s.farbeSVG("fill", farbe)
		s.WriteString(`/>`)
	}
}
