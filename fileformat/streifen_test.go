package fileformat

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"regexp"
	"strings"
	"testing"

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
	if _, err := LeseArbeitsdatei(bytes.NewReader(mit(document.MaxStreifen + 1))); err == nil {
		t.Error("MaxStreifen+1 ohne Fehler gelesen")
	}
}
