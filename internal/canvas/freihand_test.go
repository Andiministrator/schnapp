// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"math"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/settings"
)

// ziehZug drückt bei pts[0], zieht über alle Punkte und lässt am letzten los.
func ziehZug(s *Sitzung, pts ...document.Punkt) {
	s.Druecken(pts[0], TastePrimaer)
	for _, p := range pts[1:] {
		s.Ziehen(p)
	}
	s.Loslassen(pts[len(pts)-1])
}

// bogen liefert Fensterpunkte eines Halbkreises um (100, 60) mit Radius 30, n Schritte.
func bogen(n int) []document.Punkt {
	var r []document.Punkt
	for i := 0; i <= n; i++ {
		w := math.Pi * float64(i) / float64(n)
		r = append(r, f(100+30*math.Cos(w), 60-30*math.Sin(w)))
	}
	return r
}

// ADR-0049 Punkte 3 und 4: ein freier Zug wird ein Freihand-Strich, beim Loslassen vereinfacht und geglättet;
// Umschalt zieht eine gerade Strecke; ein zu kurzer Zug bleibt nicht liegen.
func TestFreihandZiehen(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugFreihand)
	zug := bogen(200)
	ziehZug(s, zug...)
	fh, ok := letztes(t, s).(document.Freihand)
	if !ok {
		t.Fatalf("kein Freihand-Strich: %+v", letztes(t, s))
	}
	pts := fh.Punkte()
	// Roh rund 48 Punkte (alle 2 Pixel), vereinfacht rund 16, zweimal geglättet 58 (gemessen 28.11.2026): ohne
	// Glätten oder ohne Vereinfachen fiele die Zahl aus dem Band.
	if len(pts) < 52 || len(pts) > 64 {
		t.Errorf("Punkte nach dem Glätten: %d, Soll 52 bis 64", len(pts))
	}
	if pts[0] != s.original(zug[0]) || pts[len(pts)-1] != s.original(zug[len(zug)-1]) {
		t.Errorf("Anfang/Ende %v %v", pts[0], pts[len(pts)-1])
	}
	for _, p := range pts { // auf dem Bogen, höchstens wenig innen (Chaikin schneidet Ecken)
		if r := p.Minus(document.Pt(100, 60)).Laenge(); r > 30.01 || r < 29 {
			t.Errorf("Punkt %v mit Radius %.2f", p, r)
			break
		}
	}
	if fh.Multipliziert() || fh.Strich.Breite != 6 || s.Auswahl != fh.ID {
		t.Errorf("Stil %+v, Auswahl %d", fh, s.Auswahl)
	}
	n := len(s.D.Objekte())
	s.Umschalt = true
	ziehZug(s, f(20, 20), f(40, 70), f(60, 30))
	if g := letztes(t, s).(document.Freihand); len(g.Punkte()) != 2 || g.Punkte()[1] != document.Pt(60, 30) {
		t.Errorf("Umschalt: %v", g.Punkte())
	}
	s.Umschalt = false
	ziehZug(s, f(150, 80), f(152, 80), f(154, 81)) // über der Ziehschwelle, unter MinObjekt (Länge rund 4)
	if len(s.D.Objekte()) != n+1 {
		t.Error("zu kurzer Zug blieb liegen")
	}
}

// ADR-0049 Punkt 5: Mit „Freehand“ zieht der Textmarker einen freien multiplizierenden Strich, Umschalt waagerecht;
// er gilt als Textmarker. Der Schalter gehört zum Stil des Textmarkers.
func TestTextmarkerFrei(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugTextmarker)
	if s.TextmarkerFrei() || !s.Seitenleiste().Freihand {
		t.Fatal("Voreinstellung oder Kästchen")
	}
	if !s.SchalteTextmarkerFrei() || !s.TextmarkerFrei() {
		t.Fatal("Schalter wirkt nicht")
	}
	ziehZug(s, bogen(60)...)
	fh, ok := letztes(t, s).(document.Freihand)
	if !ok || !fh.Multipliziert() || fh.Strich.Breite != TextmarkerBreite {
		t.Fatalf("freier Textmarker %+v", letztes(t, s))
	}
	if w, _ := objektWerkzeug(fh); w != WerkzeugTextmarker {
		t.Errorf("objektWerkzeug %v", w)
	}
	if st, _ := objektStandOhneSchatten(fh); st.Art != SchnellTextmarker {
		t.Errorf("Schnellstil-Art %v", st.Art)
	}
	s.Umschalt = true
	ziehZug(s, f(20, 20), f(60, 50))
	if g := letztes(t, s).(document.Freihand); len(g.Punkte()) != 2 || g.Punkte()[0].Y != g.Punkte()[1].Y {
		t.Errorf("Umschalt: %v", g.Punkte())
	}
	s.SetzeWerkzeug(WerkzeugFreihand)
	if s.Seitenleiste().Freihand {
		t.Error("Kästchen beim Werkzeug Freihand")
	}
	// tools.ini trägt den Schalter beim Textmarker.
	var gefunden bool
	for _, e := range s.WerkzeugStile() {
		if e.Art == settings.ArtTextmarker {
			gefunden = e.Freihand
		}
	}
	if !gefunden {
		t.Error("freehand nicht in den Werkzeugstilen")
	}
	s2, _, _ := sitzung(t)
	s2.SetzeWerkzeugStile(s.WerkzeugStile())
	s2.SetzeWerkzeug(WerkzeugTextmarker)
	if !s2.TextmarkerFrei() {
		t.Error("Schalter nicht übernommen")
	}
}

// Verschieben, Treffer, Strich-Setter, Schnellstile und Stil je Werkzeug kennen den Freihand-Strich.
func TestFreihandBearbeiten(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugFreihand)
	ziehZug(s, f(20, 80), f(40, 90), f(60, 80), f(80, 90))
	fh := letztes(t, s).(document.Freihand)
	if !trifft(fh, document.Pt(40, 90), 1) || trifft(fh, document.Pt(40, 60), 1) {
		t.Error("Treffer")
	}
	v := Verschiebe(fh, document.Pt(5, -3)).(document.Freihand)
	if v.Punkte()[0] != fh.Punkte()[0].Plus(document.Pt(5, -3)) || fh.Punkte()[0] != document.Pt(20, 80) {
		t.Errorf("Verschiebe %v / %v", v.Punkte()[0], fh.Punkte()[0])
	}
	s.SetzeWerkzeug(WerkzeugAuswahl)
	s.waehleNur(fh.ID)
	if !s.SetzeBreite(14) || s.Breite() != 14 || !s.SetzeStrichart(document.StrichGepunktet) ||
		s.Strichart() != document.StrichGepunktet {
		t.Error("Strich-Setter")
	}
	if !s.SetzeFarbe(TextmarkerRosa) || letztes(t, s).(document.Freihand).Strich.Farbe != TextmarkerRosa {
		t.Error("Farbe")
	}
	ein := EingebauteSchnellstile(SchnellFreihand)
	if len(ein) != len(EingebauteSchnellstile(SchnellPfeil)) || ein[0].Art != SchnellFreihand {
		t.Fatalf("eingebaute Stile %+v", ein)
	}
	if !s.WendeSchnellstilAn(ein[1]) || letztes(t, s).(document.Freihand).Strich.Farbe != ein[1].Farbe {
		t.Error("Schnellstil")
	}
	if werkzeugAbschnitt(WerkzeugFreihand) != settings.ArtFreihand {
		t.Error("Abschnitt in tools.ini")
	}
	if w, _ := objektWerkzeug(fh); w != WerkzeugFreihand || !strichWerkzeug(w) || fuellWerkzeug(w) {
		t.Errorf("Werkzeug %v", w)
	}
}
