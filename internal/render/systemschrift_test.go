// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fonts"
)

// Systemschriften im Kostenmodell (Karte #238, F-601, ADR-0020 Nachtrag 13): Glyphen aus Systemschriften zählen
// pessimistisch mit systemFaktor, die eingebauten Go und Go Mono mit 1.

// randDokument legt Texte der Familie schrift („@“ im Grad 74, 40 je Zeile, 20 Zeilen) auf 4000×2000, bis die
// Glyphen in der Rechnung mit Faktor 1 knapp unter anteil × GlyphBudget liegen. Nur für den Trockenlauf von Pruefe.
func randDokument(t *testing.T, schrift string, anteil float64) *document.Dokument {
	t.Helper()
	const grad, jeZeile, zeilen = 74.0, 40, 20
	d := weissesDokument(t, 4000, 2000)
	text := strings.TrimSuffix(strings.Repeat(strings.Repeat("@", jeZeile)+"\n", zeilen), "\n")
	eins := document.Text{Ort: document.Pt(10, 10), Inhalt: text, Schriftgrad: grad, Farbe: rot, Schrift: schrift}
	je := glyphKosten(jeZeile*zeilen, grad) // Faktor 1
	for n := 0; float64(n+1)*je < anteil*GlyphBudget; n++ {
		hinzu(t, d, eins)
	}
	return d
}

// TestSystemschriftAmRandAbgelehnt: Ein Dokument, dessen Glyphen mit Faktor 1 knapp unter GlyphBudget liegen, wird
// mit einer Systemschrift abgelehnt, mit Go und Go Mono angenommen. Vor Karte #238 nahm Pruefe alle an; dasselbe
// Dokument mit „Noto Serif CJK JP“ „龘“ rasterte dann in rund 40 bis 42 s statt rund 21 s (Sammelkritik Sprint 12 KS-2).
// Dazu die Lage des Faktors: Er deckt die gemessenen 2,39 × Go (ein Dokument mit GlyphBudget/2,39 wird abgelehnt) und ist
// nicht pessimistischer als 2,5 (eines mit 0,97 × GlyphBudget/2,5 wird angenommen).
func TestSystemschriftAmRandAbgelehnt(t *testing.T) {
	ladeTestkern(t)
	for _, f := range []struct {
		name, schrift string
		anteil        float64
		abgelehnt     bool
	}{
		{"Go am Rand", "", 0.98, false},
		{"Go Mono am Rand", fonts.GoMono, 0.98, false},
		{"Ersatz für fehlende Familie am Rand", "Gibt Es Nicht", 0.98, false},
		{"Systemschrift am Rand von Go", "Schnapp Testkern", 0.98, true},
		{"Systemschrift mit GlyphBudget/2,39", "Schnapp Testkern", 1 / 2.39, true},
		{"Systemschrift mit 0,97 × GlyphBudget/2,5", "Schnapp Testkern", 0.97 / 2.5, false},
	} {
		err := Pruefe(randDokument(t, f.schrift, f.anteil))
		if f.abgelehnt && !errors.Is(err, ErrZuAufwendig) {
			t.Errorf("%s: %v, Soll ErrZuAufwendig", f.name, err)
		}
		if !f.abgelehnt && err != nil {
			t.Errorf("%s: %v, Soll angenommen", f.name, err)
		}
	}
}

// TestSystemschriftFaktor: Derselbe Text kostet in einer Systemschrift genau systemFaktor-mal so viel wie in Go; eine
// fehlende Familie (Ersatz: Go wird gerastert) kostet wie Go, ebenso die Ziffern einer Nummer (immer Go Fett).
func TestSystemschriftFaktor(t *testing.T) {
	ladeTestkern(t)
	kosten := func(schrift string) float64 {
		d := weissesDokument(t, 2000, 500)
		hinzu(t, d, document.Text{Ort: document.Pt(10, 10), Inhalt: "Hallo Welt\nzweite Zeile", Schriftgrad: 40, Farbe: rot, Schrift: schrift})
		return trockenlauf(d, false).glyph
	}
	goK := kosten("")
	if goK == 0 {
		t.Fatal("Go-Text kostet nichts")
	}
	if k := kosten("Schnapp Testkern"); k != systemFaktor*goK {
		t.Errorf("Systemschrift kostet %v, Soll %v × %v = %v", k, systemFaktor, goK, systemFaktor*goK)
	}
	for _, s := range []string{fonts.Go, fonts.GoMono, "Gibt Es Nicht"} {
		if k := kosten(s); k != goK {
			t.Errorf("%q kostet %v, Soll wie Go %v", s, k, goK)
		}
	}
	d := weissesDokument(t, 500, 500)
	hinzu(t, d, document.Nummer{Mitte: document.Pt(100, 100), Radius: 40, Wert: 12, Farbe: blau})
	if k, soll := trockenlauf(d, false).glyph, glyphKosten(2, 48); k != soll {
		t.Errorf("Nummer kostet %v, Soll %v", k, soll)
	}
}

// TestAlltagsdokumentSystemschriftWeitUnterDemBudget: Das alltagsnahe Dokument (alltagsDokument) mit allen Texten in
// einer lateinischen Systemschrift bleibt weit unter GlyphBudget (Abstand mindestens 20; mit Go mindestens 50, siehe
// TestAlltagsdokumentWeitUnterDemBudget) und wird angenommen.
func TestAlltagsdokumentSystemschriftWeitUnterDemBudget(t *testing.T) {
	ladeTestkern(t)
	d := alltagsDokumentIn(t, "Schnapp Testkern")
	a := trockenlauf(d, false)
	f := GlyphBudget / a.glyph
	t.Logf("Alltagsdokument mit Systemschrift: Glyphen %.3g, GlyphBudget %.1f-mal so groß", a.glyph, f)
	if f < 20 {
		t.Errorf("GlyphBudget nur %.1f-mal so groß wie die Glyphen (%.3g), Soll mindestens 20", f, a.glyph)
	}
	if err := Pruefe(d); err != nil {
		t.Error(err)
	}
}

// BenchmarkNaehteAlleGrenzenSystemschrift (F-601, Sammelkritik Sprint 12 KS-2): wie
// BenchmarkNaehteAlleGrenzenKleineGlyphen/Klammeraffe74 (gleiche Naht-, Füll- und Glyphengrenzen), aber mit einer
// Systemschrift und einem dichten Zeichen, bis knapp unter GlyphBudget nach dem Kostenmodell (mit systemFaktor):
//
//	SCHNAPP_MESS_SCHRIFTEN=<Ordner> SCHNAPP_MESS_FAMILIE=<Familie> SCHNAPP_MESS_ZEICHEN=<Zeichen> \
//	  go test ./internal/render -run x -bench NaehteAlleGrenzenSystemschrift -cpu 1 -count 3
//
// Ohne die drei Variablen übersprungen (die Schrift muss installiert sein). Bezug: „Noto Serif CJK JP“ „龘“.
func BenchmarkNaehteAlleGrenzenSystemschrift(b *testing.B) {
	ordner, familie, zeichen := os.Getenv("SCHNAPP_MESS_SCHRIFTEN"), os.Getenv("SCHNAPP_MESS_FAMILIE"), os.Getenv("SCHNAPP_MESS_ZEICHEN")
	if ordner == "" || familie == "" || zeichen == "" {
		b.Skip("ohne SCHNAPP_MESS_SCHRIFTEN, _FAMILIE, _ZEICHEN")
	}
	fonts.SetzeOrdner([]string{ordner})
	fonts.SucheZuruecksetzen()
	fonts.Suche()
	b.Cleanup(func() {
		fonts.SetzeOrdner(nil)
		fonts.SucheZuruecksetzen()
		fonts.Suche()
	})
	const grad, jeZeile, zeilen = 74.0, 440, 20
	d := opakDokument(b, 32768, 2100, document.KanteAbriss, 90)
	text := strings.TrimSuffix(strings.Repeat(strings.Repeat(zeichen, jeZeile)+"\n", zeilen), "\n")
	eins := document.Text{Ort: document.Pt(10, 10), Inhalt: text, Schriftgrad: grad, Farbe: rot, Schrift: familie}
	if eins.Schriftdatei().Ersatz {
		b.Fatalf("Familie %q nicht gefunden", familie)
	}
	hinzu(b, d, eins)
	je := trockenlauf(d, false).glyph
	if soll := glyphKostenIn(eins.Schriftdatei(), jeZeile*zeilen, grad); je != soll {
		b.Fatalf("ein Text kostet %v, Soll %v: nicht alle Glyphen liegen im Bild", je, soll)
	}
	for n := 1; float64(n+1)*je < 0.98*GlyphBudget; n++ {
		hinzu(b, d, eins)
	}
	for Aufwand(d)+fuellGewicht*32768*2100 < 0.98*Budget {
		hinzu(b, d, document.Kasten{Bereich: document.R(document.Pt(0, 0), document.Pt(32768, 2100)), Fuellung: rot})
	}
	if err := Pruefe(d); err != nil {
		b.Fatalf("Pruefe: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Bild(d); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(Aufwand(d), "Einheiten")
	b.ReportMetric(trockenlauf(d, false).glyph, "Glyphen-Einheiten")
	b.ReportMetric(float64(len(d.Objekte())), "Objekte")
}
