// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"math"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0031 Nachtrag 3: Was die rechte Leiste je Werkzeug und Auswahl zeigt.
func TestSeitenleisteJeWerkzeug(t *testing.T) {
	faelle := []struct {
		w       Werkzeug
		farbe   bool
		kante   bool
		hinweis SeitenHinweis
	}{
		{WerkzeugAuswahl, false, false, SeitenHinweisNichts},
		{WerkzeugPfeil, true, false, SeitenHinweisKeiner},
		{WerkzeugKasten, true, false, SeitenHinweisKeiner},
		{WerkzeugText, true, false, SeitenHinweisKeiner},
		{WerkzeugNummer, true, false, SeitenHinweisKeiner},
		{WerkzeugSchwaerzen, false, false, SeitenHinweisEndgueltig},
		{WerkzeugVerpixeln, false, false, SeitenHinweisEndgueltig},
		{WerkzeugZuschneiden, false, false, SeitenHinweisOriginal},
		{WerkzeugStauchen, false, true, SeitenHinweisKeineNaht},
	}
	for _, f := range faelle {
		s := randSitzung(t)
		s.SetzeWerkzeug(f.w)
		i := s.Seitenleiste()
		if i.Auswahl || i.Werkzeug != f.w || i.Farbe != f.farbe || i.Kante != f.kante || i.Hinweis != f.hinweis {
			t.Errorf("Werkzeug %d: %+v", f.w, i)
		}
		if i.KanteAn {
			t.Errorf("Werkzeug %d: Kantenform ohne Naht bedienbar", f.w)
		}
	}
}

// Ein gewähltes Objekt geht dem Werkzeug vor und nennt seine Art; mit Naht wird die Kantenform bedienbar.
func TestSeitenleisteAuswahlUndNaht(t *testing.T) {
	s := randSitzung(t)
	id, err := s.D.Hinzufuegen(document.Kasten{Bereich: document.Rechteck{Min: document.Pt(5, 5), Max: document.Pt(20, 20)}})
	if err != nil {
		t.Fatal(err)
	}
	s.Auswahl = id
	if i := s.Seitenleiste(); !i.Auswahl || i.Werkzeug != WerkzeugKasten || !i.Farbe {
		t.Errorf("Auswahl Kasten: %+v", i)
	}
	s.Auswahl = 0
	if err := s.D.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 100, Bis: 120}}); err != nil {
		t.Fatal(err)
	}
	s.SetzeWerkzeug(WerkzeugStauchen)
	if i := s.Seitenleiste(); !i.Kante || !i.KanteAn || i.Hinweis != SeitenHinweisOriginal {
		t.Errorf("Stauchen mit Naht: %+v", i)
	}
}

// SetzeKante setzt genau die gewählte Form als einen Schritt; dieselbe Form noch einmal ist kein Schritt.
func TestSetzeKante(t *testing.T) {
	s := randSitzung(t)
	if s.SetzeKante(document.KanteWelle) {
		t.Fatal("ohne Naht gesetzt")
	}
	if err := s.D.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 100, Bis: 120}}); err != nil {
		t.Fatal(err)
	}
	vor := s.D.Stand()
	if !s.SetzeKante(document.KanteZacken) || s.D.Kante() != document.KanteZacken {
		t.Fatalf("Zacken nicht gesetzt: %v", s.D.Kante())
	}
	if s.D.Stand() == vor || !s.Ungesichert() {
		t.Error("kein Rückgängig-Schritt oder nicht ungesichert")
	}
	nach := s.D.Stand()
	if s.SetzeKante(document.KanteZacken) || s.D.Stand() != nach {
		t.Error("dieselbe Form erneut hat einen Schritt angelegt")
	}
	if !s.Rueckgaengig() || s.D.Kante() != document.KanteAbriss {
		t.Errorf("Rückgängig: %v", s.D.Kante())
	}
}

// SetzeZoom zoomt um die Fenstermitte und hebt das Einpassen auf; ungültige Stufen ändern nichts.
func TestSetzeZoom(t *testing.T) {
	s := randSitzung(t)
	s.SetzeFenster(image.Pt(400, 300))
	mitte := document.Pt(200, 150)
	vorher := s.Abb.ZuAnsicht(mitte)
	s.SetzeZoom(200)
	if math.Abs(s.Abb.Zoom-2) > 1e-9 || s.Eingepasst() {
		t.Fatalf("Zoom %v, eingepasst %v", s.Abb.Zoom, s.Eingepasst())
	}
	if d := s.Abb.ZuAnsicht(mitte).Minus(vorher); d.Laenge() > 1e-6 {
		t.Errorf("Mitte gewandert um %v", d)
	}
	s.SetzeZoom(0)
	s.SetzeZoom(100000)
	if math.Abs(s.Abb.Zoom-2) > 1e-9 {
		t.Errorf("ungültige Stufe hat den Zoom geändert: %v", s.Abb.Zoom)
	}
	for _, z := range ZoomStufen {
		s.SetzeZoom(z)
		if math.Abs(s.Abb.Zoom*100-float64(z)) > 1e-6 {
			t.Errorf("Stufe %d: Zoom %v", z, s.Abb.Zoom)
		}
	}
}
