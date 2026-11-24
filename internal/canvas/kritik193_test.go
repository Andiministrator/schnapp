// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"math"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Kritik Karte #193 — Erkundung, kein Produkttest.

// K193: Große Quelle (mehr als die halbe Ansicht in beiden Richtungen): Wo liegt das Ziel, ist es in der Ansicht?
func TestK193ZielGrosseQuelle(t *testing.T) {
	s := lupenSitzung(t)
	l := legeLupe(t, s, document.Pt(20, 20), document.Pt(260, 140))
	g := s.D.Groesse()
	ansicht := document.R(document.Pt(0, 0), document.Pt(float64(g.X), float64(g.Y)))
	t.Logf("Quelle %v Ziel %v Ansicht %v", l.Quelle, l.Ziel, ansicht)
	sichtbar := l.Ziel.Min.X < ansicht.Max.X && l.Ziel.Max.X > 0 && l.Ziel.Min.Y < ansicht.Max.Y && l.Ziel.Max.Y > 0
	if !sichtbar {
		t.Errorf("Ziel liegt ganz außerhalb der Ansicht: %v", l.Ziel)
	}
	// ADR-0033 Nachtrag 2 Punkt 4: Das Ziel (480 × 240) ist größer als die Ansicht (400 × 200) und rückt so weit
	// hinein, wie seine Größe es zulässt: an die linke obere Ecke.
	if l.Ziel.Min != document.Pt(0, 0) {
		t.Errorf("Ziel %v nicht in die Ansicht gerückt, Soll ab (0,0)", l.Ziel)
	}
}

// K193-6: Passt das Ziel auf keiner Seite, aber in die Ansicht, liegt es ganz in der Ansicht, mit möglichst wenig
// Überlappung mit der Quelle.
func TestK193ZielPlatzKeineSeite(t *testing.T) {
	ansicht := image.Pt(400, 200)
	q := document.R(document.Pt(150, 70), document.Pt(250, 130))
	z := ZielPlatz(q, document.Pt(200, 120), 16, ansicht)
	if z.Min.X < 0 || z.Min.Y < 0 || z.Max.X > 400 || z.Max.Y > 200 {
		t.Fatalf("Ziel %v nicht ganz in der Ansicht 400×200", z)
	}
	ueber := func(z document.Rechteck) float64 {
		w := math.Max(0, math.Min(z.Max.X, q.Max.X)-math.Max(z.Min.X, q.Min.X))
		h := math.Max(0, math.Min(z.Max.Y, q.Max.Y)-math.Max(z.Min.Y, q.Min.Y))
		return w * h
	}
	// Bester Platz: eine Ecke der Ansicht (Überlappung 50 × 50 = 2500); jede Lage in der Ansicht überlappt mindestens so.
	if got := ueber(z); got > 2500 {
		t.Errorf("Ziel %v überlappt die Quelle mit %v, möglich sind 2500", z, got)
	}
	for x := 0.0; x <= 200; x += 10 {
		for y := 0.0; y <= 80; y += 10 {
			if o := ueber(document.R(document.Pt(x, y), document.Pt(x+200, y+120))); o < ueber(z) {
				t.Fatalf("Lage (%v,%v) überlappt nur %v < %v", x, y, o, ueber(z))
			}
		}
	}
	// Passt das Ziel neben die Quelle, bleibt es dort (Punkt 8 unverändert).
	if z := ZielPlatz(document.R(document.Pt(20, 20), document.Pt(60, 50)), document.Pt(80, 60), 16, ansicht); z.Min != document.Pt(76, 20) {
		t.Errorf("rechts neben der Quelle erwartet, Ziel %v", z)
	}
}

// K193-5 (Mutant K11): Überlappen Quelle und Ziel, trifft ein Druck in die Überlappung das Ziel (es liegt oben,
// ADR-0033 Nachtrag 1 Punkt 5).
func TestK193TrefferZielVorQuelle(t *testing.T) {
	l := document.Lupe{Quelle: document.R(document.Pt(10, 10), document.Pt(50, 40)), Ziel: document.R(document.Pt(30, 20), document.Pt(110, 80))}
	if teil, ok := lupenTeil(l, document.Pt(40, 30), 0); !ok || teil != LupeZiel {
		t.Errorf("Druck in die Überlappung: Teil %v (%v), Soll LupeZiel", teil, ok)
	}
	if teil, ok := lupenTeil(l, document.Pt(15, 15), 0); !ok || teil != LupeQuelle {
		t.Errorf("Druck nur in die Quelle: Teil %v (%v), Soll LupeQuelle", teil, ok)
	}
}

// K193: Quelle über den Bildrand hinaus aufgezogen — bleibt sie im Bild?
func TestK193QuelleUeberRand(t *testing.T) {
	s := lupenSitzung(t)
	l := legeLupe(t, s, document.Pt(350, 150), document.Pt(420, 230))
	t.Logf("Quelle %v Ziel %v", l.Quelle, l.Ziel)
	if l.Quelle.Max.X > 400 || l.Quelle.Max.Y > 200 {
		t.Errorf("Quelle ragt aus dem Bild: %v", l.Quelle)
	}
}

// K193: Rückgängig je Zug — Anlegen, Ziel verschieben, Griff, Stufe: je ein Schritt.
func TestK193SchritteJeZug(t *testing.T) {
	s := lupenSitzung(t)
	l := legeLupe(t, s, document.Pt(20, 20), document.Pt(60, 50))
	s.SetzeWerkzeug(WerkzeugAuswahl)
	n0 := schritte(t, s)
	zieh(s, f(100, 50), f(130, 70)) // Ziel verschieben
	if schritte(t, s) != n0+1 {
		t.Errorf("Verschieben: %d Schritte", schritte(t, s)-n0)
	}
	o, _ := s.D.Objekt(l.ID)
	ll := o.(document.Lupe)
	// Ziel-Griff SO
	zieh(s, f(ll.Ziel.Max.X, ll.Ziel.Max.Y), f(ll.Ziel.Max.X+40, ll.Ziel.Max.Y+30))
	if schritte(t, s) != n0+2 {
		t.Errorf("Griff: %d Schritte", schritte(t, s)-n0)
	}
	o, _ = s.D.Objekt(l.ID)
	ll = o.(document.Lupe)
	kx, ky := ll.Vergroesserung()
	t.Logf("nach Griff: Ziel %v Faktor %v %v", ll.Ziel, kx, ky)
	if kx != ky {
		t.Errorf("Faktoren ungleich %v %v", kx, ky)
	}
}

// K193: Gesperrte Lupe — Stufe und Ziehen ohne Wirkung.
func TestK193Gesperrt(t *testing.T) {
	s := lupenSitzung(t)
	l := legeLupe(t, s, document.Pt(20, 20), document.Pt(60, 50))
	e := l.Ebenenwerte()
	e.Gesperrt = true
	if err := s.D.Ersetzen(document.MitEbene(l, e)); err != nil {
		t.Fatal(err)
	}
	s.Auswahl = l.ID
	if s.SetzeVergroesserung(3) {
		t.Error("gesperrte Lupe vergrößert")
	}
	if _, an := s.VergroesserungStufe(1); an {
		t.Log("Hinweis: Knopf + gilt bei gesperrter Lupe als bedienbar (wirkt aber nicht)")
	}
}

// K193: Stauchen — Ziel einer neuen Lupe, wenn rechts der Quelle ein Streifen liegt.
func TestK193ZielNebenStreifen(t *testing.T) {
	s := lupenSitzung(t)
	if err := s.D.SetzeStreifen([]document.Streifen{{Von: 70, Bis: 170}}); err != nil {
		t.Fatal(err)
	}
	s.SetzeFenster(image.Pt(448, 248))
	q := document.R(document.Pt(20, 20), document.Pt(60, 50))
	l := s.neueLupe(q)
	a, b := s.D.ZuAnsicht(l.Ziel.Min), s.D.ZuAnsicht(l.Ziel.Max)
	t.Logf("Ziel Original %v, Ansicht %v..%v (Soll Breite 80 in der Ansicht)", l.Ziel, a, b)
	if b.X-a.X < 79.5 {
		t.Errorf("Ziel in der Ansicht nur %v breit statt 80 (Teil im Streifen)", b.X-a.X)
	}
}

// K193-2: Quelle über eine Stauch-Naht aufgezogen. In der Ansicht ist sie 40 Pixel breit, im Original liegt der
// Streifen (200 Spalten) mitten darin: Wie breit wird das Ziel, und wie viel davon bleibt leer?
func TestK193QuelleUeberNaht(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 600, 200)))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]document.Streifen{{Von: 100, Bis: 300}}); err != nil {
		t.Fatal(err)
	}
	s := NeueSitzung(d)
	s.SetzeFenster(image.Pt(448, 248)) // Ansicht 400x200, Zoom 1, Versatz 24
	if s.Abb.Zoom != 1 {
		t.Fatalf("Zoom %v", s.Abb.Zoom)
	}
	s.SchalteEinrasten()
	l := legeLupe(t, s, document.Pt(80, 100), document.Pt(120, 130)) // Ansicht: 40 x 30 quer über die Naht bei 100
	a, b := s.D.ZuAnsicht(l.Ziel.Min), s.D.ZuAnsicht(l.Ziel.Max)
	// Jede Zielspalte zeigt eine Spalte der Ansicht (ADR-0033 Nachtrag 2 Punkt 1): keine Lücke für den Streifen.
	q, leer := s.D.LupenQuelle(l), 0
	for x := l.Ziel.Min.X + 0.5; x < l.Ziel.Max.X; x++ {
		if v, ok := q.Spalte(x); !ok || v < 0 || v >= s.D.Groesse().X {
			leer++
		}
	}
	t.Logf("Quelle Original %v (Breite %v, in der Ansicht 40); Ziel Original %v, Ansicht %v..%v; leere Zielspalten %d von %v",
		l.Quelle, l.Quelle.Breite(), l.Ziel, a, b, leer, l.Ziel.Breite())
	if l.Ziel.Breite() > 2*40+1 || leer > 0 {
		t.Errorf("Ziel %v breit statt 80: die 40 Pixel der Ansicht werden nicht 2x, sondern mit dem weggestauchten Streifen vergrößert; %d Spalten des Ziels sind leer",
			l.Ziel.Breite(), leer)
	}
	// Leiste und Griffe rechnen mit der Quelle in der Ansicht (ADR-0033 Nachtrag 2 Punkt 2).
	s.Auswahl = l.ID
	if k, ok := s.Vergroesserung(); !ok || k != 2 {
		t.Errorf("Vergrößerung %v (%v), Soll 2", k, ok)
	}
	z := s.zieheGriff(l, GriffZielSO, document.Pt(l.Ziel.Min.X+120, l.Ziel.Min.Y+10)).(document.Lupe) // k = 120/40 = 3
	if z.Ziel.Breite() != 120 || z.Ziel.Hoehe() != 90 {
		t.Errorf("Ziel-Griff: Ziel %v, Soll 120 × 90 (3 × Quelle in der Ansicht)", z.Ziel)
	}
	// Quell-Griff: Quelle rechts bis vor die Naht (Ansicht 80..100, 20 breit), Vergrößerung 2 bleibt: Ziel 40 × 60.
	q2 := s.zieheGriff(l, GriffQuelleSO, document.Pt(100, 130)).(document.Lupe)
	if q2.Ziel.Breite() != 40 || q2.Ziel.Hoehe() != 60 {
		t.Errorf("Quell-Griff: Quelle %v, Ziel %v, Soll 40 × 60", q2.Quelle, q2.Ziel)
	}
	if err := s.D.Ersetzen(z); err != nil {
		t.Fatal(err)
	}
	if k, _ := s.Vergroesserung(); k != 3 {
		t.Errorf("Vergrößerung nach dem Griff %v, Soll 3", k)
	}
}
