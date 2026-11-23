// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"image/color"
	"math"
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

// TestFussFeldZeigerImOriginal (F-513): Das Feld zeigt den Ort im Original, nicht in der Ansicht: bei Zuschnitt ab
// (10,10) liegt Ansichtspunkt (2,2) bei Original (12,12); beim Zuschneiden zeigt die Leinwand die Vollansicht, das Feld
// bleibt dort vorhanden und zählt auch den abgedunkelten Teil.
func TestFussFeldZeigerImOriginal(t *testing.T) {
	s, _, _ := sitzung(t)
	if err := s.D.SetzeZuschnitt(image.Rect(10, 10, 200, 100)); err != nil {
		t.Fatal(err)
	}
	ort := func(p document.Punkt) (image.Point, bool) {
		for _, x := range s.FussFelder(&p) {
			if x.Art == FeldZeiger {
				return x.Ort, true
			}
		}
		return image.Point{}, false
	}
	if o, ok := ort(document.Pt(2.5, 2.5)); !ok || o != image.Pt(12, 12) {
		t.Errorf("mit Zuschnitt: Ort %v (da %v), erwartet (12,12)", o, ok)
	}
	if _, ok := ort(document.Pt(190, 5)); ok {
		t.Error("Zeiger rechts außerhalb der Ansicht (190 ≥ Breite 190) als Feld")
	}
	s.SetzeWerkzeug(WerkzeugZuschneiden)
	if o, ok := ort(document.Pt(-5.5, -5.5)); !ok || o != image.Pt(4, 4) {
		t.Errorf("Zuschneiden, abgedunkelter Teil: Ort %v (da %v), erwartet (4,4)", o, ok)
	}
	if o, ok := ort(document.Pt(189.5, 89.5)); !ok || o != image.Pt(199, 99) {
		t.Errorf("Zuschneiden, Vollansicht rechts unten: Ort %v (da %v), erwartet (199,99)", o, ok)
	}

	// Ein Zeilenstreifen vor dem Zuschnittsanfang (Kritik B-6, Mutant E3): Dann rechnen Ansicht und Vollansicht
	// verschieden um, und beim Zuschneiden gilt die Vollansicht (Zeigerort plus Versatz).
	s2, _, _ := sitzung(t)
	if err := s2.D.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 2, Bis: 6}}); err != nil {
		t.Fatal(err)
	}
	if err := s2.D.SetzeZuschnitt(image.Rect(10, 10, 200, 100)); err != nil {
		t.Fatal(err)
	}
	a := document.Pt(5.5, -5.5) // im abgedunkelten Teil vor dem Streifen (Ansicht setzt den Zuschnitt linear fort)
	v := s2.D.VollVersatz()
	voll := s2.D.VollZuOriginal(a.Plus(document.Pt(float64(v.X), float64(v.Y))))
	ansicht := s2.D.ZuOriginal(a)
	if math.Floor(voll.Y) == math.Floor(ansicht.Y) {
		t.Fatalf("Vorbedingung: Vollansicht %v und Ansicht %v rechnen gleich um, der Streifen wirkt nicht", voll, ansicht)
	}
	ort2 := func() (image.Point, bool) {
		for _, x := range s2.FussFelder(&a) {
			if x.Art == FeldZeiger {
				return x.Ort, true
			}
		}
		return image.Point{}, false
	}
	s2.SetzeWerkzeug(WerkzeugZuschneiden)
	if o, ok := ort2(); !ok || o != image.Pt(int(math.Floor(voll.X)), int(math.Floor(voll.Y))) {
		t.Errorf("Streifen vor dem Zuschnitt, Zuschneiden: Ort %v (da %v), erwartet Vollansicht %v", o, ok, voll)
	}
}

// TestFussFelderAuswahlBildSchrift (F-514, Mutanten U10 bis U12): Die Auswahlgröße trägt Breite und Höhe getrennt, das
// Bildfeld nennt „gestaucht“ nach einem Stauchen, und ein gewählter Text zeigt seinen eigenen Schriftgrad statt den des
// Werkzeugs.
func TestFussFelderAuswahlBildSchrift(t *testing.T) {
	s, _, _ := sitzung(t)
	id, err := s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(20, 60), document.Pt(120, 80)),
		Fuellung: color.NRGBA{A: 255}})
	if err != nil {
		t.Fatal(err)
	}
	s.Auswahl = id
	if a := feld(t, s.FussFelder(nil), FeldAuswahl); a.Groesse.X < 90 || a.Groesse.X > 130 || a.Groesse.Y < 15 || a.Groesse.Y > 30 {
		t.Errorf("Auswahl eines 100×20-Kastens: %+v (Breite und Höhe verwechselt?)", a.Groesse)
	}

	if feld(t, s.FussFelder(nil), FeldBild).Gestaucht {
		t.Error("Bild ohne Stauchen als gestaucht gemeldet")
	}
	if err := s.D.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 20, Bis: 30}}); err != nil {
		t.Fatal(err)
	}
	if b := feld(t, s.FussFelder(nil), FeldBild); !b.Gestaucht || b.Groesse.Y != 90 {
		t.Errorf("Bild nach dem Stauchen: %+v, erwartet gestaucht, Höhe 90", b)
	}

	s.Stil.Schriftgrad = 18
	tid, err := s.D.Hinzufuegen(document.Text{Ort: document.Pt(5, 15), Inhalt: "Hallo", Schriftgrad: 33,
		Farbe: color.NRGBA{A: 255}})
	if err != nil {
		t.Fatal(err)
	}
	s.Auswahl = tid
	if g := feld(t, s.FussFelder(nil), FeldSchrift); g.Grad != 33 {
		t.Errorf("Schrift des gewählten Texts: %d, erwartet 33 (nicht der Grad des Werkzeugs %v)", g.Grad, s.Stil.Schriftgrad)
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
