// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"math"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fonts"
	"github.com/Andiministrator/schnapp/internal/fonts/fonttest"
)

// ladeTestkernUI macht die Testschrift (kern-Paar A,V = -0,12 em) als Systemschrift bekannt.
func ladeTestkernUI(t *testing.T) *fonts.Schrift {
	t.Helper()
	fonts.SetzeOrdner([]string{"../../testdata/fonts"})
	fonts.SucheZuruecksetzen()
	fonts.Suche()
	t.Cleanup(func() {
		fonts.SetzeOrdner(nil)
		fonts.SucheZuruecksetzen()
		fonts.Suche()
	})
	k := fonts.Laden("Schnapp Testkern", fonts.Normal)
	if k.Ersatz {
		t.Fatalf("Testschrift nicht gefunden: %v", fonts.Familien())
	}
	return k
}

// zeile zeichnet text im Grad grad ab x auf die Grundlinie 50 in eine Ansicht 0..100 und liefert die Zahl der Umrisse.
func zeile(ts *textsatz, s *fonts.Schrift, text string, x, grad float64) int {
	gtx := layout.Context{Ops: new(op.Ops)}
	return ts.zeichneZeile(gtx, s, text, x, 50, grad, color.NRGBA{A: 255}, image.Rect(0, 0, 100, 100))
}

// erwartet zählt, wie viele Glyphen mit Umriss ab x im Bereich liegen, den zeichneZeile zeichnet: Ursprung der Glyphe =
// Summe der Vorschübe und Paarunterschneidungen davor (Renderer und Messung setzen genauso).
func erwartet(t *testing.T, s *fonts.Schrift, text string, x, grad float64) int {
	t.Helper()
	face, err := opentype.NewFace(s.Font, &opentype.FaceOptions{Size: grad, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		t.Fatal(err)
	}
	_, _, seitlich := s.Huelle()
	links, rechts := -seitlich*grad, 100+seitlich*grad
	var puf sfnt.Buffer
	dot := fixed.Int26_6(math.Round(x * 64))
	n, vorher := 0, rune(-1)
	for _, r := range text {
		if vorher >= 0 {
			dot += document.KernPaar(s.Font, &puf, vorher, r, grad)
		}
		vorher = r
		px := float64(dot) / 64
		if px > rechts {
			break
		}
		a, _ := face.GlyphAdvance(r)
		if px >= links {
			gi, _ := s.Font.GlyphIndex(&puf, r)
			if seg, err := s.Font.LoadGlyph(&puf, gi, fixed.Int26_6(math.Round(grad*64)), nil); err == nil && len(seg) > 0 {
				n++
			}
		}
		dot += a
	}
	return n
}

// F-577 (textsatz.go): Die Glyphen stehen im Abstand ihrer Vorschübe und Paarunterschneidungen; was rechts hinter der
// Ansicht beginnt, entfällt. Fehlte der Vorschub oder die Unterschneidung, rückten die Glyphen zusammen oder
// auseinander, und die Zahl der gezeichneten Umrisse wäre eine andere.
func TestZeichneZeileVorschubUndUnterschneidung(t *testing.T) {
	k := ladeTestkernUI(t)
	g := fonts.Laden(fonts.Go, fonts.Normal)
	var ts textsatz
	for _, c := range []struct {
		s    *fonts.Schrift
		text string
		x    float64
		grad float64
	}{
		{g, "gggggggg", 0, 100},
		{g, "gggggggg", -150, 100},
		{k, "AVAVAVAVAVAV", 0, 100},
		{k, "AVAVAVAVAVAV", -200, 100},
		{k, "AVAVAVAVAVAV", 0, 30},
		{k, "AAAAAAAAAAAA", 0, 100},
	} {
		want := erwartet(t, c.s, c.text, c.x, c.grad)
		if got := zeile(&ts, c.s, c.text, c.x, c.grad); got != want || want == 0 {
			t.Errorf("%q ab %v im Grad %v: %d Umrisse, erwartet %d (und mehr als 0)", c.text, c.x, c.grad, got, want)
		}
	}
	// Die Unterschneidung zählt: „AV“ rückt um 0,12 em zusammen, also kommt ein Paar mehr in die Ansicht als bei „AA“-Abstand.
	if a, b := zeile(&ts, k, "AVAVAVAVAVAV", 0, 100), zeile(&ts, k, "AAAAAAAAAAAA", 0, 100); a == b {
		t.Logf("AV…: %d, AA…: %d (gleich: kein Beleg für die Unterschneidung in diesem Ausschnitt)", a, b)
	}
}

// F-577 (textsatz.go): Ein Zeichen mit unlesbaren Umrissen zeichnet nichts, schiebt die nächste Glyphe aber um seinen
// Vorschub weiter, ohne Absturz.
func TestZeichneZeileUnlesbareGlyphe(t *testing.T) {
	s := &fonts.Schrift{Font: fonttest.Defekt(t)}
	var ts textsatz
	if n := zeile(&ts, s, "AB", 0, 40); n != 1 {
		t.Errorf("„AB“: %d Umrisse, erwartet 1 (nur das B)", n)
	}
	// Sieben A schieben das B hinter die Ansicht (7 · 0,67 em · 100 = 470 > 100 + 125): Es entfällt nur, wenn der
	// Vorschub der unlesbaren Glyphen zählt.
	if n := zeile(&ts, s, "AAAAAAAB", 0, 100); n != 0 {
		t.Errorf("sieben A vor dem B: %d Umrisse, erwartet 0 (B liegt hinter der Ansicht)", n)
	}
	if n := zeile(&ts, s, "", 0, 40); n != 0 {
		t.Errorf("leere Zeile: %d Umrisse", n)
	}
}

// F-577 (textsatz.go): Die Zwischenspeicher für Faces und Umrisse werden begrenzt.
func TestTextsatzZwischenspeicherBegrenzt(t *testing.T) {
	g := fonts.Laden(fonts.Go, fonts.Normal)
	var ts textsatz
	for i := 0; i < 300; i++ {
		zeile(&ts, g, "g", 0, 10+float64(i)/8)
	}
	if len(ts.faces) > 257 {
		t.Errorf("%d Faces im Speicher, Grenze 256", len(ts.faces))
	}
	// Umrisse: mehr verschiedene (Glyphe, ppem) als die Grenze.
	var tu textsatz
	f := g.Font
	for i := 0; i < umrissGrenze+50; i++ {
		tu.umriss(f, sfnt.GlyphIndex(1+i%40), fixed.Int26_6(640+i))
	}
	if len(tu.umrisse) > umrissGrenze {
		t.Errorf("%d Umrisse im Speicher, Grenze %d", len(tu.umrisse), umrissGrenze)
	}
	if len(tu.umrisse) == 0 {
		t.Error("Speicher leer: der letzte Umriss gehört hinein")
	}
}

// F-577 (Striche im Editor): Unterstrich und Durchstrich zeichnet der Editor als gefüllte Rechtecke an den Orten, die
// document.Text.Striche liefert (wie Renderer und SVG); ohne Stil steht dort nichts.
func TestEditorZeichnetStriche(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	farbe := color.NRGBA{R: 255, A: 255}
	tx := document.Text{Ort: document.Pt(2, 1), Inhalt: "Hallo", Schriftgrad: 10, Farbe: farbe, Unterstrichen: true}
	_, err := l.e.s.D.Hinzufuegen(tx)
	if err != nil {
		t.Fatal(err)
	}
	l.setle()
	st := tx.Striche()
	if len(st) != 1 {
		t.Fatalf("%d Striche, erwartet 1", len(st))
	}
	mitte := l.e.s.D.ZuAnsicht(document.Pt((st[0].Min.X+st[0].Max.X)/2, (st[0].Min.Y+st[0].Max.Y)/2))
	pixel := func() color.RGBA {
		b := headlessBild(t, image.Pt(400, 300), func(gtx layout.Context) { l.e.zeichneObjekte(gtx) })
		return b.RGBAAt(int(mitte.X), int(mitte.Y))
	}
	if p := pixel(); p.R < 200 || p.G > 60 || p.A < 200 {
		t.Errorf("Mitte des Unterstrichs bei %v: Pixel %v, erwartet die Textfarbe Rot", mitte, p)
	}
	// Ohne Unterstrich steht dort nichts (Text „Hallo“ hat keine Unterlänge).
	l2 := neuerMenueLauf(t, image.Pt(1200, 800))
	tx.Unterstrichen = false
	if _, err := l2.e.s.D.Hinzufuegen(tx); err != nil {
		t.Fatal(err)
	}
	l2.setle()
	b := headlessBild(t, image.Pt(400, 300), func(gtx layout.Context) { l2.e.zeichneObjekte(gtx) })
	if p := b.RGBAAt(int(mitte.X), int(mitte.Y)); p.A > 30 {
		t.Errorf("ohne Unterstrich: Pixel %v, erwartet leer", p)
	}
}
