// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"errors"
	"image"
	"io"
	"math/rand"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/saveprogress"
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
	saveprogress.Haken = func(s string) {
		if s == "bild-kodiert" {
			beimKodieren = saveprogress.Pulse()
		}
	}
	t.Cleanup(func() { saveprogress.Haken = nil })
	vorher := saveprogress.Pulse()
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
	vorher := saveprogress.Pulse()
	if _, err := decodiere(z.b); err != nil {
		t.Fatal(err)
	}
	if saveprogress.Pulse() == vorher {
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
	p.stand = append(p.stand, saveprogress.Pulse())
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

// stueckSchreiber merkt sich den Pulszähler bei jedem Write und scheitert ab dem Aufruf fehlerAb (0 = nie).
type stueckSchreiber struct {
	pulse    []uint64
	fehlerAb int
	fehler   error
}

func (s *stueckSchreiber) Write(b []byte) (int, error) {
	s.pulse = append(s.pulse, saveprogress.Pulse())
	if s.fehlerAb > 0 && len(s.pulse) >= s.fehlerAb {
		return 0, s.fehler
	}
	return len(b), nil
}

// rauschDokument liefert ein Dokument, dessen Arbeitsdatei größer als zwei Schreibstücke ist.
func rauschDokument(t *testing.T) (*document.Dokument, *image.NRGBA) {
	t.Helper()
	bild := image.NewNRGBA(image.Rect(0, 0, 700, 500))
	rand.New(rand.NewSource(298)).Read(bild.Pix)
	for i := 3; i < len(bild.Pix); i += 4 {
		bild.Pix[i] = 255
	}
	d, err := document.Neu(bild)
	if err != nil {
		t.Fatal(err)
	}
	return d, bild
}

// Kritik Sprint 9 K9-3: Jedes geschriebene Stück meldet Fortschritt — der Pulszähler steigt zwischen zwei Stücken.
func TestSchreibenMeldetJeStueckFortschritt(t *testing.T) {
	d, bild := rauschDokument(t)
	var s stueckSchreiber
	if err := SchreibeArbeitsdatei(&s, d, bild, Metadaten{}); err != nil {
		t.Fatal(err)
	}
	if len(s.pulse) < 2 {
		t.Fatalf("nur %d Schreibaufrufe", len(s.pulse))
	}
	for i := 1; i < len(s.pulse); i++ {
		if s.pulse[i] <= s.pulse[i-1] {
			t.Errorf("kein Puls vor Stück %d (%d → %d)", i+1, s.pulse[i-1], s.pulse[i])
		}
	}
}

// Kritik Sprint 9 K9-2: Scheitert das Schreiben eines Stücks, liefert SchreibeArbeitsdatei genau diesen Fehler und
// schreibt nicht weiter.
func TestSchreibenBrichtBeimFehlerEinesStuecksAb(t *testing.T) {
	d, bild := rauschDokument(t)
	platteVoll := errors.New("Platte voll")
	s := stueckSchreiber{fehlerAb: 2, fehler: platteVoll}
	if err := SchreibeArbeitsdatei(&s, d, bild, Metadaten{}); !errors.Is(err, platteVoll) {
		t.Errorf("Rückgabe %v, erwartet %v", err, platteVoll)
	}
	if len(s.pulse) != 2 {
		t.Errorf("%d Schreibaufrufe, erwartet 2 (nach dem Fehler nicht weiter)", len(s.pulse))
	}
}
