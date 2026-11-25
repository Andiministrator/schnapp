// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"golang.org/x/image/font"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fonts"
)

// tintenzeilen liefert die Zeilen von b, in denen mindestens ein Pixel nicht weiß ist, als Bänder zusammenhängender
// Zeilen [von, bis] (beide enthalten).
func tintenzeilen(b *image.NRGBA) [][2]int {
	var baender [][2]int
	for y := b.Bounds().Min.Y; y < b.Bounds().Max.Y; y++ {
		tinte := false
		for x := b.Bounds().Min.X; x < b.Bounds().Max.X; x++ {
			if b.NRGBAAt(x, y) != weiss {
				tinte = true
				break
			}
		}
		switch {
		case tinte && len(baender) > 0 && baender[len(baender)-1][1] == y-1:
			baender[len(baender)-1][1] = y
		case tinte:
			baender = append(baender, [2]int{y, y})
		}
	}
	return baender
}

// F-111: Die Grundlinie einer Textzeile liegt bei Ort.Y + Schriftgrad. „H“ hat keine Unterlänge: Die letzte Zeile mit
// Tinte ist die Zeile direkt über der Grundlinie, und die Oberkante liegt eine Kapitälchenhöhe (rund 0,7 Schriftgrad)
// darüber.
func TestTextGrundlinie(t *testing.T) {
	for _, fall := range []struct {
		ort  document.Punkt
		grad float64
	}{{document.Pt(10, 10), 40}, {document.Pt(4, 7), 24}, {document.Pt(3, 0), 60}} {
		d := weissesDokument(t, 100, 100)
		hinzu(t, d, document.Text{Ort: fall.ort, Inhalt: "H", Schriftgrad: fall.grad, Farbe: rot})
		baender := tintenzeilen(bild(t, d))
		if len(baender) != 1 {
			t.Fatalf("Ort %v Grad %v: %d Bänder %v, Soll 1", fall.ort, fall.grad, len(baender), baender)
		}
		grundlinie := int(fall.ort.Y + fall.grad)
		if baender[0][1] != grundlinie-1 {
			t.Errorf("Ort %v Grad %v: letzte Tintenzeile %d, Soll %d (Grundlinie %d)", fall.ort, fall.grad, baender[0][1], grundlinie-1, grundlinie)
		}
		if hoehe := float64(baender[0][1]-baender[0][0]+1) / fall.grad; hoehe < 0.6 || hoehe > 0.8 {
			t.Errorf("Ort %v Grad %v: Buchstabenhöhe %.2f Schriftgrade, Soll 0,6 bis 0,8", fall.ort, fall.grad, hoehe)
		}
	}
}

// F-111: Der Zeilenabstand beträgt 1,3 Schriftgrade: Die Unterkanten aufeinanderfolgender „H“-Zeilen liegen 26
// (Grad 20), 52 (Grad 40) bzw. 78 Pixel (Grad 60) auseinander.
func TestTextZeilenabstand(t *testing.T) {
	for _, fall := range []struct {
		grad    float64
		abstand int
	}{{20, 26}, {40, 52}, {60, 78}} {
		d := weissesDokument(t, 100, 300)
		hinzu(t, d, document.Text{Ort: document.Pt(5, 5), Inhalt: "H\nH\nH", Schriftgrad: fall.grad, Farbe: rot})
		baender := tintenzeilen(bild(t, d))
		if len(baender) != 3 {
			t.Fatalf("Grad %v: %d Bänder %v, Soll 3", fall.grad, len(baender), baender)
		}
		if a, b := baender[1][1]-baender[0][1], baender[2][1]-baender[1][1]; a != fall.abstand || b != fall.abstand {
			t.Errorf("Grad %v: Zeilenabstand %d und %d, Soll %d", fall.grad, a, b, fall.abstand)
		}
		if erste := int(5 + fall.grad); baender[0][1] != erste-1 {
			t.Errorf("Grad %v: erste Grundlinie bei %d, Soll %d", fall.grad, baender[0][1]+1, erste)
		}
	}
}

// F-111: Zuschnitt und Streifen verschieben, was gezeichnet wird, Pixel für Pixel. Ansicht: Original 100×40, Spalten
// 20 bis 40 herausgeschnitten (Ansicht 80 breit), Zeilen 0 bis 5 auch (35 hoch).
func TestStreifenVerschiebenObjekte(t *testing.T) {
	d := weissesDokument(t, 100, 40)
	// Liegt rechts vom Streifen: Original 60 bis 90 wird Ansicht 40 bis 70; Zeilen 5 bis 35 werden 0 bis 30.
	hinzu(t, d, document.Kasten{Bereich: document.R(document.Pt(60, 5), document.Pt(90, 35)), Fuellung: rot})
	// Überspannt den Streifen: Original 2 bis 58 wird Ansicht 2 bis 38 (der Streifen fällt weg).
	hinzu(t, d, document.Kasten{Bereich: document.R(document.Pt(2, 10), document.Pt(58, 20)), Fuellung: blau})
	// Liegt ganz im Streifen: beide Kanten fallen auf Ansicht 20, nichts bleibt.
	hinzu(t, d, document.Kasten{Bereich: document.R(document.Pt(25, 30), document.Pt(35, 40)), Fuellung: color.NRGBA{G: 255, A: 255}})
	if err := d.SetzeStreifen([]document.Streifen{{Zeilen: false, Von: 20, Bis: 40}, {Zeilen: true, Von: 0, Bis: 5}}); err != nil {
		t.Fatal(err)
	}
	b := bild(t, d)
	if b.Bounds().Size() != image.Pt(80, 35) {
		t.Fatalf("Größe %v, Soll 80×35", b.Bounds().Size())
	}
	// Über der Naht bei x = 20 liegt die Kante (Voreinstellung Abriss, ADR-0023 Nachtrag 6 und 7, Band ±16 um die Naht, also x 4 bis 36); ihre Sollwerte prüft TestKanteSollwerte (kante_faktor_test.go).
	pruefe(t, b, map[image.Point]color.NRGBA{
		{40, 15}: rot, {69, 15}: rot, {39, 15}: weiss, {70, 15}: weiss, {55, 0}: rot, {55, 29}: rot, {55, 30}: weiss,
		{2, 7}: blau, {37, 7}: blau, {1, 7}: weiss, {38, 7}: weiss, {2, 16}: weiss, {37, 16}: weiss,
		{2, 28}: weiss, {37, 28}: weiss, // der Kasten im Streifen zeichnet nichts (auch nicht neben der Kante)
	})
	// Das Grüne darf nirgends auftauchen.
	for y := 0; y < 35; y++ {
		for x := 0; x < 80; x++ {
			if c := b.NRGBAAt(x, y); c.G == 255 && c.R == 0 && c.B == 0 {
				t.Fatalf("grüner Pixel bei (%d,%d): Objekt im Streifen gezeichnet", x, y)
			}
		}
	}
}

// F-111: Ein halbdurchsichtiger Strich mischt mit dem Grund (hier Weiß): rot mit Alpha 128 ergibt (255, 127, 127).
// Auch an Ecken und Stößen der Strecken und Kreise gilt dieselbe Deckung wie auf einer geraden Strecke — alles wird in
// einem Durchgang gleich ausgerichtet gefüllt, überlappende Teile addieren sich nicht auf (und heben sich nicht auf).
func TestHalbtransparenterStrich(t *testing.T) {
	halb := color.NRGBA{R: 255, A: 128}
	soll := color.NRGBA{R: 255, G: 127, B: 127, A: 255}
	d := weissesDokument(t, 80, 80)
	hinzu(t, d, document.Kasten{Bereich: document.R(document.Pt(20, 20), document.Pt(60, 60)), Strich: document.Strich{Farbe: halb, Breite: 8}})
	// Ein gebogener Pfeil im gedrehten Umlaufsinn: mehrere Strecken mit Stößen.
	b := bild(t, d)
	pruefe(t, b, map[image.Point]color.NRGBA{
		{40, 20}: soll, {40, 18}: soll, {40, 23}: soll, // Mitte der oberen Kante
		{20, 20}: soll, {19, 19}: soll, {22, 22}: soll, // Ecke oben links, Strecken und Kreise überlagern sich
		{59, 59}: soll, {61, 61}: soll, {60, 20}: soll, {20, 60}: soll, // übrige Ecken
		{40, 40}: weiss, {5, 5}: weiss, // Inneres frei, außen frei
	})
	// Die Zeile durch die obere Kante hat überall außerhalb der Ecken dieselbe Farbe.
	for x := 24; x < 56; x++ {
		if c := b.NRGBAAt(x, 21); c != soll {
			t.Errorf("Pixel (%d,21) = %v, Soll %v", x, c, soll)
		}
	}
	// Dasselbe mit einer Füllung unter dem Strich: Der Strich mischt über die Füllung, die Füllung bleibt unberührt.
	e := weissesDokument(t, 80, 80)
	hinzu(t, e, document.Kasten{Bereich: document.R(document.Pt(20, 20), document.Pt(60, 60)), Fuellung: blau,
		Strich: document.Strich{Farbe: halb, Breite: 8}})
	be := bild(t, e)
	if c := be.NRGBAAt(40, 40); c != blau {
		t.Errorf("Füllung innen %v, Soll %v", c, blau)
	}
	// Rot (Alpha 128) über Blau: R = 128, B = 127 (Gegenstück zu 127 über Weiß), an Kante und Ecke gleich.
	gemischt := color.NRGBA{R: 128, B: 127, A: 255}
	pruefe(t, be, map[image.Point]color.NRGBA{{40, 21}: gemischt, {20, 20}: gemischt, {22, 22}: gemischt, {59, 59}: gemischt})
}

// F-133: Rechter Rand der Glyphen in text: Eine Glyphe, deren Ursprung bis 1,25 Schriftgrade rechts vom Bild liegt,
// wird gezeichnet, eine dahinter nicht. Bild 60 breit, Grad 20: Grenze bei 85. Der Text beginnt links vom Bild, damit
// die Zeile selbst sichtbar bleibt (zeileSichtbar, Testhilfe in zeichnerhilfen_test.go, prüft nur den Anfang).
func TestTextRechterRand(t *testing.T) {
	d := weissesDokument(t, 60, 30)
	z := zeichner{d: d, faces: map[faceSchluessel]font.Face{}}
	defer z.schliessen()
	f, err := z.schrift(fonts.Laden(fonts.Go, fonts.Normal), 20)
	if err != nil {
		t.Fatal(err)
	}
	adv := float64(font.MeasureString(f, "H")) / 64
	for _, fall := range []struct {
		ursprung float64 // Ort der vierten Glyphe (Index 3)
		soll     int     // gerasterte Glyphen
	}{{84, 4}, {85, 4}, {86.5, 3}, {88, 3}, {92, 3}} {
		d := weissesDokument(t, 60, 30)
		hinzu(t, d, document.Text{Ort: document.Pt(fall.ursprung-3*adv, 2), Inhalt: strings.Repeat("H", 8), Schriftgrad: 20, Farbe: rot})
		if a := zaehle(t, d); a.glyphen != fall.soll {
			t.Errorf("vierte Glyphe bei %v: %d Glyphen gerastert, Soll %d (Rand bei 85)", fall.ursprung, a.glyphen, fall.soll)
		}
	}
}

// F-133: Die Ziffern einer Nummer kosten nichts, wenn ihr Kasten das Bild nicht schneidet. Kasten: halbe Breite
// Schriftgrad·(0,4·Stellen + 1), Schriftgrad 1,2·Radius. Seit F-181 fällt eine Nummer, deren Kreis und Ziffern ganz
// neben dem Bild liegen, schon vorher weg (ObjektSichtbar nach der echten Tinte): Radius 20, eine Stelle — Mitte bei
// −15 reicht der Kreis ins Bild (Kosten 1700, obere Schranke), ab −20 liegt alles daneben (Kosten 0; vor F-181 zählte
// der Kasten bis −33 noch Kosten für Ziffern, die nichts zeichneten).
func TestNummerZiffernKostenNurImBild(t *testing.T) {
	for _, fall := range []struct {
		mitte float64
		soll  float64
	}{{-15, glyphGrund}, {-19, glyphGrund}, {-20, 0}, {-30, 0}, {-33, 0}, {-40, 0}, {-1e6, 0}, {50, glyphGrund}} {
		d := weissesDokument(t, 100, 100)
		hinzu(t, d, document.Nummer{Mitte: document.Pt(fall.mitte, 50), Radius: 20, Wert: 7, Farbe: blau})
		if a := trockenlauf(d, false); a.glyph != fall.soll {
			t.Errorf("Mitte %v: Ziffernkosten %v, Soll %v", fall.mitte, a.glyph, fall.soll)
		}
		// Das Rendern zählt dieselben Kosten; gerastert wird nur im Bild (die Kosten sind eine obere Schranke).
		if a := zaehle(t, d); a.glyph != fall.soll || (fall.mitte == 50 && a.glyphen != 1) || (fall.soll == 0 && a.glyphen != 0) {
			t.Errorf("Mitte %v: Rendern zählt Kosten %v und %d Ziffern", fall.mitte, a.glyph, a.glyphen)
		}
	}
	// Zwei Stellen: halbe Breite 24·(0,8+1) = 43,2 — bei −19 reicht der Kreis ins Bild und der Kasten schneidet es, bei
	// −40 und −50 liegt die Nummer ganz daneben.
	for _, fall := range []struct {
		mitte float64
		soll  float64
	}{{-19, 2 * glyphGrund}, {-40, 0}, {-50, 0}} {
		d := weissesDokument(t, 100, 100)
		hinzu(t, d, document.Nummer{Mitte: document.Pt(fall.mitte, 50), Radius: 20, Wert: 77, Farbe: blau})
		if a := trockenlauf(d, false); a.glyph != fall.soll {
			t.Errorf("zweistellig, Mitte %v: Ziffernkosten %v, Soll %v", fall.mitte, a.glyph, fall.soll)
		}
	}
}

// F-133: Kosten für Zeilen, die das Bild nicht schneiden, fallen weg — im Trockenlauf wie beim Rendern.
func TestKostenUnsichtbarerTextzeilen(t *testing.T) {
	// Bild 100×30, Grad 20: Grundlinien bei 20, 46, 72 … — nur die ersten beiden Zeilen reichen ins Bild.
	d := weissesDokument(t, 100, 30)
	hinzu(t, d, document.Text{Ort: document.Pt(0, 0), Inhalt: strings.Repeat("a\n", 99) + "a", Schriftgrad: 20, Farbe: rot})
	if a := trockenlauf(d, false); a.glyph != 2*glyphGrund {
		t.Errorf("100 Zeilen, 2 sichtbar: Kosten %v, Soll %v", a.glyph, 2*glyphGrund)
	}
	// Über dem Bild, unter dem Bild und rechts davon.
	for _, ort := range []document.Punkt{document.Pt(0, -5000), document.Pt(0, 5000), document.Pt(500, 0)} {
		e := weissesDokument(t, 100, 30)
		hinzu(t, e, document.Text{Ort: ort, Inhalt: "a\na\na", Schriftgrad: 20, Farbe: rot})
		if a := trockenlauf(e, false); a.glyph != 0 {
			t.Errorf("Text bei %v: Kosten %v, Soll 0", ort, a.glyph)
		}
	}
}

// F-133: Der Tabulator kostet vier Glyphen (Zuschlag in zeichenzahl): Zeichenzahl ist obere Schranke der gerasterten
// Glyphen, und die Kosten folgen ihr.
func TestZeichenzahlTabulatorZuschlag(t *testing.T) {
	for _, fall := range []struct {
		zeile string
		soll  int
	}{{"", 0}, {"ab", 2}, {"\t", 4}, {"a\tb", 6}, {"\t\t", 8}, {"äö😀", 3}} {
		if got := zeichenzahl(fall.zeile); got != fall.soll {
			t.Errorf("zeichenzahl(%q) = %d, Soll %d", fall.zeile, got, fall.soll)
		}
	}
	d := weissesDokument(t, 200, 40)
	hinzu(t, d, document.Text{Ort: document.Pt(5, 5), Inhalt: "\t\t", Schriftgrad: 20, Farbe: rot})
	trocken := trockenlauf(d, false)
	if trocken.glyph != 8*glyphGrund {
		t.Errorf("zwei Tabulatoren kosten %v, Soll %v (8 Glyphen)", trocken.glyph, 8*glyphGrund)
	}
	if a := zaehle(t, d); a.glyphen != 8 || a.glyph != trocken.glyph {
		t.Errorf("gerastert %d Glyphen (Soll 8), Kosten %v (Trockenlauf %v)", a.glyphen, a.glyph, trocken.glyph)
	}
}

// F-133: Sollwerte des Kostenmodells, von Hand gerechnet (Formeln in den Kommentaren), damit ein Gewicht auf 0
// nicht unbemerkt bleibt.
func TestKostenmodellSollwerte(t *testing.T) {
	// Gefüllter Kasten (10,10)–(30,30), keine Ecken: 4 Punkte. Hülle (10,10)–(31,31) = 21×21 Pixel.
	//   Fläche  = fuellGewicht · 21 · 21 = 25 · 441 = 11025
	//   Kanten  = punktGewicht · 4 + kantenGewicht · Schritte; die Schritte: zwei senkrechte Kanten à
	//             1 + 20 (Zeilen) + 2·20 (Spanne) = 61, zwei waagerechte à 1 → 124; also 400 + 3 · 124 = 772,
	//             dazu aufrufGewicht 12000 je Füllung (F-750) → 12772
	d := weissesDokument(t, 100, 100)
	hinzu(t, d, document.Kasten{Bereich: document.R(document.Pt(10, 10), document.Pt(30, 30)), Fuellung: rot})
	if a := trockenlauf(d, false); a.flaeche != 11025 || a.kanten != 12772 || a.glyph != 0 {
		t.Errorf("Kasten: %+v, Soll Fläche 11025, Kanten 12772, Glyphen 0", a)
	}
	// Nummer Mitte (50,50), Radius 20, Wert 7: Kreis mit kreisEcken(20) = 40 Ecken, Hülle (30,30)–(71,71) = 41×41.
	//   Fläche = 25 · 41 · 41 = 42025
	//   Kanten = punktGewicht · 40 + kantenGewicht · kreisSchritte; kreisSchritte = 2·40 (Aufrufe)
	//            + 2·40 (Zeilen von Rand zu Rand) + 2·(2·40 + 40) + 4·20 = 80 + 80 + 240 + 80 = 480 → 4000 + 1440 = 5440,
	//            dazu aufrufGewicht 12000 → 17440
	//   Glyphen = eine Ziffer, Grad 24 (Grundkosten 1700)
	e := weissesDokument(t, 100, 100)
	hinzu(t, e, document.Nummer{Mitte: document.Pt(50, 50), Radius: 20, Wert: 7, Farbe: blau})
	if a := trockenlauf(e, false); a.flaeche != 42025 || a.kanten != 17440 || a.glyph != glyphGrund {
		t.Errorf("Nummer: %+v, Soll Fläche 42025, Kanten 17440, Glyphen %v", a, glyphGrund)
	}
	if a := Aufwand(e); a != 42025+17440+glyphGrund {
		t.Errorf("Aufwand der Nummer %v, Soll %v", a, 42025+17440+glyphGrund)
	}
}

// F-133: kreisSchritte schätzt die Schritte des Rasterizers für die Ecken eines Kreises nach oben ab (nie weniger als
// kantenSchritte des wirklichen Polygons) — für Kreise ganz im Ausschnitt, am Rand, darüber hinaus und außerhalb.
func TestKreisSchritteSchaetztNachObenAb(t *testing.T) {
	for _, r := range []float64{0.5, 3, 8, 20, 64, 200} {
		for _, hoehe := range []int{10, 100, 700} {
			for _, oy := range []float64{0, 17.5} {
				for _, my := range []float64{-300, -r, 0, r / 2, 50, 100, 400, 1000} {
					for _, mx := range []float64{-50, 3, 60, 900} {
						k := kreisform{document.Pt(mx, my), r}
						schaetzung := kreisSchritte([]kreisform{k}, oy, hoehe)
						genau := kantenSchritte([][]document.Punkt{kreis(k.m, k.r)}, oy, hoehe)
						if schaetzung < genau {
							t.Fatalf("Kreis %v r=%v oy=%v hoehe=%d: Schätzung %v < gezählt %v", k.m, r, oy, hoehe, schaetzung, genau)
						}
					}
				}
			}
		}
	}
	// Ein Kreis im Ausschnitt kostet mehr als 2·Ecken (Aufrufe), ein weit außerhalb nur die Aufrufe.
	k := kreisform{document.Pt(50, 50), 20}
	if s := kreisSchritte([]kreisform{k}, 30, 41); s != 480 {
		t.Errorf("Kreis im Ausschnitt: %v Schritte, Soll 480", s)
	}
	if s := kreisSchritte([]kreisform{{document.Pt(50, 5000), 20}}, 0, 100); s != 80 {
		t.Errorf("Kreis weit außerhalb: %v Schritte, Soll 80 (nur die Aufrufe)", s)
	}
}
