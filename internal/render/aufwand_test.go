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

	"github.com/Andiministrator/schnapp/internal/document"
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

// gemischtesDokument enthält jede Objektart, sichtbar, angeschnitten und außerhalb, mit breiten Strichen, Eckradius,
// Texthintergrund, Tabulator und Zeilen über und unter dem Bild.
func gemischtesDokument(t *testing.T) *document.Dokument {
	t.Helper()
	halb := color.NRGBA{R: 20, G: 200, B: 40, A: 120}
	d := weissesDokument(t, 320, 200)
	for i := 0; i < 6; i++ {
		f := float64(i)
		hinzu(t, d, document.Pfeil{A: document.Pt(-30+f*17, 5+f*9), B: document.Pt(300-f*11, 190-f*3), K: document.Pt(150, f*20),
			Strich: document.Strich{Farbe: halb, Breite: 0.5 + f*1.7}})
		hinzu(t, d, document.Kasten{Bereich: document.R(document.Pt(-20+f*30, -10+f*20), document.Pt(100+f*25, 80+f*15)),
			Eckradius: f * 5, Fuellung: halb, Strich: document.Strich{Farbe: blau, Breite: f}})
		hinzu(t, d, document.Nummer{Mitte: document.Pt(-40+f*60, -30+f*45), Radius: 3 + f*9, Wert: i * 1370, Farbe: blau})
	}
	hinzu(t, d, document.Pfeil{A: document.Pt(-5000, -5000), B: document.Pt(5000, 5000), K: document.Pt(5000, -5000),
		Strich: document.Strich{Farbe: rot, Breite: 3000}})
	hinzu(t, d, document.Nummer{Mitte: document.Pt(-2000, 100), Radius: 900, Wert: 1000000, Farbe: rot})
	hinzu(t, d, document.Text{Ort: document.Pt(-35, 3), Inhalt: "A\tB😀C\r\nzweite Zeile\n\tdritte", Schriftgrad: 17, Farbe: rot, Hintergrund: halb})
	hinzu(t, d, document.Text{Ort: document.Pt(250, 100), Inhalt: "rechts raus WWWWW", Schriftgrad: 30, Farbe: blau})
	hinzu(t, d, document.Text{Ort: document.Pt(400, 100), Inhalt: "ganz rechts", Schriftgrad: 30, Farbe: blau})
	hinzu(t, d, document.Text{Ort: document.Pt(10, -60), Inhalt: "oben\nmitte\nunten\n\n\n\n\n\n\n\n\nweg", Schriftgrad: 25, Farbe: rot})
	return d
}

// Der Trockenlauf (Aufwand) zählt dieselben Kosten wie das echte Rendern, Teil für Teil (Karte #71).
func TestTrockenlaufGleichRendern(t *testing.T) {
	d := gemischtesDokument(t)
	trocken := trockenlauf(d, false)
	_, echt, err := bildMitAufwand(d)
	if err != nil {
		t.Fatal(err)
	}
	if trocken.glyph != echt.glyph || trocken.flaeche != echt.flaeche || trocken.kanten != echt.kanten {
		t.Errorf("Trockenlauf %+v, Rendern %+v", trocken, echt)
	}
	if trocken.glyph <= 0 || trocken.flaeche <= 0 || trocken.kanten <= 0 {
		t.Errorf("Teil ohne Kosten: %+v", trocken)
	}
	if Aufwand(d) != echt.summe() {
		t.Errorf("Aufwand %v, Summe beim Rendern %v", Aufwand(d), echt.summe())
	}
	if trocken.glyphen != 0 || echt.glyphen == 0 {
		t.Errorf("Glyphen: Trockenlauf %d (Soll 0), Rendern %d", trocken.glyphen, echt.glyphen)
	}
}

// F-131: Bild bricht über dem Budget nicht ab — alle Glyphen und Füllungen werden gezeichnet, nur Pruefe lehnt ab.
func TestBildBrichtUeberDemBudgetNichtAb(t *testing.T) {
	alt := grenzen
	grenzen.gesamt, grenzen.glyph, grenzen.kanten = 1, 1, 1
	defer func() { grenzen = alt }()
	d := weissesDokument(t, 100, 40)
	for i := 0; i < 3; i++ {
		hinzu(t, d, document.Text{Ort: document.Pt(float64(10+30*i), 0), Inhalt: "W", Schriftgrad: 20, Farbe: rot})
	}
	hinzu(t, d, document.Kasten{Bereich: document.R(document.Pt(80, 20), document.Pt(98, 38)), Fuellung: blau})
	if err := Pruefe(d); !errors.Is(err, ErrZuAufwendig) {
		t.Fatalf("Pruefe: %v, Soll ErrZuAufwendig", err)
	}
	b, a, err := bildMitAufwand(d)
	if err != nil {
		t.Fatalf("Bild über dem Budget: %v", err)
	}
	if a.glyphen != 3 {
		t.Errorf("%d Glyphen gerastert, Soll 3 (kein Abbruch)", a.glyphen)
	}
	pruefe(t, b, map[image.Point]color.NRGBA{{90, 30}: blau})
	if _, err := Bild(d); err != nil {
		t.Errorf("Bild: %v", err)
	}
}

// Jede Grenze von Pruefe wirkt für sich: Glyphen, Kanten, Summe (je knapp darunter erlaubt, darüber abgelehnt).
func TestPruefeGrenzenEinzeln(t *testing.T) {
	alt := grenzen
	defer func() { grenzen = alt }()
	d := gemischtesDokument(t)
	a := trockenlauf(d, false)
	for _, fall := range []struct {
		name   string
		setze  func(f float64)
		anteil float64
	}{
		{"gesamt", func(f float64) { grenzen.gesamt = f }, a.summe()},
		{"glyph", func(f float64) { grenzen.glyph = f }, a.glyph},
		{"kanten", func(f float64) { grenzen.kanten = f }, a.kanten},
	} {
		grenzen = alt
		fall.setze(fall.anteil)
		if err := Pruefe(d); err != nil {
			t.Errorf("%s: genau auf der Grenze abgelehnt: %v", fall.name, err)
		}
		fall.setze(fall.anteil * 0.999)
		if err := Pruefe(d); !errors.Is(err, ErrZuAufwendig) {
			t.Errorf("%s: über der Grenze %v, Soll ErrZuAufwendig", fall.name, err)
		}
	}
}

// F-132: Kosten, die nicht von der Fläche abhängen, zählen — Grundkosten winziger Glyphen und Kanten breiter Striche
// außerhalb eines winzigen Bildes.
func TestKostenmodellLueckenF132(t *testing.T) {
	d := weissesDokument(t, 10, 10)
	hinzu(t, d, document.Text{Ort: document.Pt(0, 0), Inhalt: strings.Repeat("W", 100), Schriftgrad: 0.01, Farbe: rot})
	if a := trockenlauf(d, false); a.glyph != 100*glyphGrund {
		t.Errorf("100 winzige Glyphen kosten %v, Soll %v", a.glyph, 100*glyphGrund)
	}
	e := weissesDokument(t, 10, 10)
	hinzu(t, e, document.Pfeil{A: document.Pt(-50, -50), B: document.Pt(70, 70), K: document.Pt(35, 3),
		Strich: document.Strich{Farbe: rot, Breite: 1e5}})
	a := trockenlauf(e, false)
	// Gemessen (Messtest Karte #71): rund 9 ms Rechenzeit je solchem Pfeil, also über 1e7 Einheiten, fast alles Kanten.
	if a.kanten < 1e7 || a.flaeche > 1e5 {
		t.Errorf("breiter Pfeil auf 10×10: %+v, Soll Kanten über 1e7 bei winziger Fläche", a)
	}
}

// alltagsDokument ist ein großes, aber alltagsnahes Dokument (F-131): zwei 4K-Monitore, 50 bildgroße Rahmen, 50
// diagonale Pfeile über das ganze Bild, 100 Nummern, 200 Texte à 50 Zeichen mit Hintergrund.
func alltagsDokument(t testing.TB) *document.Dokument {
	t.Helper()
	d := weissesDokument(t, 7680, 2160)
	for i := 0; i < 50; i++ {
		f := float64(i)
		hinzu(t, d, document.Kasten{Bereich: document.R(document.Pt(5+f, 5+f), document.Pt(7675-f, 2155-f)),
			Strich: document.Strich{Farbe: rot, Breite: 4}})
	}
	for i := 0; i < 50; i++ {
		a, b := document.Pt(5, 5+float64(i)), document.Pt(7670, 2150)
		hinzu(t, d, document.Pfeil{A: a, B: b, K: document.Pt((a.X+b.X)/2, (a.Y+b.Y)/2), Strich: document.Strich{Farbe: rot, Breite: 6}})
	}
	for i := 0; i < 100; i++ {
		hinzu(t, d, document.Nummer{Mitte: document.Pt(100+float64(i*70), 300+float64(i*15)), Radius: 20, Wert: i + 1, Farbe: blau})
	}
	for i := 0; i < 200; i++ {
		hinzu(t, d, document.Text{Ort: document.Pt(30+float64(i*30), float64(i*10)), Inhalt: strings.Repeat("Hallo Welt ", 5)[:50],
			Schriftgrad: 24, Farbe: rot, Hintergrund: color.NRGBA{R: 255, G: 255, A: 200}})
	}
	return d
}

// F-131: Das alltagsnahe Dokument liegt weit unter jeder Grenze (Abstände aus der Messung, siehe Budget).
func TestAlltagsdokumentWeitUnterDemBudget(t *testing.T) {
	d := alltagsDokument(t)
	a := trockenlauf(d, false)
	if f := Budget / a.summe(); f < 3 {
		t.Errorf("Budget nur %.2f-mal so groß wie das Alltagsdokument (%.3g), Soll mindestens 3", f, a.summe())
	}
	if f := GlyphBudget / a.glyph; f < 50 {
		t.Errorf("GlyphBudget nur %.1f-mal so groß wie die Glyphen (%.3g), Soll mindestens 50", f, a.glyph)
	}
	if f := KantenBudget / a.kanten; f < 100 {
		t.Errorf("KantenBudget nur %.1f-mal so groß wie die Kanten (%.3g), Soll mindestens 100", f, a.kanten)
	}
	if err := Pruefe(d); err != nil {
		t.Error(err)
	}
}

// F-125, F-132: Präparierte Dokumente überschreiten eine Grenze und werden beim Öffnen abgelehnt (nur gerechnet,
// nicht gerastert). Die bildgroßen Kästen stehen auf 2000×2000 statt 8000×8000 (Speicher im Test), mit
// entsprechend mehr Kästen.
func TestPraeparierteDokumenteUeberDemBudget(t *testing.T) {
	viele := func(w, h, n int, o document.Objekt) *document.Dokument {
		d := weissesDokument(t, w, h)
		for i := 0; i < n; i++ {
			hinzu(t, d, o)
		}
		return d
	}
	w := strings.Repeat("W", document.MaxTextZeichen)
	for name, d := range map[string]*document.Dokument{
		// Grundlinie bei y = 6, Tinte (beginnt rund 49 px rechts vom Ort) reicht ins Bild; bei Ort (0, 0) läge das W
		// ganz unter und neben dem Bild und wäre unsichtbar (F-181).
		"10000 W Grad 4096": viele(10, 10, document.MaxObjekte,
			document.Text{Ort: document.Pt(-45, 6-4096), Inhalt: "W", Schriftgrad: 4096, Farbe: rot}),
		"10000 Nummern Radius 4096": viele(10, 10, document.MaxObjekte,
			document.Nummer{Mitte: document.Pt(5, 5), Radius: 4096, Wert: 7, Farbe: blau}),
		"10000 Pfeile Breite 1e5": viele(10, 10, document.MaxObjekte,
			document.Pfeil{A: document.Pt(-50, -50), B: document.Pt(70, 70), K: document.Pt(35, 3), Strich: document.Strich{Farbe: rot, Breite: 1e5}}),
		"10000 Texte à 10000 W Grad 0,01": viele(10, 10, document.MaxObjekte,
			document.Text{Ort: document.Pt(0, 0), Inhalt: w, Schriftgrad: 0.01, Farbe: rot}),
		"2000 Kästen 2000×2000": viele(2000, 2000, 2000,
			document.Kasten{Bereich: document.R(document.Pt(0, 0), document.Pt(2000, 2000)), Fuellung: rot, Strich: document.Strich{Farbe: blau, Breite: 4}}),
		// Je Kasten zwei bildgroße Füllungen (Fläche, Rand) à 2000² × fuellGewicht = 2e8: 630 Kästen = 1,26e11 > Budget.
		"630 Kästen 2000×2000 (knapp über Budget)": viele(2000, 2000, 630,
			document.Kasten{Bereich: document.R(document.Pt(0, 0), document.Pt(2000, 2000)), Fuellung: rot, Strich: document.Strich{Farbe: blau, Breite: 4}}),
	} {
		if err := Pruefe(d); !errors.Is(err, ErrZuAufwendig) {
			t.Errorf("%s: %v (Aufwand %+v), Soll ErrZuAufwendig", name, err, trockenlauf(d, false))
		}
	}
}

// Ein realistisches Dokument rastert alle sichtbaren Glyphen: 200 Texte à 50 Zeichen Grad 24 und 100 Nummern auf
// 3840×2160.
func TestRealistischesDokumentRastertAlleGlyphen(t *testing.T) {
	d := weissesDokument(t, 3840, 2160)
	for i := 0; i < 200; i++ {
		hinzu(t, d, document.Text{Ort: document.Pt(10, float64(i*10)), Inhalt: strings.Repeat("Hallo Welt ", 5)[:50], Schriftgrad: 24, Farbe: rot})
	}
	for i := 0; i < 100; i++ {
		hinzu(t, d, document.Nummer{Mitte: document.Pt(100, float64(20*i)), Radius: 20, Wert: i, Farbe: blau})
	}
	a := zaehle(t, d)
	if a.summe() > Budget/1000 {
		t.Errorf("Aufwand %.3g, Soll weniger als ein Tausendstel des Budgets %.3g", a.summe(), Budget)
	}
	if a.glyphen < 10000 {
		t.Errorf("nur %d Glyphen gerastert", a.glyphen)
	}
}
