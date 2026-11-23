// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
)

// glyphenDokument legt auf ein Bild von 4000×4000 ein Textobjekt mit n sichtbaren Glyphen im Schriftgrad grad
// (Buchstaben, Zeilenabstand 1,3 Grad, in Zeilen bis zur Bildbreite, auf mehrere Textobjekte verteilt). Liefert das Dokument und die Zahl der Glyphen,
// die wirklich ins Bild passen (höchstens n).
func glyphenDokument(b testing.TB, grad float64, n int) (*document.Dokument, int) {
	b.Helper()
	d := weissesDokument(b, 4000, 4000)
	const muster = "Schnapp zeichnet Beschriftungen schnell und sauber "
	proZeile := int(3900 / (0.5 * grad)) // Go-Schrift: im Mittel rund 0,5 Grad je Zeichen
	zeilen := int(3900 / (1.3 * grad))
	n = min(n, proZeile*zeilen*8/10)
	// Ein Text trägt höchstens 10000 Zeichen: Zeilen auf mehrere Objekte verteilen, je unter dem anderen.
	zeilenJeText := max(1, 9000/proZeile)
	rest, z := n, 0
	for rest > 0 {
		var sb strings.Builder
		erste := z
		for ; rest > 0 && z-erste < zeilenJeText; z++ {
			k := min(proZeile, rest)
			if z > erste {
				sb.WriteByte('\n')
			}
			for i := 0; i < k; i++ {
				sb.WriteByte(muster[(z*7+i)%len(muster)])
			}
			rest -= k
		}
		hinzu(b, d, document.Text{Ort: document.Pt(10, 10+float64(erste)*1.3*grad), Inhalt: sb.String(), Schriftgrad: grad, Farbe: rot})
	}
	return d, n
}

// BenchmarkGlyphKosten misst die Rechenzeit je Glyphe nach Schriftgrad (F-589, ADR-0020 Nachtrag 3): ein Kern
// (`go test ./internal/render -run X -bench GlyphKosten -cpu 1`, ohne -race), die Zeit des Bildes ohne Text ist
// abgezogen. Die Metriken `ns/Glyphe` und `ns/Einheit` (Einheit des Kostenmodells, glyphKosten) sind die Grundlage von
// glyphGrund und GlyphBudget.
func BenchmarkGlyphKosten(b *testing.B) {
	leer := weissesDokument(b, 4000, 4000)
	grund := func() time.Duration { // Bild ohne Objekte, mehrfach, das Minimum
		best := time.Duration(1 << 62)
		for i := 0; i < 5; i++ {
			t0 := time.Now()
			if _, err := Bild(leer); err != nil {
				b.Fatal(err)
			}
			best = min(best, time.Since(t0))
		}
		return best
	}
	for _, grad := range []float64{4, 6, 8, 12, 16, 24, 32, 48, 64, 128, 256} {
		b.Run(fmt.Sprintf("Grad%v", grad), func(b *testing.B) {
			d, n := glyphenDokument(b, grad, 60000)
			a := trockenlauf(d, false)
			if a.glyph == 0 {
				b.Fatal("keine Glyphen gezählt")
			}
			g0 := grund()
			b.ResetTimer()
			var summe time.Duration
			for i := 0; i < b.N; i++ {
				t0 := time.Now()
				if _, err := Bild(d); err != nil {
					b.Fatal(err)
				}
				summe += time.Since(t0)
			}
			je := (summe/time.Duration(b.N) - g0).Seconds() * 1e9
			b.ReportMetric(je/float64(n), "ns/Glyphe")
			b.ReportMetric(je/a.glyph, "ns/Einheit")
			b.ReportMetric(a.glyph/float64(n), "Einheiten/Glyphe")
			b.ReportMetric(float64(n), "Glyphen")
		})
	}
}
