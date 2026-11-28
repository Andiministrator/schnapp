// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0045 Punkt 2: Objekte der Zwischenablage überstehen Schreiben und Lesen mit den Angaben der Quelle und ohne
// die übrigen Objekte.
func TestObjekteRundlauf(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 120, 90)))
	if err != nil {
		t.Fatal(err)
	}
	k := document.Kasten{Bereich: document.R(document.Pt(10, 10), document.Pt(40, 30)), Fuellung: color.NRGBA{R: 9, A: 255}}
	a, _ := d.Hinzufuegen(k)
	_, _ = d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(50, 50), document.Pt(60, 60))})
	n, _ := d.Hinzufuegen(document.Nummer{Mitte: document.Pt(70, 20), Wert: 3, Radius: 8})
	if err := d.SetzeZuschnitt(image.Rect(5, 0, 120, 90)); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 40, Bis: 45}}); err != nil {
		t.Fatal(err)
	}
	q, err := d.NurObjekte([]int{n, a})
	if err != nil {
		t.Fatal(err)
	}
	g, err := LeseObjekte(bytes.NewReader(AngabenSVG(q)))
	if err != nil {
		t.Fatal(err)
	}
	os := g.Objekte()
	if len(os) != 2 || !document.GleichOhneKennung(os[0], k) {
		t.Fatalf("gelesen %v", os)
	}
	if _, ok := os[1].(document.Nummer); !ok {
		t.Errorf("zweites Objekt %T, erwartet Nummer (Stapelreihenfolge)", os[1])
	}
	if g.Zuschnitt() != d.Zuschnitt() || len(g.Streifen()) != 1 || g.OriginalGroesse() != d.OriginalGroesse() {
		t.Errorf("Angaben %v %v %v", g.Zuschnitt(), g.Streifen(), g.OriginalGroesse())
	}
	if g.KannRueckgaengig() {
		t.Error("Verlauf nicht leer")
	}
	if _, err := LeseObjekte(bytes.NewReader([]byte("<svg/>"))); err == nil {
		t.Error("kaputtes SVG angenommen")
	}
}

// ADR-0045 Punkt 4: OeffneBild liest PNG und JPEG als einfaches Bild, auch ein PNG mit Schnapp-Blöcken, und lehnt
// anderes ab.
func TestOeffneBild(t *testing.T) {
	b := image.NewNRGBA(image.Rect(0, 0, 7, 5))
	b.Set(1, 1, color.NRGBA{R: 255, A: 255})
	var p, j bytes.Buffer
	_ = png.Encode(&p, b)
	_ = jpeg.Encode(&j, b, nil)
	for name, daten := range map[string][]byte{"png": p.Bytes(), "jpeg": j.Bytes()} {
		g, err := OeffneBild(daten)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !g.Fremd || g.Dokument.Groesse() != image.Pt(7, 5) {
			t.Errorf("%s: Fremd %v, Größe %v", name, g.Fremd, g.Dokument.Groesse())
		}
	}
	// Arbeitsdatei mit Objekt: als Bild, ohne Objekte
	d, _ := document.Neu(b)
	_, _ = d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(1, 1), document.Pt(3, 3))})
	var a bytes.Buffer
	if err := SchreibeArbeitsdatei(&a, d, b, Metadaten{}); err != nil {
		t.Fatal(err)
	}
	g, err := OeffneBild(a.Bytes())
	if err != nil || !g.Fremd || len(g.Dokument.Objekte()) != 0 {
		t.Errorf("Arbeitsdatei: %v, Fremd %v", err, g != nil && g.Fremd)
	}
	if _, err := OeffneBild([]byte("GIF89a")); !errors.Is(err, ErrKeinPNG) {
		t.Errorf("GIF: %v", err)
	}
}
