// SPDX-License-Identifier: GPL-3.0-or-later

package fonts

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/gomediumitalic"
	"golang.org/x/image/font/sfnt"

	"github.com/Andiministrator/schnapp/internal/fonts/fonttest"
)

// untertitel liefert den Namen des Schnitts (Subfamily) einer geladenen Schrift.
func untertitel(t *testing.T, s *Schrift) string {
	t.Helper()
	var b sfnt.Buffer
	u, err := s.Font.Name(&b, sfnt.NameIDSubfamily)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// F-577: Eine Sammlung (.ttc) trägt mehrere Schnitte; jeder Eintrag der Familie merkt seinen Index in der Datei. Normal
// ist der erste, Kursiv der zweite Eintrag: Mit dem falschen Index käme der falsche Schnitt (oder ein Fehler).
func TestSammlungMitIndex(t *testing.T) {
	d := t.TempDir()
	schreibe(t, filepath.Join(d, "medium.ttc"), fonttest.Sammlung(t, gomedium.TTF, gomediumitalic.TTF))
	mitOrdnern(t, d)
	medium, _ := familieName(t, gomedium.TTF)
	if !Bekannt(medium) {
		t.Fatalf("Familie %q nicht gefunden: %v", medium, Familien())
	}
	n, k := Laden(medium, Normal), Laden(medium, Kursiv)
	if n.Ersatz || k.Ersatz || n.Schnitt != Normal || k.Schnitt != Kursiv {
		t.Fatalf("normal %+v, kursiv %+v", n, k)
	}
	if u := untertitel(t, n); u != "Regular" {
		t.Errorf("normal ist %q, erwartet Regular (Index 0)", u)
	}
	if u := untertitel(t, k); u != "Italic" {
		t.Errorf("kursiv ist %q, erwartet Italic (Index 1)", u)
	}
}

// F-577: Eine Datei über MaxDateiGroesse wird bei der Suche übergangen, auch wenn sie sich als Schrift lesen ließe.
func TestSucheUeberMaxDateiGroesse(t *testing.T) {
	d := t.TempDir()
	pfad := filepath.Join(d, "gross.ttf")
	schreibe(t, pfad, gomedium.TTF)
	if err := os.Truncate(pfad, MaxDateiGroesse+1); err != nil { // Nullen hinter der Schrift; lesbar bliebe sie
		t.Fatal(err)
	}
	mitOrdnern(t, d)
	if f := Familien(); len(f) != 2 {
		t.Errorf("Familien %v: die zu große Datei wurde gelesen", f)
	}
	// Genau an der Grenze gilt sie noch.
	if err := os.Truncate(pfad, MaxDateiGroesse); err != nil {
		t.Fatal(err)
	}
	Suche()
	if f := Familien(); len(f) != 3 {
		t.Errorf("Familien %v: die Datei an der Grenze fehlt", f)
	}
}

// F-577: Wächst die Datei nach der Suche über die Grenze, lädt Laden sie nicht mehr (Ersatz Go) und liest sie nicht ein.
func TestLadenUeberMaxDateiGroesse(t *testing.T) {
	d := t.TempDir()
	pfad := filepath.Join(d, "wachsend.ttf")
	schreibe(t, pfad, gomedium.TTF)
	mitOrdnern(t, d)
	medium, _ := familieName(t, gomedium.TTF)
	if !Bekannt(medium) {
		t.Fatalf("Familie %q nicht gefunden", medium)
	}
	if err := os.Truncate(pfad, MaxDateiGroesse+1); err != nil {
		t.Fatal(err)
	}
	if s := Laden(medium, Normal); !s.Ersatz {
		t.Errorf("zu große Datei geladen: %+v", s)
	}
	if len(dateien) != 0 {
		t.Errorf("zu große Datei im Speicher: %d Dateien", len(dateien))
	}
}

// F-577: Farb-Bitmap-Schriften (Emoji) stehen nicht in der Liste, weil Renderer und Editor sie nicht zeichnen; eine
// normale Schrift schon.
func TestSucheOhneFarbBitmapSchriften(t *testing.T) {
	d := t.TempDir()
	schreibe(t, filepath.Join(d, "farbe.ttf"), fonttest.FarbBitmapRoh(t, gomedium.TTF))
	mitOrdnern(t, d)
	if f := Familien(); len(f) != 2 {
		t.Errorf("Familien %v: die Farb-Bitmap-Schrift steht in der Liste", f)
	}
	schreibe(t, filepath.Join(d, "normal.ttf"), gomedium.TTF)
	Suche()
	if f := Familien(); len(f) != 3 || f[2] != "Go Medium" {
		t.Errorf("Familien %v, erwartet Go, Go Mono, Go Medium", f)
	}
}
