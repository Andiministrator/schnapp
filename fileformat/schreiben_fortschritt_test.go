// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"image"
	"io"
	"math/rand"
	"testing"

	"github.com/Andiministrator/schnapp/document"
	"github.com/Andiministrator/schnapp/internal/speicherschritt"
)

// zaehlSchreiber zählt Write-Aufrufe und geschriebene Bytes.
type zaehlSchreiber struct {
	aufrufe int
	groesse int
}

func (z *zaehlSchreiber) Write(b []byte) (int, error) {
	z.aufrufe++
	z.groesse += len(b)
	return len(b), nil
}

// F-298 (Verify B-2 Mutation F1, B-6): SchreibeArbeitsdatei meldet beim Kodieren Fortschritt (vor der Marke
// „bild-kodiert“) und schreibt eine große Datei in Stücken.
func TestSchreibeArbeitsdateiMeldetFortschritt(t *testing.T) {
	bild := image.NewNRGBA(image.Rect(0, 0, 700, 500))
	rand.New(rand.NewSource(298)).Read(bild.Pix)
	for i := 3; i < len(bild.Pix); i += 4 {
		bild.Pix[i] = 255
	}
	d, err := document.Neu(bild)
	if err != nil {
		t.Fatal(err)
	}
	var beimKodieren uint64
	speicherschritt.Haken = func(s string) {
		if s == "bild-kodiert" {
			beimKodieren = speicherschritt.Pulse()
		}
	}
	t.Cleanup(func() { speicherschritt.Haken = nil })
	vorher := speicherschritt.Pulse()
	var z zaehlSchreiber
	if err := SchreibeArbeitsdatei(&z, d, bild, Metadaten{}); err != nil {
		t.Fatal(err)
	}
	if beimKodieren <= vorher {
		t.Errorf("kein Fortschritt beim Kodieren (%d → %d)", vorher, beimKodieren)
	}
	if z.groesse <= schreibStueck || z.aufrufe < 2 {
		t.Errorf("Datei %d Bytes in %d Aufrufen — erwartet mehr als ein Stück (%d) in mehreren Aufrufen",
			z.groesse, z.aufrufe, schreibStueck)
	}
}

// F-298 (Verify B-4): Dekodieren meldet Fortschritt.
func TestDecodiereMeldetFortschritt(t *testing.T) {
	bild := image.NewNRGBA(image.Rect(0, 0, 300, 200))
	d, err := document.Neu(bild)
	if err != nil {
		t.Fatal(err)
	}
	var z bytesSchreiber
	if err := SchreibeArbeitsdatei(&z, d, bild, Metadaten{}); err != nil {
		t.Fatal(err)
	}
	vorher := speicherschritt.Pulse()
	if _, err := decodiere(z.b); err != nil {
		t.Fatal(err)
	}
	if speicherschritt.Pulse() == vorher {
		t.Error("kein Fortschritt beim Dekodieren")
	}
}

type bytesSchreiber struct{ b []byte }

func (s *bytesSchreiber) Write(p []byte) (int, error) { s.b = append(s.b, p...); return len(p), nil }

// pulsBeobachter merkt sich bei jedem Read den Pulszähler.
type pulsBeobachter struct {
	r     io.Reader
	stand []uint64
}

func (p *pulsBeobachter) Read(b []byte) (int, error) {
	p.stand = append(p.stand, speicherschritt.Pulse())
	if len(b) > 4096 {
		b = b[:4096] // kleine Stücke wie von einem langsamen Medium
	}
	return p.r.Read(b)
}

// F-298 (Verify B-2, Mutation L1): LeseArbeitsdatei meldet schon beim Einlesen der Datei Fortschritt, nicht erst beim
// Dekodieren — auf einem langsamen Medium dauert das Lesen.
func TestLeseArbeitsdateiMeldetFortschrittBeimLesen(t *testing.T) {
	bild := image.NewNRGBA(image.Rect(0, 0, 200, 150))
	rand.New(rand.NewSource(298)).Read(bild.Pix)
	d, err := document.Neu(bild)
	if err != nil {
		t.Fatal(err)
	}
	var z bytesSchreiber
	if err := SchreibeArbeitsdatei(&z, d, bild, Metadaten{}); err != nil {
		t.Fatal(err)
	}
	p := &pulsBeobachter{r: bytes.NewReader(z.b)}
	if _, err := LeseArbeitsdatei(p); err != nil {
		t.Fatal(err)
	}
	if n := len(p.stand); n < 3 || p.stand[n-1] == p.stand[0] {
		t.Errorf("kein Fortschritt zwischen den Leseaufrufen: %d Aufrufe, Pulse %v … %v", n, p.stand[0], p.stand[n-1])
	}
}
