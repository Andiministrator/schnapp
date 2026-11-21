// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"compress/zlib"
	"errors"
	"image/color"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// F-127: Eine Arbeitsdatei mit mehr als MaxObjekte Objekten wird beim Lesen abgelehnt (die Prüfung sitzt in document
// und greift über Hinzufuegen); genau MaxObjekte bleibt lesbar. Der Fehler gilt als beschädigt (Öffnen als Bild).
func TestLesenZuvieleObjekte(t *testing.T) {
	d, err := document.Neu(testbild(80, 60))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(1, 1), document.Pt(5, 5)),
		Strich: document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 1}}); err != nil {
		t.Fatal(err)
	}
	gut := schreibe(t, d, meta)
	zr, err := zlib.NewReader(bytes.NewReader(blockDaten(t, gut, blockSVG)))
	if err != nil {
		t.Fatal(err)
	}
	svg, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`<rect s:typ="kasten"[^>]*/>`)
	if len(re.FindAll(svg, -1)) != 1 {
		t.Fatalf("Kasten-Element nicht eindeutig im SVG: %s", svg)
	}
	mit := func(n int) []byte {
		neu := re.ReplaceAllLiteral(svg, []byte(strings.Repeat(string(re.Find(svg)), n)))
		return ersetzeBlock(t, gut, blockSVG, zlibVon(string(neu)))
	}
	g, err := LeseArbeitsdatei(bytes.NewReader(mit(document.MaxObjekte)))
	if err != nil {
		t.Fatalf("MaxObjekte Objekte abgelehnt: %v", err)
	}
	if n := len(g.Dokument.Objekte()); n != document.MaxObjekte {
		t.Errorf("%d Objekte gelesen, soll %d", n, document.MaxObjekte)
	}
	_, err = LeseArbeitsdatei(bytes.NewReader(mit(document.MaxObjekte + 1)))
	if err == nil {
		t.Fatal("MaxObjekte+1 Objekte ohne Fehler gelesen")
	}
	if !errors.Is(err, ErrBeschaedigt) || !IstBeschaedigt(err) || errors.Is(err, ErrFormatNeuer) {
		t.Errorf("MaxObjekte+1: Fehler %v, Soll ErrBeschaedigt", err)
	}
}
