// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// F-1051: Ist unter der Gesamtgrenze kein Platz mehr für einen ganzen Freihand-Strich, lehnen Freihand und freier
// Textmarker den Druck mit einer eigenen Ablehnung ab; der gerade Textmarker und andere Werkzeuge zeichnen weiter.
func TestFreihandGesamtgrenzeImWerkzeug(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 200, 100)))
	if err != nil {
		t.Fatal(err)
	}
	pts := make([]document.Punkt, document.MaxFreihandPunkte)
	for i := range pts {
		pts[i] = document.Pt(float64(i%100), float64(i%2))
	}
	var os []document.Objekt
	for range document.MaxFreihandGesamt/document.MaxFreihandPunkte - 1 {
		os = append(os, document.NeuerFreihand(pts, document.Strich{Breite: 2}))
	}
	if _, err := d.HinzufuegenMehrere(os); err != nil {
		t.Fatal(err)
	}
	if p := d.FreihandPlatz(); p != document.MaxFreihandPunkte {
		t.Fatalf("Platz %d", p)
	}
	s := NeueSitzung(d)
	s.SetzeFenster(image.Pt(248, 148))
	s.SetzeWerkzeug(WerkzeugFreihand)
	if a := s.Voll(); a != AblehnungKeine {
		t.Fatalf("Platz für einen ganzen Strich: %v", a)
	}
	// Zwei Punkte mehr belegen: kein Platz für einen ganzen Strich.
	if _, err := d.Hinzufuegen(document.NeuerFreihand([]document.Punkt{document.Pt(1, 1), document.Pt(2, 2)},
		document.Strich{Breite: 2})); err != nil {
		t.Fatal(err)
	}
	if a := s.Voll(); a != AblehnungFreihand {
		t.Errorf("Freihand: %v, Soll AblehnungFreihand", a)
	}
	n := len(d.Objekte())
	s.Druecken(document.Pt(60, 60), TastePrimaer)
	s.Ziehen(document.Pt(90, 70))
	s.Loslassen(document.Pt(90, 70))
	if s.Abgelehnt() != AblehnungFreihand || len(d.Objekte()) != n {
		t.Errorf("Druck mit Freihand: Ablehnung %v, Objekte %d (vorher %d)", s.Abgelehnt(), len(d.Objekte()), n)
	}
	s.SetzeWerkzeug(WerkzeugTextmarker)
	if s.TextmarkerFrei() {
		s.SchalteTextmarkerFrei()
	}
	if a := s.Voll(); a != AblehnungKeine {
		t.Errorf("gerader Textmarker: %v", a)
	}
	s.SchalteTextmarkerFrei()
	if a := s.Voll(); a != AblehnungFreihand {
		t.Errorf("freier Textmarker: %v", a)
	}
	s.SetzeWerkzeug(WerkzeugPfeil)
	if a := s.Voll(); a != AblehnungKeine {
		t.Errorf("Pfeil: %v", a)
	}
}
