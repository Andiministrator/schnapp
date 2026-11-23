// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"fmt"
	"image"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// F-422 (Prüfbefehl): Die Kante einer Naht darf den SVG-Export nur um wenig vergrößern. Das Bild ist einfarbig, damit die PNG-Kacheln
// nichts beitragen; verglichen wird mit demselben Bild ohne Naht. Grenze: 100 kB je Naht und Kantenform, in beiden
// Achsen (Zeilen-Naht über 1920 Pixel Länge, Spalten-Naht über 1080).
const nahtZuwachsGrenze = 100 << 10

func nahtGroesse(t *testing.T, form document.Kante, zeilen bool, mitNaht bool) int {
	t.Helper()
	b := image.NewNRGBA(image.Rect(0, 0, 1920, 1080))
	for i := 0; i < len(b.Pix); i += 4 {
		copy(b.Pix[i:i+4], []byte{0x80, 0x80, 0x80, 0xFF})
	}
	d, err := document.Neu(b)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeKante(form); err != nil && form != document.KanteAbriss {
		t.Fatal(err)
	}
	if mitNaht {
		von := 500
		if zeilen {
			von = 300
		}
		if err := d.SetzeStreifen([]document.Streifen{{Zeilen: zeilen, Von: von, Bis: von + 40}}); err != nil {
			t.Fatal(err)
		}
	}
	var buf bytes.Buffer
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
	return buf.Len()
}

// Bei Faktor 2 (kurze Seite 2160) sind Breiten und Versätze doppelt so groß wie bei Faktor 1: Das äußerste Stück des
// Fasersaums liegt von −9,6 bis −8,8, also bei −19,2 mit der Breite 160 Hundertstel.
func TestSVGNahtFaktor2(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 3840, 2160)))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 300, Bis: 340}}); err != nil {
		t.Fatal(err)
	}
	if f := d.KantenFaktor(); f != 2 {
		t.Fatalf("Faktor %d, Soll 2", f)
	}
	var buf bytes.Buffer
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, `<use href="#band-zeilen-80" y="-19.2" fill="#ffffff" fill-opacity="`) {
		t.Errorf("Stück bei −19,2 fehlt (Breite 80 im Pfad, Versatz verdoppelt):\n%.1500s", s)
	}
	if !strings.Contains(s, `<path id="band-zeilen-80" d="M0,`) {
		t.Errorf("Basispfad der Breite 80 fehlt:\n%.600s", s)
	}
}

func TestSVGNahtZuwachs(t *testing.T) {
	for _, form := range []document.Kante{document.KanteAbriss, document.KanteWelle, document.KanteZacken} {
		for _, zeilen := range []bool{false, true} {
			t.Run(fmt.Sprintf("%v/zeilen=%v", form, zeilen), func(t *testing.T) {
				ohne := nahtGroesse(t, form, zeilen, false)
				mit := nahtGroesse(t, form, zeilen, true)
				t.Logf("ohne Naht %d B, mit Naht %d B, Zuwachs %d B", ohne, mit, mit-ohne)
				if mit-ohne > nahtZuwachsGrenze {
					t.Errorf("Zuwachs %d B > %d B", mit-ohne, nahtZuwachsGrenze)
				}
			})
		}
	}
}
