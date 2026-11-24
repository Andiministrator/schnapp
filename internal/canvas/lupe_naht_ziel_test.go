// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

// Sitzungs-QA vom 24.11.2026 (Kritiker Fable), QS-3: Das Ziel einer neuen Lupe fiel über eine Stauch-Naht; im Original
// reichte es dann in den weggestauchten Streifen, und in der Ansicht fehlte der Teil hinter der Naht (ADR-0033
// Nachtrag 5 Punkt 3: das Ziel einer neuen Lupe kreuzt keine Naht).

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/render"
)

// nahtSitzung: Bild 600×200, jede Spalte eigene Farbe, Streifen Spalten 100–300 gestaucht → Ansicht 400×200, Naht bei 100.
func nahtSitzung(t *testing.T) *Sitzung {
	t.Helper()
	b := image.NewNRGBA(image.Rect(0, 0, 600, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 600; x++ {
			b.SetNRGBA(x, y, color.NRGBA{uint8(x), uint8(x >> 8), uint8(y), 255})
		}
	}
	d, err := document.Neu(b)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]document.Streifen{{Von: 100, Bis: 300}}); err != nil {
		t.Fatal(err)
	}
	s := NeueSitzung(d)
	s.SetzeFenster(image.Pt(448, 248))
	if s.Abb.Zoom != 1 {
		t.Fatalf("Abbildung %+v", s.Abb)
	}
	s.SchalteEinrasten()
	return s
}

// zielZeigtQuelle prüft, dass das Ziel von l in der Ansicht 2 × die Quelle in der Ansicht groß ist und alle Quellspalten
// zeigt (in der mittleren Zeile des Ziels).
func zielZeigtQuelle(t *testing.T, s *Sitzung, l document.Lupe, von, bis int) {
	t.Helper()
	za, zb := s.D.ZuAnsicht(l.Ziel.Min), s.D.ZuAnsicht(l.Ziel.Max)
	qa, qb := s.D.ZuAnsicht(l.Quelle.Min), s.D.ZuAnsicht(l.Quelle.Max)
	if w, soll := zb.X-za.X, 2*(qb.X-qa.X); w < soll-1e-9 || w > soll+1e-9 {
		t.Errorf("Ziel in der Ansicht x %v..%v, Breite %v statt %v (Ziel im Original %v)", za.X, zb.X, w, soll, l.Ziel)
	}
	if h, soll := zb.Y-za.Y, 2*(qb.Y-qa.Y); h < soll-1e-9 || h > soll+1e-9 {
		t.Errorf("Ziel in der Ansicht Höhe %v statt %v", h, soll)
	}
	inhalt := render.LupenInhalt(s.D, l)
	gezeigt := map[int]bool{}
	if inhalt != nil {
		y := inhalt.Rect.Min.Y + inhalt.Rect.Dy()/2
		for x := inhalt.Rect.Min.X; x < inhalt.Rect.Max.X; x++ {
			if c := inhalt.NRGBAAt(x, y); c.A != 0 {
				gezeigt[int(c.R)|int(c.G)<<8] = true
			}
		}
	}
	fehlen := 0
	for x := von; x < bis; x++ {
		if !gezeigt[x] {
			fehlen++
		}
	}
	if fehlen > 0 {
		t.Errorf("von %d Quellspalten fehlen %d im Ziel (Ziel %v)", bis-von, fehlen, l.Ziel)
	}
}

// TestQSZielUeberNaht: Quelle links der Naht, das Ziel rechts im Abstand 16 fiele über die Naht (Ansicht 76..156). Das
// Ziel einer neuen Lupe kreuzt keine Naht: Es ist in der Ansicht 2 × die Quelle und zeigt alle Quellspalten.
func TestQSZielUeberNaht(t *testing.T) {
	s := nahtSitzung(t)
	l := legeLupe(t, s, document.Pt(20, 100), document.Pt(60, 130)) // Ansicht = Original (links der Naht)
	zielZeigtQuelle(t, s, l, 20, 60)
	za, zb := s.D.ZuAnsicht(l.Ziel.Min), s.D.ZuAnsicht(l.Ziel.Max)
	if za.X < 100 && zb.X > 100 {
		t.Errorf("Ziel kreuzt die Naht bei 100: Ansicht x %v..%v", za.X, zb.X)
	}
}

// TestQSZielUeberNahtZeilen: dasselbe mit einer waagerechten Naht (Zeilen 60–160 gestaucht, Naht bei Ansicht y 60): Die
// Quelle liegt oben links, rechts daneben fiele das Ziel über die Naht.
func TestQSZielUeberNahtZeilen(t *testing.T) {
	b := image.NewNRGBA(image.Rect(0, 0, 400, 300))
	for y := 0; y < 300; y++ {
		for x := 0; x < 400; x++ {
			b.SetNRGBA(x, y, color.NRGBA{uint8(x), uint8(x >> 8), uint8(y), 255})
		}
	}
	d, err := document.Neu(b)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 60, Bis: 160}}); err != nil {
		t.Fatal(err)
	}
	s := NeueSitzung(d)
	s.SetzeFenster(image.Pt(448, 248))
	s.SchalteEinrasten()
	l := legeLupe(t, s, document.Pt(20, 20), document.Pt(60, 50))
	zielZeigtQuelle(t, s, l, 20, 60)
	za, zb := s.D.ZuAnsicht(l.Ziel.Min), s.D.ZuAnsicht(l.Ziel.Max)
	if za.Y < 60 && zb.Y > 60 {
		t.Errorf("Ziel kreuzt die Naht bei y 60: Ansicht y %v..%v", za.Y, zb.Y)
	}
}

// TestZielPlatzMeidetNaehte: ZielPlatz lässt Plätze aus, deren Ziel eine Naht kreuzt; ein Ziel darf an einer Naht enden
// oder beginnen. Gibt es keinen Platz ohne Naht, gilt die Regel ohne Nähte (Notfall).
func TestZielPlatzMeidetNaehte(t *testing.T) {
	ansicht := image.Pt(400, 200)
	q := document.R(document.Pt(20, 100), document.Pt(60, 130))
	senk := []document.Naht{{Lage: 100}}
	// rechts (76..156) kreuzt, links passt nicht, darunter (146..206) passt nicht, darüber (24..84) bei x 20..100: endet an der Naht
	if z := ZielPlatz(q, document.Pt(80, 60), 16, ansicht, senk); z != document.R(document.Pt(20, 24), document.Pt(100, 84)) {
		t.Errorf("Ziel %v, erwartet darüber (20,24)–(100,84)", z)
	}
	// ohne Nähte bleibt es rechts
	if z := ZielPlatz(q, document.Pt(80, 60), 16, ansicht, nil); z.Min != document.Pt(76, 100) {
		t.Errorf("ohne Naht: Ziel %v, erwartet rechts ab (76,100)", z)
	}
	// eine waagerechte Naht bei y 120 trifft das Ziel rechts (y 100..160): darüber (24..84) kreuzt sie nicht
	if z := ZielPlatz(q, document.Pt(80, 60), 16, ansicht, []document.Naht{{Zeilen: true, Lage: 120}}); z.Min != document.Pt(20, 24) {
		t.Errorf("waagerechte Naht: Ziel %v, erwartet darüber ab (20,24)", z)
	}
	// Keine Seite passt (Ziel 200×120 um eine Quelle in der Mitte): unter den gleich guten Ecken gewinnt eine ohne Naht.
	mitte := document.R(document.Pt(150, 70), document.Pt(250, 130))
	z0 := ZielPlatz(mitte, document.Pt(200, 120), 16, ansicht, []document.Naht{{Lage: 300}})
	if z0.Min.X < 300 && z0.Max.X > 300 {
		t.Errorf("ohne Seite: Ziel %v kreuzt die Naht bei 300", z0)
	}
	if w, hh := math.Min(z0.Max.X, 250)-math.Max(z0.Min.X, 150), math.Min(z0.Max.Y, 130)-math.Max(z0.Min.Y, 70); w*hh > 2500 {
		t.Errorf("ohne Seite: Ziel %v überlappt die Quelle mit %v, möglich 2500", z0, w*hh)
	}
	// Nähte überall (alle 30 Pixel in beiden Richtungen): kein Platz ohne Naht, das Ziel liegt trotzdem in der Ansicht
	var dicht []document.Naht
	for v := 30; v < 400; v += 30 {
		dicht = append(dicht, document.Naht{Lage: v}, document.Naht{Zeilen: true, Lage: v})
	}
	z := ZielPlatz(q, document.Pt(80, 60), 16, ansicht, dicht)
	if z.Min.X < 0 || z.Min.Y < 0 || z.Max.X > 400 || z.Max.Y > 200 || z.Breite() != 80 || z.Hoehe() != 60 {
		t.Errorf("dichte Nähte: Ziel %v nicht 80×60 in der Ansicht", z)
	}
}

// TestQSZielUeberNahtGezogen: Wird das Ziel später über eine Naht gezogen, verhält es sich wie jedes Objekt (ADR-0033
// Nachtrag 5 Punkt 3): Im Original bleibt es 80 breit, in der Ansicht fällt der Teil im Streifen weg. Festgehalten, damit
// eine Änderung dieser Entscheidung bewusst geschieht.
func TestQSZielUeberNahtGezogen(t *testing.T) {
	s := nahtSitzung(t)
	l := legeLupe(t, s, document.Pt(150, 20), document.Pt(190, 50)) // Ansicht rechts der Naht
	s.SetzeWerkzeug(WerkzeugAuswahl)
	za := s.D.ZuAnsicht(l.Ziel.Min)
	zieh(s, f(za.X+10, za.Y+10), f(za.X+10-(za.X-60), za.Y+10+60)) // Ziel so ziehen, dass es bei Ansicht x = 60 beginnt
	o, _ := s.D.Objekt(l.ID)
	l = o.(document.Lupe)
	a, b := s.D.ZuAnsicht(l.Ziel.Min), s.D.ZuAnsicht(l.Ziel.Max)
	t.Logf("nach dem Ziehen: Ziel im Original %v, in der Ansicht x %v..%v", l.Ziel, a.X, b.X)
	if l.Ziel.Breite() != 80 {
		t.Errorf("Ziehen änderte die Größe im Original: %v", l.Ziel)
	}
	if a.X != 60 || b.X != 100 {
		t.Errorf("gezogenes Ziel in der Ansicht x %v..%v, erwartet 60..100 (Rest im Streifen)", a.X, b.X)
	}
}
