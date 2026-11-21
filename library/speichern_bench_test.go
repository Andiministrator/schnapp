// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/document"
	"github.com/Andiministrator/schnapp/fileformat"
	"github.com/Andiministrator/schnapp/internal/speicherschritt"
	"github.com/Andiministrator/schnapp/render"
)

// Messung F-297: Wie verteilt sich die Zeit beim Speichern eines 4K-Dokuments auf Rendern, Lesen der alten Datei,
// PNG-Kodierung (Bild, Original), Schreiben, Index und Vorschau? Aufruf:
//
//	GOFLAGS=-tags=novulkan,nox11 go test -run '^$' -bench 'Speichern4K|Kodierung4K' -benchtime 3x ./library
//
// Die Bilder sind synthetisch (nie echte Aufnahmen); die Dateien liegen unter b.TempDir().
const (
	breite4K = 3840
	hoehe4K  = 2160
)

// bild4KGlatt ist ein glatter Verlauf mit einem feinen Raster (kompressionsfreundlich, wie Oberflächen).
func bild4KGlatt() *image.NRGBA {
	bild := image.NewNRGBA(image.Rect(0, 0, breite4K, hoehe4K))
	for y := 0; y < hoehe4K; y++ {
		for x := 0; x < breite4K; x++ {
			c := color.NRGBA{uint8(x * 255 / breite4K), uint8(y * 255 / hoehe4K), 120, 255}
			if x%64 == 0 || y%64 == 0 {
				c = color.NRGBA{30, 30, 30, 255}
			}
			bild.SetNRGBA(x, y, c)
		}
	}
	return bild
}

// bild4KRauschen ist Pseudozufall mit festem Seed (kaum komprimierbar, wie Fotos mit Rauschen).
func bild4KRauschen() *image.NRGBA {
	bild := image.NewNRGBA(image.Rect(0, 0, breite4K, hoehe4K))
	rand.New(rand.NewSource(297)).Read(bild.Pix)
	for i := 3; i < len(bild.Pix); i += 4 {
		bild.Pix[i] = 255
	}
	return bild
}

// schwaerzeFlaechen schwärzt n Rechtecke (Größe etwa 400×120, über das Bild verteilt).
func schwaerzeFlaechen(b testing.TB, d *document.Dokument, n int) {
	b.Helper()
	for i := 0; i < n; i++ {
		x, y := 200+i*500, 150+i*300
		r := image.Rect(x, y, x+400, y+120)
		if err := d.Schwaerzen(r, document.Schwaerzung{Farbe: color.NRGBA{0, 0, 0, 255}}); err != nil {
			b.Fatal(err)
		}
	}
}

// speichern4K misst den Speicherweg wie ablegenIn/speichernIn: Rendern, dann Bibliothek.Speichern (alte Datei lesen,
// Bild und Original kodieren, schreiben, ersetzen, Index, Vorschau).
func speichern4K(b *testing.B, bild func() *image.NRGBA, schwaerzungen int) {
	heim := b.TempDir()
	o, err := OrteAus(func(k string) string {
		if k == "HOME" {
			return heim
		}
		return ""
	})
	if err != nil {
		b.Fatal(err)
	}
	bib := Neu(o)
	bib.Jetzt = func() time.Time { return time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC) }

	d, err := document.Neu(bild())
	if err != nil {
		b.Fatal(err)
	}
	sicht, err := render.Bild(d)
	if err != nil {
		b.Fatal(err)
	}
	e, err := bib.Ablegen(d, sicht, fileformat.Metadaten{})
	if err != nil {
		b.Fatal(err)
	}
	stand, err := StandVon(bib.Orte.Ablage + "/" + e.Pfad)
	if err != nil {
		b.Fatal(err)
	}
	schwaerzeFlaechen(b, d, schwaerzungen)

	var mu sync.Mutex
	zeiten := map[string]time.Duration{}
	var reihe []string
	var vorig time.Time
	speicherschritt.Haken = func(s string) {
		mu.Lock()
		defer mu.Unlock()
		jetzt := time.Now()
		if s == "start" {
			vorig, reihe = jetzt, nil
		}
		zeiten[s] += jetzt.Sub(vorig)
		vorig = jetzt
		reihe = append(reihe, s)
	}
	defer func() { speicherschritt.Haken = nil }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		speicherschritt.Marke("start")
		sichtbar, err := render.Bild(d)
		if err != nil {
			b.Fatal(err)
		}
		speicherschritt.Marke("gerendert")
		_, neu, err := bib.Speichern(e.Pfad, stand, d, sichtbar)
		if err != nil {
			b.Fatal(err)
		}
		stand = neu
	}
	b.StopTimer()
	mu.Lock()
	defer mu.Unlock()
	for _, s := range reihe {
		if s == "start" {
			continue
		}
		b.ReportMetric(float64(zeiten[s].Milliseconds())/float64(b.N), s+"-ms/op")
	}
}

func BenchmarkSpeichern4KGlatt(b *testing.B)    { speichern4K(b, bild4KGlatt, 6) }
func BenchmarkSpeichern4KRauschen(b *testing.B) { speichern4K(b, bild4KRauschen, 1) }

// BenchmarkKodierung4K vergleicht die PNG-Kodierung von Bild und Original einzeln: Voreinstellung, BestSpeed und
// beide nebenläufig (Vorstufe der Empfehlung zu F-297; kein Produktcode).
func BenchmarkKodierung4K(b *testing.B) {
	for _, fall := range []struct {
		name string
		bild func() *image.NRGBA
	}{{"glatt", bild4KGlatt}, {"rauschen", bild4KRauschen}} {
		img := fall.bild()
		kodiere := func(stufe png.CompressionLevel) int {
			var buf bytes.Buffer
			enc := png.Encoder{CompressionLevel: stufe}
			if err := enc.Encode(&buf, img); err != nil {
				b.Fatal(err)
			}
			return buf.Len()
		}
		for _, s := range []struct {
			name  string
			stufe png.CompressionLevel
		}{{"default", png.DefaultCompression}, {"speed", png.BestSpeed}} {
			b.Run(fall.name+"/"+s.name+"/einzeln", func(b *testing.B) {
				var n int
				for i := 0; i < b.N; i++ {
					n = kodiere(s.stufe)
				}
				b.ReportMetric(float64(n)/1e6, "MB")
			})
			b.Run(fall.name+"/"+s.name+"/zwei-parallel", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					var wg sync.WaitGroup
					wg.Add(2)
					for k := 0; k < 2; k++ {
						go func() { defer wg.Done(); kodiere(s.stufe) }()
					}
					wg.Wait()
				}
			})
		}
	}
}
