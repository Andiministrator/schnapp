// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"errors"
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Lupe im Renderer (ADR-0033): Sollwerte je Pixel.

var gruen = color.NRGBA{G: 255, A: 255}

// musterDokument hat in jedem Pixel eine eigene, deckende Farbe.
func musterDokument(t *testing.T, w, h int) *document.Dokument {
	t.Helper()
	b := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			b.SetNRGBA(x, y, color.NRGBA{uint8(x * 5), uint8(y * 7), uint8(100 + x + y), 255})
		}
	}
	d, err := document.Neu(b)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// ohneStrich: nur das Bild, damit jedes Zielpixel eine Sollfarbe hat.
func lupeOhneStrich(q, z document.Rechteck) document.Lupe {
	return document.Lupe{Quelle: q, Ziel: z, Strich: document.Strich{Farbe: gruen}}
}

// TestLupeVergroessert: Jedes Zielpixel ist das Quellpixel nach dem nächsten Nachbarn (2× und 3×), außerhalb des
// Ziels bleibt das Bild; LupenInhalt liefert dieselben Pixel.
func TestLupeVergroessert(t *testing.T) {
	for _, f := range []struct {
		name string
		z    document.Rechteck
		k    int
	}{{"2x", document.R(document.Pt(20, 10), document.Pt(28, 16)), 2}, {"3x", document.R(document.Pt(20, 10), document.Pt(32, 19)), 3},
		{"2x versetzt", document.R(document.Pt(20.6, 10.6), document.Pt(28.6, 16.6)), 2}} {
		d := musterDokument(t, 40, 30)
		orig := d.Original()
		l := lupeOhneStrich(document.R(document.Pt(2, 3), document.Pt(6, 6)), f.z)
		hinzu(t, d, l)
		b := bild(t, d)
		inhalt := LupenInhalt(d, l)
		q := d.LupenQuelle(l) // ohne Zuschnitt und Stauchen: Ansicht = Original
		for y := 0; y < 30; y++ {
			for x := 0; x < 40; x++ {
				want := orig.NRGBAAt(x, y)
				mx, my := float64(x)+0.5, float64(y)+0.5 // Mittelpunkt des Pixels
				drin := mx >= f.z.Min.X && mx < f.z.Max.X && my >= f.z.Min.Y && my < f.z.Max.Y
				if drin {
					sx, _ := q.Spalte(mx)
					sy, _ := q.Zeile(my)
					want = orig.NRGBAAt(sx, sy)
					if gx, gy := 2+int(mx-f.z.Min.X)/f.k, 3+int(my-f.z.Min.Y)/f.k; sx != gx || sy != gy {
						t.Fatalf("%s: Quellpixel (%d,%d), erwartet (%d,%d)", f.name, sx, sy, gx, gy)
					}
					if got := inhalt.NRGBAAt(x, y); got != want {
						t.Errorf("%s: LupenInhalt (%d,%d) = %v, erwartet %v", f.name, x, y, got, want)
					}
				}
				if got := b.NRGBAAt(x, y); got != want {
					t.Errorf("%s: (%d,%d) = %v, erwartet %v", f.name, x, y, got, want)
				}
			}
		}
	}
}

// TestLupeGeschwaerzt: Eine Schwärzung in der Quelle zeigt die Lupe sofort, auch eine spätere (ADR-0033 Punkt 2).
func TestLupeGeschwaerzt(t *testing.T) {
	d := musterDokument(t, 40, 30)
	l := lupeOhneStrich(document.R(document.Pt(2, 3), document.Pt(6, 6)), document.R(document.Pt(20, 10), document.Pt(28, 16)))
	hinzu(t, d, l)
	if err := d.Schwaerzen(image.Rect(2, 3, 4, 5), document.Schwaerzung{Farbe: color.NRGBA{A: 255}}); err != nil {
		t.Fatal(err)
	}
	b := bild(t, d)
	for y := 10; y < 14; y++ {
		for x := 20; x < 24; x++ {
			if got := b.NRGBAAt(x, y); got != (color.NRGBA{A: 255}) {
				t.Errorf("(%d,%d) = %v, erwartet schwarz", x, y, got)
			}
		}
	}
	if got := b.NRGBAAt(24, 10); got == (color.NRGBA{A: 255}) {
		t.Error("ungeschwärzter Teil schwarz")
	}
}

// TestLupeVerborgenDurchsichtig (ADR-0033 Punkt 2 und 4, Nachtrag 2 Punkt 1): Die Lupe vergrößert die Ansicht.
// Was vor dem Zuschnitt liegt, bleibt durchsichtig (das Bild darunter bleibt), ein weggestauchter Streifen in der
// Quelle fällt heraus, und der Spalt der Naht bleibt durchsichtig; die Zielpixel liegen über ZuOriginal.
func TestLupeVerborgenDurchsichtig(t *testing.T) {
	// Das Ziel liegt weit unter der Naht des Streifens: Die Kante (Spalt, Fasersaum, ±16 Pixel) berührt es nicht.
	d := musterDokument(t, 60, 200)
	l := lupeOhneStrich(document.R(document.Pt(2, 10), document.Pt(14, 70)), document.R(document.Pt(30, 80), document.Pt(54, 180)))
	id, err := d.Hinzufuegen(l)
	if err != nil {
		t.Fatal(err)
	}
	// Zuschnitt ab x = 4, Zeilenstreifen 40..50: Naht in der Ansicht bei y = 40. Quelle in der Ansicht −2..10 × 10..60
	// (12 × 50), Ziel 24 × 100: in beiden Achsen 2×. Ziel in der Ansicht 26..50 × 70..170.
	if err := d.SetzeZuschnitt(image.Rect(4, 0, 60, 200)); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 40, Bis: 50}}); err != nil {
		t.Fatal(err)
	}
	b := bild(t, d)
	orig := d.Original()
	q := d.LupenQuelle(l)
	if err := d.Entfernen(id); err != nil {
		t.Fatal(err)
	}
	grund := bild(t, d) // dieselbe Ansicht ohne Lupe
	spalt, nachNaht := 0, 0
	for vy := 66; vy < 174; vy++ {
		for vx := 22; vx < 54; vx++ {
			o := d.ZuOriginal(document.Pt(float64(vx)+0.5, float64(vy)+0.5))
			want := grund.NRGBAAt(vx, vy)
			if o.X >= 30 && o.X < 54 && o.Y >= 80 && o.Y < 180 {
				sx := int(math.Floor(-2 + (o.X-30)/2)) // Ansichtsspalte der Quelle
				sy := int(math.Floor(10 + (o.Y-80)/2)) // Ansichtszeile der Quelle
				oy := sy
				if sy >= 40 {
					oy = sy + 10 // hinter dem Streifen
				}
				switch {
				case sx < 0: // vor dem Zuschnitt
				case q.ImSpalt(sx, sy):
					spalt++
				default:
					want = orig.NRGBAAt(sx+4, oy)
					if sy >= 40 {
						nachNaht++
					}
				}
			}
			if got := b.NRGBAAt(vx, vy); got != want {
				t.Errorf("Ansicht (%d,%d) = %v, erwartet %v", vx, vy, got, want)
			}
		}
	}
	if spalt == 0 || nachNaht == 0 {
		t.Errorf("Prüfung unvollständig: %d Zielpixel im Spalt, %d hinter der Naht", spalt, nachNaht)
	}
	// Sollwerte an Stellen: Ansicht (26,70) zeigt Quellspalte −2 (vor dem Zuschnitt) → Grund; (30,70) Quellpixel
	// Ansicht (0,10) = Original (4,10); (30,168) Ansicht (0,59) = Original (4,69).
	if got, want := b.NRGBAAt(26, 70), grund.NRGBAAt(26, 70); got != want {
		t.Errorf("vor dem Zuschnitt: %v, erwartet Grund %v", got, want)
	}
	for p, o := range map[image.Point]image.Point{{30, 70}: {4, 10}, {30, 168}: {4, 69}} {
		if got, want := b.NRGBAAt(p.X, p.Y), orig.NRGBAAt(o.X, o.Y); got != want {
			t.Errorf("Zielpixel %v: %v, erwartet Original %v = %v", p, got, o, want)
		}
	}
	if _, err := d.Hinzufuegen(l); err != nil {
		t.Fatal(err)
	}
	inhalt := LupenInhalt(d, l)
	if inhalt.Rect != image.Rect(26, 70, 50, 170) {
		t.Errorf("LupenInhalt %v", inhalt.Rect)
	}
	if inhalt.NRGBAAt(26, 70).A != 0 || inhalt.NRGBAAt(30, 168) != orig.NRGBAAt(4, 69) {
		t.Error("LupenInhalt weicht ab")
	}
}

// TestLupeLinieUndRahmen: Linie zwischen den Rechtecken, nicht in ihrem Inneren; keine Linie bei Überlappung; Rahmen
// auf der Kante.
func TestLupeLinieUndRahmen(t *testing.T) {
	d := weissesDokument(t, 80, 40)
	l := document.Lupe{Quelle: document.R(document.Pt(5, 10), document.Pt(15, 20)), Ziel: document.R(document.Pt(50, 10), document.Pt(70, 30)),
		Strich: document.Strich{Farbe: gruen, Breite: 2}}
	hinzu(t, d, l)
	b := bild(t, d)
	// Mitten (10,15) und (60,20): die Linie läuft bei x = 32 durch y = 15 + 5·22/50 = 17,2.
	pruefe(t, b, map[image.Point]color.NRGBA{
		{32, 17}: gruen, // Linie
		{10, 15}: weiss, // Inneres der Quelle: keine Linie
		{12, 16}: weiss, // Inneres der Quelle, auf der Geraden
		{5, 15}:  gruen, // Rahmen der Quelle (mittig auf der Kante)
		{15, 12}: gruen, // rechte Kante der Quelle
		{69, 29}: gruen, // Rahmen des Ziels, Ecke innen
		{40, 30}: weiss, // neben der Linie
		{60, 20}: weiss, // Bild im Ziel (weiße Quelle)
		{1, 1}:   weiss, // fern
		{4, 9}:   gruen, // spitze Außenecke der Quelle
		{3, 8}:   weiss, // außerhalb der Ecke
		{70, 30}: gruen, // spitze Außenecke des Ziels
		{71, 31}: weiss, // außerhalb
	})
	// Überlappung: keine Linie, das Innere der Quelle außerhalb des Ziels bleibt weiß.
	d2 := weissesDokument(t, 80, 40)
	l2 := l
	l2.Ziel = document.R(document.Pt(12, 12), document.Pt(32, 32))
	hinzu(t, d2, l2)
	b2 := bild(t, d2)
	pruefe(t, b2, map[image.Point]color.NRGBA{{9, 13}: weiss, {10, 15}: weiss, {8, 14}: weiss})
}

// TestLupeKosten: Das Bild kostet wie eine Füllung seiner aufs Bild beschnittenen Fläche; Pruefe lehnt über dem
// Budget ab (ADR-0033 Punkt 7).
func TestLupeKosten(t *testing.T) {
	d := weissesDokument(t, 100, 50)
	// Ohne Strich: nur das Bild. Ziel 60..120 × 20..60 ragt rechts und unten hinaus: beschnitten 40 × 30. Dazu die
	// Quelle 30 × 20 (ADR-0033 Nachtrag 2 Punkt 3).
	hinzu(t, d, lupeOhneStrich(document.R(document.Pt(1, 1), document.Pt(31, 21)), document.R(document.Pt(60, 20), document.Pt(120, 60))))
	kosten := fuellGewicht * (40*30 + 30*20)
	if got, want := Aufwand(d), kosten; got != want {
		t.Errorf("Aufwand %v, erwartet %v", got, want)
	}
	_, a, err := bildMitAufwand(d)
	if err != nil {
		t.Fatal(err)
	}
	if a.summe() != Aufwand(d) {
		t.Errorf("Bild zählt %v, Trockenlauf %v", a.summe(), Aufwand(d))
	}
	alt := grenzen
	defer func() { grenzen = alt }()
	grenzen.gesamt = kosten - 1
	if err := Pruefe(d); !errors.Is(err, ErrZuAufwendig) {
		t.Errorf("Pruefe über dem Budget: %v", err)
	}
	grenzen.gesamt = kosten
	if err := Pruefe(d); err != nil {
		t.Errorf("Pruefe am Budget: %v", err)
	}
	// Ziel ganz außerhalb der Ansicht: kostet nichts (auch die Quelle nicht, der SVG-Export bettet dann kein Bild ein).
	d2 := weissesDokument(t, 100, 50)
	hinzu(t, d2, lupeOhneStrich(document.R(document.Pt(1, 1), document.Pt(3, 3)), document.R(document.Pt(200, 200), document.Pt(204, 204))))
	if got := Aufwand(d2); got != 0 {
		t.Errorf("Lupe außerhalb kostet %v", got)
	}
}

// TestLupeDeckkraft: Gruppendeckkraft 50 % über das Bild (ADR-0032 Punkt 10).
func TestLupeDeckkraft(t *testing.T) {
	d := weissesDokument(t, 40, 30)
	if err := d.Schwaerzen(image.Rect(2, 2, 6, 6), document.Schwaerzung{Farbe: color.NRGBA{A: 255}}); err != nil {
		t.Fatal(err)
	}
	l := lupeOhneStrich(document.R(document.Pt(2, 2), document.Pt(6, 6)), document.R(document.Pt(20, 10), document.Pt(28, 18)))
	l.Ebene = l.Ebene.MitDeckkraft(50)
	hinzu(t, d, l)
	b := bild(t, d)
	if got := b.NRGBAAt(22, 12); got.R < 120 || got.R > 135 || got.A != 255 {
		t.Errorf("halbe Deckkraft: %v", got)
	}
}
