// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

func arten(f []FussFeld) []FeldArt {
	var a []FeldArt
	for _, x := range f {
		a = append(a, x.Art)
	}
	return a
}

func feld(t *testing.T, f []FussFeld, a FeldArt) FussFeld {
	t.Helper()
	for _, x := range f {
		if x.Art == a {
			return x
		}
	}
	t.Fatalf("Feld %d fehlt in %v", a, arten(f))
	return FussFeld{}
}

// TestFussFelder: Reihenfolge und Inhalt der Felder (ADR-0031 Punkt 4); Auswahl, Zeiger und Schrift nur, wenn sie
// etwas zu sagen haben.
func TestFussFelder(t *testing.T) {
	s, k, _ := sitzung(t)
	ff := s.FussFelder(nil)
	want := []FeldArt{FeldFarbe, FeldBild, FeldZoom, FeldStand}
	if got := arten(ff); len(got) != len(want) {
		t.Fatalf("ohne Auswahl und Zeiger: %v, erwartet %v", got, want)
	}
	if b := feld(t, ff, FeldBild); b.Groesse != image.Pt(200, 100) || b.Gestaucht {
		t.Errorf("Bild: %+v", b)
	}
	if z := feld(t, ff, FeldZoom); z.Zoom != 100 || !z.Klickbar {
		t.Errorf("Zoom: %+v", z)
	}
	if st := feld(t, ff, FeldStand); st.Ungesichert {
		t.Error("frische Sitzung ungesichert")
	}
	if fa := feld(t, ff, FeldFarbe); fa.Farbe != s.Stil.Farbe || !fa.Klickbar || Deckkraft(fa.Farbe) != 100 {
		t.Errorf("Farbe: %+v", fa)
	}

	s.Druecken(f(30, 30), TastePrimaer) // wählt den Kasten (10,10)–(50,50), ohne zu verschieben
	s.Loslassen(f(30, 30))
	if s.Auswahl != k {
		t.Fatalf("Vorbedingung: Auswahl %d, erwartet %d", s.Auswahl, k)
	}
	p := document.Pt(12.7, 99.9)
	ff = s.FussFelder(&p)
	want = []FeldArt{FeldAuswahl, FeldZeiger, FeldFarbe, FeldBild, FeldZoom, FeldStand}
	got := arten(ff)
	for i := range want {
		if i >= len(got) || got[i] != want[i] {
			t.Fatalf("mit Auswahl und Zeiger: %v, erwartet %v", got, want)
		}
	}
	if a := feld(t, ff, FeldAuswahl); a.Groesse.X < 40 || a.Groesse.Y < 40 {
		t.Errorf("Auswahl: %+v", a)
	}
	if z := feld(t, ff, FeldZeiger); z.Ort != image.Pt(12, 99) {
		t.Errorf("Zeiger: %+v", z)
	}
	aussen := document.Pt(200, 5)
	for _, x := range s.FussFelder(&aussen) {
		if x.Art == FeldZeiger {
			t.Error("Zeiger außerhalb des Bilds als Feld")
		}
	}

	s.Abwaehlen()
	s.SetzeWerkzeug(WerkzeugText)
	if g := feld(t, s.FussFelder(nil), FeldSchrift); g.Grad != int(s.Stil.Schriftgrad) {
		t.Errorf("Schrift: %+v", g)
	}
}

// TestFussFeldUngesichert: Nach einer Änderung meldet das Feld „ungesichert“, nach Gespeichert nicht mehr.
func TestFussFeldUngesichert(t *testing.T) {
	s, k, _ := sitzung(t)
	s.Auswahl = k
	if !s.Loeschen() {
		t.Fatal("Vorbedingung: Löschen")
	}
	if !feld(t, s.FussFelder(nil), FeldStand).Ungesichert {
		t.Error("geändert, aber nicht ungesichert")
	}
	s.Gespeichert()
	if feld(t, s.FussFelder(nil), FeldStand).Ungesichert {
		t.Error("gespeichert, aber ungesichert")
	}
}
