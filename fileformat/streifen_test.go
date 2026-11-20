package fileformat

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/document"
)

// F-136: Eine Arbeitsdatei mit mehr als MaxStreifen Streifen wird beim Lesen abgelehnt; genau MaxStreifen bleibt lesbar.
func TestLesenZuvieleStreifen(t *testing.T) {
	d, err := document.Neu(testbild(4*document.MaxStreifen, 8))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]document.Streifen{{Zeilen: false, Von: 0, Bis: 1}}); err != nil {
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
	re := regexp.MustCompile(`<s:streifen[^>]*/>`)
	if len(re.FindAll(svg, -1)) != 1 {
		t.Fatalf("Streifen-Element nicht eindeutig im SVG: %s", svg)
	}
	mit := func(n int) []byte {
		var sb strings.Builder
		for i := 0; i < n; i++ {
			fmt.Fprintf(&sb, `<s:streifen s:zeilen="false" s:von="%d" s:bis="%d"/>`, 2*i, 2*i+1)
		}
		neu := re.ReplaceAllLiteral(svg, []byte(sb.String()))
		return ersetzeBlock(t, gut, blockSVG, zlibVon(string(neu)))
	}
	g, err := LeseArbeitsdatei(bytes.NewReader(mit(document.MaxStreifen)))
	if err != nil {
		t.Fatalf("MaxStreifen abgelehnt: %v", err)
	}
	if n := len(g.Dokument.Streifen()); n != document.MaxStreifen {
		t.Errorf("%d Streifen gelesen, soll %d", n, document.MaxStreifen)
	}
	// Beschädigt (F-142), nicht etwa „neuere Fassung“: Der Editor öffnet so eine Datei als einfaches Bild.
	if _, err := LeseArbeitsdatei(bytes.NewReader(mit(document.MaxStreifen + 1))); err == nil {
		t.Error("MaxStreifen+1 ohne Fehler gelesen")
	} else if !errors.Is(err, ErrBeschaedigt) || errors.Is(err, ErrFormatNeuer) {
		t.Errorf("MaxStreifen+1: Fehler %v, Soll ErrBeschaedigt", err)
	}
}

// F-141: An der Kachelgrenze (63 Spalten- und 63 Zeilenstreifen, 4096 Kacheln) bleibt der SVG-Export kurz; vor der
// Grenze brauchten erlaubte 512+512 Streifen auf 2100×2100 rund 14 s.
func TestExportSVGKachelgrenze(t *testing.T) {
	if testing.Short() {
		t.Skip("Zeitmessung")
	}
	d, err := document.Neu(testbild(2100, 2100))
	if err != nil {
		t.Fatal(err)
	}
	var st []document.Streifen
	for i := 0; i < 63; i++ {
		st = append(st, document.Streifen{Zeilen: false, Von: 30*i + 15, Bis: 30*i + 16},
			document.Streifen{Zeilen: true, Von: 30*i + 15, Bis: 30*i + 16})
	}
	if err := d.SetzeStreifen(st); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	var b bytes.Buffer
	if err := ExportiereSVG(&b, d); err != nil {
		t.Fatal(err)
	}
	if dauer := time.Since(start); kachelgrenzeZeit > 0 && dauer > kachelgrenzeZeit {
		t.Errorf("ExportiereSVG an der Kachelgrenze %v, soll unter %v", dauer, kachelgrenzeZeit)
	}
	if n := strings.Count(b.String(), `<use href="#objekte"/>`); n != document.MaxKacheln {
		t.Errorf("%d Kacheln im Export, soll %d", n, document.MaxKacheln)
	}
}
