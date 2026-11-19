// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"math"
	"strings"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"github.com/Andiministrator/schnapp/document"
)

// zaehle rendert und liefert den Aufwand samt Zählung der gerasterten Glyphen (Zählnahtstelle ohne Uhr).
func zaehle(t *testing.T, d *document.Dokument) aufwand {
	t.Helper()
	_, a, err := bildMitAufwand(d)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// F-118/F-127: Was das Bild nicht schneidet, wird nicht gerastert — gezählt, nicht gemessen.
func TestTextAusserhalbRastertKeineGlyphe(t *testing.T) {
	for _, ort := range []document.Punkt{document.Pt(50000, 50000), document.Pt(-900000, 5), document.Pt(5, -900000), document.Pt(500000, 5)} {
		d := weissesDokument(t, 10, 10)
		hinzu(t, d, document.Text{Ort: ort, Inhalt: strings.Repeat("Wg", 100), Schriftgrad: 4096, Farbe: rot})
		vor := append([]byte(nil), d.Ansichtsbild().Pix...)
		b, a, err := bildMitAufwand(d)
		if err != nil {
			t.Fatal(err)
		}
		if a.glyphen != 0 {
			t.Errorf("Text bei %v: %d Glyphen gerastert, Soll 0", ort, a.glyphen)
		}
		if !bytes.Equal(b.Pix, vor) {
			t.Errorf("Text bei %v verändert das Bild", ort)
		}
	}
}

// Zeilen über oder unter dem Bild entfallen (senkrechtes Überspringen): Bild 100×30, Schriftgrad 20, Grundlinien bei
// 20, 46, 72 … — nur die ersten beiden Zeilen reichen ins Bild.
func TestTextZeilenAusserhalbEntfallen(t *testing.T) {
	d := weissesDokument(t, 100, 30)
	hinzu(t, d, document.Text{Ort: document.Pt(0, 0), Inhalt: strings.Repeat("a\n", 99) + "a", Schriftgrad: 20, Farbe: rot})
	if a := zaehle(t, d); a.glyphen != 2 {
		t.Errorf("%d Glyphen gerastert, Soll 2 (nur sichtbare Zeilen)", a.glyphen)
	}
	e := weissesDokument(t, 100, 30)
	hinzu(t, e, document.Text{Ort: document.Pt(0, -5000), Inhalt: strings.Repeat("a\n", 99) + "a", Schriftgrad: 20, Farbe: rot})
	if a := zaehle(t, e); a.glyphen != 0 {
		t.Errorf("Zeilen über dem Bild: %d Glyphen, Soll 0", a.glyphen)
	}
}

// Ein Text, der links aus dem Bild ragt, rastert nur den Teil, der ins Bild (plus Rand) reicht; rechts endet er früher.
func TestTextHalbSichtbarRastertNurSichtbares(t *testing.T) {
	d := weissesDokument(t, 60, 30)
	hinzu(t, d, document.Text{Ort: document.Pt(-300, 2), Inhalt: strings.Repeat("H", 40), Schriftgrad: 20, Farbe: rot})
	a := zaehle(t, d)
	// Fenster −25 … 85 bei rund 14,6 px Vorschub: 7 bis 8 Glyphen.
	if a.glyphen < 6 || a.glyphen > 9 {
		t.Errorf("%d Glyphen gerastert, Soll 6 bis 9 von 40", a.glyphen)
	}
	r := weissesDokument(t, 60, 30)
	hinzu(t, r, document.Text{Ort: document.Pt(5, 2), Inhalt: strings.Repeat("H", 400), Schriftgrad: 20, Farbe: rot})
	if a := zaehle(t, r); a.glyphen < 6 || a.glyphen > 9 {
		t.Errorf("rechts abgeschnitten: %d Glyphen, Soll 6 bis 9 von 400", a.glyphen)
	}
}

// F-124: Pixelgleich zu font.Drawer.DrawString ohne Überspringen — auch wenn der übersprungene Teil Zeichen ohne Glyphe
// enthält (Emoji, CJK; Vorschub des Ersatzkästchens) oder sehr breite Zeichen (₧ ist 1,055 em breit).
func TestTextLinksAusserhalbPixelgleichZuDrawString(t *testing.T) {
	texte := []string{"A😀V😀A", "漢字AV漢VA", "₧₧W", "😀 Hinweis", "😀😀😀😀😀😀😀😀😀😀 Hinweis"}
	for _, grad := range []float64{20, 33} {
		for _, inhalt := range texte {
			for _, x := range []float64{-37.3, -45.1, -61.7, -88.2, -120.4, -300.5} {
				vergleiche(t, inhalt, grad, x)
			}
		}
		// Randfall: ₧ (1,055 em breit) knapp links der Grenze, in feinen Schritten über den Bereich −1 … −1,1 Schriftgrad.
		for i := 0; i <= 20; i++ {
			vergleiche(t, "₧", grad, -grad*(1+0.005*float64(i)))
			vergleiche(t, "₧W", grad, -grad*(1+0.005*float64(i)))
		}
	}
}

// vergleiche zeichnet inhalt mit Bild und mit font.Drawer.DrawString (ohne Überspringen) und verlangt gleiche Pixel.
func vergleiche(t *testing.T, inhalt string, grad, x float64) {
	t.Helper()
	d := weissesDokument(t, 60, 50)
	hinzu(t, d, document.Text{Ort: document.Pt(x, 3.7), Inhalt: inhalt, Schriftgrad: grad, Farbe: rot})
	got := bild(t, d)

	ref := weissesDokument(t, 60, 50)
	z := zeichner{d: ref, faces: map[faceSchluessel]font.Face{}}
	defer z.schliessen()
	f, err := z.schrift(false, grad)
	if err != nil {
		t.Fatal(err)
	}
	soll := ref.Ansichtsbild()
	dr := font.Drawer{Dst: soll, Src: image.NewUniform(color.NRGBA{R: 255, A: 255}), Face: f,
		Dot: fixed.Point26_6{X: fixed.Int26_6(math.Round(x * 64)), Y: fixed.Int26_6(math.Round((3.7 + grad) * 64))}}
	dr.DrawString(inhalt)
	if !bytes.Equal(got.Pix, soll.Pix) {
		t.Errorf("%q Grad %v bei x=%v weicht von DrawString ab", inhalt, grad, x)
	}
}

// F-125: Viele große Glyphen überschreiten das Budget → ErrZuAufwendig, und es wird nicht weiter gerastert.
func TestBudgetBegrenztGlyphen(t *testing.T) {
	alt := budget
	budget = 1e8 // ~6 Glyphen bei Grad 4096
	defer func() { budget = alt }()
	d := weissesDokument(t, 10, 10)
	for i := 0; i < 50; i++ {
		hinzu(t, d, document.Text{Ort: document.Pt(0, 0), Inhalt: "W", Schriftgrad: 4096, Farbe: rot})
	}
	_, a, err := bildMitAufwand(d)
	if !errors.Is(err, ErrZuAufwendig) {
		t.Fatalf("Fehler %v, Soll ErrZuAufwendig", err)
	}
	if a.glyphen > 8 {
		t.Errorf("%d Glyphen gerastert, das Budget hätte bei etwa 6 stoppen müssen", a.glyphen)
	}
	if _, err := Bild(d); !errors.Is(err, ErrZuAufwendig) {
		t.Errorf("Bild: %v", err)
	}
}

// Nummern mit großem Radius (Kreis plus Ziffer) und große Füllungen fallen unter dasselbe Budget.
func TestBudgetNummernUndFuellungen(t *testing.T) {
	alt := budget
	budget = 1e8
	defer func() { budget = alt }()
	d := weissesDokument(t, 10, 10)
	for i := 0; i < 50; i++ {
		hinzu(t, d, document.Nummer{Mitte: document.Pt(5, 5), Radius: 4096, Wert: 7, Farbe: blau})
	}
	if _, err := Bild(d); !errors.Is(err, ErrZuAufwendig) {
		t.Errorf("Nummern: %v", err)
	}
	e := weissesDokument(t, 2000, 2000)
	for i := 0; i < 50; i++ {
		hinzu(t, e, document.Kasten{Bereich: document.R(document.Pt(0, 0), document.Pt(2000, 2000)), Fuellung: rot})
	}
	if _, err := Bild(e); !errors.Is(err, ErrZuAufwendig) {
		t.Errorf("Füllungen: %v", err)
	}
}

// Das echte Budget ist für Dokumente, die in der Praxis vorkommen, weit entfernt: 200 Texte à 50 Zeichen Grad 24 und
// 100 Nummern auf 3840×2160.
func TestRealistischesDokumentLiegtWeitUnterDemBudget(t *testing.T) {
	d := weissesDokument(t, 3840, 2160)
	for i := 0; i < 200; i++ {
		hinzu(t, d, document.Text{Ort: document.Pt(10, float64(i*10)), Inhalt: strings.Repeat("Hallo Welt ", 5)[:50], Schriftgrad: 24, Farbe: rot})
	}
	for i := 0; i < 100; i++ {
		hinzu(t, d, document.Nummer{Mitte: document.Pt(100, float64(20*i)), Radius: 20, Wert: i, Farbe: blau})
	}
	a := zaehle(t, d)
	if a.summe > Budget/100 {
		t.Errorf("Aufwand %.3g, Soll weniger als ein Hundertstel des Budgets %.3g", a.summe, Budget)
	}
	if a.glyphen < 10000 {
		t.Errorf("nur %d Glyphen gerastert", a.glyphen)
	}
}
