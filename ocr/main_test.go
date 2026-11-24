// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strings"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

func TestLeseArgumente(t *testing.T) {
	gut := []struct {
		args []string
		soll auftrag
	}{
		{[]string{"--list-langs"}, auftrag{liste: true, psm: 3}},
		{[]string{"--version"}, auftrag{ver: true, psm: 3}},
		{[]string{"-", "-"}, auftrag{sprache: "eng", psm: 3}},
		{[]string{"-", "-", "-l", "deu", "--psm", "6", "--progress"}, auftrag{sprache: "deu", psm: 6, fortschritt: true}},
	}
	for _, f := range gut {
		ist, err := leseArgumente(f.args)
		if err != nil || ist != f.soll {
			t.Errorf("%q: %+v, %v; soll %+v", f.args, ist, err, f.soll)
		}
	}
	for _, args := range [][]string{
		nil, {"bild.png", "-"}, {"-", "datei"}, {"-", "-", "-l"}, {"-", "-", "-l", "deu+eng"}, {"-", "-", "-l", "fra"},
		{"-", "-", "--psm", "x"}, {"-", "-", "--psm", "14"}, {"-", "-", "--oem", "1"}, {"--list-langs", "x"},
	} {
		if _, err := leseArgumente(args); err == nil {
			t.Errorf("%q: kein Fehler", args)
		}
	}
}

// textBild zeichnet zeilen mit Go Regular schwarz auf Weiß.
func textBild(t *testing.T, px float64, zeilen ...string) []byte {
	t.Helper()
	ft, err := opentype.Parse(goregular.TTF)
	if err != nil {
		t.Fatal(err)
	}
	fc, err := opentype.NewFace(ft, &opentype.FaceOptions{Size: px, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		t.Fatal(err)
	}
	h := int(px*1.6) + 4
	b := image.NewNRGBA(image.Rect(0, 0, int(px*30), h*len(zeilen)+20))
	draw.Draw(b, b.Bounds(), image.White, image.Point{}, draw.Src)
	for i, z := range zeilen {
		d := &font.Drawer{Dst: b, Src: image.NewUniform(color.NRGBA{0x20, 0x20, 0x20, 0xFF}), Face: fc,
			Dot: fixed.P(10, 10+(i+1)*h-h/4)}
		d.DrawString(z)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, b); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// TestErkenne fährt die eingebaute WASM-Fassung mit beiden Sprachen (einige Sekunden).
func TestErkenne(t *testing.T) {
	if testing.Short() {
		t.Skip("WASM-Erkennung dauert einige Sekunden")
	}
	for _, f := range []struct{ sprache, zeile string }{
		{"deu", "Grüße aus Köln, Straße 12"},
		{"eng", "The quick brown fox jumps"},
	} {
		var fehler bytes.Buffer
		text, err := erkenne(context.Background(), auftrag{sprache: f.sprache, psm: 3, fortschritt: true},
			textBild(t, 28, f.zeile), &fehler)
		if err != nil {
			t.Fatalf("%s: %v", f.sprache, err)
		}
		if strings.TrimSpace(text) != f.zeile {
			t.Errorf("%s: %q, soll %q", f.sprache, text, f.zeile)
		}
		if !strings.Contains(fehler.String(), "progress 100\n") {
			t.Errorf("%s: kein Fortschritt bis 100: %q", f.sprache, fehler.String())
		}
	}
}

func TestErkenneOhneBild(t *testing.T) {
	if _, err := erkenne(context.Background(), auftrag{sprache: "eng", psm: 3}, nil, &bytes.Buffer{}); err == nil {
		t.Error("leere Eingabe ohne Fehler")
	}
}

func TestErkenneAbgebrochen(t *testing.T) {
	ctx, ende := context.WithCancel(context.Background())
	ende()
	if _, err := erkenne(ctx, auftrag{sprache: "eng", psm: 3}, textBild(t, 20, "abc"), &bytes.Buffer{}); err == nil {
		t.Error("abgebrochener Kontext ohne Fehler")
	}
}
