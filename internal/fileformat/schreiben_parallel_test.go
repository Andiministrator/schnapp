// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"compress/zlib"
	"image"
	"image/color"
	"image/png"
	"io"
	"math/rand"
	"sync"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/saveprogress"
)

func rauschbild(b, h int, seed int64) *image.NRGBA {
	bild := image.NewNRGBA(image.Rect(0, 0, b, h))
	rand.New(rand.NewSource(seed)).Read(bild.Pix)
	for i := 3; i < len(bild.Pix); i += 4 {
		bild.Pix[i] = 255
	}
	return bild
}

// sequentiell baut die Arbeitsdatei so, wie SchreibeArbeitsdatei es vor F-310 tat: erst das Bild, dann das Original
// kodieren, dann zusammensetzen. Referenz für die Bitgleichheit.
func sequentiell(t *testing.T, d *document.Dokument, sichtbar image.Image, m Metadaten) []byte {
	t.Helper()
	var sicht, orig, svg bytes.Buffer
	if err := png.Encode(&sicht, sichtbar); err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(&orig, d.OriginalBild()); err != nil {
		t.Fatal(err)
	}
	zw, _ := zlib.NewWriterLevel(&svg, zlib.BestCompression)
	if _, err := zw.Write(svgArbeitsdatei(d, m)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	daten, err := fuegeEin(sicht.Bytes(), block{blockSVG, svg.Bytes()}, block{blockOriginal, orig.Bytes()})
	if err != nil {
		t.Fatal(err)
	}
	return daten
}

// F-310: Die nebenläufige Kodierung liefert dieselben Bytes wie die sequentielle Fassung (Bild und Original
// verschieden, damit ein Vertauschen der Puffer auffällt).
func TestSchreibeArbeitsdateiBitgleichZurSequentiellen(t *testing.T) {
	orig := rauschbild(320, 200, 310)
	d, err := document.Neu(orig)
	if err != nil {
		t.Fatal(err)
	}
	sichtbar := image.NewNRGBA(orig.Bounds())
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			sichtbar.SetNRGBA(x, y, color.NRGBA{uint8(x), uint8(y), 7, 255})
		}
	}
	m := Metadaten{Bearbeitet: true}
	soll := sequentiell(t, d, sichtbar, m)
	for i := 0; i < 5; i++ {
		var got bytes.Buffer
		if err := SchreibeArbeitsdatei(&got, d, sichtbar, m); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got.Bytes(), soll) {
			t.Fatalf("Durchlauf %d: Ausgabe weicht von der sequentiellen Fassung ab (%d zu %d Bytes)", i, got.Len(), len(soll))
		}
	}
}

// pulseWaehrend zählt die Pulse, die bis zur Marke schritt gemeldet sind.
func pulseBis(t *testing.T, schritt string, lauf func()) uint64 {
	t.Helper()
	var mu sync.Mutex
	var bei uint64
	alt := saveprogress.SetzeHaken(func(s string) {
		if s == schritt {
			mu.Lock()
			bei = saveprogress.Pulse()
			mu.Unlock()
		}
	})
	t.Cleanup(func() { saveprogress.SetzeHaken(alt) })
	vorher := saveprogress.Pulse()
	lauf()
	mu.Lock()
	defer mu.Unlock()
	return bei - vorher
}

// F-310 (Meldestelle je Kodierung): Jede der zwei Kodierungen pulst für sich. Ein winziges Original und ein großes
// sichtbares Bild (und umgekehrt) trennen die Quellen; die Pulse bis zur Marke der großen Kodierung können nur von
// ihr stammen, denn die kleine schreibt einen einzigen Puffer.
func TestSchreibeArbeitsdateiPulstJeKodierung(t *testing.T) {
	gross, klein := rauschbild(700, 500, 1), rauschbild(700, 500, 2)
	// Original groß, sichtbar identisch groß, aber die Kodierung mit mehr Stücken wird einzeln belegt: Das Original
	// wird durch ein Dokument kleiner Pixelzahl ersetzt, das sichtbare Bild bleibt groß.
	kleinesDok, err := document.Neu(klein.SubImage(image.Rect(0, 0, 8, 8)))
	if err != nil {
		t.Fatal(err)
	}
	grossesDok, err := document.Neu(gross)
	if err != nil {
		t.Fatal(err)
	}
	var z zaehlSchreiber
	// sichtbar groß, Original klein: Pulse bis „bild-kodiert“ stammen vom Bild.
	if n := pulseBis(t, "bild-kodiert", func() {
		if err := SchreibeArbeitsdatei(&z, kleinesDok, gross, Metadaten{}); err != nil {
			t.Fatal(err)
		}
	}); n < 20 {
		t.Errorf("sichtbares Bild: nur %d Pulse bis zur Marke bild-kodiert", n)
	}
	// Original groß, sichtbar klein: Pulse bis „original-kodiert“ stammen vom Original.
	if n := pulseBis(t, "original-kodiert", func() {
		if err := SchreibeArbeitsdatei(&z, grossesDok, klein.SubImage(image.Rect(0, 0, 8, 8)), Metadaten{}); err != nil {
			t.Fatal(err)
		}
	}); n < 20 {
		t.Errorf("Original: nur %d Pulse bis zur Marke original-kodiert", n)
	}
}

// F-310: Scheitert die Kodierung des Bilds, kehrt SchreibeArbeitsdatei erst zurück, wenn auch die des Originals
// beendet ist (kein auslaufender Schreiber), und meldet den Fehler des Bilds.
func TestSchreibeArbeitsdateiFehlerWartetAufBeide(t *testing.T) {
	d, err := document.Neu(rauschbild(700, 500, 3))
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	orig := false
	alt := saveprogress.SetzeHaken(func(s string) {
		if s == "original-kodiert" {
			mu.Lock()
			orig = true
			mu.Unlock()
		}
	})
	t.Cleanup(func() { saveprogress.SetzeHaken(alt) })
	var z zaehlSchreiber
	err = SchreibeArbeitsdatei(&z, d, image.NewNRGBA(image.Rect(0, 0, 0, 0)), Metadaten{})
	if err == nil {
		t.Fatal("leeres sichtbares Bild ohne Fehler geschrieben")
	}
	if want := png.Encode(io.Discard, image.NewNRGBA(image.Rect(0, 0, 0, 0))); want == nil || err.Error() != want.Error() {
		t.Errorf("Fehler %q, erwartet den der Bild-Kodierung %v", err, want)
	}
	mu.Lock()
	defer mu.Unlock()
	if !orig {
		t.Error("Rückkehr, bevor die Kodierung des Originals beendet war")
	}
	if z.aufrufe != 0 {
		t.Errorf("trotz Fehler %d Schreibaufrufe", z.aufrufe)
	}
}
