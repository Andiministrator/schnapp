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

// ADR-0021 Nachtrag 7: Die Voreinstellung wächst mit der kurzen Seite des Bilds (Faktor wie die Stauch-Kante).
func TestStilFuerBildgroesse(t *testing.T) {
	for _, f := range []struct {
		g      image.Point
		faktor float64
	}{{image.Pt(1920, 1080), 1}, {image.Pt(1200, 800), 1}, {image.Pt(3840, 2160), 2}, {image.Pt(11472, 2560), 2},
		{image.Pt(7680, 4320), 4}} {
		st, std := StilFuer(f.g), StandardStil()
		if st.Breite != std.Breite*f.faktor || st.Schriftgrad != std.Schriftgrad*f.faktor || st.Radius != std.Radius*f.faktor ||
			st.Farbe != std.Farbe || st.Eckradius != std.Eckradius {
			t.Errorf("%v: %+v, Faktor %v", f.g, st, f.faktor)
		}
	}
}

// SetzeBreite ändert Werkzeug und gewählten Pfeil als einen Schritt; BreiteStufe geht von Stufe zu Stufe und hält an
// den Enden; ein Kasten zeigt seine eigene Breite.
func TestSetzeBreiteUndStufen(t *testing.T) {
	s := randSitzung(t)
	if s.Breite() != 6 {
		t.Fatalf("Voreinstellung %v", s.Breite())
	}
	if b, ok := s.BreiteStufe(1); !ok || b != 8 {
		t.Errorf("Stufe hoch von 6: %v %v", b, ok)
	}
	if b, ok := s.BreiteStufe(-1); !ok || b != 4 {
		t.Errorf("Stufe runter von 6: %v %v", b, ok)
	}
	id, err := s.D.Hinzufuegen(document.GeraderPfeil(document.Pt(10, 10), document.Pt(80, 60),
		document.Strich{Farbe: Beschriftungsfarben[0], Breite: 5}))
	if err != nil {
		t.Fatal(err)
	}
	s.Auswahl = id
	if s.Breite() != 5 {
		t.Fatalf("Breite des gewählten Pfeils %v", s.Breite())
	}
	if b, _ := s.BreiteStufe(1); b != 6 {
		t.Errorf("zwischen den Stufen hoch: %v", b)
	}
	if b, _ := s.BreiteStufe(-1); b != 4 {
		t.Errorf("zwischen den Stufen runter: %v", b)
	}
	vor := s.D.Stand()
	if !s.SetzeBreite(12) || s.Stil.Breite != 12 || s.Breite() != 12 || s.D.Stand() == vor {
		t.Fatalf("SetzeBreite 12: Stil %v, Objekt %v", s.Stil.Breite, s.Breite())
	}
	nach := s.D.Stand()
	if s.SetzeBreite(12) || s.D.Stand() != nach {
		t.Error("dieselbe Stärke ist ein Schritt")
	}
	if s.SetzeBreite(0) || s.SetzeBreite(-1) || s.Breite() != 12 {
		t.Error("ungültige Stärke angenommen")
	}
	if !s.Rueckgaengig() || s.Breite() != 5 {
		t.Errorf("Rückgängig: %v", s.Breite())
	}
	s.SetzeBreite(StrichStufen[len(StrichStufen)-1])
	if _, ok := s.BreiteStufe(1); ok {
		t.Error("über die größte Stufe hinaus")
	}
	s.SetzeBreite(StrichStufen[0])
	if _, ok := s.BreiteStufe(-1); ok {
		t.Error("unter die kleinste Stufe")
	}
	s.Auswahl = 0
	s.SetzeWerkzeug(WerkzeugKasten)
	if i := s.Seitenleiste(); !i.Strich {
		t.Error("Kasten ohne Strichstärke in der Leiste")
	}
	s.SetzeWerkzeug(WerkzeugText)
	if i := s.Seitenleiste(); i.Strich {
		t.Error("Text mit Strichstärke in der Leiste")
	}
}

// NeueSitzung nimmt die Voreinstellung nach der Größe des Originals (ADR-0021 Nachtrag 7): ein 4K-Bild beginnt mit
// Strich 12.
func TestNeueSitzungStilNachBildgroesse(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 3840, 2160)))
	if err != nil {
		t.Fatal(err)
	}
	if s := NeueSitzung(d); s.Stil != StilFuer(image.Pt(3840, 2160)) || s.Stil.Breite != 12 {
		t.Errorf("Stil %+v", s.Stil)
	}
}

// Sitzungs-QA N-7: Der Faktor ist wie bei der Kante durch die Ansicht gedeckelt: ein 4K-Original, auf 300×80
// zugeschnitten, beginnt mit der Grundeinstellung.
func TestNeueSitzungFaktorGedeckelt(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 3840, 2160)))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeZuschnitt(image.Rect(100, 100, 400, 180)); err != nil {
		t.Fatal(err)
	}
	if s := NeueSitzung(d); s.Stil != StandardStil() {
		t.Errorf("zugeschnittenes 4K-Bild: Stil %+v, soll die Grundeinstellung", s.Stil)
	}
}
