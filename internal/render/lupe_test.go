// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"errors"
	"image"
	"image/color"
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
	}{{"2x", document.R(document.Pt(20, 10), document.Pt(28, 16)), 2}, {"3x", document.R(document.Pt(20, 10), document.Pt(32, 19)), 3}} {
		d := musterDokument(t, 40, 30)
		orig := d.Original()
		l := lupeOhneStrich(document.R(document.Pt(2, 3), document.Pt(6, 6)), f.z)
		hinzu(t, d, l)
		b := bild(t, d)
		inhalt := LupenInhalt(d, l)
		for y := 0; y < 30; y++ {
			for x := 0; x < 40; x++ {
				want := orig.NRGBAAt(x, y)
				drin := image.Pt(x, y).In(image.Rect(int(f.z.Min.X), int(f.z.Min.Y), int(f.z.Max.X), int(f.z.Max.Y)))
				if drin {
					want = orig.NRGBAAt(2+(x-20)/f.k, 3+(y-10)/f.k)
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

// TestLupeVerborgenDurchsichtig: Quellpixel außerhalb des Zuschnitts und in einem Streifen sind in der Lupe
// durchsichtig, das Bild darunter bleibt; die Zielpixel liegen über ZuOriginal (ADR-0033 Punkte 2 und 4).
func TestLupeVerborgenDurchsichtig(t *testing.T) {
	// Das Ziel liegt weit unter der Naht des Streifens: Die Kante (Spalt, Fasersaum, ±16 Pixel) berührt es nicht.
	d := musterDokument(t, 40, 70)
	l := lupeOhneStrich(document.R(document.Pt(2, 3), document.Pt(8, 9)), document.R(document.Pt(20, 40), document.Pt(32, 52)))
	id, err := d.Hinzufuegen(l)
	if err != nil {
		t.Fatal(err)
	}
	// Zuschnitt ab x = 4: Quellspalten 2, 3 verborgen. Zeilenstreifen 5..7: Quellzeilen 5, 6 verborgen. Die Ansicht
	// beginnt bei Original x = 4, alles unter dem Streifen rückt um 2 nach oben: Ziel in der Ansicht 16..28 × 38..50.
	if err := d.SetzeZuschnitt(image.Rect(4, 0, 40, 70)); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 5, Bis: 7}}); err != nil {
		t.Fatal(err)
	}
	b := bild(t, d)
	orig := d.Original()
	if err := d.Entfernen(id); err != nil {
		t.Fatal(err)
	}
	grund := bild(t, d) // dieselbe Ansicht ohne Lupe
	for vy := 36; vy < 52; vy++ {
		for vx := 14; vx < 30; vx++ {
			o := d.ZuOriginal(document.Pt(float64(vx)+0.5, float64(vy)+0.5))
			sx, sy := l.QuellSpalte(o.X), l.QuellZeile(o.Y)
			want := grund.NRGBAAt(vx, vy)
			if o.X >= 20 && o.X < 32 && o.Y >= 40 && o.Y < 52 && sx >= 4 && (sy < 5 || sy >= 7) {
				want = orig.NRGBAAt(sx, sy)
			}
			if got := b.NRGBAAt(vx, vy); got != want {
				t.Errorf("Ansicht (%d,%d) = %v, erwartet %v", vx, vy, got, want)
			}
		}
	}
	// Sollwerte an Stellen: Ansicht (16,38) zeigt Quellspalte 2 (verborgen) → Grund; (20,42) Quellzeile 5 (verborgen)
	// → Grund; (20,46) Quellpixel (4,7); (26,40) Quellpixel (7,4).
	for _, p := range []image.Point{{16, 38}, {20, 42}} {
		if got, want := b.NRGBAAt(p.X, p.Y), grund.NRGBAAt(p.X, p.Y); got != want {
			t.Errorf("verborgen bei %v: %v, erwartet Grund %v", p, got, want)
		}
	}
	for p, q := range map[image.Point]image.Point{{20, 46}: {4, 7}, {26, 40}: {7, 4}} {
		if got, want := b.NRGBAAt(p.X, p.Y), orig.NRGBAAt(q.X, q.Y); got != want {
			t.Errorf("Zielpixel %v: %v, erwartet Quelle %v = %v", p, got, q, want)
		}
	}
	if _, err := d.Hinzufuegen(l); err != nil {
		t.Fatal(err)
	}
	inhalt := LupenInhalt(d, l)
	if inhalt.Rect != image.Rect(16, 38, 28, 50) {
		t.Errorf("LupenInhalt %v", inhalt.Rect)
	}
	if inhalt.NRGBAAt(16, 38).A != 0 || inhalt.NRGBAAt(20, 42).A != 0 || inhalt.NRGBAAt(20, 46) != orig.NRGBAAt(4, 7) {
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
	// Ohne Strich: nur das Bild. Ziel 60..120 × 20..60 ragt rechts und unten hinaus: beschnitten 40 × 30.
	hinzu(t, d, lupeOhneStrich(document.R(document.Pt(1, 1), document.Pt(31, 21)), document.R(document.Pt(60, 20), document.Pt(120, 60))))
	if got, want := Aufwand(d), fuellGewicht*40*30; got != want {
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
	grenzen.gesamt = fuellGewicht*40*30 - 1
	if err := Pruefe(d); !errors.Is(err, ErrZuAufwendig) {
		t.Errorf("Pruefe über dem Budget: %v", err)
	}
	grenzen.gesamt = fuellGewicht * 40 * 30
	if err := Pruefe(d); err != nil {
		t.Errorf("Pruefe am Budget: %v", err)
	}
	// Ganz außerhalb der Ansicht: kostet nichts.
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
