// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"os"
	"os/exec"
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

// TestMain spielt das gebaute Programm, wenn ein Test sich selbst mit SCHNAPP_OCR_ALS_PROGRAMM=1 startet: So prüfen
// die Tests main mit echtem Prozess, stdin, stdout, stderr und Exit-Code (F-643).
func TestMain(m *testing.M) {
	if os.Getenv("SCHNAPP_OCR_ALS_PROGRAMM") == "1" {
		main()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// programm startet dieses Testprogramm als schnapp-ocr mit args und eingabe auf stdin.
func programm(t *testing.T, eingabe []byte, args ...string) (aus, fehler string, code int) {
	t.Helper()
	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = append(os.Environ(), "SCHNAPP_OCR_ALS_PROGRAMM=1")
	cmd.Stdin = bytes.NewReader(eingabe)
	var a, f bytes.Buffer
	cmd.Stdout, cmd.Stderr = &a, &f
	err := cmd.Run()
	var ee *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &ee):
		code = ee.ExitCode()
	default:
		t.Fatal(err)
	}
	return a.String(), f.String(), code
}

// pngKopf ist der Anfang eines PNG mit den Maßen b × h: Signatur und IHDR, ohne Bilddaten (für die Pixelgrenze reicht
// der Kopf; so kostet der Test keinen Speicher).
func pngKopf(b, h uint32) []byte {
	var ihdr bytes.Buffer
	ihdr.WriteString("IHDR")
	binary.Write(&ihdr, binary.BigEndian, b)
	binary.Write(&ihdr, binary.BigEndian, h)
	ihdr.Write([]byte{8, 0, 0, 0, 0}) // 8 Bit Grau
	var r bytes.Buffer
	r.WriteString("\x89PNG\r\n\x1a\n")
	binary.Write(&r, binary.BigEndian, uint32(ihdr.Len()-4))
	r.Write(ihdr.Bytes())
	binary.Write(&r, binary.BigEndian, crc32.ChecksumIEEE(ihdr.Bytes()))
	return r.Bytes()
}

// Das Programm über main: Exit-Codes, stdout nur Text, ohne --progress kein Fortschritt, mit --progress bis 100,
// kurze Fehlertexte ohne Go-Interna (F-643, F-649).
func TestProgramm(t *testing.T) {
	if _, f, code := programm(t, nil, "bild.png", "-"); code != 2 || !strings.HasPrefix(f, "schnapp-ocr: usage:") {
		t.Errorf("falscher Aufruf: Exit %d, %q", code, f)
	}
	if a, _, code := programm(t, nil, "--list-langs"); code != 0 || a != "List of available languages in \"schnapp-ocr\" (2):\ndeu\neng\n" {
		t.Errorf("--list-langs: Exit %d, %q", code, a)
	}
	if a, f, code := programm(t, []byte("das ist kein Bild"), "-", "-", "-l", "eng"); code != 1 || a != "" ||
		f != "schnapp-ocr: "+errUnlesbar.Error()+"\n" {
		t.Errorf("kein Bild: Exit %d, stdout %q, stderr %q", code, a, f)
	}
	gut := textBild(t, 28, "The quick brown fox jumps")
	for _, kaputt := range [][]byte{gut[:len(gut)/2], gut[:60]} { // Kopf lesbar, Bilddaten abgeschnitten
		_, f, code := programm(t, kaputt, "-", "-", "-l", "eng")
		if code != 1 || f != "schnapp-ocr: "+errUnlesbar.Error()+"\n" {
			t.Errorf("abgeschnitten (%d Byte): Exit %d, %q", len(kaputt), code, f)
		}
	}
	if testing.Short() {
		return
	}
	a, f, code := programm(t, gut, "-", "-", "-l", "eng")
	if code != 0 || strings.TrimSpace(a) != "The quick brown fox jumps" || f != "" {
		t.Errorf("ohne --progress: Exit %d, stdout %q, stderr %q (erwartet leer)", code, a, f)
	}
	a, f, code = programm(t, gut, "-", "-", "-l", "eng", "--progress")
	if code != 0 || strings.TrimSpace(a) != "The quick brown fox jumps" || !strings.HasSuffix(f, "progress 100\n") ||
		strings.Count(f, "\n") != strings.Count(f, "progress ") {
		t.Errorf("mit --progress: Exit %d, stdout %q, stderr %q", code, a, f)
	}
}

// Ein Bild über der Pixel- oder Seitengrenze wird nach dem Kopf abgelehnt, ohne es zu laden (F-646).
func TestLaufPixelgrenze(t *testing.T) {
	for _, m := range [][2]uint32{{30000, 30000}, {8000, 5001}, {32768, 10}, {10, 32768}} {
		var a, f bytes.Buffer
		if code := lauf([]string{"-", "-", "-l", "eng"}, bytes.NewReader(pngKopf(m[0], m[1])), &a, &f); code != 1 ||
			f.String() != "schnapp-ocr: "+errZuGross.Error()+"\n" {
			t.Errorf("%v: Exit %d, %q", m, code, f.String())
		}
	}
	// An der Grenze geht es weiter (der Kopf allein ist dann nicht lesbar, aber nicht zu groß).
	if err := pruefeBild(pngKopf(8000, 5000)); errors.Is(err, errZuGross) {
		t.Errorf("8000 × 5000: %v", err)
	}
	if err := pruefeBild(pngKopf(32767, 1)); errors.Is(err, errZuGross) {
		t.Errorf("32767 × 1: %v", err)
	}
}

// Die Eingabe ist auf grenze Bytes begrenzt.
func TestLeseBildGrenze(t *testing.T) {
	if b, err := leseBild(strings.NewReader("0123456789"), 10); err != nil || len(b) != 10 {
		t.Errorf("an der Grenze: %d, %v", len(b), err)
	}
	if _, err := leseBild(strings.NewReader("0123456789x"), 10); !errors.Is(err, errZuGrossDatei) {
		t.Errorf("über der Grenze: %v", err)
	}
}

// Die Einstellung trägt --psm und verwirft die Ausgaben von tesseract selbst.
func TestKonfig(t *testing.T) {
	cfg := konfig(auftrag{sprache: "deu", psm: 6}, []byte("x"))
	if cfg.Variables["tessedit_pageseg_mode"] != "6" || cfg.Language != "deu" {
		t.Errorf("Einstellung %+v", cfg)
	}
	if cfg.Stdout != io.Discard || cfg.Stderr != io.Discard {
		t.Error("Ausgaben von tesseract werden nicht verworfen")
	}
}
