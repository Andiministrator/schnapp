// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"fmt"
	"math"
	"strings"

	"github.com/Andiministrator/schnapp/internal/document"
)

// leseSchatten liest s:schatten „WINKEL LÄNGE WEICH #RRGGBBAA“ (docs/format.md 8.5); ein unlesbarer oder ungültiger Wert
// ist kein Schatten (ADR-0021 Nachtrag 26 Punkt 6).
func leseSchatten(v string) document.Schatten {
	if len(v) > maxListeAttr { // gültig sind nur wenige Dutzend Zeichen (F-1035)
		return document.Schatten{}
	}
	teile := strings.Fields(v)
	if len(teile) != 4 {
		return document.Schatten{}
	}
	var werte [3]float64
	for i := range werte {
		f, ok := parseZahl(teile[i])
		if !ok {
			return document.Schatten{}
		}
		werte[i] = f
	}
	l := leser{a: map[string]string{"_f": teile[3]}}
	farbe := l.farbe("_f")
	if l.err != nil {
		return document.Schatten{}
	}
	s := document.Schatten{An: true, Winkel: document.NormWinkel(werte[0]), Laenge: werte[1], Weich: werte[2], Farbe: farbe}
	if !gueltigerSchatten(s) {
		return document.Schatten{}
	}
	return s
}

// gueltigerSchatten prüft die Grenzen aus document (Länge, Weich, endliche Werte).
func gueltigerSchatten(s document.Schatten) bool {
	return !math.IsNaN(s.Winkel) && !math.IsInf(s.Winkel, 0) && s.Laenge >= 0 && s.Laenge <= document.MaxSchattenLaenge &&
		s.Weich >= 0 && s.Weich <= document.MaxSchattenWeich
}

// schattenFilter schreibt vor einem Objekt mit sichtbarem Schatten dessen Filter (ADR-0021 Nachtrag 26 Punkt 5) und merkt
// die id für das filter-Attribut (ebene): Unschärfe der Deckung, Versatz, Farbe, darüber das Objekt. Die Fläche des
// Filters fasst Objekt, Versatz und Unschärfe großzügig (userSpaceOnUse, Koordinaten des Originals).
func (s *schreiber) schattenFilter(o document.Objekt) {
	sch := o.Ebenenwerte().Schatten
	if !sch.Sichtbar() {
		return
	}
	s.schattenNr++
	s.schattenID = fmt.Sprintf("schatten-%d", s.schattenNr)
	g := o.Grenzen()
	switch v := o.(type) {
	case document.Pfeil:
		g = g.Vereint(v.Form().Huelle(v.Strich.Breite))
	case document.Text:
		if v.Spitze != nil {
			g = g.Vereint(v.Huelle())
		}
		// Schreibt der Export nur das Dreieck (Schild verborgen, F-692), darf die Fläche des Filters die Maße des
		// Schilds nicht verraten (F-842, wie F-728): nur das Dreieck mit den Schenkeln.
		if s.ohneText != nil && s.ohneText(v) {
			if a, sp, b, ok := v.Spitzendreieck(); ok {
				g = document.R(a, sp).Vereint(document.R(sp, b))
				g = document.Rechteck{Min: g.Min.Minus(document.Pt(v.SchenkelBreite(), v.SchenkelBreite())),
					Max: g.Max.Plus(document.Pt(v.SchenkelBreite(), v.SchenkelBreite()))}
			}
		}
	}
	rand := sch.Rand() + 8
	v := sch.Versatz()
	v = document.Pt(math.Round(v.X*1e6)/1e6, math.Round(v.Y*1e6)/1e6) // kein 3.67e-16 aus cos(180°)
	s.WriteString(`<filter`)
	s.WriteString(` id="` + s.schattenID + `"`)
	s.WriteString(` filterUnits="userSpaceOnUse" color-interpolation-filters="sRGB"`)
	s.WriteString(fmt.Sprintf(` x="%s" y="%s" width="%s" height="%s">`, zahl(g.Min.X-rand), zahl(g.Min.Y-rand),
		zahl(g.Breite()+2*rand), zahl(g.Hoehe()+2*rand)))
	// Ohne Unschärfe kein feGaussianBlur: stdDeviation 0 legen ältere Fassungen der Spezifikation anders aus
	// (abgeschaltetes Primitiv statt durchgereichter Eingabe); der Versatz liest dann die Deckung direkt.
	quelle := ""
	if sch.Weich > 0 {
		s.WriteString(`<feGaussianBlur in="SourceAlpha" stdDeviation="` + zahl(sch.Weich/2) + `"/>`)
	} else {
		quelle = ` in="SourceAlpha"`
	}
	s.WriteString(`<feOffset` + quelle + ` dx="` + zahl(v.X) + `" dy="` + zahl(v.Y) + `" result="schatten"/>`)
	s.WriteString(fmt.Sprintf(`<feFlood flood-color="#%02x%02x%02x" flood-opacity="%s"/>`, sch.Farbe.R, sch.Farbe.G,
		sch.Farbe.B, zahl(float64(sch.Farbe.A)/255)))
	s.WriteString(`<feComposite in2="schatten" operator="in"/>`)
	s.WriteString(`<feMerge><feMergeNode/><feMergeNode in="SourceGraphic"/></feMerge></filter>`)
}
