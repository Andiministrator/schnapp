// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image/color"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/settings"
)

// ADR-0047: Das Werkzeug Ellipse legt einen Kasten mit Form Ellipse an (Umschalt: Kreis); die Leiste, die Schnellstile
// und der Stil je Werkzeug kennen es als eigenes Werkzeug; der Treffer folgt dem Oval.

func TestEllipseAnlegen(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugEllipse)
	zieh(s, f(90, 80), f(60, 60))
	k, ok := letztes(t, s).(document.Kasten)
	if !ok || k.Form != document.FormEllipse || k.Bereich != document.R(document.Pt(60, 60), document.Pt(90, 80)) {
		t.Fatalf("Ellipse %+v", letztes(t, s))
	}
	if k.Fuellung.A != 0 || k.Strich.Breite != 6 || k.Strich.Farbe != Beschriftungsfarben[0] {
		t.Errorf("Stil %+v", k)
	}
	s.Umschalt = true
	zieh(s, f(120, 60), f(140, 70))
	if k = letztes(t, s).(document.Kasten); k.Form != document.FormEllipse ||
		k.Bereich != document.R(document.Pt(120, 60), document.Pt(140, 80)) {
		t.Errorf("Kreis %+v", k)
	}
	// Der Kasten bleibt ein Rechteck.
	s.Umschalt = false
	s.SetzeWerkzeug(WerkzeugKasten)
	zieh(s, f(150, 20), f(190, 40))
	if k = letztes(t, s).(document.Kasten); k.Form != document.FormRechteck {
		t.Errorf("Kasten mit Form %v", k.Form)
	}
}

func TestEllipseInLeisteUndSchnellstilen(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugEllipse)
	zieh(s, f(60, 60), f(90, 80))
	id := letztes(t, s).Kennung()
	s.SetzeWerkzeug(WerkzeugAuswahl)
	s.waehleNur(id)
	if w, ok := objektWerkzeug(letztes(t, s)); !ok || w != WerkzeugEllipse {
		t.Errorf("objektWerkzeug %v", w)
	}
	if !strichWerkzeug(WerkzeugEllipse) || !fuellWerkzeug(WerkzeugEllipse) || !farbWerkzeug(WerkzeugEllipse) {
		t.Error("Ellipse ohne Strich, Füllung oder Farbe")
	}
	if a, ok := werkzeugArt(WerkzeugEllipse); !ok || a != SchnellEllipse {
		t.Errorf("Art %v", a)
	}
	if w, ok := schnellWerkzeug(SchnellEllipse); !ok || w != WerkzeugEllipse {
		t.Errorf("Werkzeug der Art %v", w)
	}
	ell, kas := EingebauteSchnellstile(SchnellEllipse), EingebauteSchnellstile(SchnellKasten)
	if len(ell) != len(kas) || len(ell) == 0 {
		t.Fatalf("eingebaute Stile %d, Kasten %d", len(ell), len(kas))
	}
	for i := range ell {
		k := kas[i]
		k.Art = SchnellEllipse
		if ell[i] != k {
			t.Errorf("Stil %d: %+v, Kasten %+v", i, ell[i], kas[i])
		}
	}
	if st, ok := objektStandOhneSchatten(letztes(t, s)); !ok || st.Art != SchnellEllipse {
		t.Errorf("Stand der Ellipse: %+v", st)
	}
	if werkzeugAbschnitt(WerkzeugEllipse) != settings.ArtEllipse || werkzeugAbschnitt(WerkzeugKasten) != settings.ArtKasten {
		t.Error("Abschnitte in tools.ini")
	}
}

// Der Stil der Ellipse ist ein eigener: Eine andere Breite am Werkzeug Ellipse ändert den Kasten nicht (ADR-0041).
func TestEllipseEigenerStil(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugEllipse)
	s.SetzeBreite(20)
	s.SetzeWerkzeug(WerkzeugKasten)
	if s.Stil.Breite == 20 {
		t.Error("Breite der Ellipse am Kasten")
	}
	s.SetzeWerkzeug(WerkzeugEllipse)
	if s.Stil.Breite != 20 {
		t.Errorf("Breite der Ellipse %v", s.Stil.Breite)
	}
}

func TestEllipseTreffer(t *testing.T) {
	r := document.R(document.Pt(0, 0), document.Pt(100, 60))
	leer := document.Kasten{Bereich: r, Form: document.FormEllipse, Strich: document.Strich{Breite: 4, Farbe: rot.Farbe}}
	voll := leer
	voll.Fuellung = color.NRGBA{A: 255}
	for _, fall := range []struct {
		p          document.Punkt
		leer, voll bool
	}{
		{document.Pt(50, 30), false, true},  // Mitte
		{document.Pt(100, 30), true, true},  // rechter Scheitel auf dem Strich
		{document.Pt(50, 0), true, true},    // oberer Scheitel
		{document.Pt(3, 3), false, false},   // Ecke des Bereichs: außerhalb des Ovals
		{document.Pt(97, 57), false, false}, // Ecke
		{document.Pt(104, 30), true, true},  // knapp außen, innerhalb halber Strich + Toleranz
		{document.Pt(110, 30), false, false},
	} {
		if g := trifft(leer, fall.p, 2); g != fall.leer {
			t.Errorf("ohne Füllung %v: %v, Soll %v", fall.p, g, fall.leer)
		}
		if g := trifft(voll, fall.p, 2); g != fall.voll {
			t.Errorf("mit Füllung %v: %v, Soll %v", fall.p, g, fall.voll)
		}
	}
}
