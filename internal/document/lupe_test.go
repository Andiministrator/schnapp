// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"image"
	"image/color"
	"math"
	"testing"
)

func testLupe() Lupe {
	return Lupe{Quelle: R(Pt(10, 10), Pt(20, 16)), Ziel: R(Pt(40, 10), Pt(60, 22)),
		Strich: Strich{Farbe: color.NRGBA{255, 0, 0, 255}, Breite: 2}}
}

// TestLupePruefe: Breite und Höhe beider Rechtecke > 0 und ≤ MaxGroesse, Koordinaten im Bereich, Strich gültig.
func TestLupePruefe(t *testing.T) {
	if err := testLupe().pruefe(); err != nil {
		t.Fatal(err)
	}
	kaputt := map[string]func(*Lupe){
		"Quelle ohne Breite": func(l *Lupe) { l.Quelle.Max.X = l.Quelle.Min.X },
		"Quelle verkehrt":    func(l *Lupe) { l.Quelle.Max.Y = l.Quelle.Min.Y - 1 },
		"Ziel ohne Höhe":     func(l *Lupe) { l.Ziel.Max.Y = l.Ziel.Min.Y },
		"Ziel zu breit":      func(l *Lupe) { l.Ziel.Max.X = l.Ziel.Min.X + MaxGroesse + 1 },
		"Quelle zu hoch":     func(l *Lupe) { l.Quelle.Max.Y = l.Quelle.Min.Y + MaxGroesse + 0.5 },
		"Koordinate NaN":     func(l *Lupe) { l.Ziel.Min.X = math.NaN() },
		"Koordinate zu groß": func(l *Lupe) { l.Quelle.Min.X, l.Quelle.Max.X = MaxKoordinate+1, MaxKoordinate+2 },
		"Strich negativ":     func(l *Lupe) { l.Strich.Breite = -1 },
		"Ebene kaputt":       func(l *Lupe) { l.Transparenz = 101 },
	}
	for name, f := range kaputt {
		l := testLupe()
		f(&l)
		if l.pruefe() == nil {
			t.Errorf("%s: angenommen", name)
		}
	}
	// Genau an der Grenze gültig.
	l := testLupe()
	l.Ziel.Max.X = l.Ziel.Min.X + MaxGroesse
	if err := l.pruefe(); err != nil {
		t.Errorf("Ziel genau MaxGroesse: %v", err)
	}
	d, _ := Neu(image.NewNRGBA(image.Rect(0, 0, 8, 8)))
	if _, err := d.Hinzufuegen(Lupe{Quelle: R(Pt(1, 1), Pt(1, 2)), Ziel: R(Pt(3, 3), Pt(4, 4))}); err == nil {
		t.Error("Hinzufuegen nimmt eine Lupe ohne Quellbreite")
	}
}

// TestLupeGrenzen: Grenzen umfassen beide Rechtecke mit halbem Strich; die Vergrößerung je Achse.
func TestLupeGrenzen(t *testing.T) {
	l := testLupe()
	if g, want := l.Grenzen(), R(Pt(9, 9), Pt(61, 23)); g != want {
		t.Errorf("Grenzen %v, erwartet %v", g, want)
	}
	if x, y := l.Vergroesserung(); x != 2 || y != 2 {
		t.Errorf("Vergrößerung %v, %v", x, y)
	}
	if l.Kennung() != 0 || l.mitKennung(7).Kennung() != 7 {
		t.Error("Kennung")
	}
	if MitEbene(l, Ebene{Name: "Lupe"}).Ebenenwerte().Name != "Lupe" {
		t.Error("MitEbene")
	}
}

// TestLupeQuellpixel: nächster Nachbar nach ADR-0033 Punkt 3 (ohne Zuschnitt und Stauchen ist die Quelle in der
// Ansicht die Quelle im Original).
func TestLupeQuellpixel(t *testing.T) {
	d, err := Neu(image.NewNRGBA(image.Rect(0, 0, 100, 40)))
	if err != nil {
		t.Fatal(err)
	}
	l := testLupe() // Quelle x 10..20, Ziel x 40..60: Zielspalten 40 und 41 zeigen 10, 58 und 59 zeigen 19
	q := d.LupenQuelle(l)
	for _, f := range []struct {
		z    float64
		want int
	}{{40.5, 10}, {41.5, 10}, {42.5, 11}, {58.5, 19}, {59.5, 19}} {
		if got, ok := q.Spalte(f.z); !ok || got != f.want {
			t.Errorf("Spalte(%v) = %d %v, erwartet %d", f.z, got, ok, f.want)
		}
	}
	for _, z := range []float64{39.5, 60, 60.5} {
		if _, ok := q.Spalte(z); ok {
			t.Errorf("Spalte(%v) außerhalb des Ziels angenommen", z)
		}
	}
	l.Ziel = R(Pt(40, 10), Pt(70, 28)) // 3×
	if got, ok := d.LupenQuelle(l).Zeile(10.5 + 3*4); !ok || got != 14 {
		t.Errorf("Zeile = %d, erwartet 14", got)
	}
	l.Quelle = R(Pt(10.5, 10), Pt(20.5, 16)) // halbe Pixel: Ziel 40.5 zeigt floor(10.5 + 0.25) = 10
	if got, _ := d.LupenQuelle(l).Spalte(40.5); got != 10 {
		t.Errorf("halbe Quelle: %d", got)
	}
}

// TestLupeLinie: von Kante zu Kante zwischen den Mittelpunkten, keine Linie bei Überlappung oder Berührung.
func TestLupeLinie(t *testing.T) {
	l := testLupe() // Mitten (15,13) und (50,16)
	a, b, ok := l.Linie()
	if !ok {
		t.Fatal("keine Linie")
	}
	nah := func(p, q Punkt) bool { return math.Abs(p.X-q.X) < 1e-9 && math.Abs(p.Y-q.Y) < 1e-9 }
	// Austritt aus der Quelle bei x = 20 (t = 5/35), Eintritt ins Ziel bei x = 40 (t = 25/35).
	if !nah(a, Pt(20, 13+3*5.0/35)) || !nah(b, Pt(40, 13+3*25.0/35)) {
		t.Errorf("Linie %v – %v", a, b)
	}
	// Senkrecht übereinander: Austritt unten, Eintritt oben.
	l.Ziel = R(Pt(5, 30), Pt(25, 42))
	a, b, ok = l.Linie()
	if !ok || !nah(a, Pt(15, 16)) || !nah(b, Pt(15, 30)) {
		t.Errorf("senkrecht: %v – %v %v", a, b, ok)
	}
	// Überlappung: keine Linie.
	l.Ziel = R(Pt(18, 14), Pt(38, 26))
	if _, _, ok := l.Linie(); ok {
		t.Error("Linie trotz Überlappung")
	}
	// Überlappung an einer Ecke, die Gerade zwischen den Mitten läuft außerhalb der Schnittfläche: trotzdem keine Linie.
	l.Ziel = R(Pt(19, 15), Pt(110, 17))
	if _, _, ok := l.Linie(); ok {
		t.Error("Linie trotz Überlappung an der Ecke")
	}
	// Berühren an der Kante: nichts bleibt übrig.
	l.Ziel = R(Pt(20, 10), Pt(40, 16))
	if _, _, ok := l.Linie(); ok {
		t.Error("Linie trotz Berührung")
	}
	// Ziel enthält die Quelle: keine Linie.
	l.Ziel = R(Pt(0, 0), Pt(100, 100))
	if _, _, ok := l.Linie(); ok {
		t.Error("Linie trotz Enthaltensein")
	}
}

// lupenMuster ist ein Dokument mit eigener Farbe je Pixel.
func lupenMuster(t *testing.T, w, h int) *Dokument {
	t.Helper()
	b := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			b.SetNRGBA(x, y, color.NRGBA{uint8(x), uint8(y), uint8(x/256 + 1), 255})
		}
	}
	d, err := Neu(b)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// TestLupenQuelleAnsicht (ADR-0033 Nachtrag 2 Punkt 1): Die Lupe vergrößert die Ansicht. Ein weggestauchter Streifen
// in der Quelle fällt heraus (keine Lücke im Ziel), was vor dem Zuschnitt liegt, behält seine Breite und bleibt leer,
// und Pixel im Spalt der Naht bleiben leer.
func TestLupenQuelleAnsicht(t *testing.T) {
	d := lupenMuster(t, 300, 100)
	if err := d.SetzeZuschnitt(image.Rect(20, 0, 300, 100)); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]Streifen{{Von: 100, Bis: 140}}); err != nil {
		t.Fatal(err)
	}
	// Naht in der Ansicht bei x = 80 (Original 100 − 20). Quelle Original 10..180: Ansicht −10..120, also 130 breit.
	l := Lupe{Quelle: R(Pt(10, 60), Pt(180, 70)), Ziel: R(Pt(0, 0), Pt(260, 20))}
	q := d.LupenQuelle(l)
	if qa := q.InAnsicht(); qa != R(Pt(-10, 60), Pt(120, 70)) {
		t.Fatalf("Quelle in der Ansicht %v", qa)
	}
	orig := d.OriginalBild()
	gezeigt, leer, spalt := 0, 0, 0
	q.Abtasten(260, 20, func(i int) float64 { return float64(i) + 0.5 }, func(j int) float64 { return float64(j) + 0.5 },
		func(i, j int, px []byte) {
			vx, vy := -10+i/2, 60+j/2 // 2× in beiden Achsen
			ox := vx + 20
			if vx >= 80 {
				ox = vx + 60
			}
			if vx < 0 || q.ImSpalt(vx, vy) {
				t.Fatalf("Ziel (%d,%d) zeigt Ansicht (%d,%d) (vor dem Zuschnitt oder im Spalt)", i, j, vx, vy)
			}
			if ox >= 100 && ox < 140 {
				t.Fatalf("Ziel (%d,%d) zeigt Streifenspalte %d", i, j, ox)
			}
			o := orig.PixOffset(ox, vy)
			if string(px) != string(orig.Pix[o:o+4]) {
				t.Fatalf("Ziel (%d,%d): %v, erwartet Original (%d,%d)", i, j, px, ox, vy)
			}
			gezeigt++
		}, nil)
	for j := 0; j < 20; j++ {
		for i := 0; i < 260; i++ {
			vx, vy := -10+i/2, 60+j/2
			if vx < 0 || q.ImSpalt(vx, vy) {
				leer++
				if vx >= 0 {
					spalt++
				}
			}
		}
	}
	// Leer bleiben genau die Pixel vor dem Zuschnitt (10 Ansichtsspalten, also 20 Zielspalten) und im Spalt.
	if gezeigt+leer != 260*20 {
		t.Errorf("Abtasten zeigte %d Pixel, %d bleiben leer: Summe nicht 5200", gezeigt, leer)
	}
	if spalt == 0 {
		t.Error("kein Pixel im Spalt: der Test prüft den Spalt nicht")
	}
	// Spaltband: ±8 um die Mittellinie (Faktor 1), die höchstens 6 auslenkt.
	for vy := 60; vy < 70; vy++ {
		if q.ImSpalt(80+15, vy) || q.ImSpalt(80-16, vy) || !q.ImSpalt(80, vy) && !q.ImSpalt(79, vy) {
			t.Errorf("Spalt in Zeile %d falsch begrenzt", vy)
		}
	}
	// AnsichtsAusschnitt: die ganzen Ansichtspixel unter der Quelle, auf die Ansicht begrenzt, Spalt durchsichtig.
	a := q.AnsichtsAusschnitt()
	if a.Rect != image.Rect(0, 60, 120, 70) {
		t.Fatalf("Ansichtsausschnitt %v", a.Rect)
	}
	for vy := 60; vy < 70; vy++ {
		for vx := 0; vx < 120; vx++ {
			ox := vx + 20
			if vx >= 80 {
				ox = vx + 60
			}
			want := orig.NRGBAAt(ox, vy)
			if q.ImSpalt(vx, vy) {
				want = color.NRGBA{}
			}
			if got := a.NRGBAAt(vx, vy); got != want {
				t.Errorf("Ausschnitt (%d,%d) = %v, erwartet %v", vx, vy, got, want)
			}
		}
	}
	// Zielbild: dieselben Pixel wie Abtasten auf dem Raster des Originals, außerhalb des Ziels durchsichtig.
	z := q.Zielbild(image.Rect(-2, -2, 262, 22))
	q.Abtasten(260, 20, func(i int) float64 { return float64(i) + 0.5 }, func(j int) float64 { return float64(j) + 0.5 },
		func(i, j int, px []byte) {
			if got := z.NRGBAAt(i, j); got != (color.NRGBA{px[0], px[1], px[2], px[3]}) {
				t.Fatalf("Zielbild (%d,%d) = %v", i, j, got)
			}
		}, nil)
	if z.NRGBAAt(-1, 5) != (color.NRGBA{}) || z.NRGBAAt(261, 5) != (color.NRGBA{}) {
		t.Error("Zielbild außerhalb des Ziels nicht durchsichtig")
	}
	// Liegt die Quelle ganz im Streifen, hat sie in der Ansicht keine Breite: Das Ziel bleibt leer.
	l.Quelle = R(Pt(105, 60), Pt(135, 70))
	if _, ok := d.LupenQuelle(l).Spalte(5.5); ok {
		t.Error("Quelle ganz im Streifen zeigt etwas")
	}
}
