// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image/color"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// F-748, F-749 (ADR-0023 Nachtrag 17): Pfeil und runder Kasten über einer Naht sind abgeschnitten wie im SVG-Export:
// außerhalb des Kantenbands gleicht das gestauchte Bild dem ungestauchten. Vorher bildete der Renderer die 48
// Stützpunkte bzw. die Ecken ab und zog dazwischen neu (Sitzungs-Kritik 25.11. A-4, A-5, C-4).

func TestPfeilUeberNahtWieAbgeschnitten(t *testing.T) {
	rot := color.NRGBA{R: 200, A: 255}
	for _, breite := range []float64{4, 12, 24, 32, 48} {
		for _, fall := range []struct {
			name     string
			a, b     document.Punkt
			von, bis int
		}{
			{"flach, Streifen 150-190", document.Pt(20, 40), document.Pt(380, 160), 150, 190},
			{"schräg, Streifen 100-300", document.Pt(20, 10), document.Pt(380, 190), 100, 300},
			{"Spitze im Streifen", document.Pt(20, 40), document.Pt(240, 120), 150, 190},
		} {
			o := document.GeraderPfeil(fall.a, fall.b, document.Strich{Farbe: rot, Breite: breite})
			links, rechts, _ := vergleicheUmNaht(t, o, fall.von, fall.bis, rot)
			if links != 0 || rechts != 0 {
				t.Errorf("Breite %v, %s: abweichend links %d, rechts %d (Soll 0)", breite, fall.name, links, rechts)
			}
		}
	}
}

// Langer Pfeil: 48 Stützpunkte liegen weiter auseinander als das Kantenband breit ist.
func TestLangerPfeilUeberNahtWieAbgeschnitten(t *testing.T) {
	baue := func() *document.Dokument {
		d := weissesDokument(t, 3200, 3200)
		hinzu(t, d, document.GeraderPfeil(document.Pt(20, 20), document.Pt(3100, 3100), document.Strich{Farbe: qaSchwarz, Breite: 6}))
		return d
	}
	voll, err := Bild(baue())
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range [][2]int{{1000, 1040}, {1030, 1070}, {1040, 1090}} {
		d := baue()
		if err := d.SetzeStreifen([]document.Streifen{{Von: s[0], Bis: s[1]}}); err != nil {
			t.Fatal(err)
		}
		g, err := Bild(d)
		if err != nil {
			t.Fatal(err)
		}
		band, breite := int(document.KantenBandHalb)*d.KantenFaktor()+4, s[1]-s[0]
		l, r := 0, 0
		for y := 0; y < 3200; y++ {
			for x := s[0] - 200; x < s[0]-band; x++ {
				if !fastGleich(g.NRGBAAt(x, y), voll.NRGBAAt(x, y)) {
					l++
				}
			}
			for x := s[0] + band; x < s[0]+200; x++ {
				if !fastGleich(g.NRGBAAt(x, y), voll.NRGBAAt(x+breite, y)) {
					r++
				}
			}
		}
		if l+r != 0 {
			t.Errorf("Streifen %v: abweichend links %d, rechts %d (Soll 0)", s, l, r)
		}
	}
}

func TestRunderKastenUeberNahtWieAbgeschnitten(t *testing.T) {
	gruen := color.NRGBA{G: 160, A: 255}
	for _, fall := range []struct {
		name     string
		bereich  document.Rechteck
		von, bis int
	}{
		{"rechte Kante 25 px hinter dem Streifen", document.R(document.Pt(40, 30), document.Pt(215, 170)), 150, 190},
		{"rechte Rundung ganz im Streifen", document.R(document.Pt(40, 30), document.Pt(170, 170)), 110, 190},
	} {
		k := document.Kasten{Bereich: fall.bereich, Eckradius: 60, Strich: document.Strich{Farbe: gruen, Breite: 4}, Fuellung: gruen}
		links, rechts, _ := vergleicheUmNaht(t, k, fall.von, fall.bis, gruen)
		if links != 0 || rechts != 0 {
			t.Errorf("%s: abweichend links %d, rechts %d (Soll 0)", fall.name, links, rechts)
		}
	}
}

// Pruefe bleibt über vielen Nähten schnell, obwohl der Pfeil je Bereich zeichnet: Jeder Bereich baut nur die Strecken in
// seiner Nähe (document.StreckenStuecke), nicht den ganzen Pfeil (F-136). Sollwert: die Zahl der Füllungen, also der
// berührten Bereiche plus die Spitze, nicht Nähte mal Strecken.
func TestPfeilUeberVielenNaehtenNurBeruehrteBereiche(t *testing.T) {
	d := weissesDokument(t, 4200, 520)
	if err := d.SetzeStreifen(spaltenStreifen(1023, 4)); err != nil {
		t.Fatal(err)
	}
	hinzu(t, d, document.GeraderPfeil(document.Pt(5, 5), document.Pt(4100, 500), document.Strich{Farbe: qaSchwarz, Breite: 6}))
	pts := make([]document.Punkt, 0, 49)
	p := d.Objekte()[0].(document.Pfeil)
	for i := 0; i <= 48; i++ {
		pts = append(pts, p.Punkt(float64(i)/48))
	}
	ss := d.StreckenStuecke(pts, false, 4)
	if len(ss) < 1000 || len(ss) > 1030 {
		t.Errorf("%d Bereiche mit Strecken, Soll rund 1024 (einer je Spalte)", len(ss))
	}
	n := 0
	for _, s := range ss {
		n += len(s.Strecken)
	}
	if n > 2*len(ss)+48 {
		t.Errorf("%d Strecken über alle Bereiche, Soll höchstens %d (je Bereich die in seiner Nähe)", n, 2*len(ss)+48)
	}
}
