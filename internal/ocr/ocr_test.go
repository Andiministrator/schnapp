// SPDX-License-Identifier: GPL-3.0-or-later

package ocr

import (
	"context"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// attrappe legt ein Skript an, das tesseract spielt: --list-langs nennt Sprachen, sonst schreibt es seine Argumente
// und die Länge von stdin in eine Datei und gibt je nach ATTRAPPE_ART Text, einen Fehler oder nichts aus bzw. hängt.
func attrappe(t *testing.T, art string) (*Erkenner, string) {
	t.Helper()
	ordner := t.TempDir()
	protokoll := filepath.Join(ordner, "protokoll")
	skript := filepath.Join(ordner, "tesseract")
	inhalt := `#!/bin/sh
if [ "$1" = "--list-langs" ]; then
  echo 'List of available languages in "/x/" (4):'
  printf 'deu\neng\nosd\nfra\n'
  exit 0
fi
echo "$@" > "` + protokoll + `"
wc -c >> "` + protokoll + `"
case "` + art + `" in
  text) printf 'Hallo \r\nWelt\t\n\n\f' ;;
  fehl) echo 'Error opening data file /x/deu.traineddata' >&2; echo 'Failed loading language deu' >&2; exit 1 ;;
  haengt) exec sleep 30 ;;
  leer) printf '\f' ;;
esac
`
	if err := os.WriteFile(skript, []byte(inhalt), 0o700); err != nil {
		t.Fatal(err)
	}
	alt := suchePfad
	suchePfad = func(string) (string, error) { return skript, nil }
	t.Cleanup(func() { suchePfad = alt })
	e, err := Suche(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return e, protokoll
}

func testbild(b, h int) image.Image {
	m := image.NewNRGBA(image.Rect(0, 0, b, h))
	draw.Draw(m, m.Bounds(), image.White, image.Point{}, draw.Src)
	return m
}

func TestSucheUndSprachen(t *testing.T) {
	e, _ := attrappe(t, "text")
	if strings.Join(e.sprachen, ",") != "deu,eng,fra" {
		t.Fatalf("Sprachen %v", e.sprachen)
	}
	for ober, soll := range map[string]string{"de": "deu+eng", "": "eng", "en": "eng", "fr": "fra+eng", "pt_BR": "eng", "de_AT": "deu+eng"} {
		if got := e.Sprachen(ober); got != soll {
			t.Errorf("Sprachen(%q) = %q, erwartet %q", ober, got, soll)
		}
	}
	ohneEng := &Erkenner{sprachen: []string{"afr", "deu"}}
	if got := ohneEng.Sprachen("ja"); got != "afr" {
		t.Errorf("ohne passende Sprache: %q", got)
	}

	alt := suchePfad
	defer func() { suchePfad = alt }()
	suchePfad = func(string) (string, error) { return "", exec.ErrNotFound }
	if _, err := Suche(context.Background()); !errors.Is(err, ErrKeinProgramm) || !strings.HasPrefix(err.Error(), "ocr: ") {
		t.Errorf("ohne Programm: %v", err)
	}
}

func TestLeseSprachen(t *testing.T) {
	aus := "List of available languages in \"/usr/share/tessdata/\" (5):\nafr\nosd\n-l\neng\nchi_sim\nbad;rm\neng\n\n"
	if got := strings.Join(leseSprachen(aus), ","); got != "afr,eng,chi_sim" {
		t.Errorf("leseSprachen = %q", got)
	}
}

func TestErkenneAttrappe(t *testing.T) {
	e, protokoll := attrappe(t, "text")
	text, err := e.Erkenne(context.Background(), testbild(800, 700), "deu+eng")
	if err != nil {
		t.Fatal(err)
	}
	if text != "Hallo\nWelt" {
		t.Errorf("Text %q", text)
	}
	p, _ := os.ReadFile(protokoll)
	zeilen := strings.Split(strings.TrimSpace(string(p)), "\n")
	if zeilen[0] != "- - -l deu+eng --psm 3" {
		t.Errorf("Argumente %q", zeilen[0])
	}
	if n := strings.TrimSpace(zeilen[1]); n == "0" {
		t.Error("kein Bild auf stdin")
	}
	if _, err := e.Erkenne(context.Background(), testbild(10, 10), "deu+klingon"); err == nil {
		t.Error("unbekannte Sprache angenommen")
	}
}

func TestErkenneFehlerFristLeer(t *testing.T) {
	e, _ := attrappe(t, "fehl")
	_, err := e.Erkenne(context.Background(), testbild(10, 10), "deu")
	if err == nil || !strings.Contains(err.Error(), "Failed loading language deu") || !strings.HasPrefix(err.Error(), "ocr: ") {
		t.Errorf("Fehler %v", err)
	}

	e, _ = attrappe(t, "haengt")
	ctx, ende := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer ende()
	start := time.Now()
	_, err = e.Erkenne(ctx, testbild(10, 10), "deu")
	if !errors.Is(err, ErrFrist) {
		t.Errorf("Frist: %v", err)
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Errorf("Abbruch dauerte %v", d)
	}
	ctx2, ab := context.WithCancel(context.Background())
	ab()
	if _, err := e.Erkenne(ctx2, testbild(10, 10), "deu"); !errors.Is(err, context.Canceled) {
		t.Errorf("Abbruch: %v", err)
	}

	e, _ = attrappe(t, "leer")
	if text, err := e.Erkenne(context.Background(), testbild(10, 10), "deu"); err != nil || text != "" {
		t.Errorf("leer: %q, %v", text, err)
	}
}

func TestVorbereite(t *testing.T) {
	klein, err := Vorbereite(testbild(300, 2000))
	if err != nil || klein.Bounds().Dx() != 600 || klein.Bounds().Dy() != 4000 {
		t.Errorf("klein: %v %v", klein.Bounds(), err)
	}
	gross := testbild(800, 600)
	if b, err := Vorbereite(gross); err != nil || b != gross {
		t.Errorf("groß unverändert erwartet: %v", err)
	}
	if _, err := Vorbereite(image.NewNRGBA(image.Rect(0, 0, 18000, 590))); !errors.Is(err, ErrZuGross) {
		t.Errorf("zu groß nach dem Vergrößern: %v", err)
	}
	if _, err := Vorbereite(image.NewNRGBA(image.Rectangle{})); !errors.Is(err, ErrLeer) {
		t.Errorf("leer: %v", err)
	}
}

func TestBereinige(t *testing.T) {
	for ein, soll := range map[string]string{
		"\n\nA  \r\nB\t\n\f": "A\nB",
		"\f":                 "",
		"x\xffy\n":           "x�y",
		"eins\n\nzwei\n\n\n": "eins\n\nzwei",
	} {
		if got := bereinige(ein); got != soll {
			t.Errorf("bereinige(%q) = %q, erwartet %q", ein, got, soll)
		}
	}
}

// TestErkenneEcht fährt das installierte tesseract mit einem synthetischen Bild (übersprungen, wenn es fehlt).
func TestErkenneEcht(t *testing.T) {
	if _, err := exec.LookPath(programmName); err != nil {
		t.Skip("tesseract nicht installiert")
	}
	e, err := Suche(context.Background())
	if err != nil {
		t.Skip(err)
	}
	ft, err := opentype.Parse(goregular.TTF)
	if err != nil {
		t.Fatal(err)
	}
	fc, err := opentype.NewFace(ft, &opentype.FaceOptions{Size: 18, DPI: 72})
	if err != nil {
		t.Fatal(err)
	}
	b := image.NewNRGBA(image.Rect(0, 0, 360, 60))
	draw.Draw(b, b.Bounds(), image.NewUniform(color.NRGBA{0x1E, 0x21, 0x24, 0xFF}), image.Point{}, draw.Src)
	d := &font.Drawer{Dst: b, Src: image.NewUniform(color.White), Face: fc, Dot: fixed.P(12, 36)}
	d.DrawString("Schnapp 2026 Text")
	text, err := e.Erkenne(context.Background(), b, e.Sprachen("en"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Schnapp") || !strings.Contains(text, "2026") {
		t.Errorf("erkannt %q", text)
	}
}
