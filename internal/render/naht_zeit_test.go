// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// BenchmarkNaehteAbriss120 misst Bild für viele Abriss-Nähte (F-430, F-429, F-444): 120 Zeilen-Nähte auf 32768×2100,
// das Dokument, das vor ADR-0023 Nachtrag 7 knapp unter KantenBudget lag. Aufruf:
//
//	GOFLAGS=-tags=novulkan,nox11 go test -run '^$' -bench 'NaehteAbriss120' -benchtime 1x ./internal/render
func BenchmarkNaehteAbriss120(b *testing.B) {
	d := nahtDokument(b, 32768, 2100, document.KanteAbriss, 120)
	a := Aufwand(d)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Bild(d); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(a, "Einheiten")
}

// BenchmarkNaehteAbriss120Zackenfrei misst dasselbe in der Form Zacken (Vergleich).
func BenchmarkNaehteZacken120(b *testing.B) {
	d := nahtDokument(b, 32768, 2100, document.KanteZacken, 120)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Bild(d); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkNaehteAlleGrenzen (F-429): ein Dokument knapp unter allen drei Grenzen zugleich — 90 Abriss-Nähte auf
// 32768×2100 (knapp unter KantenBudget), 177 × „W“ Grad 4096 (knapp unter GlyphBudget, seit F-589; vorher 59) und bildgroße Rahmen, bis die
// Summe knapp unter Budget liegt. Die Zeit ist das Versprechen des Rechenbudgets für Bild (ein Kern).
func BenchmarkNaehteAlleGrenzen(b *testing.B) {
	d := opakDokument(b, 32768, 2100, document.KanteAbriss, 90)
	for i := 0; i < 177; i++ {
		hinzu(b, d, document.Text{Ort: document.Pt(float64((i%80)*100), 0), Inhalt: "W", Schriftgrad: 4096, Farbe: rot})
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
}

// BenchmarkNaehteAlleGrenzenKleineGlyphen (F-589): wie BenchmarkNaehteAlleGrenzen, aber mit kleinen Glyphen statt der
// „W“ im Grad 4096, übereinander in Textobjekten unter 10 000 Zeichen, bis knapp unter GlyphBudget:
//   - Fliesstext64: gemischte Buchstaben im Grad 64 (Kosten glyphGrund je Glyphe, gemessen rund 10 µs);
//   - Klammeraffe74: das teuerste gefundene Muster je Kosteneinheit (Kritik zu Karte #224): „@“ im Grad 74, an der
//     Knickstelle des Modells (74² = glyphGrund), gemessen rund 20 µs je Glyphe, also rund 3,7 ns je Einheit statt
//     rund 1 ns. Diese Zeit ist das Versprechen des Rechenbudgets für Bild (ein Kern).
func BenchmarkNaehteAlleGrenzenKleineGlyphen(b *testing.B) {
	for _, fall := range []struct {
		name  string
		grad  float64
		zeile string
		n     int // Zeilen je Text
	}{
		{"Fliesstext64", 64, strings.Repeat("Schnapp zeichnet ", 60)[:900], 11},
		{"Klammeraffe74", 74, strings.Repeat("@", 440), 20},
	} {
		b.Run(fall.name, func(b *testing.B) {
			d := opakDokument(b, 32768, 2100, document.KanteAbriss, 90)
			text := strings.TrimSuffix(strings.Repeat(fall.zeile+"\n", fall.n), "\n")
			eins := document.Text{Ort: document.Pt(10, 10), Inhalt: text, Schriftgrad: fall.grad, Farbe: rot}
			hinzu(b, d, eins)
			je := trockenlauf(d, false).glyph // Kosten eines solchen Texts
			if soll := glyphKosten(len(fall.zeile)*fall.n, fall.grad); je != soll {
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
		})
	}
}

// opakDokument ist wie nahtDokument, aber mit deckend weißem Bild (wie eine echte Aufnahme).
func opakDokument(tb testing.TB, b, h int, k document.Kante, zn int) *document.Dokument {
	tb.Helper()
	bild := image.NewNRGBA(image.Rect(0, 0, b, h))
	for i := range bild.Pix {
		bild.Pix[i] = 0xff
	}
	d, err := document.NeuAus(bild)
	if err != nil {
		tb.Fatal(err)
	}
	if err := d.VorgabeKante(k); err != nil {
		tb.Fatal(err)
	}
	var st []document.Streifen
	for i := 0; i < zn; i++ {
		v := (i + 1) * h / (zn + 1)
		st = append(st, document.Streifen{Zeilen: true, Von: v, Bis: v + 1})
	}
	if err := d.SetzeStreifen(st); err != nil {
		tb.Fatal(err)
	}
	return d
}

// BenchmarkNaehteAbriss120Opak: dasselbe auf deckend weißem Bild (der Normalfall einer Aufnahme).
func BenchmarkNaehteAbriss120Opak(b *testing.B) {
	d := opakDokument(b, 32768, 2100, document.KanteAbriss, 120)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Bild(d); err != nil {
			b.Fatal(err)
		}
	}
}

// F-408, F-444: Die Zahl der Abriss-Nähte, die Pruefe annimmt, hängt an der Länge der Naht (gemessen am 23.11.2026):
// auf 8192 Pixeln Länge etwa 364, auf 32768 etwa 91. Der Test hält das fest, nicht als Soll, sondern als Dokumentation:
// weit darunter nimmt Pruefe an, weit darüber lehnt es ab. Ändert sich das Kostenmodell, ändern sich die Zahlen in
// ADR-0023 Nachtrag 8 und in der Anmerkung zu KantenBudget mit. Nicht unter -short (69 MB Bild).
func TestNahtZahlGrenze(t *testing.T) {
	if testing.Short() || pruefeNahtZeit == 0 {
		t.Skip("lang (69 MB Bild); nicht unter -short oder -race")
	}
	for _, c := range []struct {
		anzahl int
		soll   bool
	}{{200, true}, {320, true}, {450, false}, {700, false}} {
		err := Pruefe(opakDokument(t, 8192, 2100, document.KanteAbriss, c.anzahl))
		if (err == nil) != c.soll {
			t.Errorf("%d Abriss-Nähte auf 8192×2100: Pruefe %v, angenommen soll %v sein", c.anzahl, err, c.soll)
		}
	}
}

// TestNahtGrenzeMessen sucht, wie viele Abriss-Nähte Pruefe auf zwei Breiten noch annimmt (F-408, F-444); läuft nur mit
// SCHNAPP_MESSEN=1 (Messung, kein Test).
func TestNahtGrenzeMessen(t *testing.T) {
	if os.Getenv("SCHNAPP_MESSEN") == "" {
		t.Skip("Messung")
	}
	for _, c := range [][2]int{{8192, 2100}, {32768, 2100}} {
		lo, hi := 1, 4000
		for lo < hi {
			m := (lo + hi + 1) / 2
			if Pruefe(opakDokument(t, c[0], c[1], document.KanteAbriss, min(m, c[1]/3))) == nil && m <= c[1]/3 {
				lo = m
			} else {
				hi = m - 1
			}
		}
		d := opakDokument(t, c[0], c[1], document.KanteAbriss, lo)
		a := trockenlauf(d, false)
		t.Logf("%d×%d: größte angenommene Nahtzahl %d (Aufwand %.3g, Kanten %.3g)", c[0], c[1], lo, a.summe(), a.kanten)
	}
}

// TestNahtPruefsumme (F-430, F-514, F-518) hält die Pixel der Nähte bitgleich fest: SHA-256 des Ergebnisses für 12
// Nähte auf 3000×700 in jeder Form gegen feste Sollwerte, vor der Beschleunigung gemessen und nach ihr unverändert.
// Ändert sich das Bild mit Absicht (Kantenform, Rasterkern), gehören die Werte mit einer Begründung im Commit neu
// eingetragen: `go test ./internal/render -run TestNahtPruefsumme -v` nennt die neuen Summen.
//
// F-518: Die festen Summen hängen an Fließkomma-Rechnung der Kantenlinien. Go darf auf Architekturen mit FMA (arm64,
// ppc64, s390x) Multiplikation und Addition ohne eigene Klammerung zusammenziehen; das Ergebnis kann dann in den
// letzten Bits abweichen, ohne dass etwas kaputt ist. Der harte Vergleich gilt deshalb nur auf amd64 (dort läuft die
// CI); überall sonst prüft der Test, was architekturunabhängig gilt: zweimal Rendern liefert dasselbe Bild, und das
// Bild mit Zwischenspeicher der Deckungen ist bitgleich zum Bild ohne (maskenCacheGrenze 0).
func TestNahtPruefsumme(t *testing.T) {
	soll := map[document.Kante]string{
		document.KanteAbriss: "82e17cfcdb1432698e71c50005238a6f4a20d6707c475731f30b22b2026087bf",
		document.KanteWelle:  "41cf31a746a2012b06066bbfc968704e1741fa89657dfed8fbae5047503073f2",
		document.KanteZacken: "ba294d195ab655a21bad2e1fdd8ec2a51d5bfca879dbc02cb70102651970e0c3",
	}
	for f, erwartet := range soll {
		d := nahtDokument(t, 3000, 700, f, 12)
		b, err := Bild(d)
		if err != nil {
			t.Fatal(err)
		}
		got := fmt.Sprintf("%x", sha256.Sum256(b.Pix))
		if runtime.GOARCH == "amd64" {
			if got != erwartet {
				t.Errorf("Summe %v: %s, erwartet %s", f, got, erwartet)
			}
			continue
		}
		b2, err := Bild(d)
		if err != nil {
			t.Fatal(err)
		}
		alt := maskenCacheGrenze
		maskenCacheGrenze = 0
		b3, err := Bild(d)
		maskenCacheGrenze = alt
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b.Pix, b2.Pix) {
			t.Errorf("Naht %v: zweimal Rendern ist nicht bitgleich (%s)", f, got)
		}
		if !bytes.Equal(b.Pix, b3.Pix) {
			t.Errorf("Naht %v: Bild mit und ohne Zwischenspeicher der Deckungen unterscheidet sich (%s)", f, got)
		}
	}
}
