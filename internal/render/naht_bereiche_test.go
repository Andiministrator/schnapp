// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"errors"
	"image/color"
	"strings"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Aufwand über vielen Nähten (F-747, F-750, ADR-0023 Nachtrag 16).

// zeilenStreifen liefert n Zeilenstreifen der Breite 1 im Abstand abstand.
func zeilenStreifen(n, abstand int) []document.Streifen {
	var st []document.Streifen
	for i := 0; i < n; i++ {
		st = append(st, document.Streifen{Zeilen: true, Von: abstand * i, Bis: abstand*i + 1})
	}
	return st
}

// spaltenStreifen liefert n Spaltenstreifen der Breite 1 im Abstand abstand.
func spaltenStreifen(n, abstand int) []document.Streifen {
	var st []document.Streifen
	for i := 0; i < n; i++ {
		st = append(st, document.Streifen{Von: abstand * i, Bis: abstand*i + 1})
	}
	return st
}

// kacheln liefert 63 Spalten- und 63 Zeilenstreifen im Abstand 4: 4096 Bereiche auf 256×256.
func kacheln() []document.Streifen {
	var st []document.Streifen
	for i := 0; i < 63; i++ {
		st = append(st, document.Streifen{Von: 4 * i, Bis: 4*i + 1}, document.Streifen{Zeilen: true, Von: 4 * i, Bis: 4*i + 1})
	}
	return st
}

// F-747: 20 Texte à 10 000 Zeilen über 1024 Zeilenstreifen. Vorher lief Pruefe je Bereich über alle Zeilen (5,2 s statt
// 22 ms); jetzt sucht jede Zeile ihre Bereiche per Binärsuche. Grenze 1 s auch unter -race auf einem langsamen Runner
// (gemessen rund 30 ms ohne -race).
func TestPruefeVieleZeilenUeberVielenNaehten(t *testing.T) {
	d := weissesDokument(t, 64, 4096)
	if err := d.SetzeStreifen(zeilenStreifen(document.MaxStreifen, 2)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		hinzu(t, d, document.Text{Ort: document.Pt(5, 5), Inhalt: "x" + strings.Repeat("\n", 9998) + "x", Schriftgrad: 8,
			Farbe: qaSchwarz})
	}
	_ = Pruefe(d) // Tinte einmal vermessen (Zwischenspeicher), wie beim Öffnen vor jedem Bild
	s := time.Now()
	err := Pruefe(d)
	if dt := time.Since(s); dt > time.Second {
		t.Errorf("Pruefe dauerte %v, Grenze 1 s (vorher 5,2 s)", dt)
	}
	if err != nil {
		t.Errorf("Pruefe: %v, Soll nil (20 sichtbare Zeichen)", err)
	}
	// Sollwert: sichtbar ist je Text nur die erste Zeile ("x"); die leeren Zeilen kosten nichts, die letzte liegt weit
	// unter dem Bild.
	if a := trockenlauf(d, false); a.glyph != 20*glyphGrund {
		t.Errorf("Glyphkosten %v, Soll %v (20 Glyphen)", a.glyph, 20*glyphGrund)
	}
}

// F-750: Über 1023 Spaltenstreifen rasterte der Renderer jede Glyphe in jedem Bereich neu, in dessen Nähe sie lag
// (76 650 statt 1 740 Glyphen, 4,6 s statt 33 ms), gezählt war jede nur einmal. Jetzt wird jede Glyphe einmal gerastert
// und je Bereich verschoben eingemischt: gleiche Zahl gerasterter Glyphen mit und ohne Streifen, gleiche Kosten.
func TestGlyphenEinmalUeberVielenNaehten(t *testing.T) {
	zeile := strings.Repeat("@", 60)
	inhalt := strings.TrimSuffix(strings.Repeat(zeile+"\n", 6), "\n")
	zaehlen := func(st []document.Streifen) aufwand {
		d := weissesDokument(t, 4200, 520)
		if err := d.SetzeStreifen(st); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 5; i++ {
			hinzu(t, d, document.Text{Ort: document.Pt(4, 4), Inhalt: inhalt, Schriftgrad: 74, Farbe: qaSchwarz})
		}
		_, a, err := bildMitAufwand(d)
		if err != nil {
			t.Fatal(err)
		}
		return a
	}
	ohne := zaehlen(nil)
	mit := zaehlen(spaltenStreifen(1023, 4))
	if mit.glyphen != ohne.glyphen || mit.glyph != ohne.glyph {
		t.Errorf("1023 Spaltenstreifen: %d Glyphen gerastert, Kosten %v; ohne Streifen %d, %v (Soll gleich)",
			mit.glyphen, mit.glyph, ohne.glyphen, ohne.glyph)
	}
	if ohne.glyphen == 0 {
		t.Error("keine Glyphe gerastert")
	}
}

// F-750, Schritt-Nummer: Die Ziffern rasterte der Renderer je Bereich neu und zählte sie auch je Bereich; jetzt einmal.
func TestNummerZiffernEinmalUeberKacheln(t *testing.T) {
	zaehlen := func(st []document.Streifen) aufwand {
		d := weissesDokument(t, 256, 256)
		if err := d.SetzeStreifen(st); err != nil {
			t.Fatal(err)
		}
		hinzu(t, d, document.Nummer{Mitte: document.Pt(128, 128), Wert: 42, Radius: 60, Farbe: color.NRGBA{R: 200, A: 255}})
		_, a, err := bildMitAufwand(d)
		if err != nil {
			t.Fatal(err)
		}
		return a
	}
	ohne, mit := zaehlen(nil), zaehlen(kacheln())
	if mit.glyphen != ohne.glyphen || mit.glyph != ohne.glyph || ohne.glyphen != 2 {
		t.Errorf("4096 Kacheln: %d Ziffern, Kosten %v; ohne Streifen %d, %v (Soll gleich, 2 Ziffern)",
			mit.glyphen, mit.glyph, ohne.glyphen, ohne.glyph)
	}
}

// F-750, Sprechblasen: Jedes Dreieck ist je Bereich eine eigene Füllung, und deren Grundkosten (rund 5,5 µs) zählte das
// Budget nicht: 1000 Sprechblasen über 4096 Kacheln rasterten in 26 s bei 63 % von KantenBudget. Mit aufrufGewicht lehnt
// Pruefe das ab; wenige Sprechblasen über denselben Kacheln bleiben erlaubt.
func TestSprechblasenUeberKachelnImBudget(t *testing.T) {
	sp := document.Pt(250, 250)
	blasen := func(n int) *document.Dokument {
		d := weissesDokument(t, 256, 256)
		if err := d.SetzeStreifen(kacheln()); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < n; i++ {
			hinzu(t, d, document.Text{Ort: document.Pt(4, 4), Inhalt: "a", Schriftgrad: 20, Farbe: qaSchwarz,
				Hintergrund: color.NRGBA{R: 30, G: 90, B: 200, A: 255}, Spitze: &sp})
		}
		return d
	}
	if err := Pruefe(blasen(1000)); !errors.Is(err, ErrZuAufwendig) {
		t.Errorf("1000 Sprechblasen über 4096 Kacheln: Pruefe %v, Soll ErrZuAufwendig", err)
	}
	if err := Pruefe(blasen(20)); err != nil {
		t.Errorf("20 Sprechblasen über 4096 Kacheln: Pruefe %v, Soll nil", err)
	}
}

// BenchmarkSprechblasenKantenGrenze rastert Sprechblasen über 4096 Kacheln knapp unter KantenBudget: die Wartezeit, die
// dieses Muster beim Speichern erzwingen kann (ADR-0023 Nachtrag 16; Zeitversprechen rund 21 s, ADR-0020 Nachtrag 12).
func BenchmarkSprechblasenKantenGrenze(b *testing.B) {
	sp := document.Pt(250, 250)
	d := weissesDokument(b, 256, 256)
	if err := d.SetzeStreifen(kacheln()); err != nil {
		b.Fatal(err)
	}
	n := 0
	for {
		hinzu(b, d, document.Text{Ort: document.Pt(4, 4), Inhalt: "a", Schriftgrad: 20, Farbe: qaSchwarz,
			Hintergrund: color.NRGBA{R: 30, G: 90, B: 200, A: 255}, Spitze: &sp})
		if Pruefe(d) != nil {
			if err := d.Entfernen(d.Objekte()[len(d.Objekte())-1].Kennung()); err != nil {
				b.Fatal(err)
			}
			break
		}
		n++
	}
	b.Logf("%d Sprechblasen, Kanten %.3g", n, trockenlauf(d, false).kanten)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Bild(d); err != nil {
			b.Fatal(err)
		}
	}
}
