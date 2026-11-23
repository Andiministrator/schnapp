package ui

import (
	"image"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// F-498: Der Editor zeichnet den Strich eines Kastens ohne Rundung aus überlappungsfreien Rechtecken mit spitzen
// Außenecken (Gios Strich kennt nur runde Ecken). Strich 8: das Eckpixel (Rand − 4) liegt im Rahmen.
func TestKastenRahmenSpitz(t *testing.T) {
	rs := kastenRahmen(image.Rect(30, 30, 80, 70), 8)
	if len(rs) != 4 {
		t.Fatalf("%d Rechtecke, Soll 4", len(rs))
	}
	deckt := func(p document.Punkt) int {
		n := 0
		for _, r := range rs {
			if p.X >= r.Min.X && p.X < r.Max.X && p.Y >= r.Min.Y && p.Y < r.Max.Y {
				n++
			}
		}
		return n
	}
	for _, p := range []document.Punkt{document.Pt(26.5, 26.5), document.Pt(83.5, 73.5), document.Pt(33, 50), document.Pt(55, 33)} {
		if n := deckt(p); n != 1 {
			t.Errorf("Punkt %v von %d Rechtecken gedeckt, Soll 1", p, n)
		}
	}
	for _, p := range []document.Punkt{document.Pt(50, 50), document.Pt(25.5, 25.5)} {
		if n := deckt(p); n != 0 {
			t.Errorf("Punkt %v gedeckt, Soll frei", p)
		}
	}
}

// F-498: schmaler Kasten, Innenraum weg: ein einziges Rechteck über die ganze Fläche.
func TestKastenRahmenSchmal(t *testing.T) {
	rs := kastenRahmen(image.Rect(40, 40, 46, 80), 8)
	if len(rs) != 1 || rs[0].Min.X != 36 || rs[0].Max.Y != 84 {
		t.Errorf("Rahmen %v", rs)
	}
}
