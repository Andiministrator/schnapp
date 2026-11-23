// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"image/color"
	"strconv"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

func TestHSVRundlauf(t *testing.T) {
	for _, c := range []color.NRGBA{
		{R: 0xD9, G: 0x3A, B: 0x33, A: 0xFF}, {R: 0x2F, G: 0x6F, B: 0xDB, A: 0x80}, {R: 0x2E, G: 0x8B, B: 0x57, A: 0},
		{A: 0xFF}, {R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}, {R: 0x80, G: 0x80, B: 0x80, A: 0xFF},
		{R: 0xFF, A: 0xFF}, {G: 0xFF, A: 0xFF}, {B: 0xFF, A: 0xFF}, {R: 0xFF, B: 0x01, A: 0xFF},
	} {
		if got := HSVAus(c).Farbe(c.A); got != c {
			t.Errorf("%v → %+v → %v", c, HSVAus(c), got)
		}
	}
	if h := HSVAus(color.NRGBA{G: 0xFF, A: 0xFF}); h.H != 120 || h.S != 1 || h.V != 1 {
		t.Errorf("Grün: %+v", h)
	}
	if c := (HSV{H: 360 + 240, S: 2, V: -1}).Farbe(9); c != (color.NRGBA{A: 9}) {
		t.Errorf("Begrenzung: %v", c)
	}
	if c := (HSV{H: -120, S: 1, V: 1}).Farbe(255); c != (color.NRGBA{B: 255, A: 255}) {
		t.Errorf("negativer Farbton: %v", c)
	}
}

func TestLeseHex(t *testing.T) {
	alt := color.NRGBA{A: 0x40}
	for text, want := range map[string]color.NRGBA{
		"#D93A33":   {R: 0xD9, G: 0x3A, B: 0x33, A: 0x40},
		"d93a33":    {R: 0xD9, G: 0x3A, B: 0x33, A: 0x40},
		" #fa0 ":    {R: 0xFF, G: 0xAA, B: 0x00, A: 0x40},
		"#11223380": {R: 0x11, G: 0x22, B: 0x33, A: 0x80},
		"112233ff":  {R: 0x11, G: 0x22, B: 0x33, A: 0xFF},
	} {
		if c, ok := LeseHex(text, alt); !ok || c != want {
			t.Errorf("%q: %v %v, erwartet %v", text, c, ok, want)
		}
	}
	for _, text := range []string{"", "#", "12345", "#1234567", "#gg0000", "#+12345", "123456789", "# 12345"} {
		if c, ok := LeseHex(text, alt); ok {
			t.Errorf("%q gilt als %v", text, c)
		}
	}
	if HexText(color.NRGBA{R: 0xD9, G: 0x3A, B: 0x03, A: 0xFF}) != "#D93A03" ||
		HexText(color.NRGBA{R: 1, G: 2, B: 3, A: 0x80}) != "#01020380" || HexText(color.NRGBA{}) != "#00000000" {
		t.Error("HexText")
	}
}

func TestLeseDeckkraft(t *testing.T) {
	for text, want := range map[string]uint8{"100": 255, "0": 0, "50 %": 128, " 50% ": 128, "1": 3, "99": 252} {
		if a, ok := LeseDeckkraft(text); !ok || a != want {
			t.Errorf("%q: %d %v, erwartet %d", text, a, ok, want)
		}
	}
	for _, text := range []string{"", "%", "101", "-1", "5.5", "x"} {
		if _, ok := LeseDeckkraft(text); ok {
			t.Errorf("%q gilt", text)
		}
	}
	for a := 0; a < 256; a++ { // die Anzeige trifft beim Zurücklesen dasselbe Prozent
		p := Deckkraft(color.NRGBA{A: uint8(a)})
		b, _ := LeseDeckkraft(strconv.Itoa(p))
		if Deckkraft(color.NRGBA{A: b}) != p {
			t.Errorf("Alpha %d → %d %% → Alpha %d → %d %%", a, p, b, Deckkraft(color.NRGBA{A: b}))
		}
	}
}

// TestFarbZugEinSchritt: Ziehen in der Farbfläche färbt das gewählte Objekt laufend und hinterlässt beim Loslassen
// genau einen Rückgängig-Schritt (ADR-0021 Nachtrag 9 Punkt 2).
func TestFarbZugEinSchritt(t *testing.T) {
	s, _, p := sitzung(t)
	s.Auswahl = p
	if !s.BeginneFarbZug() || !s.Zieht() || !s.FarbZugLaeuft() {
		t.Fatal("Farbzug beginnt nicht")
	}
	for _, v := range []float64{0.2, 0.5, 0.8} {
		s.FarbZugHSV(HSV{H: 200, S: 1, V: v}, 0x80)
		if o, _ := s.D.Objekt(p); o.(document.Pfeil).Strich.Farbe != (HSV{H: 200, S: 1, V: v}).Farbe(0x80) {
			t.Fatalf("Pfeil färbt sich nicht mit: %v", o.(document.Pfeil).Strich.Farbe)
		}
	}
	if s.SetzeWerkzeug(WerkzeugKasten) || s.Rueckgaengig() {
		t.Error("Werkzeugwechsel oder Rückgängig während des Farbzugs")
	}
	s.BeendeFarbZug()
	want := (HSV{H: 200, S: 1, V: 0.8}).Farbe(0x80)
	if n := schritte(t, s); n != 1 {
		t.Errorf("%d Schritte, erwartet 1", n)
	}
	if s.Stil.Farbe != want || s.AktuelleFarbe() != want || len(s.Zuletzt()) != 1 || s.Zuletzt()[0] != want {
		t.Errorf("Stil %v, aktuell %v, zuletzt %v", s.Stil.Farbe, s.AktuelleFarbe(), s.Zuletzt())
	}
	s.Rueckgaengig()
	if o, _ := s.D.Objekt(p); o.(document.Pfeil).Strich != rot {
		t.Errorf("Rückgängig: %+v", o)
	}
	// Abbruch (Zeiger-Cancel) stellt das Objekt her und hinterlässt nichts.
	s.Auswahl = p
	vor := s.D.Stand()
	s.BeginneFarbZug()
	s.FarbZugHSV(HSV{H: 10, S: 1, V: 1}, 255)
	s.BrecheFarbZugAb()
	if o, _ := s.D.Objekt(p); o.(document.Pfeil).Strich != rot || s.D.Stand() != vor || s.Zieht() {
		t.Errorf("Abbruch: %+v, Stand %d/%d", o, s.D.Stand(), vor)
	}
	// Ohne Auswahl: nur die Werkzeugfarbe, kein Schritt.
	s.Auswahl = 0
	s.BeginneFarbZug()
	s.FarbZugHSV(HSV{H: 60, S: 1, V: 1}, 255)
	s.BeendeFarbZug()
	if s.D.Stand() != vor || s.Stil.Farbe != (color.NRGBA{R: 255, G: 255, A: 255}) {
		t.Errorf("ohne Auswahl: Stand %d/%d, Stil %v", s.D.Stand(), vor, s.Stil.Farbe)
	}
}

// TestFarbZugBehaeltFarbton: Schwarz und Grau verlieren beim Ziehen den Farbton nicht (Nachtrag 9 Punkt 3).
func TestFarbZugBehaeltFarbton(t *testing.T) {
	s, _, _ := sitzung(t)
	s.BeginneFarbZug()
	s.FarbZugHSV(HSV{H: 200, S: 1, V: 0}, 255) // schwarz
	if h := s.FarbeHSV(); h.H != 200 {
		t.Errorf("Farbton nach Schwarz: %+v", h)
	}
	s.BeendeFarbZug()
	if h := s.FarbeHSV(); h.H != 200 || h.S != 1 {
		t.Errorf("nach dem Loslassen: %+v", h)
	}
	s.SetzeFarbe(Beschriftungsfarben[1]) // eine andere Farbe: HSV aus ihr
	if h := s.FarbeHSV(); h != HSVAus(Beschriftungsfarben[1]) {
		t.Errorf("nach Farbfeld: %+v", h)
	}
}

func TestFarbZugInDerTexteingabe(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugText)
	s.Druecken(f(120, 80), TastePrimaer)
	s.Loslassen(f(120, 80))
	s.Tippe("x")
	vor := schritte(t, s)
	s.BeginneFarbZug()
	s.FarbZugHSV(HSV{H: 120, S: 1, V: 0.5}, 255)
	s.BeendeFarbZug()
	if !s.TextLaeuft() {
		t.Fatal("Texteingabe endete")
	}
	s.Enter(false)
	if tx := letztes(t, s).(document.Text); tx.Hintergrund != (HSV{H: 120, S: 1, V: 0.5}).Farbe(255) {
		t.Errorf("Schild %v", tx.Hintergrund)
	}
	if n := schritte(t, s); n != vor+1 {
		t.Errorf("%d Schritte, erwartet %d", n, vor+1)
	}
}

func TestHexUndDeckkraftEingabe(t *testing.T) {
	s, k, p := sitzung(t)
	s.Auswahl = p
	if s.SetzeHex("#12") || s.SetzeDeckkraftText("200") || schritte(t, s) != 0 {
		t.Fatal("ungültige Eingabe wirkte")
	}
	if !s.SetzeHex("#336699") {
		t.Fatal("Hex abgelehnt")
	}
	if o, _ := s.D.Objekt(p); o.(document.Pfeil).Strich.Farbe != (color.NRGBA{R: 0x33, G: 0x66, B: 0x99, A: 255}) {
		t.Errorf("Pfeil %v", o.(document.Pfeil).Strich.Farbe)
	}
	if !s.SetzeDeckkraftText("50") {
		t.Fatal("Deckkraft abgelehnt")
	}
	want := color.NRGBA{R: 0x33, G: 0x66, B: 0x99, A: 128}
	if o, _ := s.D.Objekt(p); o.(document.Pfeil).Strich.Farbe != want || s.AktuelleFarbe() != want {
		t.Errorf("Pfeil %v", o.(document.Pfeil).Strich.Farbe)
	}
	if n := schritte(t, s); n != 2 {
		t.Errorf("%d Schritte, erwartet 2", n)
	}
	if z := s.Zuletzt(); len(z) != 2 || z[0] != want {
		t.Errorf("zuletzt %v", z)
	}
	// Der Kasten der Testsitzung hat keinen Strich (Alpha 0): 6 Ziffern behalten diese Deckkraft, 8 setzen sie.
	s.Auswahl = k
	if st := s.D.Stand(); !s.SetzeHex("#000000") || s.D.Stand() != st {
		t.Error("6 Ziffern änderten den Kasten ohne Strich")
	}
	s.SetzeHex("#000000ff")
	if o, _ := s.D.Objekt(k); o.(document.Kasten).Strich.Farbe != (color.NRGBA{A: 255}) {
		t.Errorf("Kasten %v", o.(document.Kasten).Strich)
	}
}

func TestZuletztBenutzt(t *testing.T) {
	s, _, _ := sitzung(t)
	a, b, c, d := color.NRGBA{R: 1, A: 255}, color.NRGBA{R: 2, A: 255}, color.NRGBA{R: 3, A: 255}, color.NRGBA{R: 4, A: 255}
	for _, x := range []color.NRGBA{a, b, Beschriftungsfarben[2], c, b, d} {
		s.SetzeHex(HexText(x))
		for _, z := range s.Zuletzt() {
			if z == Beschriftungsfarben[2] {
				t.Fatalf("Beschriftungsfarbe in den zuletzt benutzten: %v", s.Zuletzt())
			}
		}
	}
	z := s.Zuletzt()
	if len(z) != ZuletztGrenze || z[0] != d || z[1] != b || z[2] != c {
		t.Errorf("zuletzt %v", z)
	}
}

// TestPipette: Der Klick nimmt die Farbe des Bildpunkts aus dem Original (ohne Beschriftungen, deckend), trifft kein
// Objekt und beendet die Pipette; daneben, mit Esc oder Werkzeugwechsel endet sie ohne Wirkung.
func TestPipette(t *testing.T) {
	o := image.NewNRGBA(image.Rect(0, 0, 200, 100))
	o.SetNRGBA(30, 30, color.NRGBA{R: 10, G: 20, B: 30, A: 77})
	o.SetNRGBA(199, 99, color.NRGBA{R: 9, A: 255})
	d, err := document.Neu(o)
	if err != nil {
		t.Fatal(err)
	}
	k, _ := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(10, 10), document.Pt(50, 50)),
		Fuellung: color.NRGBA{G: 255, A: 255}})
	s := NeueSitzung(d)
	s.SetzeFenster(image.Pt(248, 148))
	s.Auswahl = k
	if !s.SchaltePipette() || !s.PipetteLaeuft() {
		t.Fatal("Pipette geht nicht an")
	}
	s.Druecken(f(30.5, 30.5), TastePrimaer)
	s.Loslassen(f(30.5, 30.5))
	want := color.NRGBA{R: 10, G: 20, B: 30, A: 255}
	if s.PipetteLaeuft() || s.Stil.Farbe != want || s.Zieht() {
		t.Fatalf("Pipette %v, Stil %v", s.PipetteLaeuft(), s.Stil.Farbe)
	}
	if ob, _ := s.D.Objekt(k); ob.(document.Kasten).Strich.Farbe != want || ob.(document.Kasten).Bereich != document.R(document.Pt(10, 10), document.Pt(50, 50)) {
		t.Errorf("Kasten %+v", ob)
	}
	// Rand des Bilds: der letzte Bildpunkt.
	s.SchaltePipette()
	s.Druecken(f(200, 100), TastePrimaer)
	s.Loslassen(f(200, 100))
	if s.Stil.Farbe != (color.NRGBA{R: 9, A: 255}) {
		t.Errorf("Rand: %v", s.Stil.Farbe)
	}
	vor, farbe := s.D.Stand(), s.Stil.Farbe
	s.SchaltePipette()
	s.Druecken(document.Pt(2, 2), TastePrimaer) // neben dem Bild
	s.Loslassen(document.Pt(2, 2))
	if s.PipetteLaeuft() || s.D.Stand() != vor || s.Stil.Farbe != farbe {
		t.Error("Klick neben das Bild wirkte")
	}
	s.SchaltePipette()
	s.Esc()
	if s.PipetteLaeuft() || s.Auswahl != k {
		t.Error("Esc beendet die Pipette nicht oder hob die Auswahl auf")
	}
	s.SchaltePipette()
	s.SetzeWerkzeug(WerkzeugPfeil)
	if s.PipetteLaeuft() {
		t.Error("Werkzeugwechsel beendet die Pipette nicht")
	}
}

// TestFussFeldZeigtAktuelleFarbe: Leiste, Farbfeld links und Fußzeile zeigen dieselbe Farbe — die des gewählten
// Objekts, sonst die des Werkzeugs (Sichtprobe #189: nach Strg+Z zeigte die Fußzeile noch die Werkzeugfarbe).
func TestFussFeldZeigtAktuelleFarbe(t *testing.T) {
	s, _, p := sitzung(t)
	s.SetzeFarbe(Beschriftungsfarben[2])
	s.Auswahl = p // roter Pfeil, Werkzeugfarbe grün
	if c := feld(t, s.FussFelder(nil), FeldFarbe).Farbe; c != rot.Farbe || s.AktuelleFarbe() != rot.Farbe {
		t.Errorf("mit Auswahl: Fußfeld %v, aktuell %v, erwartet %v", c, s.AktuelleFarbe(), rot.Farbe)
	}
	s.Auswahl = 0
	if c := feld(t, s.FussFelder(nil), FeldFarbe).Farbe; c != Beschriftungsfarben[2] {
		t.Errorf("ohne Auswahl: Fußfeld %v", c)
	}
}
